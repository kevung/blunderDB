// Package report builds the self-contained HTML statistics report: one file,
// no external image, style sheet or script, diagrams inlined as SVG, printable
// to PDF by any browser. A report is data, not a view: the desktop app, the CLI
// (`stats report`) and the daemon (`stats.report`) all come through Build, so
// the three cannot drift. The report names its scope (the filter) because
// figures that do not say what they count mean nothing.
package report

import (
	"context"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TopBlunders is how many decisions the report details: enough to see a
// pattern, few enough for a document people read.
const TopBlunders = 10

// DefaultLanguage is used when the requested language has no catalogue.
const DefaultLanguage = "en"

// Languages lists the languages a report can be written in.
func Languages() []string {
	return []string{"fr", "en", "de", "el", "es", "fi", "it", "ja", "ru"}
}

// Diagrammer draws a position as a standalone SVG document. The desktop app
// supplies its own, which follows the board palette on screen; Diagram is the
// one the CLI and the daemon use.
type Diagrammer func(p *domain.Position) string

// Blunder is one decision the report details.
type Blunder struct {
	Rank     int
	Kind     int // storage.BlunderEntry.DecisionType: 0 checker, 1 cube
	ErrorMP  int64
	Players  string
	Date     string
	Diagram  string // SVG, "" when the position is unavailable
	BestMove string // "" when no analysis names one
}

// Data is everything Render writes.
type Data struct {
	Language  string
	Generated time.Time
	Totals    storage.StatsTotals
	PRGlobal  float64
	PRChecker float64
	PRCube    float64
	Blunders  []Blunder
}

// Build computes the statistics of filter in scope and renders the report.
// diagram nil means the built-in Diagram.
func Build(ctx context.Context, s storage.Storage, scope string, filter storage.StatsFilter, lang string, diagram Diagrammer) (string, error) {
	stats, err := s.Stats().Compute(ctx, scope, filter)
	if err != nil {
		return "", fmt.Errorf("report: compute stats: %w", err)
	}
	if diagram == nil {
		diagram = Diagram
	}
	d := Data{
		Language:  lang,
		Generated: time.Now(),
		Totals:    stats.Totals,
		PRGlobal:  stats.PRGlobal,
		PRChecker: stats.PRChecker,
		PRCube:    stats.PRCube,
	}
	top := stats.TopBlunders
	if len(top) > TopBlunders {
		top = top[:TopBlunders]
	}
	var ids []int64
	for _, b := range top {
		if b.PositionID > 0 {
			ids = append(ids, b.PositionID)
		}
	}
	byID := map[int64]*domain.Position{}
	if len(ids) > 0 {
		positions, err := s.Positions().LoadByIDs(ctx, scope, ids)
		if err != nil {
			return "", fmt.Errorf("report: load positions: %w", err)
		}
		for i := range positions {
			byID[positions[i].ID] = &positions[i]
		}
	}
	for i, b := range top {
		item := Blunder{Rank: i + 1, Kind: b.DecisionType, ErrorMP: b.ErrorMP, Players: b.PlayerNames, Date: b.MatchDate}
		if p := byID[b.PositionID]; p != nil {
			item.Diagram = diagram(p)
		}
		// An unreadable analysis costs the section its best-move line, not
		// the report.
		if a, err := s.Analyses().Load(ctx, scope, b.PositionID); err == nil && a != nil &&
			a.CheckerAnalysis != nil && len(a.CheckerAnalysis.Moves) > 0 {
			item.BestMove = a.CheckerAnalysis.Moves[0].Move
		}
		d.Blunders = append(d.Blunders, item)
	}
	return Render(d), nil
}

func label(lang, key string) string {
	if m, ok := labels[lang]; ok {
		return m[key]
	}
	return labels[DefaultLanguage][key]
}

// Render writes the document. Every fragment that comes from imported data is
// escaped one by one: the body carries SVG, so no blanket escaping.
func Render(d Data) string {
	lang := d.Language
	if _, ok := labels[lang]; !ok {
		lang = DefaultLanguage
	}
	esc := html.EscapeString
	var b strings.Builder
	title := label(lang, "title")
	fmt.Fprintf(&b, "<!doctype html>\n<html lang=\"%s\">\n<head>\n<meta charset=\"utf-8\">\n<title>%s</title>\n<style>\n%s</style>\n</head>\n<body>\n", esc(lang), esc(title), css)
	fmt.Fprintf(&b, "<h1>%s</h1>\n", esc(title))
	date := d.Generated.Format("2006-01-02 15:04")
	fmt.Fprintf(&b, "<p class=\"meta\">%s</p>\n<table>\n", esc(strings.ReplaceAll(label(lang, "generated"), "{date}", date)))
	rows := [][2]string{
		{label(lang, "positions"), fmt.Sprint(d.Totals.NumPositions)},
		{label(lang, "matches"), fmt.Sprint(d.Totals.NumMatches)},
		{label(lang, "decisions"), fmt.Sprint(d.Totals.NumDecisions)},
		{label(lang, "prGlobal"), pr(d.PRGlobal)},
		{label(lang, "prChecker"), pr(d.PRChecker)},
		{label(lang, "prCube"), pr(d.PRCube)},
	}
	for _, r := range rows {
		fmt.Fprintf(&b, "  <tr><td>%s</td><td class=\"value\">%s</td></tr>\n", esc(r[0]), esc(r[1]))
	}
	fmt.Fprintf(&b, "</table>\n<h2>%s</h2>\n", esc(label(lang, "worst")))
	if len(d.Blunders) == 0 {
		fmt.Fprintf(&b, "<p>%s</p>\n", esc(label(lang, "noBlunder")))
	}
	for _, x := range d.Blunders {
		kind := label(lang, "checker")
		if x.Kind == 1 {
			kind = label(lang, "cube")
		}
		meta := esc(x.Players)
		if len(x.Date) >= 10 {
			meta += " · " + esc(x.Date[:10])
		}
		fmt.Fprintf(&b, "\n<section class=\"blunder\">\n  <h3>%d. %s — %.3f</h3>\n  <p class=\"meta\">%s</p>\n  <div class=\"diagram\">%s</div>\n", x.Rank, esc(kind), float64(x.ErrorMP)/1000, meta, x.Diagram)
		if x.BestMove != "" {
			fmt.Fprintf(&b, "  <p class=\"best\">%s <b>%s</b></p>\n", esc(label(lang, "bestMove")), esc(x.BestMove))
		}
		b.WriteString("</section>\n")
	}
	fmt.Fprintf(&b, "<p class=\"footer\">%s</p>\n</body>\n</html>\n", esc(label(lang, "footer")))
	return b.String()
}

func pr(v float64) string {
	if v > 0 {
		return fmt.Sprintf("%.2f", v)
	}
	return "—"
}

const css = `  body { font-family: system-ui, sans-serif; margin: 2rem auto; max-width: 46rem; color: #222; }
  h1 { font-size: 1.4rem; }
  h3 { font-size: 1rem; margin-bottom: 0.2rem; }
  table { border-collapse: collapse; margin: 1rem 0; }
  td { padding: 0.15rem 1rem 0.15rem 0; }
  td.value { font-weight: 600; text-align: right; }
  .meta { color: #666; margin: 0 0 0.4rem 0; font-size: 0.9rem; }
  .diagram svg { max-width: 100%; height: auto; }
  .blunder { break-inside: avoid; page-break-inside: avoid; margin-bottom: 1.6rem; }
  .footer { color: #666; font-size: 0.85rem; margin-top: 2rem; }
  @media print { body { margin: 0; max-width: none; } }
`
