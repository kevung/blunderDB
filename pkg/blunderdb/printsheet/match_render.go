package printsheet

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/arkqr"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// codeModuleMM is the module of the reimport code: read by a scanner at
// 300 dpi and by a phone photograph taken flat.
const codeModuleMM = 0.33

// BuildMatches writes the sheets of matches, one per match, as one printable
// document. It writes nothing to the library.
func BuildMatches(ctx context.Context, s storage.Storage, scope string, ids []int64, o MatchOptions) ([]byte, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("no match to print: %w", storage.ErrInvalid)
	}
	if len(ids) > MaxMatches {
		return nil, fmt.Errorf("%d matches to print, at most %d at once: %w", len(ids), MaxMatches, storage.ErrInvalid)
	}
	lang, c := catalogue(o.Language)
	var docs []string
	var code []arkqr.Frame
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		ms, err := ReadMatch(ctx, s, scope, id, o)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return nil, fmt.Errorf("match %d: %w", id, err)
			}
			return nil, err
		}
		doc, err := RenderMatch(ms, o, c)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
		code = append(code, ms.Code...)
	}
	tail := ""
	if len(code) > 0 {
		tail = arkqr.EmbedFrames(code)
	}
	return Document(lang, "blunderDB · "+label(c, "title"), docs, tail), nil
}

// RenderMatch writes one match's sheet in the labels c.
func RenderMatch(ms *MatchSheet, o MatchOptions, c map[string]string) (string, error) {
	esc := html.EscapeString
	var b strings.Builder

	// Header.
	b.WriteString(`<header class="mh"><div><div class="mt">`)
	fmt.Fprintf(&b, "%s %d – %d %s", esc(ms.Players[0]), ms.Score[0], ms.Score[1], esc(ms.Players[1]))
	b.WriteString(`</div><div class="mm">`)
	var meta []string
	if ms.Length > 0 {
		meta = append(meta, label(c, "length", "n", strconv.Itoa(ms.Length)))
	} else {
		meta = append(meta, label(c, "money"))
	}
	ev := ms.Event
	if ms.Round != "" {
		ev = strings.TrimSpace(ev + ", " + ms.Round)
		ev = strings.TrimPrefix(ev, ", ")
	}
	if ev != "" {
		meta = append(meta, esc(ev))
	}
	if !ms.Date.IsZero() {
		meta = append(meta, ms.Date.Format("2006-01-02"))
	}
	meta = append(meta, label(c, "games", "n", strconv.Itoa(len(ms.Games))), label(c, "plies", "n", strconv.Itoa(ms.Plies)))
	if ms.TotalMS != nil {
		meta = append(meta, duration(*ms.TotalMS))
	}
	b.WriteString(strings.Join(meta, " · "))
	b.WriteString(`</div></div><table class="st"><tr><th></th><th>PR</th><th>` + label(c, "errors") + `</th><th>` + label(c, "blunders") + `</th>`)
	if ms.TotalMS != nil {
		b.WriteString(`<th>` + label(c, "time") + `</th>`)
	}
	b.WriteString(`</tr>`)
	for i, st := range ms.Stats {
		pr := "—"
		if st.PR != nil {
			pr = strconv.FormatFloat(*st.PR, 'f', 1, 64)
		}
		fmt.Fprintf(&b, `<tr><td>%s</td><td>%s</td><td>%d</td><td>%d</td>`, esc(ms.Players[i]), pr, st.Errors, st.Blunders)
		if ms.TotalMS != nil {
			t := "—"
			if st.TimeMS != nil {
				t = duration(*st.TimeMS)
			}
			b.WriteString(`<td>` + t + `</td>`)
		}
		b.WriteString(`</tr>`)
	}
	b.WriteString(`</table></header>`)
	if o.Comments && strings.TrimSpace(ms.Comment) != "" {
		b.WriteString(`<div class="mc">` + esc(ms.Comment) + `</div>`)
	}
	b.WriteString(`<div class="lg"><b>?</b> ` + label(c, "lgError") + ` · <b>??</b> ` + label(c, "lgBlunder") +
		` · <span class="ref">n</span> ` + label(c, "lgRef") + ` · <span class="ref o">n</span> ` + label(c, "lgRemark"))
	if ms.TotalMS != nil {
		b.WriteString(` · <i>47s</i> ` + label(c, "lgSlow"))
	}
	b.WriteString(` · ` + esc(label(c, "lgArk")) + `</div>`)

	// Transcription.
	b.WriteString(`<div class="tr" data-flow>`)
	for _, g := range ms.Games {
		b.WriteString(`<div class="gh"><div>` + label(c, "game") + " " + strconv.Itoa(g.Number))
		if ms.Length > 0 {
			crawford := ""
			if g.Crawford {
				crawford = "c"
			}
			fmt.Fprintf(&b, ` <span>%da–%da%s</span>`, g.Away[0], g.Away[1], crawford)
		}
		b.WriteString(`</div><span>`)
		var res []string
		if g.Winner >= 0 {
			res = append(res, fmt.Sprintf("%s +%d", esc(ms.Players[g.Winner]), g.Points))
		}
		if g.TimeMS != nil {
			res = append(res, duration(*g.TimeMS))
		}
		b.WriteString(strings.Join(res, " · ") + `</span></div><div class="gb">`)
		for i, row := range g.Rows {
			fmt.Fprintf(&b, `<div class="rw"><span class="no">%d</span><span class="c1">%s</span><span class="c2">%s</span></div>`,
				i+1, cellHTML(row[0], c), cellHTML(row[1], c))
		}
		b.WriteString(`</div>`)
	}
	if o.ReimportCode {
		switch {
		case len(ms.Code) > 0:
			b.WriteString(`<div class="qrb">`)
			for _, f := range ms.Code {
				svg, _, err := arkqr.SVG(f, codeModuleMM)
				if err != nil {
					return "", err
				}
				b.WriteString(svg)
			}
			b.WriteString(`<div><b>` + label(c, "reimport") + `</b> · ` + label(c, "bytes", "n", strconv.Itoa(ms.CodeSize)) +
				`<br>` + label(c, "reimportNote") + `</div></div>`)
		case ms.CodeErr != "":
			b.WriteString(`<div class="qrb">` + esc(label(c, "noCode", "err", ms.CodeErr)) + `</div>`)
		}
	}
	b.WriteString(`</div>`)

	// Positions.
	if len(ms.Entries) > 0 {
		blunders, remarks := 0, 0
		for _, e := range ms.Entries {
			if e.Kind == KindBlunder {
				blunders++
			} else {
				remarks++
			}
		}
		who := label(c, "blundersOf", "n", strconv.Itoa(blunders), "p", esc(ms.Players[ms.Self]))
		if o.OpponentBlunders {
			who = label(c, "blundersBoth", "n", strconv.Itoa(blunders))
		}
		b.WriteString(`<div data-verso><div class="ph"><div>` + label(c, "positions") + `</div><span>` + who + ` · ` +
			label(c, "remarks", "n", strconv.Itoa(remarks)) + `</span></div><div class="pl">`)
		for _, e := range ms.Entries {
			b.WriteString(entryHTML(ms, e, o, c))
		}
		b.WriteString(`</div></div>`)
	}
	return Sheet("match", b.String()), nil
}

