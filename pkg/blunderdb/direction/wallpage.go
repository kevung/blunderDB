package direction

import (
	"fmt"
	"html"
	"strings"

	"github.com/PileOfCells/backgammon-tournoi/render"
)

// The room's wall page (ADR-0056 §6): one file at the root of the Rencontre's output folder, one
// line per table whichever event plays on it, a link to each event's own page. It shares the
// per-event display page's plumbing — style, credit, no script, meta refresh — because it is the
// same wall, just wider: TableGrid renders one Direction's tables, this renders every member's.

// WallPageName is the room's wall page — `<dossier>/index.html`.
const WallPageName = "index.html"

// WallTable is one line of the wall page: one table, whichever event plays on it.
type WallTable struct {
	Number      int
	Unavailable bool
	Event       string // empty when the table is free
	A, B        string // the two players, when Event is set
}

// WallEvent is one member Tournament, as the wall page links to it: its name, where its own page
// lives (relative to the Rencontre's folder), and how many rounds it has announced so far.
type WallEvent struct {
	Name   string
	Slug   string
	Rounds int
}

// WallPageInput is everything the wall page needs. The caller — the database layer, which knows
// the Rencontre and replays each member's Direction — gathers it; this package only renders.
type WallPageInput struct {
	Name   string
	Tables []WallTable
	Events []WallEvent
}

func wallEsc(s string) string { return html.EscapeString(s) }

// WallPage renders the room's wall page in the host's language, falling back to the engine's own
// French labels when no catalogue was published (the CLI's case).
func WallPage(in WallPageInput, cat *Catalog, lang string) string {
	l := NewLabeler(cat, nil)
	credit := render.DefaultCredit
	if c, ok := cat.t("credit.by", nil); ok && c != "" {
		credit = c
	}
	if lang == "" {
		lang = "fr"
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<h1>%s</h1>`, wallEsc(in.Name))
	b.WriteString(`<ul class="tables">`)
	for _, tb := range in.Tables {
		classe, contenu := "libre", wallEsc(l.Term(render.TermFree, 0))
		switch {
		case tb.Unavailable:
			classe, contenu = "hs", wallEsc(l.Term(render.TermUnavailable, 0))
		case tb.Event != "":
			classe = "occupee"
			contenu = fmt.Sprintf(`<span class="evt">%s</span><br>%s %s %s`,
				wallEsc(tb.Event), wallEsc(tb.A), wallEsc(l.Term(render.TermVersus, 0)), wallEsc(tb.B))
		}
		fmt.Fprintf(&b, `<li class="%s"><span class="num">%s %d</span>%s</li>`,
			classe, wallEsc(l.Term(render.TermTable, 0)), tb.Number, contenu)
	}
	b.WriteString(`</ul>`)
	if len(in.Events) > 0 {
		b.WriteString(`<ul class="events">`)
		for _, ev := range in.Events {
			rounds := wallTerm(cat, "rencontre.wall.rounds", "{n} rondes annoncées", map[string]any{"n": ev.Rounds})
			fmt.Fprintf(&b, `<li><a href="%s/%s">%s</a> — %s</li>`,
				wallEsc(ev.Slug), PageName, wallEsc(ev.Name), wallEsc(rounds))
		}
		b.WriteString(`</ul>`)
	}

	var page strings.Builder
	fmt.Fprintf(&page, `<!doctype html><html lang="%s"><head><meta charset="utf-8">`+
		`<meta http-equiv="refresh" content="30">`+
		`<meta name="viewport" content="width=device-width,initial-scale=1"><title>%s</title><style>%s</style></head><body>`,
		wallEsc(lang), wallEsc(in.Name), render.DefaultStyle)
	page.WriteString(b.String())
	fmt.Fprintf(&page, `<p class="credit">%s</p></body></html>`, wallEsc(credit))
	return page.String()
}

// wallTerm reads a key of the host catalogue, or falls back to a French sentence — the wall
// page's own vocabulary is not part of the engine's Term set, which the per-event page draws
// from.
func wallTerm(cat *Catalog, key, fallback string, params map[string]any) string {
	if s, ok := cat.t(key, params); ok {
		return s
	}
	return interpolate(fallback, params)
}

// WriteWallPage writes the wall page into dir, atomically, and returns the path written.
func WriteWallPage(dir, page string) (string, error) {
	return WriteFileAtomically(dir, WallPageName, page)
}
