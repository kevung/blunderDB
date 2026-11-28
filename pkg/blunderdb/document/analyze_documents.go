package document

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	_ "image/gif" // pictures a word processor embeds as GIF
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	_ "golang.org/x/image/bmp"  // pictures a word processor embeds as BMP
	_ "golang.org/x/image/tiff" // and as TIFF
	"golang.org/x/net/html"

	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf"
	"github.com/kevung/blunderdb/pkg/blunderdb/document/structure"
)

// A written document — DOCX, ODT, EPUB or HTML — holds its diagrams as
// pictures placed in its text. Each picture, in the order of the text, is a
// page of the structure reader, the text around it laid out beside it as
// that page's text layer, so the printed pips and the dice it cites cross-
// check the reading as a PDF's text layer does. An entry's page is then the
// picture's rank among those read, its chapter the heading it follows.

// Bounds of a written document, before and while it is unpacked. The file
// itself is bounded as a PDF is; its zip archive by its number of entries,
// by the size of each part and by the bytes it unpacks in all, so a small
// file that would unpack without end is refused.
const (
	MaxDocumentEntries = 10_000
	MaxDocumentPart    = 64 << 20
	MaxDocumentUnpack  = 1 << 30
)

var (
	// ErrLegacyWord refuses a Word 97-2003 binary document: its pictures
	// are not read without an outside tool.
	ErrLegacyWord = errors.New("document: a Word 97-2003 .doc file is not read; save it as PDF or DOCX and import that file")
	// ErrDocumentTooLarge refuses a written document past its bounds.
	ErrDocumentTooLarge = errors.New("document: the document is too large to read")
	// ErrNotWrittenDocument refuses a zip archive that is no DOCX, ODT or
	// EPUB.
	ErrNotWrittenDocument = errors.New("document: the archive is not a DOCX, ODT or EPUB document")
)

// IsWrittenDocument reports whether a file name is a written document the
// reader takes (and the Word 97-2003 one it refuses with a reason).
func IsWrittenDocument(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".docx", ".odt", ".epub", ".html", ".htm", ".xhtml", ".doc":
		return true
	}
	return false
}

// picture is one picture of a written document, in the order of its text:
// how to read its bytes again, and the text around it.
type picture struct {
	name                  string
	read                  func() ([]byte, error)
	before, inside, after string
	section               string
}

// written is a written document unpacked: its metadata and its pictures.
type written struct {
	title, author string
	pictures      []picture
	// remote counts the pictures not read because they lie on the network.
	remote int
}

// AnalyzeWrittenFile reads the diagrams of a DOCX, ODT, EPUB or HTML file.
func AnalyzeWrittenFile(path string, opts AnalyzeOptions) (*Releve, error) {
	data, err := readDocumentFile(path)
	if err != nil {
		return nil, err
	}
	if opts.FileName == "" {
		opts.FileName = path
	}
	w, err := unpackWritten(path, data)
	if err != nil {
		return nil, err
	}
	if w.remote > 0 && opts.Warn != nil {
		opts.Warn(fmt.Sprintf("%d pictures on the network not fetched", w.remote))
	}
	if strings.TrimSpace(opts.Title) == "" {
		opts.Title = w.title
	}
	if strings.TrimSpace(opts.Author) == "" {
		opts.Author = w.author
	}
	if len(w.pictures) == 0 {
		return nil, fmt.Errorf("document: no picture in the document: %w", structure.ErrNoDiagram)
	}
	if len(w.pictures) > structure.MaxPages {
		return nil, fmt.Errorf("document: %d pictures, at most %d", len(w.pictures), structure.MaxPages)
	}
	src := &writtenPages{pics: w.pictures, unread: map[int]error{}}
	r, err := releveOfPages(src, len(w.pictures), sha256Hex(data), opts, func(page int) string {
		return w.pictures[page-1].section
	}, "pictures")
	if errors.Is(err, structure.ErrNoDiagram) {
		// The pictures passed over are why none was found: they are named.
		var us []structure.PageError
		for i := range w.pictures {
			if e := src.unread[i]; e != nil {
				us = append(us, structure.PageError{Page: i, Err: e})
			}
		}
		if s := unreadNamed(us, "pictures"); s != "" {
			return nil, fmt.Errorf("%w (%s)", err, s)
		}
	}
	if r != nil {
		r.Document.Pictures = true
	}
	return r, err
}

func readDocumentFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	limit := pdf.DefaultLimits.MaxFileSize
	data, err := io.ReadAll(io.LimitReader(f, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		return nil, fmt.Errorf("%w: %s is larger than %d bytes", ErrDocumentTooLarge, path, limit)
	}
	return data, nil
}

// unpackWritten tells the document's kind by its content, not its name: a
// .doc saved as DOCX or HTML is read, a binary one refused.
func unpackWritten(path string, data []byte) (*written, error) {
	switch {
	case bytes.HasPrefix(data, []byte("\xD0\xCF\x11\xE0\xA1\xB1\x1A\xE1")), bytes.HasPrefix(data, []byte(`{\rtf`)):
		return nil, ErrLegacyWord
	case bytes.HasPrefix(data, []byte("PK\x03\x04")):
		return unpackZip(data)
	}
	return unpackHTML(path, data)
}

// budget is what a document may still unpack, in bytes.
type budget struct{ left int64 }

// readPart reads one zip entry, bounded by MaxDocumentPart and by the
// document's budget whatever its header claims.
func (b *budget) readPart(f *zip.File) ([]byte, error) {
	if f.UncompressedSize64 > MaxDocumentPart {
		return nil, fmt.Errorf("%w: %s unpacks to %d bytes", ErrDocumentTooLarge, f.Name, f.UncompressedSize64)
	}
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("document: %s: %w", f.Name, err)
	}
	defer func() { _ = rc.Close() }()
	data, err := io.ReadAll(io.LimitReader(rc, MaxDocumentPart+1))
	if err != nil {
		return nil, fmt.Errorf("document: %s: %w", f.Name, err)
	}
	if len(data) > MaxDocumentPart {
		return nil, fmt.Errorf("%w: %s unpacks to more than %d bytes", ErrDocumentTooLarge, f.Name, MaxDocumentPart)
	}
	b.left -= int64(len(data))
	if b.left < 0 {
		return nil, fmt.Errorf("%w: it unpacks to more than %d bytes", ErrDocumentTooLarge, MaxDocumentUnpack)
	}
	return data, nil
}

func unpackZip(data []byte) (*written, error) {
	// archive/zip lists every central directory record before anything is
	// bounded; each one starts with this signature, so their count bounds
	// the list.
	if n := bytes.Count(data, []byte("PK\x01\x02")); n > MaxDocumentEntries {
		return nil, fmt.Errorf("%w: more than %d entries in its archive", ErrDocumentTooLarge, MaxDocumentEntries)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("document: %w", err)
	}
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}
	z := &zipDoc{files: files, budget: &budget{left: MaxDocumentUnpack}}
	mime := ""
	if f := files["mimetype"]; f != nil {
		b, err := z.budget.readPart(f)
		if err != nil {
			return nil, err
		}
		mime = strings.TrimSpace(string(b))
	}
	switch {
	case mime == "application/epub+zip":
		return z.epub()
	case strings.HasPrefix(mime, "application/vnd.oasis.opendocument"):
		return z.odt()
	case files["word/document.xml"] != nil:
		return z.docx()
	}
	return nil, ErrNotWrittenDocument
}

type zipDoc struct {
	files  map[string]*zip.File
	budget *budget
}

func (z *zipDoc) part(name string) ([]byte, error) {
	f := z.files[name]
	if f == nil {
		return nil, fmt.Errorf("document: %s missing from the archive", name)
	}
	return z.budget.readPart(f)
}

// picture is the entry name as a picture, its bytes read again when the
// reader asks for its page, bounded by what was measured the first time.
func (z *zipDoc) picture(name string) (picture, bool, error) {
	f := z.files[name]
	if f == nil {
		return picture{}, false, nil
	}
	data, err := z.budget.readPart(f)
	if err != nil {
		return picture{}, false, err
	}
	size := len(data)
	return picture{name: name, read: func() ([]byte, error) {
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer func() { _ = rc.Close() }()
		return io.ReadAll(io.LimitReader(rc, int64(size)))
	}}, true, nil
}

