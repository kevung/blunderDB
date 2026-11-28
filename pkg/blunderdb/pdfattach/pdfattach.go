// Package pdfattach embeds a file in a PDF and reads embedded files back,
// with the standard library alone. It covers what blunderDB needs and no
// more: Attach writes an incremental update (the original bytes are kept
// verbatim, signatures and all) holding an EmbeddedFile stream, its file
// specification and a new revision of the catalog naming it; Files returns
// the content of every EmbeddedFile stream of a document, Flate-compressed
// or not, so a file rewritten by another tool still yields its attachments
// as long as it keeps them outside object streams.
package pdfattach

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// ErrUnsupported is a PDF whose structure Attach does not rewrite: no
// readable trailer or catalog, or a catalog that already names files.
var ErrUnsupported = errors.New("pdfattach: unsupported PDF structure")

var (
	startxrefRe = regexp.MustCompile(`startxref\s+(\d+)\s+%%EOF`)
	rootRe      = regexp.MustCompile(`/Root\s+(\d+)\s+(\d+)\s+R`)
	sizeRe      = regexp.MustCompile(`/Size\s+(\d+)`)
	infoRe      = regexp.MustCompile(`/Info\s+\d+\s+\d+\s+R`)
	idRe        = regexp.MustCompile(`/ID\s*\[[^\]]*\]`)
	objRe       = regexp.MustCompile(`(\d+)\s+(\d+)\s+obj\b`)
	embeddedRe  = regexp.MustCompile(`/Type\s*/EmbeddedFile\b`)
	flateRe     = regexp.MustCompile(`/Filter\s*(?:\[\s*)?/FlateDecode\b`)
	lengthRe    = regexp.MustCompile(`/Length\s+(\d+)(\s+\d+\s+R)?`)
)

// Attach returns pdf with data embedded as a file named name, of MIME type
// mime, listed among the document's attachments.
func Attach(pdf []byte, name, mime string, data []byte) ([]byte, error) {
	sx := startxrefRe.FindAllSubmatchIndex(pdf, -1)
	if len(sx) == 0 {
		return nil, fmt.Errorf("%w: no startxref", ErrUnsupported)
	}
	last := sx[len(sx)-1]
	prev := string(pdf[last[2]:last[3]])
	trailer, err := trailerDict(pdf, last[0], prev)
	if err != nil {
		return nil, err
	}
	rm := rootRe.FindStringSubmatch(trailer)
	sm := sizeRe.FindStringSubmatch(trailer)
	if rm == nil || sm == nil {
		return nil, fmt.Errorf("%w: trailer without /Root or /Size", ErrUnsupported)
	}
	rootNum, _ := strconv.Atoi(rm[1])
	rootGen, _ := strconv.Atoi(rm[2])
	size, _ := strconv.Atoi(sm[1])
	catalog, err := objectDict(pdf, rootNum, rootGen)
	if err != nil {
		return nil, err
	}
	if strings.Contains(catalog, "/Names") || strings.Contains(catalog, "/AF") {
		return nil, fmt.Errorf("%w: the catalog already names files", ErrUnsupported)
	}

	var b bytes.Buffer
	b.Write(pdf)
	if n := len(pdf); n > 0 && pdf[n-1] != '\n' {
		b.WriteByte('\n')
	}
	efNum, specNum := size, size+1
	offsets := map[int]int{}

	offsets[efNum] = b.Len()
	fmt.Fprintf(&b, "%d 0 obj\n<< /Type /EmbeddedFile /Subtype %s /Length %d /Params << /Size %d >> >>\nstream\n",
		efNum, pdfName(mime), len(data), len(data))
	b.Write(data)
	b.WriteString("\nendstream\nendobj\n")

	offsets[specNum] = b.Len()
	fmt.Fprintf(&b, "%d 0 obj\n<< /Type /Filespec /F %s /UF %s /EF << /F %d 0 R >> /AFRelationship /Data >>\nendobj\n",
		specNum, pdfString(name), pdfString(name), efNum)

	offsets[rootNum] = b.Len()
	body := strings.TrimSuffix(strings.TrimSpace(catalog), ">>")
	fmt.Fprintf(&b, "%d %d obj\n%s /Names << /EmbeddedFiles << /Names [%s %d 0 R] >> >> /AF [%d 0 R] >>\nendobj\n",
		rootNum, rootGen, body, pdfString(name), specNum, specNum)

	xref := b.Len()
	b.WriteString("xref\n")
	fmt.Fprintf(&b, "%d 1\n%010d %05d n \n", rootNum, offsets[rootNum], rootGen)
	fmt.Fprintf(&b, "%d 2\n%010d 00000 n \n%010d 00000 n \n", efNum, offsets[efNum], offsets[specNum])
	extra := ""
	if m := infoRe.FindString(trailer); m != "" {
		extra += " " + m
	}
	if m := idRe.FindString(trailer); m != "" {
		extra += " " + m
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root %d %d R /Prev %s%s >>\nstartxref\n%d\n%%%%EOF\n",
		size+2, rootNum, rootGen, prev, extra, xref)
	return b.Bytes(), nil
}

