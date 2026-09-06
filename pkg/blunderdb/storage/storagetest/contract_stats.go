// Contract cases for statistics: aggregate counts, cube directions and the
// Snowie denominator.
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testStatsAggregateCounts checks the StatsStore wiring and its behaviour on an
// empty database. Rich correctness (PR/MWC/Snowie aggregation against real
// matches) is covered by the SQLite parity test against the legacy Database
// implementation. Backends that have not implemented Stats yet return
// ErrInternal ("not implemented"); the case skips for them so it lights up
// automatically once the family lands.
func testStatsAggregateCounts(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ss := s.Stats()

	dr, err := ss.DateRange(ctx, "")
	if errors.Is(err, storage.ErrInternal) {
		t.Skip("Stats not implemented on this backend")
	}
	if err != nil {
		t.Fatalf("DateRange: %v", err)
	}
	if dr.DateFrom != "" || dr.DateTo != "" {
		t.Errorf("empty DateRange: got %+v, want both empty", dr)
	}

	res, err := ss.Compute(ctx, "", storage.StatsFilter{DecisionType: -1})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if res == nil {
		t.Fatal("Compute returned nil result")
	}
	if res.Totals != (storage.StatsTotals{}) {
		t.Errorf("empty totals: got %+v, want zero", res.Totals)
	}
	if res.PRGlobal != 0 || res.MWCAvailable {
		t.Errorf("empty result: PRGlobal=%v MWCAvailable=%v, want 0/false", res.PRGlobal, res.MWCAvailable)
	}

	players, err := ss.PlayerNames(ctx, "")
	if err != nil {
		t.Fatalf("PlayerNames: %v", err)
	}
	if len(players) != 0 {
		t.Errorf("empty PlayerNames: got %v, want none", players)
	}

	ids, err := ss.PositionIDsBySelection(ctx, "",
		storage.StatsFilter{DecisionType: -1}, storage.SelectionSpec{Kind: "all"})
	if err != nil {
		t.Fatalf("PositionIDsBySelection: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("empty selection: got %v, want none", ids)
	}

	mb, err := ss.MatchBadges(ctx, "", nil)
	if err != nil {
		t.Fatalf("MatchBadges: %v", err)
	}
	if len(mb) != 0 {
		t.Errorf("empty MatchBadges: got %v, want none", mb)
	}

	tb, err := ss.TournamentBadges(ctx, "")
	if err != nil {
		t.Fatalf("TournamentBadges: %v", err)
	}
	if len(tb) != 0 {
		t.Errorf("empty TournamentBadges: got %v, want none", tb)
	}
}

// statsDicePairs are the 21 distinct unordered dice pairs (faces 1..6,
// doubles included). Zobrist hashes dice as an unordered pair (see
// engine/zobrist.go), so this list is exactly the set of values that cannot
// collide with one another.
var statsDicePairs = [][2]int{
	{1, 1}, {1, 2}, {1, 3}, {1, 4}, {1, 5}, {1, 6},
	{2, 2}, {2, 3}, {2, 4}, {2, 5}, {2, 6},
	{3, 3}, {3, 4}, {3, 5}, {3, 6},
	{4, 4}, {4, 5}, {4, 6},
	{5, 5}, {5, 6},
	{6, 6},
}

// statsDecisionPos returns a position unique to slot via its dice (rather
// than score, unlike provenancePos): the stats fixtures below need a
// realistic, non-degenerate score so the MWC-loss computation doesn't hit an
// edge case, so uniqueness comes from statsDicePairs instead.
// statsCubeDecision saves one cube decision: a position, the move that carries
// the action actually played, and the analysis carrying the engine's ruling.
//
// The equities are what make the fixture real rather than decorative. gnuBG's
// "close cube" predicate (engine.ComputeIsCloseCube) compares the optimal line
// with the double/take line, and statsCountedExpr drops non-close no-doubles —
// so a fixture with careless equities would be silently excluded from every
// aggregate and the test would pass on an empty set.
func statsCubeDecision(t *testing.T, s storage.Storage, gameID int64, slot int,
	player int32, played, best string, ndEq, dtEq, dpEq float64) {
	t.Helper()
	ctx := context.Background()

	pos := statsDecisionPos(t, slot)
	pos.DecisionType = domain.CubeAction
	posID, err := s.Positions().Save(ctx, "", &pos)
	if err != nil {
		t.Fatalf("Save cube position (slot %d): %v", slot, err)
	}
	mv := domain.Move{GameID: gameID, MoveNumber: int32(slot), MoveType: "cube",
		PositionID: posID, Player: player, CubeAction: played}
	if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
		t.Fatalf("CreateMove (slot %d): %v", slot, err)
	}
	a := domain.PositionAnalysis{
		PlayedCubeActions: []string{played},
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
			BestCubeAction:          best,
			CubefulNoDoubleEquity:   ndEq,
			CubefulDoubleTakeEquity: dtEq,
			CubefulDoublePassEquity: dpEq,
			CubefulNoDoubleError:    -0.100,
			CubefulDoubleTakeError:  -0.100,
			CubefulDoublePassError:  -0.100,
		},
	}
	if err := s.Analyses().Save(ctx, "", posID, &a); err != nil {
		t.Fatalf("Save cube analysis (slot %d): %v", slot, err)
	}
}