// block is a paragraph of a document's text, with the pictures it holds.
type block struct {
	text     strings.Builder
	heading  bool
	pictures []string
}

// picturesOf places each picture of the blocks between the text before and
// after it, under the last heading.
func picturesOf(blocks []*block, get func(ref string) (picture, bool, error)) ([]picture, error) {
	var out []picture
	section := ""
	text := func(i int) string {
		if i < 0 || i >= len(blocks) {
			return ""
		}
		return cleanText(blocks[i].text.String())
	}
	for i, b := range blocks {
		if b.heading && text(i) != "" {
			section = text(i)
		}
		for _, ref := range b.pictures {
			p, ok, err := get(ref)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			p.before, p.inside, p.after, p.section = text(i-1), text(i), text(i+1), section
			out = append(out, p)
			if len(out) > structure.MaxPages {
				return nil, fmt.Errorf("document: more than %d pictures", structure.MaxPages)
			}
		}
	}
	return out, nil
}

func cleanText(s string) string { return strings.Join(strings.Fields(s), " ") }

// xmlTokens walks an XML part, its elements by local name.
func xmlTokens(data []byte, start func(xml.StartElement), end func(xml.EndElement), text func(string)) error {
	d := xml.NewDecoder(bytes.NewReader(data))
	d.Strict = false
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("document: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			start(t)
		case xml.EndElement:
			end(t)
		case xml.CharData:
			text(string(t))
		}
	}
}

