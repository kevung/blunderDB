package direction

import (
	"fmt"
	"html"
	"strings"
	"time"

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
	Number int
	// Name is shown beside the number; Room groups the lines when the Rencontre has rooms.
	Name        string
	Room        string
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
	// Statuses answer "am I playing?" for the event's players (Direction.Statuses).
	Statuses []PlayerStatus
}

// WallPageInput is everything the wall page needs. The caller — the database layer, which knows
// the Rencontre and replays each member's Direction — gathers it; this package only renders.
type WallPageInput struct {
	Name   string
	Tables []WallTable
	Events []WallEvent
	// Brackets are the events currently in a bracket phase; each gets a view of its own, in
	// rotation with the tables. None: the page is the plain one.
	Brackets []WallBracket
	// Now places the rotation on the wall clock; it is the render time.
	Now time.Time
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
	// One list per room, in the order of their first table, when tables carry rooms; tables in
	// no room close the page under no heading (ADR-0058 §12).
	var rooms []string
	byRoom := map[string][]WallTable{}
	for _, tb := range in.Tables {
		if _, seen := byRoom[tb.Room]; !seen && tb.Room != "" {
			rooms = append(rooms, tb.Room)
		}
		byRoom[tb.Room] = append(byRoom[tb.Room], tb)
	}
	if _, ok := byRoom[""]; ok || len(rooms) == 0 {
		rooms = append(rooms, "")
	}
	for _, room := range rooms {
		if room != "" {
			fmt.Fprintf(&b, `<h2>%s</h2>`, wallEsc(room))
		}
		wallTables(&b, byRoom[room], l)
	}
	if len(in.Events) > 0 {
		b.WriteString(`<ul class="events">`)
		for _, ev := range in.Events {
			rounds := wallTerm(cat, "rencontre.wall.rounds", "{n} rondes annoncées", map[string]any{"n": ev.Rounds})
			fmt.Fprintf(&b, `<li><a href="%s/%s">%s</a> — %s</li>`,
				wallEsc(ev.Slug), PageName, wallEsc(ev.Name), wallEsc(rounds))
		}
		b.WriteString(`</ul>`)
	}
	for _, ev := range in.Events {
		b.WriteString(StatusBlock(ev.Statuses, ev.Name, cat))
	}

	style := render.DefaultStyle
	content := b.String()
	if views := wallBracketViews(in, l); len(views) > 0 {
		var css string
		css, content = rotation(append([]string{content}, views...), in.Now)
		style += css
	}

	var page strings.Builder
	fmt.Fprintf(&page, `<!doctype html><html lang="%s"><head><meta charset="utf-8">`+
		`<meta http-equiv="refresh" content="%d">`+
		`<meta name="viewport" content="width=device-width,initial-scale=1"><title>%s</title><style>%s</style></head><body>`,
		wallEsc(lang), minRefresh, wallEsc(in.Name), style)
	page.WriteString(content)
	fmt.Fprintf(&page, `<p class="credit">%s</p></body></html>`, wallEsc(credit))
	return page.String()
}

// wallTables writes one list of table lines, each number followed by the table's name when it
// has one.
func wallTables(b *strings.Builder, tables []WallTable, l render.Labeler) {
	b.WriteString(`<ul class="tables">`)
	for _, tb := range tables {
		classe, contenu := "libre", wallEsc(l.Term(render.TermFree, 0))
		switch {
		case tb.Unavailable:
			classe, contenu = "hs", wallEsc(l.Term(render.TermUnavailable, 0))
		case tb.Event != "":
			classe = "occupee"
			contenu = fmt.Sprintf(`<span class="evt">%s</span><br>%s %s %s`,
				wallEsc(tb.Event), wallEsc(tb.A), wallEsc(l.Term(render.TermVersus, 0)), wallEsc(tb.B))
		}
		num := fmt.Sprintf("%s %d", l.Term(render.TermTable, 0), tb.Number)
		if tb.Name != "" {
			num += " — " + tb.Name
		}
		fmt.Fprintf(b, `<li class="%s"><span class="num">%s</span>%s</li>`, classe, wallEsc(num), contenu)
	}
	b.WriteString(`</ul>`)
}

// wallBracketViews renders one view per event that has a bracket to show.
func wallBracketViews(in WallPageInput, l render.Labeler) []string {
	var views []string
	for _, br := range in.Brackets {
		if svg := BracketSVG(br, l); svg != "" {
			views = append(views, fmt.Sprintf(`<h1>%s — %s</h1>%s`, wallEsc(in.Name), wallEsc(br.Event), svg))
		}
	}
	return views
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
