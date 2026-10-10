package apkg

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"html"
	"io"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
	"github.com/kevung/blunderdb/pkg/blunderdb/report"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Source names what is exported: a study deck of blunderDB, or a collection.
// Exactly one of the two is set.
type Source struct {
	DeckID       int64 `json:"deckId,omitempty"`
	CollectionID int64 `json:"collectionId,omitempty"`
}

// Result reports one export. Truncated is set when a living collection
// selects more positions than its declared ceiling: the package then holds the
// first Notes of Total.
type Result struct {
	Name      string `json:"name"`
	Notes     int    `json:"notes"`
	Total     int    `json:"total"`
	Truncated bool   `json:"truncated"`
}

// loadBatch bounds one LoadByIDs call of a living collection.
const loadBatch = 500

// Export writes the .apkg of src to w, its cards written in lang. It reads and
// never writes: a deck is exported as it stands, without the resynchronisation
// a study session performs, and a living collection's query is evaluated now.
func Export(ctx context.Context, st storage.Storage, scope string, src Source, lang string, now time.Time, w io.Writer) (*Result, error) {
	if (src.DeckID == 0) == (src.CollectionID == 0) {
		return nil, fmt.Errorf("apkg: name a deck or a collection: %w", storage.ErrInvalid)
	}
	res := &Result{}
	var positions []domain.Position
	var err error
	if src.DeckID != 0 {
		res.Name, positions, err = deckPositions(ctx, st, scope, src.DeckID)
	} else {
		res.Name, positions, err = collectionPositions(ctx, st, scope, src.CollectionID, res)
	}
	if err != nil {
		return nil, err
	}
	md, err := st.Metadata().Load(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("apkg: metadata: %w", err)
	}
	pkg := Package{
		DeckID:          DeckID(res.Name),
		DeckName:        res.Name,
		DeckDescription: description(issuance.Carried(md)),
		Modified:        now,
	}
	seen := map[string]bool{}
	for i := range positions {
		p := &positions[i]
		a, err := st.Analyses().Load(ctx, scope, p.ID)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return nil, fmt.Errorf("apkg: analysis of position %d: %w", p.ID, err)
		}
		n, m := card(p, a, lang)
		// One note per position: a position listed twice is one card.
		if seen[n.GUID] {
			continue
		}
		seen[n.GUID] = true
		pkg.Notes = append(pkg.Notes, n)
		pkg.Media = append(pkg.Media, m)
	}
	res.Notes = len(pkg.Notes)
	res.Total = max(res.Total, res.Notes)
	return res, Write(ctx, w, pkg)
}

func deckPositions(ctx context.Context, st storage.Storage, scope string, deckID int64) (string, []domain.Position, error) {
	var name string
	found := false
	for d, err := range st.Anki().ListDecks(ctx, scope) {
		if err != nil {
			return "", nil, err
		}
		if d.ID == deckID {
			name, found = d.Name, true
			break
		}
	}
	if !found {
		return "", nil, fmt.Errorf("anki deck %d: %w", deckID, storage.ErrNotFound)
	}
	var out []domain.Position
	for p, err := range st.Anki().DeckPositions(ctx, scope, deckID) {
		if err != nil {
			return "", nil, err
		}
		out = append(out, *p)
	}
	return name, out, nil
}

func collectionPositions(ctx context.Context, st storage.Storage, scope string, id int64, res *Result) (string, []domain.Position, error) {
	c, err := st.Collections().Get(ctx, scope, id)
	if err != nil {
		return "", nil, err
	}
	_, _, living, err := storage.LivingFilters(ctx, st, scope, id)
	if err != nil {
		return "", nil, err
	}
	var out []domain.Position
	if !living {
		for p, err := range st.Collections().Positions(ctx, scope, id, storage.ListOpts{}) {
			if err != nil {
				return "", nil, err
			}
			out = append(out, *p)
		}
		return c.Name, out, nil
	}
	ev, err := storage.EvaluateCollection(ctx, st, scope, id, 0)
	if err != nil {
		return "", nil, err
	}
	res.Total, res.Truncated = ev.Total, ev.Truncated
	for start := 0; start < len(ev.PositionIDs); start += loadBatch {
		ids := ev.PositionIDs[start:min(start+loadBatch, len(ev.PositionIDs))]
		ps, err := st.Positions().LoadByIDs(ctx, scope, ids)
		if err != nil {
			return "", nil, err
		}
		out = append(out, ps...)
	}
	return c.Name, out, nil
}

// description states the producer's carried metadata in the deck's
// description, so the package says who made it.
func description(md map[string]string) string {
	var parts []string
	for _, k := range issuance.CarriedMetadataKeys {
		if v := md[k]; v != "" {
			parts = append(parts, html.EscapeString(v))
		}
	}
	return strings.Join(parts, "<br>")
}

// DeckID is the Anki deck id of a package named name: derived from the name,
// so a re-export lands in the deck the first import created.
func DeckID(name string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte("blunderdb-deck:" + name))
	return stableID(h.Sum64())
}

// stableID folds a 64-bit hash into a positive id below 2^53, the integers
// every Anki client (JavaScript included) holds exactly. Zero and one are
// Anki's own default ids, so they are avoided.
func stableID(h uint64) int64 {
	return int64(h&(1<<53-1)) | 1<<52
}

// NoteID and NoteGUID are a position's note identity, derived from its
// Zobrist hash: a re-export updates the note instead of duplicating it.
func NoteID(p *domain.Position) int64 { return stableID(engine.ZobristHash(p)) }

// NoteGUID is the guid Anki matches a re-imported note on.
func NoteGUID(p *domain.Position) string {
	return fmt.Sprintf("blunderdb-%016x", engine.ZobristHash(p))
}

