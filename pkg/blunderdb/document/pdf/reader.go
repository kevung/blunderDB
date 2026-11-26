package pdf

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
)

var (
	// ErrNotPDF: the data does not start like a PDF file.
	ErrNotPDF = errors.New("pdf: not a PDF file")
	// ErrMalformed: the file breaks the PDF syntax beyond what is repaired.
	ErrMalformed = errors.New("pdf: malformed file")
	// ErrEncrypted: the file is encrypted; its strings and streams are not read.
	ErrEncrypted = errors.New("pdf: encrypted file")
	// ErrTooLarge: a limit of Limits is exceeded.
	ErrTooLarge = errors.New("pdf: file exceeds a size limit")
	// ErrUnsupported: a stream uses a filter this reader does not decode.
	ErrUnsupported = errors.New("pdf: unsupported filter")
)

func malformed(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrMalformed, fmt.Sprintf(format, args...))
}

// Limits bound what a file may make the reader do.
type Limits struct {
	// MaxFileSize is the largest file Open accepts, in bytes.
	MaxFileSize int
	// MaxStreamSize is the largest decoded stream, in bytes.
	MaxStreamSize int
	// MaxDecoded is the sum of every stream the reader decodes, in bytes.
	MaxDecoded int64
	// MaxPages is the largest page count.
	MaxPages int
	// MaxObjects is the largest object number the cross-reference may name.
	MaxObjects int
	// MaxGlyphs is the most glyphs one page may draw.
	MaxGlyphs int
}

// DefaultLimits are generous for a book of a thousand pages and stop a
// decompression bomb or a looping structure early.
var DefaultLimits = Limits{
	MaxFileSize:   512 << 20,
	MaxStreamSize: 64 << 20,
	MaxDecoded:    2 << 30,
	MaxPages:      20000,
	MaxObjects:    4 << 20,
	MaxGlyphs:     1 << 20,
}

type xrefEntry struct {
	inStream bool
	off      int // byte offset, or the object stream's number
	idx      int // index inside the object stream
}

// Reader reads one PDF file held in memory. It is not safe for concurrent use.
type Reader struct {
	data      []byte
	lim       Limits
	xref      map[int]xrefEntry
	scanned   map[int]xrefEntry // objects found by scanning, when xref fails
	trailer   Dict
	cache     map[int]Object
	resolving map[int]bool
	decoded   int64
	fonts     map[int]*Font
	objStm    map[int][]byte
	pages     []Page
}