// testStatsCubeDirections is the parity gate for the cube-direction matrix: the
// SQL that reads the raw (best, played) cells differs between the backends, the
// classification behind it must not. It also pins the two rulings a single
// bestCubeAction carries — a missed double and a wrong pass are different
// players' mistakes on different axes.
func testStatsCubeDirections(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	if _, err := s.Stats().DateRange(ctx, ""); errors.Is(err, storage.ErrInternal) {
		t.Skip("Stats not implemented on this backend")
	}

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7,
		MatchDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	gameID, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1})
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	// Offer axis. "Double, Take" makes the decision close by construction
	// (optimal line == double/take line), which is what keeps the missed double
	// inside statsCountedExpr.
	statsCubeDecision(t, s, gameID, 0, 1, "No Double", "Double, Take", 0.40, 0.55, 1.00) // missed
	statsCubeDecision(t, s, gameID, 1, 1, "Double", "No Double", 0.60, 0.45, 1.00)       // premature
	statsCubeDecision(t, s, gameID, 2, 1, "Double", "Double, Take", 0.40, 0.55, 1.00)    // right
	// Answer axis.
	statsCubeDecision(t, s, gameID, 3, -1, "Pass", "Double, Take", 0.40, 0.55, 1.00) // wrong pass
	statsCubeDecision(t, s, gameID, 4, -1, "Take", "Double, Pass", 0.90, 1.30, 1.00) // wrong take
	statsCubeDecision(t, s, gameID, 5, -1, "Pass", "Double, Pass", 0.90, 1.30, 1.00) // right

	res, err := s.Stats().Compute(ctx, "", storage.StatsFilter{DecisionType: -1})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	d := res.CubeDirections
	if d.Offer.Missed != 1 || d.Offer.Premature != 1 || d.Offer.Right != 1 {
		t.Errorf("offer axis: got %+v, want 1 right / 1 missed / 1 premature", d.Offer)
	}
	if d.Answer.WrongPass != 1 || d.Answer.WrongTake != 1 || d.Answer.Right != 1 {
		t.Errorf("answer axis: got %+v, want 1 right / 1 wrong-pass / 1 wrong-take", d.Answer)
	}

	// Every cell must be clickable: a figure that leads nowhere is the defect
	// this panel already avoids everywhere else ("ce qu'on clique = ce qu'on
	// voit"). One decision per cell in this fixture, so one id each.
	for _, cell := range []string{
		storage.CubeCellOfferRight, storage.CubeCellOfferMissed, storage.CubeCellOfferPremature,
		storage.CubeCellAnswerRight, storage.CubeCellAnswerWrongPass, storage.CubeCellAnswerWrongTake,
	} {
		ids, err := s.Stats().PositionIDsBySelection(ctx, "",
			storage.StatsFilter{DecisionType: -1},
			storage.SelectionSpec{Kind: "cube_direction", CubeCell: cell})
		if err != nil {
			t.Fatalf("PositionIDsBySelection(%s): %v", cell, err)
		}
		if len(ids) != 1 {
			t.Errorf("PositionIDsBySelection(%s): got %d ids, want 1", cell, len(ids))
		}
	}
	// A drill-down naming no cell selects nothing: returning everything would
	// pass for a working feature.
	ids, err := s.Stats().PositionIDsBySelection(ctx, "",
		storage.StatsFilter{DecisionType: -1}, storage.SelectionSpec{Kind: "cube_direction"})
	if err != nil {
		t.Fatalf("PositionIDsBySelection(no cell): %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("PositionIDsBySelection(no cell): got %d ids, want none", len(ids))
	}

	// A player who signed under several spellings must be matched under all of
	// them. Asking for "Alice" plus an alias that happens to be Bob is the
	// sharpest check that aliases are really OR-ed in rather than ignored: the
	// two seats together must give back exactly the unfiltered result. The
	// duplicate alias also pins that repeating a name does not double-count.
	both, err := s.Stats().Compute(ctx, "", storage.StatsFilter{
		DecisionType: -1, PlayerName: "Alice", PlayerAliases: []string{"Bob", "Alice"},
	})
	if err != nil {
		t.Fatalf("Compute(aliases): %v", err)
	}
	if both.CubeDirections != d {
		t.Errorf("aliases covering both seats: got %+v, want the unfiltered %+v", both.CubeDirections, d)
	}

	// Scoping to one player must split the two axes: in this fixture Alice only
	// ever holds the cube and Bob only ever answers.
	alice, err := s.Stats().Compute(ctx, "", storage.StatsFilter{DecisionType: -1, PlayerName: "Alice"})
	if err != nil {
		t.Fatalf("Compute(Alice): %v", err)
	}
	if alice.CubeDirections.Answer != (storage.CubeAnswerCounts{}) {
		t.Errorf("Alice never answered a cube, got %+v", alice.CubeDirections.Answer)
	}
	if alice.CubeDirections.Offer.Missed != 1 {
		t.Errorf("Alice's offer axis: got %+v, want the missed double", alice.CubeDirections.Offer)
	}
}