func attr(e xml.StartElement, local string) string {
	for _, a := range e.Attr {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}

// metaOf reads the Dublin Core title and creator of an XML part.
func metaOf(data []byte) (title, author string) {
	var in string
	_ = xmlTokens(data, func(e xml.StartElement) { in = e.Name.Local }, func(xml.EndElement) { in = "" }, func(s string) {
		switch {
		case in == "title" && title == "":
			title = cleanText(s)
		case in == "creator" && author == "":
			author = cleanText(s)
		}
	})
	return title, author
}

// reHeadingStyle names a heading's paragraph style in the languages a word
// processor names it in.
var reHeadingStyle = regexp.MustCompile(`(?i)^(heading|titre|title|berschrift|kop|titolo|encabezado|t[ií]tulo|nagł[oó]wek)`)

func (z *zipDoc) docx() (*written, error) {
	doc, err := z.part("word/document.xml")
	if err != nil {
		return nil, err
	}
	rels := map[string]string{}
	if data, err := z.part("word/_rels/document.xml.rels"); err == nil {
		_ = xmlTokens(data, func(e xml.StartElement) {
			if e.Name.Local == "Relationship" && !strings.EqualFold(attr(e, "TargetMode"), "External") {
				rels[attr(e, "Id")] = path.Clean(path.Join("word", attr(e, "Target")))
			}
		}, func(xml.EndElement) {}, func(string) {})
	}
	w := &written{}
	if data, err := z.part("docProps/core.xml"); err == nil {
		w.title, w.author = metaOf(data)
	}
	var blocks []*block
	cur := &block{}
	flush := func() {
		blocks = append(blocks, cur)
		cur = &block{}
	}
	inText, fallback := false, 0
	err = xmlTokens(doc, func(e xml.StartElement) {
		if e.Name.Local == "Fallback" || fallback > 0 {
			// mc:Fallback repeats mc:Choice's picture for older readers.
			fallback++
			return
		}
		switch e.Name.Local {
		case "p":
			flush()
		case "pStyle":
			if reHeadingStyle.MatchString(attr(e, "val")) {
				cur.heading = true
			}
		case "outlineLvl":
			cur.heading = true
		case "t":
			inText = true
		case "tab", "br", "cr":
			cur.text.WriteByte(' ')
		case "blip":
			if id := attr(e, "embed"); rels[id] != "" {
				cur.pictures = append(cur.pictures, rels[id])
			}
		case "imagedata":
			if id := attr(e, "id"); rels[id] != "" {
				cur.pictures = append(cur.pictures, rels[id])
			}
		}
	}, func(e xml.EndElement) {
		if fallback > 0 {
			fallback--
			return
		}
		switch e.Name.Local {
		case "t":
			inText = false
		case "p":
			flush()
		}
	}, func(s string) {
		if inText && fallback == 0 {
			cur.text.WriteString(s)
		}
	})
	if err != nil {
		return nil, err
	}
	flush()
	w.pictures, err = picturesOf(nonEmpty(blocks), z.picture)
	return w, err
}

// nonEmpty drops the blocks with neither text nor picture, so the text
// beside a picture is the nearest paragraph that says something.
func nonEmpty(bs []*block) []*block {
	var out []*block
	for _, b := range bs {
		if strings.TrimSpace(b.text.String()) != "" || len(b.pictures) > 0 {
			out = append(out, b)
		}
	}
	return out
}

func (z *zipDoc) odt() (*written, error) {
	doc, err := z.part("content.xml")
	if err != nil {
		return nil, err
	}
	w := &written{}
	if data, err := z.part("meta.xml"); err == nil {
		w.title, w.author = metaOf(data)
	}
	var blocks []*block
	cur := &block{}
	var stack []*block
	var frame []string
	inFrame := 0
	err = xmlTokens(doc, func(e xml.StartElement) {
		switch e.Name.Local {
		case "p", "h":
			// A paragraph inside another (a frame's caption) is a block of
			// its own; the outer one resumes after it.
			stack = append(stack, cur)
			cur = &block{heading: e.Name.Local == "h"}
		case "s":
			cur.text.WriteByte(' ')
		case "tab", "line-break":
			cur.text.WriteByte(' ')
		case "frame":
			inFrame++
		case "image":
			if inFrame > 0 {
				frame = append(frame, attr(e, "href"))
			}
		}
	}, func(e xml.EndElement) {
		switch e.Name.Local {
		case "p", "h":
			blocks = append(blocks, cur)
			if n := len(stack); n > 0 {
				cur, stack = stack[n-1], stack[:n-1]
			} else {
				cur = &block{}
			}
		case "frame":
			inFrame--
			// A frame may hold one picture under several formats (an SVG
			// and its bitmap replacement): the bitmap is read.
			if p := preferredPicture(frame); p != "" && !strings.Contains(p, "://") {
				cur.pictures = append(cur.pictures, path.Clean(strings.TrimPrefix(p, "./")))
			}
			if inFrame <= 0 {
				inFrame, frame = 0, nil
			}
		}
	}, func(s string) {
		cur.text.WriteString(s)
	})
	if err != nil {
		return nil, err
	}
	blocks = append(blocks, cur)
	w.pictures, err = picturesOf(nonEmpty(blocks), z.picture)
	return w, err
}

// preferredPicture is the first bitmap of a frame's pictures, else its
// first picture.
func preferredPicture(refs []string) string {
	for _, r := range refs {
		switch strings.ToLower(path.Ext(r)) {
		case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".tif", ".tiff":
			return r
		}
	}
	if len(refs) > 0 {
		return refs[0]
	}
	return ""
}

func (z *zipDoc) epub() (*written, error) {
	container, err := z.part("META-INF/container.xml")
	if err != nil {
		return nil, err
	}
	opfPath := ""
	_ = xmlTokens(container, func(e xml.StartElement) {
		if e.Name.Local == "rootfile" && opfPath == "" {
			opfPath = attr(e, "full-path")
		}
	}, func(xml.EndElement) {}, func(string) {})
	opf, err := z.part(opfPath)
	if err != nil {
		return nil, err
	}
	w := &written{}
	w.title, w.author = metaOf(opf)
	base := path.Dir(opfPath)
	items := map[string]string{}
	var spine []string
	_ = xmlTokens(opf, func(e xml.StartElement) {
		switch e.Name.Local {
		case "item":
			items[attr(e, "id")] = resolveRef(base, attr(e, "href"))
		case "itemref":
			spine = append(spine, attr(e, "idref"))
		}
	}, func(xml.EndElement) {}, func(string) {})
	var blocks []*block
	for _, id := range spine {
		name := items[id]
		if name == "" {
			continue
		}
		data, err := z.part(name)
		if err != nil {
			return nil, err
		}
		bs, remote := htmlBlocks(data, func(src string) string {
			if hasScheme(src) {
				return ""
			}
			return resolveRef(path.Dir(name), src)
		}, nil)
		w.remote += remote
		blocks = append(blocks, bs...)
	}
	w.pictures, err = picturesOf(nonEmpty(blocks), z.picture)
	return w, err
}

// resolveRef is a reference inside a document's archive, relative to dir,
// without its fragment or query, decoded.
func resolveRef(dir, ref string) string {
	if i := strings.IndexAny(ref, "#?"); i >= 0 {
		ref = ref[:i]
	}
	if u, err := url.PathUnescape(ref); err == nil {
		ref = u
	}
	if ref == "" {
		return ""
	}
	return strings.TrimPrefix(path.Clean(path.Join(dir, ref)), "/")
}

var reScheme = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)

