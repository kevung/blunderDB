package engine

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/net/html/charset"
)

// MET is a match equity table a library imported from a gnubg .xml file
// (ADR-0068). A nil *MET is the built-in Kazaross-XG2: every lookup through a
// nil table reads the built-in arrays, bit for bit.
//
// Only explicit tables are held, up to their declared length; past it a
// lookup falls back to the built-in table (Kazaross-XG2 to 25 points, the
// Zadeh model beyond), so a 15-point table still answers a 21-point match.
type MET struct {
	Name   string
	Length int
	pre    []float64 // Length×Length, row-major: [i*Length+j]
	post   []float64 // Length
}

func (m *MET) preAt(i, j int) float64 {
	if m == nil || i < 0 || j < 0 || i >= m.Length || j >= m.Length {
		return metPre(i, j)
	}
	return m.pre[i*m.Length+j]
}

func (m *MET) postAt(n int) float64 {
	if m == nil || n < 0 || n >= m.Length {
		return metPost(n)
	}
	return m.post[n]
}

// ErrMETFormat is a .xml this importer cannot read as an explicit gnubg MET.
var ErrMETFormat = errors.New("not an explicit gnubg match equity table")

type gnubgMETRow struct {
	ME []string `xml:"me"`
}

type gnubgMETTable struct {
	Type   string        `xml:"type,attr"`
	Player string        `xml:"player,attr"`
	Rows   []gnubgMETRow `xml:"row"`
}

type gnubgMETDoc struct {
	XMLName xml.Name `xml:"met"`
	Info    struct {
		Name   string `xml:"name"`
		Length string `xml:"length"`
	} `xml:"info"`
	Pre  gnubgMETTable   `xml:"pre-crawford-table"`
	Post []gnubgMETTable `xml:"post-crawford-table"`
}

// ParseGnubgMET reads a gnubg match equity table (met.dtd). Parametric
// tables (zadeh, mec) and tables whose post-Crawford values differ by player
// are refused: blunderDB stores values, and one post-Crawford row serves
// both sides.
func ParseGnubgMET(data []byte) (*MET, error) {
	var doc gnubgMETDoc
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.CharsetReader = charset.NewReaderLabel
	dec.Strict = false
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMETFormat, err)
	}
	length, err := strconv.Atoi(strings.TrimSpace(doc.Info.Length))
	if err != nil || length < 1 || length > MaxScore {
		return nil, fmt.Errorf("%w: length %q outside 1..%d", ErrMETFormat, doc.Info.Length, MaxScore)
	}
	if t := strings.TrimSpace(doc.Pre.Type); t != "" && t != "explicit" {
		return nil, fmt.Errorf("%w: pre-Crawford table of type %q (only explicit tables are supported)", ErrMETFormat, t)
	}
	if len(doc.Post) != 1 {
		return nil, fmt.Errorf("%w: %d post-Crawford tables (one, for both players, is supported)", ErrMETFormat, len(doc.Post))
	}
	post := doc.Post[0]
	if t := strings.TrimSpace(post.Type); t != "" && t != "explicit" {
		return nil, fmt.Errorf("%w: post-Crawford table of type %q (only explicit tables are supported)", ErrMETFormat, t)
	}
	if p := strings.TrimSpace(post.Player); p != "" && p != "both" {
		return nil, fmt.Errorf("%w: post-Crawford table for player %q (one table for both players is supported)", ErrMETFormat, p)
	}
	m := &MET{
		Name:   strings.Join(strings.Fields(doc.Info.Name), " "),
		Length: length,
		pre:    make([]float64, 0, length*length),
		post:   make([]float64, 0, length),
	}
	if len(doc.Pre.Rows) < length {
		return nil, fmt.Errorf("%w: %d pre-Crawford rows for a %d-point table", ErrMETFormat, len(doc.Pre.Rows), length)
	}
	for i, row := range doc.Pre.Rows[:length] {
		if len(row.ME) < length {
			return nil, fmt.Errorf("%w: pre-Crawford row %d has %d values, want %d", ErrMETFormat, i+1, len(row.ME), length)
		}
		for _, s := range row.ME[:length] {
			v, err := parseMWC(s)
			if err != nil {
				return nil, err
			}
			m.pre = append(m.pre, v)
		}
	}
	if len(post.Rows) != 1 || len(post.Rows[0].ME) < length {
		return nil, fmt.Errorf("%w: the post-Crawford table needs one row of %d values", ErrMETFormat, length)
	}
	for _, s := range post.Rows[0].ME[:length] {
		v, err := parseMWC(s)
		if err != nil {
			return nil, err
		}
		m.post = append(m.post, v)
	}
	return m, nil
}

func parseMWC(s string) (float64, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || v < 0 || v > 1 {
		return 0, fmt.Errorf("%w: match winning chance %q outside [0, 1]", ErrMETFormat, s)
	}
	return v, nil
}

// Digest identifies the table by its values alone, in a canonical text fixed
// here once: two files differing only by layout, comments or name share it,
// and so do two libraries that hold the table (ADR-0068). The built-in
// Kazaross-XG2 has KazarossXG2Digest.
func (m *MET) Digest() string {
	if m == nil {
		return KazarossXG2Digest()
	}
	return metDigest(m.Length, m.pre, m.post)
}

func metDigest(length int, pre, post []float64) string {
	var b strings.Builder
	fmt.Fprintf(&b, "blunderdb-met/1\n%d\n", length)
	for i, v := range pre {
		b.WriteString(strconv.FormatFloat(v, 'g', -1, 64))
		if (i+1)%length == 0 {
			b.WriteByte('\n')
		} else {
			b.WriteByte(' ')
		}
	}
	for i, v := range post {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(strconv.FormatFloat(v, 'g', -1, 64))
	}
	b.WriteByte('\n')
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

var kazarossXG2Digest = sync.OnceValue(func() string {
	pre := make([]float64, 0, 25*25)
	for i := range 25 {
		pre = append(pre, kazarossXG2PreCrawford[i][:]...)
	}
	return metDigest(25, pre, kazarossXG2PostCrawford[:])
})

// KazarossXG2Digest is the digest of the built-in table: an imported table
// that carries it is the built-in one under another file name.
func KazarossXG2Digest() string { return kazarossXG2Digest() }
