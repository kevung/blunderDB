package direction

import (
	"fmt"
	"html"
	"slices"
	"strings"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/PileOfCells/backgammon-tournoi/render"
)

// The bracket of an event on a wall page, drawn as inline SVG in Go — a wall page carries no
// script (ADR-0039), and what it shows is read from several metres away, which the engine's own
// board (small type, sections side by side) is not built for. The geometry is the same idea as
// the in-app view's (a column per round, a match centred between the two it is fed by) but is
// written here from the same BracketPhase data: the Svelte view computes it in JavaScript, and
// nothing in a static file can call that.

// WallBracketMatch is one place of a graph, reduced to what the wall draws.
type WallBracketMatch struct {
	Key    string
	Label  tournoi.Label
	Length int
	// Round is the column the place sits in.
	Round        int
	AName, BName string
	// Winner is 0 while undecided, 1 when A won, 2 when B won.
	Winner         int
	ScoreA, ScoreB int
	Done, Running  bool
	Skipped        bool
	Feeds          []WallBracketFeed
}

// WallBracketFeed links a seat of a place to the place it comes from.
type WallBracketFeed struct {
	Side    int    // 0 = A, 1 = B
	Section string // empty when the source is in the same graph
	Key     string
	Loser   bool
}

// WallBracketSection is one graph of a phase: main draw, consolation, final…
type WallBracketSection struct {
	Name    string
	Matches []WallBracketMatch
}

// WallBracket is an event's bracket phase: Event names it, Sections are its graphs. An event with
// no bracket to show has no WallBracket at all.
type WallBracket struct {
	Event    string
	Sections []WallBracketSection
}

const (
	wbBoxW, wbBoxH = 330.0, 76.0
	wbGapX, wbGapY = 70.0, 34.0
	wbHead         = 44.0
	wbSecGap       = 30.0
	wbMargin       = 12.0
)

type wbPos struct{ x, y float64 }

// BracketSVG draws the bracket: sections stacked top to bottom, one column per round, the
// matches that send a loser to another graph linked by a dashed line. The viewBox makes it scale
// to the width of the screen.
func BracketSVG(b WallBracket, l render.Labeler) string {
	pos := map[string]wbPos{}
	key := func(sec, k string) string { return sec + "\x00" + k }
	var body strings.Builder
	type seg struct {
		name string
		ms   []WallBracketMatch
		y0   float64
	}
	var segs []seg
	y0, width := wbMargin, 0.0
	for _, s := range b.Sections {
		var ms []WallBracketMatch
		for _, m := range s.Matches {
			if !m.Skipped {
				ms = append(ms, m)
			}
		}
		if len(ms) == 0 {
			continue
		}
		rounds := 0
		for _, m := range ms {
			rounds = max(rounds, m.Round+1)
		}
		ys := placeWallSection(ms)
		bottom := 0.0
		for i, m := range ms {
			pos[key(s.Name, m.Key)] = wbPos{wbMargin + float64(m.Round)*(wbBoxW+wbGapX), y0 + wbHead + ys[i]}
			bottom = max(bottom, ys[i]+wbBoxH)
		}
		width = max(width, wbMargin*2+float64(rounds)*(wbBoxW+wbGapX)-wbGapX)
		segs = append(segs, seg{s.Name, ms, y0})
		y0 += wbHead + bottom + wbSecGap
	}
	if len(segs) == 0 {
		return ""
	}
	height := y0 - wbSecGap + wbMargin

	// Links under the boxes.
	for _, sg := range segs {
		for _, m := range sg.ms {
			here := pos[key(sg.name, m.Key)]
			for _, f := range m.Feeds {
				from := sg.name
				if f.Section != "" {
					from = f.Section
				}
				src, ok := pos[key(from, f.Key)]
				if !ok {
					continue
				}
				dash, colour := "", "#999"
				if from != sg.name || f.Loser {
					dash, colour = ` stroke-dasharray="10 8"`, "#b0308a"
				}
				seat := here.y + wbBoxH*(0.3+0.4*float64(f.Side))
				fmt.Fprintf(&body, `<path d="M%.0f %.0f H%.0f V%.0f H%.0f" fill="none" stroke="%s" stroke-width="3"%s/>`,
					src.x+wbBoxW, src.y+wbBoxH/2, here.x-wbGapX/2, seat, here.x, colour, dash)
			}
		}
	}
	for _, sg := range segs {
		fmt.Fprintf(&body, `<text x="%.0f" y="%.0f" font-size="30" font-weight="700">%s</text>`,
			wbMargin, sg.y0+30, html.EscapeString(l.SectionName(sg.name)))
		for _, m := range sg.ms {
			wbBox(&body, m, pos[key(sg.name, m.Key)], l)
		}
	}
	return fmt.Sprintf(`<svg class="arbre-mur" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %.0f %.0f" width="100%%" font-family="system-ui, sans-serif" font-size="24" role="img" aria-label="%s">%s</svg>`,
		width, height, html.EscapeString(b.Event), body.String())
}