// Open reads the cross-reference and the trailer of a PDF file. The data is
// kept, not copied: the caller must not change it while the Reader is used.
func Open(data []byte, lim Limits) (r *Reader, err error) {
	defer func() {
		if p := recover(); p != nil {
			r, err = nil, malformed("%v", p)
		}
	}()
	if len(data) > lim.MaxFileSize {
		return nil, fmt.Errorf("%w: %d bytes, at most %d", ErrTooLarge, len(data), lim.MaxFileSize)
	}
	head := data[:min(len(data), 1024)]
	if !bytes.Contains(head, []byte("%PDF-")) {
		return nil, ErrNotPDF
	}
	r = &Reader{data: data, lim: lim, cache: map[int]Object{}, resolving: map[int]bool{}, fonts: map[int]*Font{}, objStm: map[int][]byte{}}
	if err := r.readXref(); err != nil || r.rootDict() == nil {
		r.xref, r.trailer = nil, nil
		r.cache = map[int]Object{}
		if err := r.reconstruct(); err != nil {
			return nil, err
		}
	}
	if r.trailer["Encrypt"] != nil {
		return nil, ErrEncrypted
	}
	if r.rootDict() == nil {
		return nil, malformed("no document catalog")
	}
	if err := r.loadPages(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Reader) rootDict() Dict {
	d, _ := r.Resolve(r.trailer["Root"]).(Dict)
	return d
}

var startxrefRE = regexp.MustCompile(`startxref\s+(\d+)`)

func (r *Reader) readXref() error {
	tail := r.data[max(0, len(r.data)-4096):]
	m := startxrefRE.FindAllSubmatch(tail, -1)
	if len(m) == 0 {
		return malformed("no startxref")
	}
	off, err := strconv.Atoi(string(m[len(m)-1][1]))
	if err != nil {
		return malformed("startxref")
	}
	r.xref = map[int]xrefEntry{}
	seen := map[int]bool{}
	for k := 0; ; k++ {
		if k > 256 || seen[off] || off < 0 || off >= len(r.data) {
			if k == 0 {
				return malformed("startxref %d out of the file", off)
			}
			break
		}
		seen[off] = true
		l := &lexer{b: r.data, pos: off}
		l.skipWhite()
		var tr Dict
		if l.hasPrefix("xref") {
			l.pos += 4
			if tr, err = r.xrefTable(l); err != nil {
				return err
			}
			if s, ok := tr["XRefStm"].(int64); ok && !seen[int(s)] {
				seen[int(s)] = true
				if _, err := r.xrefStream(int(s)); err != nil {
					return err
				}
			}
		} else if tr, err = r.xrefStream(off); err != nil {
			return err
		}
		if r.trailer == nil {
			r.trailer = Dict{}
		}
		for key, v := range tr {
			if _, ok := r.trailer[key]; !ok {
				r.trailer[key] = v
			}
		}
		prev, ok := tr["Prev"].(int64)
		if !ok {
			break
		}
		off = int(prev)
	}
	return nil
}

func (r *Reader) setEntry(num int, e xrefEntry) error {
	if num < 0 || num > r.lim.MaxObjects {
		return fmt.Errorf("%w: object number %d", ErrTooLarge, num)
	}
	if _, ok := r.xref[num]; !ok { // the newest section wins
		r.xref[num] = e
	}
	return nil
}

func (r *Reader) xrefTable(l *lexer) (Dict, error) {
	for {
		t := l.token()
		if k, ok := t.(keyword); ok && k == "trailer" {
			o, err := l.object(true, 0)
			if err != nil {
				return nil, err
			}
			d, ok := o.(Dict)
			if !ok {
				return nil, malformed("trailer is not a dictionary")
			}
			return d, nil
		}
		start, ok1 := t.(int64)
		count, ok2 := l.token().(int64)
		if !ok1 || !ok2 || start < 0 || count < 0 || count > int64(r.lim.MaxObjects) {
			return nil, malformed("cross-reference subsection")
		}
		for i := int64(0); i < count; i++ {
			off, ok1 := l.token().(int64)
			_, ok2 := l.token().(int64)
			kind, ok3 := l.token().(keyword)
			if !ok1 || !ok2 || !ok3 {
				return nil, malformed("cross-reference entry")
			}
			if kind == "n" && off > 0 {
				if err := r.setEntry(int(start+i), xrefEntry{off: int(off)}); err != nil {
					return nil, err
				}
			} else if kind == "f" {
				// A free entry still hides an older section's object.
				if _, ok := r.xref[int(start+i)]; !ok && start+i <= int64(r.lim.MaxObjects) {
					r.xref[int(start+i)] = xrefEntry{off: -1}
				}
			}
		}
	}
}

func (r *Reader) xrefStream(off int) (Dict, error) {
	_, o, err := r.parseAt(off)
	if err != nil {
		return nil, err
	}
	s, ok := o.(*Stream)
	if !ok || s.Dict["Type"] != Name("XRef") {
		return nil, malformed("startxref points at neither a table nor a stream")
	}
	data, err := r.Decode(s)
	if err != nil {
		return nil, err
	}
	w, _ := s.Dict["W"].(Array)
	if len(w) != 3 {
		return nil, malformed("cross-reference stream /W")
	}
	var ws [3]int
	for i := range ws {
		n, _ := w[i].(int64)
		if n < 0 || n > 8 {
			return nil, malformed("cross-reference stream /W")
		}
		ws[i] = int(n)
	}
	row := ws[0] + ws[1] + ws[2]
	if row == 0 {
		return nil, malformed("cross-reference stream /W")
	}
	size, _ := s.Dict["Size"].(int64)
	index := Array{int64(0), size}
	if a, ok := s.Dict["Index"].(Array); ok {
		index = a
	}
	field := func(b []byte, def int64) int64 {
		if len(b) == 0 {
			return def
		}
		var v int64
		for _, c := range b {
			v = v<<8 | int64(c)
		}
		return v
	}
	p := 0
	for i := 0; i+1 < len(index); i += 2 {
		start, _ := index[i].(int64)
		count, _ := index[i+1].(int64)
		if start < 0 || count < 0 || count > int64(r.lim.MaxObjects) {
			return nil, malformed("cross-reference stream /Index")
		}
		for k := int64(0); k < count; k++ {
			if p+row > len(data) {
				return s.Dict, nil
			}
			b := data[p : p+row]
			p += row
			typ := field(b[:ws[0]], 1)
			f2 := field(b[ws[0]:ws[0]+ws[1]], 0)
			f3 := field(b[ws[0]+ws[1]:], 0)
			num := int(start + k)
			var err error
			switch typ {
			case 1:
				err = r.setEntry(num, xrefEntry{off: int(f2)})
			case 2:
				err = r.setEntry(num, xrefEntry{inStream: true, off: int(f2), idx: int(f3)})
			case 0:
				err = r.setEntry(num, xrefEntry{off: -1})
			}
			if err != nil {
				return nil, err
			}
		}
	}
	return s.Dict, nil
}

var objRE = regexp.MustCompile(`(\d+)[\x00\t\n\f\r ]+(\d+)[\x00\t\n\f\r ]+obj\b`)

// reconstruct rebuilds the cross-reference by scanning the file for "n g obj",
// the repair every viewer does for a file whose offsets are wrong.
func (r *Reader) reconstruct() error {
	r.xref = map[int]xrefEntry{}
	r.scanned = nil
	for _, m := range objRE.FindAllSubmatchIndex(r.data, -1) {
		num, err := strconv.Atoi(string(r.data[m[2]:m[3]]))
		if err != nil || num > r.lim.MaxObjects {
			continue
		}
		r.xref[num] = xrefEntry{off: m[0]} // the last definition wins
	}
	if len(r.xref) == 0 {
		return malformed("no object found")
	}
	// Objects inside object streams, and a trailer from the last trailer
	// dictionary, the last cross-reference stream, or the catalog.
	var trailer Dict
	for num := range r.xref {
		o := r.object(num)
		s, ok := o.(*Stream)
		if !ok {
			if d, ok := o.(Dict); ok && d["Type"] == Name("Catalog") && trailer == nil {
				trailer = Dict{"Root": Ref{Num: num}}
			}
			continue
		}
		switch s.Dict["Type"] {
		case Name("ObjStm"):
			nums, _, err := r.objStreamHeader(s)
			if err != nil {
				continue
			}
			for i, n := range nums {
				if _, ok := r.xref[n]; !ok && n <= r.lim.MaxObjects {
					r.xref[n] = xrefEntry{inStream: true, off: num, idx: i}
				}
			}
		case Name("XRef"):
			if _, ok := s.Dict["Root"]; ok {
				trailer = s.Dict
			}
		}
	}
	if i := bytes.LastIndex(r.data, []byte("trailer")); i >= 0 {
		l := &lexer{b: r.data, pos: i + len("trailer")}
		if o, err := l.object(true, 0); err == nil {
			if d, ok := o.(Dict); ok && d["Root"] != nil {
				trailer = d
			}
		}
	}
	if trailer == nil {
		return malformed("no trailer and no catalog")
	}
	r.trailer = trailer
	return nil
}

// parseAt reads the indirect object "n g obj … endobj" at off.
func (r *Reader) parseAt(off int) (int, Object, error) {
	if off < 0 || off >= len(r.data) {
		return 0, nil, malformed("offset %d out of the file", off)
	}
	l := &lexer{b: r.data, pos: off}
	num, ok1 := l.token().(int64)
	_, ok2 := l.token().(int64)
	k, ok3 := l.token().(keyword)
	if !ok1 || !ok2 || !ok3 || k != "obj" {
		return 0, nil, malformed("no object at offset %d", off)
	}
	o, err := l.object(true, 0)
	if err != nil {
		return 0, nil, err
	}
	d, isDict := o.(Dict)
	if !isDict {
		if _, ok := o.(keyword); ok {
			o = nil
		}
		return int(num), o, nil
	}
	save := l.pos
	if k, ok := l.token().(keyword); !ok || k != "stream" {
		l.pos = save
		return int(num), d, nil
	}
	// The data starts after the end of line that follows "stream".
	if l.pos < len(r.data) && r.data[l.pos] == '\r' {
		l.pos++
	}
	if l.pos < len(r.data) && r.data[l.pos] == '\n' {
		l.pos++
	}
	start := l.pos
	end := -1
	if n, ok := r.lengthOf(d["Length"]); ok && n >= 0 && start+n <= len(r.data) {
		rest := r.data[start+n:]
		i := 0
		for i < len(rest) && i < 32 && isWhite(rest[i]) {
			i++
		}
		if bytes.HasPrefix(rest[i:], []byte("endstream")) {
			end = start + n
		}
	}
	if end < 0 { // a wrong /Length: the data runs to "endstream"
		i := bytes.Index(r.data[start:], []byte("endstream"))
		if i < 0 {
			end = len(r.data)
		} else {
			end = start + i
			for end > start && (r.data[end-1] == '\n' || r.data[end-1] == '\r') {
				end--
			}
		}
	}
	return int(num), &Stream{Dict: d, Raw: r.data[start:end]}, nil
}

func (r *Reader) lengthOf(o Object) (int, bool) {
	if ref, ok := o.(Ref); ok {
		if r.resolving[ref.Num] {
			return 0, false
		}
		o = r.Resolve(ref)
	}
	n, ok := o.(int64)
	if !ok || n > int64(len(r.data)) {
		return 0, false
	}
	return int(n), true
}

// Resolve follows a reference to its object; any other object is returned
// as is. A missing or unreadable object is nil, as the format prescribes.
func (r *Reader) Resolve(o Object) Object {
	for range 32 {
		ref, ok := o.(Ref)
		if !ok {
			return o
		}
		o = r.object(ref.Num)
	}
	return nil
}

func (r *Reader) object(num int) Object {
	if o, ok := r.cache[num]; ok {
		return o
	}
	if r.resolving[num] || len(r.resolving) > maxNesting {
		return nil
	}
	r.resolving[num] = true
	defer delete(r.resolving, num)
	e, ok := r.xref[num]
	var o Object
	switch {
	case !ok || e.off < 0:
	case e.inStream:
		o = r.fromObjStream(e.off, e.idx, num)
	default:
		n, obj, err := r.parseAt(e.off)
		if err != nil || n != num {
			obj = r.scanFor(num)
		}
		o = obj
	}
	r.cache[num] = o
	return o
}

// scanFor finds object num by scanning, when the cross-reference points
// elsewhere.
func (r *Reader) scanFor(num int) Object {
	if r.scanned == nil {
		r.scanned = map[int]xrefEntry{}
		for _, m := range objRE.FindAllSubmatchIndex(r.data, -1) {
			n, err := strconv.Atoi(string(r.data[m[2]:m[3]]))
			if err == nil && n <= r.lim.MaxObjects {
				r.scanned[n] = xrefEntry{off: m[0]}
			}
		}
	}
	e, ok := r.scanned[num]
	if !ok {
		return nil
	}
	n, o, err := r.parseAt(e.off)
	if err != nil || n != num {
		return nil
	}
	return o
}

func (r *Reader) objStreamHeader(s *Stream) ([]int, []byte, error) {
	data, err := r.Decode(s)
	if err != nil {
		return nil, nil, err
	}
	n, _ := s.Dict["N"].(int64)
	first, _ := s.Dict["First"].(int64)
	if n < 0 || n > int64(r.lim.MaxObjects) || first < 0 || first > int64(len(data)) {
		return nil, nil, malformed("object stream header")
	}
	l := &lexer{b: data[:first]}
	nums := make([]int, 0, min(n, 1<<16))
	for range n {
		num, ok1 := l.token().(int64)
		off, ok2 := l.token().(int64)
		if !ok1 || !ok2 || off < 0 || first+off > int64(len(data)) {
			return nil, nil, malformed("object stream header")
		}
		nums = append(nums, int(num))
	}
	return nums, data, nil
}

func (r *Reader) fromObjStream(snum, idx, num int) Object {
	s, ok := r.object(snum).(*Stream)
	if !ok {
		return nil
	}
	data, ok := r.objStm[snum]
	if !ok {
		var err error
		if data, err = r.Decode(s); err != nil {
			data = nil
		}
		r.objStm[snum] = data
	}
	n, _ := s.Dict["N"].(int64)
	first, _ := s.Dict["First"].(int64)
	if idx < 0 || int64(idx) >= n || first < 0 || first > int64(len(data)) {
		return nil
	}
	l := &lexer{b: data[:first]}
	for i := 0; ; i++ {
		on, ok1 := l.token().(int64)
		off, ok2 := l.token().(int64)
		if !ok1 || !ok2 {
			return nil
		}
		if i == idx {
			if int(on) != num || off < 0 || first+off >= int64(len(data)) {
				return nil
			}
			ol := &lexer{b: data, pos: int(first + off)}
			o, err := ol.object(true, 0)
			if err != nil {
				return nil
			}
			if _, ok := o.(keyword); ok {
				return nil
			}
			return o
		}
	}
}

// Decode returns a stream's data with its filters undone. An image codec
// (DCT, CCITT fax, JBIG2, JPEG 2000) is ErrUnsupported here: only Scan
// decodes images.
func (r *Reader) Decode(s *Stream) (out []byte, err error) {
	defer func() {
		if p := recover(); p != nil {
			out, err = nil, malformed("%v", p)
		}
	}()
	data, codec, _, err := r.imageFilters(s)
	if err != nil {
		return nil, err
	}
	if codec != "" {
		return nil, fmt.Errorf("%w: %s", ErrUnsupported, codec)
	}
	r.decoded += int64(len(data))
	if r.decoded > r.lim.MaxDecoded {
		return nil, fmt.Errorf("%w: the file decodes to more than %d bytes", ErrTooLarge, r.lim.MaxDecoded)
	}
	return data, nil
}

func (r *Reader) inflate(in []byte) ([]byte, error) {
	var rd io.ReadCloser
	zr, err := zlib.NewReader(bytes.NewReader(in))
	if err == nil {
		rd = zr
	} else {
		rd = flate.NewReader(bytes.NewReader(in)) // a raw deflate stream, without its header
	}
	defer func() { _ = rd.Close() }()
	out, err := io.ReadAll(io.LimitReader(rd, int64(r.lim.MaxStreamSize)+1))
	if len(out) > r.lim.MaxStreamSize {
		return nil, fmt.Errorf("%w: a stream decodes to more than %d bytes", ErrTooLarge, r.lim.MaxStreamSize)
	}
	if err != nil && len(out) == 0 {
		return nil, malformed("flate: %v", err)
	}
	// A truncated or checksum-broken stream keeps what was inflated, as
	// viewers do.
	return out, nil
}

func predict(data []byte, p Dict) ([]byte, error) {
	pred, _ := p["Predictor"].(int64)
	if pred <= 1 {
		return data, nil
	}
	colors, bpc, cols := int64(1), int64(8), int64(1)
	if v, ok := p["Colors"].(int64); ok {
		colors = v
	}
	if v, ok := p["BitsPerComponent"].(int64); ok {
		bpc = v
	}
	if v, ok := p["Columns"].(int64); ok {
		cols = v
	}
	if colors < 1 || colors > 32 || bpc < 1 || bpc > 16 || cols < 1 || cols > 1<<20 {
		return nil, malformed("predictor parameters")
	}
	bpp := int((colors*bpc + 7) / 8)
	rowLen := int((colors*bpc*cols + 7) / 8)
	if pred == 2 {
		if bpc != 8 {
			return nil, fmt.Errorf("%w: TIFF predictor with %d bits", ErrUnsupported, bpc)
		}
		for r := 0; r+rowLen <= len(data); r += rowLen {
			for i := bpp; i < rowLen; i++ {
				data[r+i] += data[r+i-bpp]
			}
		}
		return data, nil
	}
	out := make([]byte, 0, len(data))
	prev := make([]byte, rowLen)
	for pos := 0; pos+1+rowLen <= len(data); pos += 1 + rowLen {
		ft := data[pos]
		row := append([]byte(nil), data[pos+1:pos+1+rowLen]...)
		for i := range row {
			var a, b, c byte
			if i >= bpp {
				a = row[i-bpp]
				c = prev[i-bpp]
			}
			b = prev[i]
			switch ft {
			case 1:
				row[i] += a
			case 2:
				row[i] += b
			case 3:
				row[i] += byte((int(a) + int(b)) / 2)
			case 4:
				row[i] += paeth(a, b, c)
			}
		}
		out = append(out, row...)
		prev = row
	}
	return out, nil
}

func paeth(a, b, c byte) byte {
	p := int(a) + int(b) - int(c)
	pa, pb, pc := abs(p-int(a)), abs(p-int(b)), abs(p-int(c))
	switch {
	case pa <= pb && pa <= pc:
		return a
	case pb <= pc:
		return b
	}
	return c
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func asciiHex(in []byte) []byte {
	l := &lexer{b: append(append([]byte{'<'}, in...), '>')}
	s, _ := l.hexString().(String)
	return []byte(s)
}

func ascii85(in []byte) []byte {
	var out []byte
	var group [5]byte
	n := 0
	for _, c := range in {
		switch {
		case c == '~':
			goto done
		case c == 'z' && n == 0:
			out = append(out, 0, 0, 0, 0)
		case c >= '!' && c <= 'u':
			group[n] = c - '!'
			n++
			if n == 5 {
				var v uint32
				for _, g := range group {
					v = v*85 + uint32(g)
				}
				out = append(out, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
				n = 0
			}
		}
	}
done:
	if n > 1 {
		for i := n; i < 5; i++ {
			group[i] = 84
		}
		var v uint32
		for _, g := range group {
			v = v*85 + uint32(g)
		}
		b := []byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}
		out = append(out, b[:n-1]...)
	}
	return out
}