func hasScheme(ref string) bool {
	return strings.HasPrefix(ref, "//") || (reScheme.MatchString(ref) && !(len(ref) > 1 && ref[1] == ':' && unicode.IsLetter(rune(ref[0]))))
}

// htmlBlock names the elements that open a paragraph of their own.
var htmlBlock = map[string]bool{
	"p": true, "div": true, "li": true, "dt": true, "dd": true, "td": true, "th": true, "tr": true,
	"figure": true, "figcaption": true, "caption": true, "blockquote": true, "pre": true,
	"section": true, "article": true, "table": true, "ul": true, "ol": true, "body": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
}

// htmlBlocks cuts an HTML or XHTML page into paragraphs, each picture (an
// img, or an SVG image) named by ref, "" when it is on the network. title,
// when set, receives the page's title.
func htmlBlocks(data []byte, ref func(src string) string, title *string) ([]*block, int) {
	root, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, 0
	}
	var blocks []*block
	cur := &block{}
	remote := 0
	flush := func() {
		blocks = append(blocks, cur)
		cur = &block{}
	}
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		switch n.Type {
		case html.TextNode:
			cur.text.WriteString(n.Data)
			return
		case html.ElementNode:
			switch n.Data {
			case "script", "style", "noscript", "template":
				return
			case "title":
				if title != nil && n.FirstChild != nil && *title == "" {
					*title = cleanText(n.FirstChild.Data)
				}
				return
			case "br":
				cur.text.WriteByte(' ')
			case "img", "image":
				src := ""
				for _, a := range n.Attr {
					if (n.Data == "img" && a.Key == "src") || (n.Data == "image" && (a.Key == "href" || strings.HasSuffix(a.Key, "href"))) {
						src = strings.TrimSpace(a.Val)
					}
				}
				if src != "" {
					if r := ref(src); r != "" {
						cur.pictures = append(cur.pictures, r)
					} else {
						remote++
					}
				}
			}
		}
		block := n.Type == html.ElementNode && htmlBlock[n.Data]
		if block {
			flush()
			if len(n.Data) == 2 && n.Data[0] == 'h' && n.Data[1] >= '1' && n.Data[1] <= '6' {
				cur.heading = true
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if block {
			flush()
		}
	}
	walk(root)
	flush()
	return blocks, remote
}