// trailerDict is the dictionary of the last cross-reference section: the
// classic trailer before startxref, or the dictionary of the cross-reference
// stream at offset prev.
func trailerDict(pdf []byte, startxref int, prev string) (string, error) {
	if i := bytes.LastIndex(pdf[:startxref], []byte("trailer")); i >= 0 {
		if d, ok := dictAt(pdf, i+len("trailer")); ok {
			return d, nil
		}
	}
	off, err := strconv.Atoi(prev)
	if err != nil || off >= len(pdf) {
		return "", fmt.Errorf("%w: bad startxref", ErrUnsupported)
	}
	if loc := objRe.FindIndex(pdf[off:]); loc != nil && loc[0] < 16 {
		if d, ok := dictAt(pdf, off+loc[1]); ok {
			return d, nil
		}
	}
	return "", fmt.Errorf("%w: no trailer", ErrUnsupported)
}

// objectDict is the dictionary of the last definition of object num gen.
func objectDict(pdf []byte, num, gen int) (string, error) {
	re := regexp.MustCompile(`(?:^|[^0-9])` + strconv.Itoa(num) + `\s+` + strconv.Itoa(gen) + `\s+obj\b`)
	locs := re.FindAllIndex(pdf, -1)
	for i := len(locs) - 1; i >= 0; i-- {
		if d, ok := dictAt(pdf, locs[i][1]); ok {
			return d, nil
		}
	}
	return "", fmt.Errorf("%w: catalog %d %d not found", ErrUnsupported, num, gen)
}

// dictAt returns the dictionary that starts at the first non-blank byte at
// or after i, "<<" to its matching ">>", strings skipped.
func dictAt(pdf []byte, i int) (string, bool) {
	for i < len(pdf) && isSpace(pdf[i]) {
		i++
	}
	if i+1 >= len(pdf) || pdf[i] != '<' || pdf[i+1] != '<' {
		return "", false
	}
	start, depth := i, 0
	for i < len(pdf) {
		switch c := pdf[i]; {
		case c == '(':
			i = skipString(pdf, i)
			continue
		case c == '<' && i+1 < len(pdf) && pdf[i+1] == '<':
			depth++
			i += 2
			continue
		case c == '>' && i+1 < len(pdf) && pdf[i+1] == '>':
			depth--
			i += 2
			if depth == 0 {
				return string(pdf[start:i]), true
			}
			continue
		}
		i++
	}
	return "", false
}

// skipString returns the index just past the literal string opening at i.
func skipString(pdf []byte, i int) int {
	depth := 0
	for ; i < len(pdf); i++ {
		switch pdf[i] {
		case '\\':
			i++
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return i
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\n' || c == '\r' || c == '\t' || c == '\f' || c == 0
}

// pdfString is s as a literal string.
func pdfString(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`)
	return "(" + r.Replace(s) + ")"
}

// pdfName is s as a name object, every byte outside the regular printable
// set written as #xx.
func pdfName(s string) string {
	var b strings.Builder
	b.WriteByte('/')
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c > 0x20 && c < 0x7f && !strings.ContainsRune("#/()<>[]{}%", rune(c)) {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "#%02X", c)
		}
	}
	return b.String()
}

// Files returns the content of every EmbeddedFile stream in pdf, in file
// order. A stream it cannot decode is left out.
func Files(pdf []byte) [][]byte {
	var out [][]byte
	for _, loc := range objRe.FindAllIndex(pdf, -1) {
		dict, ok := dictAt(pdf, loc[1])
		if !ok || !embeddedRe.MatchString(dict) {
			continue
		}
		i := bytes.Index(pdf[loc[1]:], []byte(dict)) + loc[1] + len(dict)
		for i < len(pdf) && isSpace(pdf[i]) {
			i++
		}
		if !bytes.HasPrefix(pdf[i:], []byte("stream")) {
			continue
		}
		i += len("stream")
		if i < len(pdf) && pdf[i] == '\r' {
			i++
		}
		if i < len(pdf) && pdf[i] == '\n' {
			i++
		}
		raw := streamData(pdf, i, dict)
		if raw == nil {
			continue
		}
		if flateRe.MatchString(dict) {
			zr, err := zlib.NewReader(bytes.NewReader(raw))
			if err != nil {
				continue
			}
			dec, err := io.ReadAll(zr)
			if err != nil {
				continue
			}
			raw = dec
		}
		out = append(out, raw)
	}
	return out
}

// streamData is the stream body starting at i: /Length bytes when the
// dictionary states it directly, otherwise up to endstream.
func streamData(pdf []byte, i int, dict string) []byte {
	if m := lengthRe.FindStringSubmatch(dict); m != nil && m[2] == "" {
		if n, err := strconv.Atoi(m[1]); err == nil && i+n <= len(pdf) {
			return pdf[i : i+n]
		}
	}
	j := bytes.Index(pdf[i:], []byte("endstream"))
	if j < 0 {
		return nil
	}
	return bytes.TrimRight(pdf[i:i+j], "\r\n")
}