// card writes one position's note and its board image.
func card(p *domain.Position, a *domain.PositionAnalysis, lang string) (Note, Media) {
	hash := engine.ZobristHash(p)
	img := fmt.Sprintf("blunderdb-%016x.svg", hash)
	ident := fmt.Sprintf("%016x", hash)
	if a != nil && a.XGID != "" {
		ident = a.XGID
	}
	answer, equity, played := back(p, a, lang)
	kind := "checker"
	if p.DecisionType == domain.CubeAction {
		kind = "cube"
	}
	n := Note{
		ID:   NoteID(p),
		GUID: NoteGUID(p),
		Fields: []string{
			html.EscapeString(ident),
			fmt.Sprintf(`<img src="%s">`, img),
			situation(p, lang),
			answer, equity, played,
		},
		Tags: []string{"blunderDB", "blunderDB::" + kind},
	}
	return n, Media{Name: img, Data: []byte(report.Diagram(p))}
}

// situation is the front's text: score, cube, dice and the side on roll. The
// diagram puts the player on roll at the bottom, so the score reads from them.
func situation(p *domain.Position, lang string) string {
	var lines []string
	if p.IsMoney() {
		lines = append(lines, label(lang, "money"))
	} else {
		me, opp := p.Score[0], p.Score[1]
		if p.PlayerOnRoll == domain.White {
			me, opp = opp, me
		}
		lines = append(lines, fmt.Sprintf("%s : %s – %s", label(lang, "score"),
			fmt.Sprintf(label(lang, "away"), me), fmt.Sprintf(label(lang, "away"), opp)))
	}
	cube := fmt.Sprintf("%s : %d", label(lang, "cube"), 1<<max(p.Cube.Value, 0))
	switch p.Cube.Owner {
	case p.PlayerOnRoll:
		cube += " " + label(lang, "ownOnRoll")
	case domain.None:
		cube += " " + label(lang, "centred")
	default:
		cube += " " + label(lang, "ownOpp")
	}
	lines = append(lines, cube)
	if p.DecisionType == domain.CubeAction {
		lines = append(lines, label(lang, "cubeDecision"))
	} else if p.Dice[0] > 0 && p.Dice[1] > 0 {
		lines = append(lines, fmt.Sprintf(label(lang, "toPlay"), p.Dice[0], p.Dice[1]))
	}
	lines = append(lines, label(lang, "onRoll"))
	for i := range lines {
		lines[i] = html.EscapeString(lines[i])
	}
	return strings.Join(lines, "<br>")
}

// back is the verso: the right decision, the equities it rests on and the
// error of what was played, each "" when the analysis does not say.
func back(p *domain.Position, a *domain.PositionAnalysis, lang string) (answer, equity, played string) {
	if p.DecisionType == domain.CubeAction {
		if a == nil || a.DoublingCubeAnalysis == nil {
			return html.EscapeString(label(lang, "noAnalysis")), "", ""
		}
		return cubeBack(a, lang)
	}
	if a == nil || a.CheckerAnalysis == nil || len(a.CheckerAnalysis.Moves) == 0 {
		return html.EscapeString(label(lang, "noAnalysis")), "", ""
	}
	moves := a.CheckerAnalysis.Moves
	best := moves[0]
	answer = html.EscapeString(best.Move)
	equity = fmt.Sprintf("%s : %+.3f", html.EscapeString(label(lang, "equity")), best.Equity)
	if pms := playedMoves(a); len(pms) > 0 {
		pm := pms[0]
		played = fmt.Sprintf("%s : %s", html.EscapeString(label(lang, "played")), html.EscapeString(pm))
		for _, m := range moves {
			if engine.CanonicalMove(m.Move) == engine.CanonicalMove(pm) {
				played += fmt.Sprintf(" (%s %.3f)", html.EscapeString(label(lang, "error")), best.Equity-m.Equity)
				break
			}
		}
	}
	return answer, equity, played
}

func cubeBack(a *domain.PositionAnalysis, lang string) (answer, equity, played string) {
	d := a.DoublingCubeAnalysis
	key := ""
	if v, ok := engine.BestCubeVerdict(d.BestCubeAction); ok {
		switch {
		case v.ShouldDouble && v.ShouldPass:
			key = "dp"
		case v.ShouldDouble:
			key = "dt"
		case v.ShouldPass:
			key = "tg"
		default:
			key = "nd"
		}
	}
	if key != "" {
		answer = html.EscapeString(label(lang, key))
	} else {
		answer = html.EscapeString(d.BestCubeAction)
	}
	row := func(k string, v float64) string {
		return fmt.Sprintf("<tr><td>%s</td><td>%+.3f</td></tr>", html.EscapeString(label(lang, k)), v)
	}
	equity = "<table>" + row("ndEq", d.CubefulNoDoubleEquity) + row("dtEq", d.CubefulDoubleTakeEquity) + row("dpEq", d.CubefulDoublePassEquity) + "</table>"
	actions := a.PlayedCubeActions
	if len(actions) == 0 && a.PlayedCubeAction != "" {
		actions = []string{a.PlayedCubeAction}
	}
	if len(actions) > 0 {
		played = fmt.Sprintf("%s : %s", html.EscapeString(label(lang, "played")), html.EscapeString(actions[0]))
		if e, ok := engine.CubeActionError(d, actions[0]); ok {
			played += fmt.Sprintf(" (%s %.3f)", html.EscapeString(label(lang, "error")), e)
		}
	}
	return answer, equity, played
}

func playedMoves(a *domain.PositionAnalysis) []string {
	if len(a.PlayedMoves) > 0 {
		return a.PlayedMoves
	}
	if a.PlayedMove != "" {
		return []string{a.PlayedMove}
	}
	return nil
}