// unpackHTML reads an HTML page and the pictures it references: a data: URL,
// or a file in the page's folder or below it. Nothing is fetched from the
// network, and no file outside that folder is opened.
func unpackHTML(file string, data []byte) (*written, error) {
	if len(data) > MaxDocumentPart {
		return nil, fmt.Errorf("%w: the page is larger than %d bytes", ErrDocumentTooLarge, MaxDocumentPart)
	}
	dir, err := filepath.Abs(filepath.Dir(file))
	if err != nil {
		return nil, err
	}
	w := &written{}
	b := &budget{left: MaxDocumentUnpack}
	var inline [][]byte
	blocks, remote := htmlBlocks(data, func(src string) string {
		if strings.HasPrefix(strings.ToLower(src), "data:") {
			if d, ok := dataURL(src); ok {
				inline = append(inline, d)
				return fmt.Sprintf("\x00%d", len(inline)-1)
			}
			return ""
		}
		if strings.HasPrefix(strings.ToLower(src), "file://") {
			if u, err := url.Parse(src); err == nil {
				src = u.Path
			}
		} else if hasScheme(src) {
			return ""
		}
		if i := strings.IndexAny(src, "#?"); i >= 0 {
			src = src[:i]
		}
		if u, err := url.PathUnescape(src); err == nil {
			src = u
		}
		p := filepath.FromSlash(src)
		if !filepath.IsAbs(p) {
			p = filepath.Join(dir, p)
		}
		return filepath.Clean(p)
	}, &w.title)
	w.remote = remote
	w.pictures, err = picturesOf(nonEmpty(blocks), func(ref string) (picture, bool, error) {
		if strings.HasPrefix(ref, "\x00") {
			var k int
			_, _ = fmt.Sscanf(ref[1:], "%d", &k)
			d := inline[k]
			return picture{name: fmt.Sprintf("data URL %d", k+1), read: func() ([]byte, error) { return d, nil }}, true, nil
		}
		if rel, err := filepath.Rel(dir, ref); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return picture{}, false, nil
		}
		st, err := os.Stat(ref)
		if err != nil || !st.Mode().IsRegular() {
			return picture{}, false, nil
		}
		if st.Size() > MaxDocumentPart {
			return picture{}, false, fmt.Errorf("%w: %s is larger than %d bytes", ErrDocumentTooLarge, ref, MaxDocumentPart)
		}
		if b.left -= st.Size(); b.left < 0 {
			return picture{}, false, fmt.Errorf("%w: its pictures are larger than %d bytes", ErrDocumentTooLarge, MaxDocumentUnpack)
		}
		size := st.Size()
		return picture{name: ref, read: func() ([]byte, error) {
			f, err := os.Open(ref)
			if err != nil {
				return nil, err
			}
			defer func() { _ = f.Close() }()
			return io.ReadAll(io.LimitReader(f, size))
		}}, true, nil
	})
	return w, err
}

// dataURL decodes a base64 data: URL.
func dataURL(s string) ([]byte, bool) {
	i := strings.IndexByte(s, ',')
	if i < 0 || !strings.HasSuffix(strings.ToLower(s[:i]), ";base64") {
		return nil, false
	}
	d, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(s[i+1:]), ""))
	if err != nil {
		return nil, false
	}
	return d, true
}

// writtenPages are a written document's pictures as the structure reader's
// pages, each decoded when asked for and dropped once read.
type writtenPages struct {
	pics   []picture
	unread map[int]error
}

func (w *writtenPages) Len() int { return len(w.pics) }

// minPicture is the side under which a picture cannot hold a diagram (at
// least 0.8 inch at the lowest resolution read): a bullet or a rule.
const minPicture = 40

func (w *writtenPages) Page(i int) (structure.Page, error) {
	pg, err := w.page(i)
	if pg.Unread != nil && w.unread != nil {
		w.unread[i] = pg.Unread
	}
	return pg, err
}

func (w *writtenPages) page(i int) (structure.Page, error) {
	p := w.pics[i]
	data, err := p.read()
	if err != nil {
		return structure.Page{Unread: err}, nil
	}
	if kind := vectorKind(p.name, data); kind != "" {
		return structure.Page{Unread: fmt.Errorf("a %s picture (drawn with vectors, not read: save the document as PDF)", kind)}, nil
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return structure.Page{Unread: errors.New("a picture in a format not read")}, nil
	}
	switch format {
	case "png", "jpeg", "gif", "bmp", "tiff":
	default:
		return structure.Page{Unread: fmt.Errorf("a %s picture, a format not read", format)}, nil
	}
	if cfg.Width < minPicture || cfg.Height < minPicture {
		return structure.Page{Unread: errSmallPicture}, nil
	}
	if cfg.Width > 40_000 || cfg.Height > 40_000 || cfg.Width*cfg.Height > structure.MaxPixels {
		return structure.Page{Unread: fmt.Errorf("a picture of %d×%d pixels", cfg.Width, cfg.Height)}, nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return structure.Page{Unread: err}, nil
	}
	dpi := imageDPI(data)
	if dpi < 50 || dpi > 1200 {
		dpi = estimatedDPI(cfg.Width, cfg.Height)
	}
	pg := structure.Page{Image: img, DPI: dpi}
	pg.Words = textBeside(cfg.Width, cfg.Height, p.before, p.inside, p.after)
	return pg, nil
}