// testStatsSnowieDenominator pins what makes the Snowie rate the Snowie rate:
// one player's errors divided by BOTH players' checker moves (gnuBG
// formatgs.c:415-424). Filtering on a player must narrow the errors, never the
// move count — the denominator only follows the matches retained.
//
// The additivity check is what catches the defect: in a database holding one
// Alice-vs-Bob match, their two filtered rates must add up to the unfiltered
// one, since they share a denominator. When the filter also narrowed the move
// count, each player was divided by their own moves alone and every filtered
// Snowie ER came out roughly twice too big.
func testStatsSnowieDenominator(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	if _, err := s.Stats().DateRange(ctx, ""); errors.Is(err, storage.ErrInternal) {
		t.Skip("Stats not implemented on this backend")
	}
	matchID, _ := statsFixtureMatch(t, s, 0, "Alice", "Bob")

	snowie := func(player string) float64 {
		t.Helper()
		res, err := s.Stats().Compute(ctx, "",
			storage.StatsFilter{DecisionType: -1, PlayerName: player})
		if err != nil {
			t.Fatalf("Compute(%q): %v", player, err)
		}
		return res.SnowieGlobal
	}

	all, alice, bob := snowie(""), snowie("Alice"), snowie("Bob")
	if all <= 0 {
		t.Fatalf("unfiltered Snowie ER = %v, want > 0 (the fixture records real errors)", all)
	}
	if math.Abs((alice+bob)-all) > 1e-9 {
		t.Errorf("Snowie ER per player: Alice=%v + Bob=%v = %v, want the unfiltered %v"+
			" — a filtered rate must keep both players in the denominator",
			alice, bob, alice+bob, all)
	}

	// Same figure, other screen: the per-match detail already divided by both
	// players' moves. On a single-match database the global rate must agree
	// with it, or the panel contradicts the match table.
	detail, err := s.Stats().MatchDetail(ctx, "", matchID)
	if err != nil {
		t.Fatalf("MatchDetail: %v", err)
	}
	if math.Abs(detail.Player1.SnowieER-alice) > 1e-9 {
		t.Errorf("Snowie ER for Alice: global %v, match detail %v — the two screens disagree",
			alice, detail.Player1.SnowieER)
	}
}

// testStatsBreakdowns pins the three breakdowns of #266: the same figures as
// the global ones, sliced by phase, by away × away score, and by tag.
//
// What it checks above all is that a slice cannot disagree with the whole:
// every breakdown must count the same decisions the global PR counts, because
// all of them read the same countedExpr. A breakdown that restated what counts
// as a decision would be a second PR under the same name.
func testStatsBreakdowns(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	// Two decisions of one match, one per player, both carrying an error.
	_, posIDs := statsFixtureMatch(t, s, 0, "Alice", "Bob")
	if _, err := s.Comments().Add(ctx, "", posIDs[0], "trop passif ici #timing #blitz"); err != nil {
		t.Fatalf("Add comment: %v", err)
	}

	res, err := s.Stats().Compute(ctx, "", storage.StatsFilter{DecisionType: -1})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	// Every breakdown sums to the same number of decisions the totals report —
	// except the tag one, which is a label and not a partition.
	var phaseDecisions int
	for _, p := range res.PerPhase {
		phaseDecisions += p.NumDecisions
	}
	if phaseDecisions != res.Totals.NumDecisions {
		t.Errorf("the per-phase breakdown counts %d decisions, the totals %d",
			phaseDecisions, res.Totals.NumDecisions)
	}
	var scoreDecisions int
	for _, c := range res.PerScore {
		scoreDecisions += c.NumDecisions
	}
	if scoreDecisions != res.Totals.NumDecisions {
		t.Errorf("the per-score breakdown counts %d decisions, the totals %d",
			scoreDecisions, res.Totals.NumDecisions)
	}
	// The score cell is (mover's away, opponent's away), read off the
	// NORMALISED position — the fixture plays at 4-away/4-away.
	if len(res.PerScore) == 0 {
		t.Fatal("the away x away matrix is empty though decisions were counted")
	}
	for _, c := range res.PerScore {
		if c.MoverAway != 4 || c.OpponentAway != 4 {
			t.Errorf("unexpected score cell %d-away/%d-away", c.MoverAway, c.OpponentAway)
		}
	}

	// The tags of one comment become two rows, each counting the ONE decision
	// of the position that carries them — a tag labels, it does not partition.
	tags := map[string]storage.TagStats{}
	for _, tag := range res.PerTag {
		tags[tag.Tag] = tag
	}
	for _, want := range []string{"#timing", "#blitz"} {
		got, ok := tags[want]
		if !ok {
			t.Errorf("the per-tag breakdown does not list %s", want)
			continue
		}
		if got.NumDecisions != 1 {
			t.Errorf("%s counts %d decisions, want 1", want, got.NumDecisions)
		}
	}
	if len(res.PerTag) != 2 {
		t.Errorf("the per-tag breakdown has %d rows, want 2", len(res.PerTag))
	}
}
