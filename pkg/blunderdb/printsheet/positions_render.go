package printsheet

import (
	"context"
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/arkqr"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// BuildPositions writes the printable list of the positions src names, one
// line a position, on as many numbered A4 pages as it takes. It writes
// nothing to the library.
func BuildPositions(ctx context.Context, s storage.Storage, scope string, src PositionSource, o PositionsOptions) ([]byte, error) {
	ids, title, err := ResolvePositions(ctx, s, scope, src)
	if err != nil {
		return nil, err
	}
	l, err := ReadPositions(ctx, s, scope, ids, o)
	if err != nil {
		return nil, err
	}
	lang, c := catalogue(o.Language)
	l.Title = strings.TrimSpace(o.Title)
	if l.Title == "" {
		l.Title = title
	}
	if l.Title == "" {
		l.Title = label(c, "listTitle")
	}
	doc, err := RenderPositions(l, o, c)
	if err != nil {
		return nil, err
	}
	var frames []arkqr.Frame
	for _, tc := range l.Codes {
		frames = append(frames, tc.Frames...)
	}
	tail := ""
	if len(frames) > 0 {
		tail = arkqr.EmbedFrames(frames)
	}
	return Document(lang, "blunderDB · "+l.Title, []string{doc}, tail), nil
}

// RenderPositions writes a position list in the labels c. With the analyses
// on the back, the front block holds the positions and the data-verso block
// the answers under the same numbers; the fitting script pairs each front
// page with its back. The reimport codes follow the lines, a .qrb block a
// tranche marked with its first number (data-from), which the script moves
// to the front page where that line falls.
func RenderPositions(l *PositionList, o PositionsOptions, c map[string]string) (string, error) {
	esc := html.EscapeString
	verso := o.Verso && o.Analysis
	inline := o.Analysis && !verso
	var b strings.Builder

	b.WriteString(`<header class="mh"><div><div class="mt">` + esc(l.Title) + `</div><div class="mm">`)
	meta := []string{label(c, "nPositions", "n", strconv.Itoa(len(l.Entries)))}
	if verso {
		meta = append(meta, label(c, "versoNote"))
	}
	if !l.Printed.IsZero() {
		meta = append(meta, label(c, "printedOn", "d", l.Printed.Format("2006-01-02")))
	}
	b.WriteString(strings.Join(meta, " · ") + `</div></div></header>`)
	b.WriteString(`<div class="lg">` + esc(label(c, "lgArk")) + ` · ` + label(c, "lgPips"))
	if inline {
		b.WriteString(` · ` + label(c, "lgAnalysis"))
	}
	b.WriteString(`</div>`)

	cls := "many"
	if inline {
		cls = "two"
	}
	b.WriteString(`<div class="dl ` + cls + `">`)
	for _, e := range l.Entries {
		b.WriteString(`<div class="de"><span class="no">` + strconv.Itoa(e.N) + `</span><span class="nt">` + esc(e.ArkID) +
			`</span><span class="an"><span class="wh">` + strconv.Itoa(e.Pips[0]) + "–" + strconv.Itoa(e.Pips[1]) + `</span>`)
		if inline {
			b.WriteString(analysisHTML(e, c))
		}
		if o.Comments && !verso {
			b.WriteString(commentsHTML(e))
		}
		b.WriteString(`</span></div>`)
	}
	b.WriteString(`</div>`)
	if o.ReimportCode {
		b.WriteString(`<div class="qrs">`)
		for _, tc := range l.Codes {
			fmt.Fprintf(&b, `<div class="qrb" data-from="%d">`, tc.From)
			for _, f := range tc.Frames {
				svg, _, err := arkqr.SVG(f, codeModuleMM)
				if err != nil {
					return "", err
				}
				b.WriteString(svg)
			}
			b.WriteString(`<div><b>` + label(c, "reimportRange", "a", strconv.Itoa(tc.From), "b", strconv.Itoa(tc.To)) + `</b> · ` +
				label(c, "bytes", "n", strconv.Itoa(tc.Size)) + `<br>` + label(c, "reimportListNote") + `</div></div>`)
		}
		if l.CodeErr != "" {
			b.WriteString(`<div class="qrb">` + esc(label(c, "noCode", "err", l.CodeErr)) + `</div>`)
		}
		b.WriteString(`</div>`)
	}

	if verso {
		b.WriteString(`<div data-verso><div class="ph"><div>` + label(c, "answers") + `</div><span>` + label(c, "lgAnalysis") +
			`</span></div><div class="dl ans">`)
		for _, e := range l.Entries {
			b.WriteString(`<div class="de"><span class="no">` + strconv.Itoa(e.N) + `</span><span class="an">` + analysisHTML(e, c))
			if o.Comments {
				b.WriteString(commentsHTML(e))
			}
			b.WriteString(`</span></div>`)
		}
		b.WriteString(`</div></div>`)
	}
	return ListSheet("positions", b.String()), nil
}

// analysisHTML is the analysis of one line: the best choice in bold, the
// others with their losses in thousandths of the stored scale, then the
// chances of winning and of a gammon.
func analysisHTML(e *ListEntry, c map[string]string) string {
	if e.Best == "" {
		return ""
	}
	name := html.EscapeString
	if e.Cube {
		name = func(s string) string { return cubeActionLabel(s, c) }
	}
	var b strings.Builder
	b.WriteString(` <b>` + name(e.Best) + `</b>`)
	for _, o := range e.Others {
		b.WriteString(` · ` + name(o.Text) + ` ` + thousandths(o.CostMP))
	}
	if e.Win != nil && e.Gammon != nil {
		b.WriteString(` · ` + strconv.Itoa(int(*e.Win+0.5)) + `/` + strconv.Itoa(int(*e.Gammon+0.5)))
	}
	return b.String()
}

func commentsHTML(e *ListEntry) string {
	var b strings.Builder
	for _, cm := range e.Comments {
		if strings.TrimSpace(cm) != "" {
			b.WriteString(`<div class="cm">` + html.EscapeString(cm) + `</div>`)
		}
	}
	return b.String()
}

// thousandths writes a loss in millipoints without its "0.": −45 for 0.045.
func thousandths(mp int) string {
	if mp == 0 {
		return "0"
	}
	return "−" + strconv.Itoa(mp)
}

// cubeActionLabel translates a cube action such as "Double, Take" part by
// part, keeping a part the catalogue does not know as it is.
func cubeActionLabel(action string, c map[string]string) string {
	parts := strings.Split(action, ",")
	for i, p := range parts {
		parts[i] = cubeLabel(strings.TrimSpace(p), c)
	}
	return strings.Join(parts, ", ")
}