var errSmallPicture = errors.New("too small to hold a diagram")

// vectorKind names a picture drawn with vectors (EMF, WMF, SVG), which the
// reader does not render: it is named, never guessed at.
func vectorKind(name string, data []byte) string {
	switch {
	case len(data) >= 44 && binary.LittleEndian.Uint32(data) == 1 && string(data[40:44]) == " EMF":
		return "EMF"
	case len(data) >= 4 && binary.LittleEndian.Uint32(data) == 0x9AC6CDD7,
		len(data) >= 6 && (binary.LittleEndian.Uint16(data) == 1 || binary.LittleEndian.Uint16(data) == 2) && binary.LittleEndian.Uint16(data[2:]) == 9 && strings.EqualFold(path.Ext(name), ".wmf"):
		return "WMF"
	case strings.EqualFold(path.Ext(name), ".svg"), strings.EqualFold(path.Ext(name), ".svgz"):
		return "SVG"
	case strings.EqualFold(path.Ext(name), ".emf"):
		return "EMF"
	case strings.EqualFold(path.Ext(name), ".wmf"):
		return "WMF"
	}
	head := bytes.TrimLeft(data[:min(len(data), 512)], " \t\r\n\xef\xbb\xbf")
	if bytes.HasPrefix(head, []byte("<svg")) || (bytes.HasPrefix(head, []byte("<?xml")) && bytes.Contains(data[:min(len(data), 2048)], []byte("<svg"))) {
		return "SVG"
	}
	return ""
}

// textBeside lays the text around a picture out as a text layer, in a
// column at its right as a book prints its pip counts beside a diagram: the
// paragraph before the picture ending at mid-height, its own paragraph and
// the next one from there down, so the upper of two counts is the upper
// player's. The column starts four points (a point being a thirteenth of
// the picture's width) right of the picture, out of reach of the numbers a
// reader takes for pips printed beside the board without a tag, and is
// three and a half points wide, within reach of the tagged ones ("Pips 139")
// and of the dice the text cites. Text without a figure says nothing the
// reader cross-checks: none is laid out, and the reader then reads the
// captions printed in the picture itself.
func textBeside(w, h int, before, inside, after string) []structure.Word {
	if !strings.ContainsFunc(before+inside+after, unicode.IsDigit) {
		return nil
	}
	pt := float64(w) / 13
	lh, cw := 0.6*pt, 0.3*pt
	x0, width := float64(w)+4*pt, 3.5*pt
	lines := func(s string) [][]string {
		var out [][]string
		var line []string
		n := 0
		for _, f := range strings.Fields(s) {
			if n > 0 && float64(n+len([]rune(f)))*cw > width {
				out = append(out, line)
				line, n = nil, 0
			}
			line = append(line, f)
			n += len([]rune(f)) + 1
		}
		if len(line) > 0 {
			out = append(out, line)
		}
		return out
	}
	var words []structure.Word
	put := func(line []string, y float64) {
		x := x0
		for _, f := range line {
			wd := float64(len([]rune(f))) * cw
			words = append(words, structure.Word{X0: x, Y0: y, X1: x + wd, Y1: y + lh, Text: f})
			x += wd + cw
		}
	}
	// Only the lines within reach of a diagram are laid out.
	keep := int((float64(h)/2+3*pt)/lh) + 1
	mid := float64(h) / 2
	up := lines(before)
	if len(up) > keep {
		up = up[len(up)-keep:]
	}
	for k, line := range up {
		put(line, mid-float64(len(up)-k)*lh)
	}
	down := lines(inside)
	down = append(down, lines(after)...)
	if len(down) > keep {
		down = down[:keep]
	}
	for k, line := range down {
		put(line, mid+float64(k)*lh)
	}
	return words
}