func refHTML(n int, kind string) string {
	cls := "ref"
	if kind == KindRemark {
		cls += " o"
	}
	return fmt.Sprintf(`<span class="%s">%d</span>`, cls, n)
}

func cellHTML(cell Cell, c map[string]string) string {
	var parts []string
	for _, p := range cell.Parts {
		var b strings.Builder
		text := html.EscapeString(p.Text)
		switch {
		case p.Dice != "" && p.Text == "":
			text = label(c, "dance")
		case p.Dice == "":
			text = cubeLabel(p.Text, c)
			if p.Cube > 0 {
				text += " " + strconv.Itoa(p.Cube)
			}
		}
		if p.Dice != "" {
			b.WriteString(`<span class="d">` + p.Dice + `</span> `)
		}
		if p.Grade == GradeBlunder {
			b.WriteString(`<span class="bl">` + text + `</span><span class="g">??</span>`)
		} else {
			b.WriteString(text)
			if p.Grade == GradeError {
				b.WriteString(`<span class="g">?</span>`)
			}
		}
		if p.Ref > 0 {
			b.WriteString(refHTML(p.Ref, p.Kind))
		}
		if p.SlowMS > 0 {
			b.WriteString(`<span class="tm">` + strconv.FormatInt(p.SlowMS/1000, 10) + `s</span>`)
		}
		parts = append(parts, b.String())
	}
	return strings.Join(parts, " ")
}

func cubeLabel(action string, c map[string]string) string {
	if _, ok := c[action]; ok || labels[DefaultLanguage][action] != "" {
		return label(c, action)
	}
	return html.EscapeString(action)
}

func entryHTML(ms *MatchSheet, e *Entry, o MatchOptions, c map[string]string) string {
	esc := html.EscapeString
	var b strings.Builder
	b.WriteString(`<div class="pe">` + refHTML(e.Ref, e.Kind) + `<span class="nt">` + esc(e.ArkID) + `</span><div>`)
	fmt.Fprintf(&b, `<span class="wh">%s%d·%d %s</span>`, gameInitial(c), e.Game, e.Row, esc(ms.Players[e.Player]))
	if o.Analysis && e.Played != "" {
		b.WriteString(`<br>` + label(c, "played") + " ")
		played := esc(e.Played)
		if e.Cube {
			played = cubeLabel(e.Played, c)
		}
		if e.CostMP != nil && *e.CostMP > 0 {
			b.WriteString(`<b>` + played + " " + cost(*e.CostMP) + `</b>`)
			if e.Best != "" {
				best := esc(e.Best)
				if e.Cube {
					best = cubeLabel(e.Best, c)
				}
				b.WriteString(" › " + best)
			}
		} else {
			b.WriteString(played)
			if e.CostMP != nil {
				b.WriteString(" (" + label(c, "best") + ")")
			}
		}
		if e.Alt != "" {
			fmt.Fprintf(&b, " · %d. %s %s", e.AltRank, esc(e.Alt), cost(e.AltCostMP))
		}
	}
	b.WriteString(`</div>`)
	if o.Comments {
		for _, cm := range e.Comments {
			b.WriteString(`<div class="cm">` + esc(cm) + `</div>`)
		}
	}
	b.WriteString(`</div>`)
	return b.String()
}

// gameInitial is the one-letter abbreviation of "game" before its number.
func gameInitial(c map[string]string) string {
	r := []rune(label(c, "game"))
	if len(r) == 0 {
		return ""
	}
	return string(r[0])
}

// cost writes a loss in millipoints as an equity, on the scale it was
// stored in: money points at money play, the current cube at a match score.
func cost(mp int) string {
	return "−" + strconv.FormatFloat(float64(mp)/1000, 'f', 3, 64)
}

func duration(ms int64) string {
	m := (ms + 30_000) / 60_000
	if m >= 60 {
		return fmt.Sprintf("%d h %02d", m/60, m%60)
	}
	return fmt.Sprintf("%d min", m)
}