// placeWallSection gives each match its ordinate inside its section: the first round in a
// column, each later match centred on the same-section matches that feed it, and boxes pushed
// apart when two would overlap.
func placeWallSection(ms []WallBracketMatch) []float64 {
	idx := map[string]int{}
	for i, m := range ms {
		idx[m.Key] = i
	}
	ys := make([]float64, len(ms))
	maxRound := 0
	for _, m := range ms {
		maxRound = max(maxRound, m.Round)
	}
	for r := 0; r <= maxRound; r++ {
		var col []int
		for i, m := range ms {
			if m.Round == r {
				col = append(col, i)
			}
		}
		for k, i := range col {
			var up []float64
			for _, f := range ms[i].Feeds {
				if j, ok := idx[f.Key]; ok && f.Section == "" && ms[j].Round < r {
					up = append(up, ys[j])
				}
			}
			if len(up) == 0 {
				ys[i] = float64(k) * (wbBoxH + wbGapY)
				continue
			}
			sum := 0.0
			for _, v := range up {
				sum += v
			}
			ys[i] = sum / float64(len(up))
		}
		slices.SortStableFunc(col, func(a, b int) int {
			switch {
			case ys[a] < ys[b]:
				return -1
			case ys[a] > ys[b]:
				return 1
			}
			return 0
		})
		for k := 1; k < len(col); k++ {
			if ys[col[k]] < ys[col[k-1]]+wbBoxH+wbGapY {
				ys[col[k]] = ys[col[k-1]] + wbBoxH + wbGapY
			}
		}
	}
	return ys
}

func wbBox(b *strings.Builder, m WallBracketMatch, p wbPos, l render.Labeler) {
	fill := "#fff"
	switch {
	case m.Done:
		fill = "#eef5ee"
	case m.Running:
		fill = "#fff7dd"
	}
	fmt.Fprintf(b, `<rect x="%.0f" y="%.0f" width="%.0f" height="%.0f" rx="6" fill="%s" stroke="#666" stroke-width="2"/>`,
		p.x, p.y, wbBoxW, wbBoxH, fill)
	for c, name := range [2]string{m.AName, m.BName} {
		if name == "" || name == string(tournoi.BYE) {
			name = "—"
		}
		weight, score := "400", ""
		if m.Winner == c+1 {
			weight = "700"
		}
		if m.Done && !(m.AName == "" && m.BName == "") {
			score = fmt.Sprintf("%d", [2]int{m.ScoreA, m.ScoreB}[c])
		}
		y := p.y + 30 + float64(c)*34
		fmt.Fprintf(b, `<text x="%.0f" y="%.0f" font-weight="%s" fill="#1c1c1c">%s</text><text x="%.0f" y="%.0f" text-anchor="end" font-weight="700">%s</text>`,
			p.x+10, y, weight, html.EscapeString(name), p.x+wbBoxW-10, y, score)
	}
	fmt.Fprintf(b, `<text x="%.0f" y="%.0f" font-size="15" fill="#555">%s</text>`,
		p.x+4, p.y-6, html.EscapeString(l.Label(m.Label)))
}

// viewSeconds is how long each view of a rotating wall page stays up: long enough to read a
// bracket from the back of the room, short enough that the tables come round again quickly.
const viewSeconds = 12

// minRefresh is the page's reload period, rotating or not: results must show within half a
// minute. A reload restarts CSS animations, so rotation() starts each view already part-way
// through the cycle, at the point the wall clock says it has reached.
const minRefresh = 30

// rotation stacks views in one grid cell and shows them one after the other with a pure-CSS
// animation: the page keeps no script. The negative animation-delay is derived from now (Unix
// time modulo the cycle), so a reloaded page resumes the rotation where it was instead of
// starting over. Reduced-motion and print show every view, stacked, instead.
func rotation(views []string, now time.Time) (css, markup string) {
	n := len(views)
	total := n * viewSeconds
	on := 100 / float64(n)
	css = fmt.Sprintf(`.rot{display:grid}.rot>.vue{grid-area:1/1;visibility:hidden;animation:vue %ds step-end infinite}`+
		`@keyframes vue{0%%{visibility:visible}%.3f%%{visibility:hidden}100%%{visibility:hidden}}`+
		`@media (prefers-reduced-motion:reduce),print{.rot{display:block}.rot>.vue{visibility:visible;animation:none}}`+
		`.arbre-mur{display:block;max-height:88vh;margin-top:8px}`, total, on)
	elapsed := int(((now.Unix() % int64(total)) + int64(total)) % int64(total))
	var b strings.Builder
	b.WriteString(`<div class="rot">`)
	for i, v := range views {
		// Seconds since this view's own turn began, in [0, total).
		local := ((elapsed-i*viewSeconds)%total + total) % total
		fmt.Fprintf(&b, `<section class="vue" style="animation-delay:%ds">%s</section>`, -local, v)
	}
	b.WriteString(`</div>`)
	return css, b.String()
}

// WithBracketView adds a rotating bracket view to an event's own display page (the engine's
// Page output): the page as it was, then the bracket, each in turn. The engine's own small
// board is left out — this one replaces it. A page with no bracket to show is returned as is.
func WithBracketView(page string, br *WallBracket, cat *Catalog, now time.Time) string {
	if br == nil {
		return page
	}
	svg := BracketSVG(*br, NewLabeler(cat, nil))
	bodyAt := strings.Index(page, "<body>")
	creditAt := strings.LastIndex(page, `<p class="credit">`)
	if svg == "" || bodyAt < 0 || creditAt < bodyAt {
		return page
	}
	bodyAt += len("<body>")
	main := page[bodyAt:creditAt]
	if i := strings.Index(main, `<div class="arbre">`); i >= 0 {
		if j := strings.Index(main[i:], `</div>`); j >= 0 {
			main = main[:i] + main[i+j+len(`</div>`):]
		}
	}
	css, markup := rotation([]string{main, `<h1>` + wallEsc(br.Event) + `</h1>` + svg}, now)
	out := page[:bodyAt] + markup + page[creditAt:]
	return strings.Replace(out, `</style>`, css+`</style>`, 1)
}
