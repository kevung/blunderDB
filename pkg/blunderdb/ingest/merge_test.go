package ingest

import (
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// This file replaces tests/analysis_merge_test.go, whose four tests were
// t.Log-only placeholders documenting intended behavior with no assertions.
// The real merge/dedup/sort logic they described (mergeCheckerMoves,
// mergePlayedMoves, sortCheckerMovesByEquity, mergeAnalysis) is unexported in
// this package, so an external "tests" package can never reach it directly —
// these tests replace the placeholders in the one package that can actually
// exercise the code. Position-level Zobrist dedup (the other half of what the
// old TestPositionDeduplication described) is covered directly in
// pkg/blunderdb/domain/position_match_test.go (NormalizeForStorage) and
// pkg/blunderdb/engine/zobrist_test.go.

// --- mergeCheckerMoves -------------------------------------------------------

func TestMergeCheckerMovesSortsByEquityDescending(t *testing.T) {
	incoming := []domain.CheckerMove{
		{Move: "13/7", Equity: -0.100, AnalysisDepth: "2-ply"},
		{Move: "24/18 13/7", Equity: 0.050, AnalysisDepth: "2-ply"},
		{Move: "6/1 6/1", Equity: -0.300, AnalysisDepth: "2-ply"},
	}
	got := mergeCheckerMoves(nil, incoming)

	wantOrder := []string{"24/18 13/7", "13/7", "6/1 6/1"}
	if len(got) != len(wantOrder) {
		t.Fatalf("got %d moves, want %d", len(got), len(wantOrder))
	}
	for i, move := range wantOrder {
		if got[i].Move != move {
			t.Errorf("position %d: move = %q, want %q (full order: %v)", i, got[i].Move, move, moveNames(got))
		}
		if got[i].Index != i {
			t.Errorf("move %q: Index = %d, want %d", got[i].Move, got[i].Index, i)
		}
	}
	if got[0].EquityError != nil {
		t.Errorf("best move should have a nil EquityError, got %v", *got[0].EquityError)
	}
	for i := 1; i < len(got); i++ {
		if got[i].EquityError == nil {
			t.Fatalf("move %q: EquityError is nil, want a computed diff", got[i].Move)
		}
		wantDiff := got[0].Equity - got[i].Equity
		if *got[i].EquityError != wantDiff {
			t.Errorf("move %q: EquityError = %v, want %v", got[i].Move, *got[i].EquityError, wantDiff)
		}
	}
}

func TestMergeCheckerMovesDedupsByMoveString(t *testing.T) {
	existing := []domain.CheckerMove{
		{Move: "13/7", Equity: -0.050, AnalysisDepth: "0-ply", AnalysisEngine: "gnubg"},
	}
	incoming := []domain.CheckerMove{
		// Same move string, deeper analysis: should replace the existing entry.
		{Move: "13/7", Equity: -0.040, AnalysisDepth: "2-ply", AnalysisEngine: "XG"},
		{Move: "24/18", Equity: -0.010, AnalysisDepth: "2-ply", AnalysisEngine: "XG"},
	}
	got := mergeCheckerMoves(existing, incoming)

	if len(got) != 2 {
		t.Fatalf("expected the duplicate move string to collapse to one entry, got %d: %v", len(got), moveNames(got))
	}
	for _, m := range got {
		if m.Move == "13/7" {
			if m.AnalysisDepth != "2-ply" || m.AnalysisEngine != "XG" {
				t.Errorf("13/7 should have been replaced by the deeper incoming analysis, got depth=%q engine=%q", m.AnalysisDepth, m.AnalysisEngine)
			}
		}
	}
}

func TestMergeCheckerMovesKeepsDeeperExistingAnalysisOnConflict(t *testing.T) {
	// mergeCheckerMoves compares AnalysisDepth as a plain string ("2-ply" >=
	// "0-ply" lexically, which happens to agree with ply order here); a
	// shallower incoming analysis for an already-analysed move must not win.
	existing := []domain.CheckerMove{
		{Move: "13/7", Equity: -0.040, AnalysisDepth: "2-ply", AnalysisEngine: "XG"},
	}
	incoming := []domain.CheckerMove{
		{Move: "13/7", Equity: -0.060, AnalysisDepth: "0-ply", AnalysisEngine: "gnubg"},
	}
	got := mergeCheckerMoves(existing, incoming)

	if len(got) != 1 {
		t.Fatalf("expected 1 move, got %d", len(got))
	}
	// The code compares m.AnalysisDepth >= existingMove.AnalysisDepth using
	// the INCOMING move's depth on the left, so it only replaces when
	// incoming's depth string sorts >= existing's. "0-ply" < "2-ply"
	// lexically, so existing (2-ply/XG) must survive.
	if got[0].AnalysisDepth != "2-ply" || got[0].AnalysisEngine != "XG" {
		t.Errorf("AnalysisDepth/Engine = %q/%q, want %q/%q (existing, deeper analysis, must survive a shallower incoming one)",
			got[0].AnalysisDepth, got[0].AnalysisEngine, "2-ply", "XG")
	}
}

func moveNames(moves []domain.CheckerMove) []string {
	names := make([]string, len(moves))
	for i, m := range moves {
		names[i] = m.Move
	}
	return names
}

// --- mergePlayedMoves ---------------------------------------------------

func TestMergePlayedMovesUnionsAndNormalizesOrder(t *testing.T) {
	existing := []string{"5/2 5/4"}
	incoming := []string{"5/4 5/2", "24/18"} // "5/4 5/2" normalizes the same as "5/2 5/4"

	got := mergePlayedMoves(existing, incoming)

	want := []string{"24/18", "5/2 5/4"} // sort.Strings order
	if len(got) != len(want) {
		t.Fatalf("mergePlayedMoves(%v, %v) = %v, want %v", existing, incoming, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("mergePlayedMoves()[%d] = %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}
}

func TestMergePlayedMovesDropsEmptyEntries(t *testing.T) {
	got := mergePlayedMoves([]string{"", "13/7"}, []string{""})
	if len(got) != 1 || got[0] != "13/7" {
		t.Errorf("mergePlayedMoves should drop empty strings: got %v", got)
	}
}

func TestMergePlayedMovesNoDuplicatesAcrossExistingAndIncoming(t *testing.T) {
	got := mergePlayedMoves([]string{"13/7"}, []string{"13/7"})
	if len(got) != 1 {
		t.Errorf("the same move on both sides should collapse to one entry, got %v", got)
	}
}

// --- sortCheckerMovesByEquity -------------------------------------------

func TestSortCheckerMovesByEquity(t *testing.T) {
	a := &domain.PositionAnalysis{
		CheckerAnalysis: &domain.CheckerAnalysis{
			Moves: []domain.CheckerMove{
				{Move: "worst", Equity: -1.0},
				{Move: "best", Equity: 0.5},
				{Move: "middle", Equity: 0.0},
			},
		},
	}
	sortCheckerMovesByEquity(a)

	moves := a.CheckerAnalysis.Moves
	wantOrder := []string{"best", "middle", "worst"}
	for i, name := range wantOrder {
		if moves[i].Move != name {
			t.Errorf("position %d: move = %q, want %q", i, moves[i].Move, name)
		}
		if moves[i].Index != i {
			t.Errorf("move %q: Index = %d, want %d", moves[i].Move, moves[i].Index, i)
		}
	}
	if moves[0].EquityError != nil {
		t.Errorf("best move EquityError should be nil, got %v", *moves[0].EquityError)
	}
	if got, want := *moves[1].EquityError, 0.5; got != want {
		t.Errorf("middle move EquityError = %v, want %v", got, want)
	}
	if got, want := *moves[2].EquityError, 1.5; got != want {
		t.Errorf("worst move EquityError = %v, want %v", got, want)
	}
}

func TestSortCheckerMovesByEquityNilOrEmptyIsANoop(t *testing.T) {
	a := &domain.PositionAnalysis{}
	sortCheckerMovesByEquity(a) // must not panic on a nil CheckerAnalysis

	a2 := &domain.PositionAnalysis{CheckerAnalysis: &domain.CheckerAnalysis{}}
	sortCheckerMovesByEquity(a2) // nor on an empty Moves slice
}

// --- mergeAnalysis (insert path: existing == nil) ---------------------------

func TestMergeAnalysisInsertPathPromotesLegacySingularFields(t *testing.T) {
	incoming := domain.PositionAnalysis{
		PlayedMove:       "13/7",
		PlayedCubeAction: "No Double",
		CheckerAnalysis: &domain.CheckerAnalysis{
			Moves: []domain.CheckerMove{{Move: "13/7", Equity: 0.1}},
		},
	}
	got := mergeAnalysis(nil, incoming)

	if len(got.PlayedMoves) != 1 || got.PlayedMoves[0] != "13/7" {
		t.Errorf("PlayedMoves = %v, want [13/7] (promoted from the legacy PlayedMove field)", got.PlayedMoves)
	}
	if got.PlayedMove != "" {
		t.Errorf("legacy PlayedMove should be cleared once promoted, got %q", got.PlayedMove)
	}
	if len(got.PlayedCubeActions) != 1 || got.PlayedCubeActions[0] != "No Double" {
		t.Errorf("PlayedCubeActions = %v, want [No Double]", got.PlayedCubeActions)
	}
	if got.PlayedCubeAction != "" {
		t.Errorf("legacy PlayedCubeAction should be cleared once promoted, got %q", got.PlayedCubeAction)
	}
	if got.CreationDate.IsZero() {
		t.Error("insert path should stamp a CreationDate when the incoming one is zero")
	}
}

func TestMergeAnalysisInsertPathKeepsExplicitCreationDate(t *testing.T) {
	stamp := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	got := mergeAnalysis(nil, domain.PositionAnalysis{CreationDate: stamp})
	if !got.CreationDate.Equal(stamp) {
		t.Errorf("CreationDate = %v, want %v (should not overwrite an explicit non-zero date)", got.CreationDate, stamp)
	}
}

// --- mergeAnalysis (update path: existing != nil) ---------------------------

func TestMergeAnalysisUpdatePathKeepsOriginalCreationDate(t *testing.T) {
	created := time.Date(2019, 6, 1, 0, 0, 0, 0, time.UTC)
	existing := &domain.PositionAnalysis{CreationDate: created}
	got := mergeAnalysis(existing, domain.PositionAnalysis{CreationDate: time.Now()})

	if !got.CreationDate.Equal(created) {
		t.Errorf("CreationDate = %v, want the original %v (update must not lose it)", got.CreationDate, created)
	}
	if got.LastModifiedDate.IsZero() {
		t.Error("update path should stamp LastModifiedDate")
	}
}

func TestMergeAnalysisUpdatePathMergesCheckerMovesAndReSorts(t *testing.T) {
	existing := &domain.PositionAnalysis{
		CheckerAnalysis: &domain.CheckerAnalysis{
			Moves: []domain.CheckerMove{{Move: "13/7", Equity: -0.100, AnalysisDepth: "0-ply"}},
		},
	}
	incoming := domain.PositionAnalysis{
		CheckerAnalysis: &domain.CheckerAnalysis{
			Moves: []domain.CheckerMove{{Move: "24/18", Equity: 0.200, AnalysisDepth: "2-ply"}},
		},
	}
	got := mergeAnalysis(existing, incoming)

	moves := got.CheckerAnalysis.Moves
	if len(moves) != 2 {
		t.Fatalf("expected both moves to survive the merge, got %d: %v", len(moves), moveNames(moves))
	}
	if moves[0].Move != "24/18" {
		t.Errorf("best move after merge = %q, want %q (highest equity first)", moves[0].Move, "24/18")
	}
}

func TestMergeAnalysisUpdatePathUnionsPlayedMovesAcrossMatches(t *testing.T) {
	existing := &domain.PositionAnalysis{PlayedMoves: []string{"13/7"}}
	incoming := domain.PositionAnalysis{PlayedMoves: []string{"24/18"}}

	got := mergeAnalysis(existing, incoming)

	if len(got.PlayedMoves) != 2 {
		t.Fatalf("PlayedMoves = %v, want both moves from different matches to be kept", got.PlayedMoves)
	}
}

func TestMergeAnalysisUpdatePathKeepsExistingCheckerAnalysisWhenIncomingHasNone(t *testing.T) {
	existing := &domain.PositionAnalysis{
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{{Move: "13/7", Equity: 0.1}}},
	}
	// incoming is a pure cube decision update: no checker analysis of its own.
	incoming := domain.PositionAnalysis{
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{AnalysisEngine: "XG"},
	}
	got := mergeAnalysis(existing, incoming)

	if got.CheckerAnalysis == nil || len(got.CheckerAnalysis.Moves) != 1 {
		t.Errorf("existing CheckerAnalysis should survive an incoming update with none: got %+v", got.CheckerAnalysis)
	}
}

func TestMergeAnalysisUpdatePathKeepsBothEnginesCubeAnalysis(t *testing.T) {
	existing := &domain.PositionAnalysis{
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{AnalysisEngine: "gnubg", BestCubeAction: "No Double"},
	}
	incoming := domain.PositionAnalysis{
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{AnalysisEngine: "XG", BestCubeAction: "Double, Take"},
	}
	got := mergeAnalysis(existing, incoming)

	if len(got.AllCubeAnalyses) != 2 {
		t.Fatalf("AllCubeAnalyses should keep both engines' analyses, got %d: %+v", len(got.AllCubeAnalyses), got.AllCubeAnalyses)
	}
	// XG is sorted first (enginePriority).
	if got.AllCubeAnalyses[0].AnalysisEngine != "XG" {
		t.Errorf("AllCubeAnalyses[0].AnalysisEngine = %q, want XG (sorted first)", got.AllCubeAnalyses[0].AnalysisEngine)
	}
	if got.AllCubeAnalyses[1].AnalysisEngine != "gnubg" {
		t.Errorf("AllCubeAnalyses[1].AnalysisEngine = %q, want gnubg", got.AllCubeAnalyses[1].AnalysisEngine)
	}
	// The incoming analysis (a.DoublingCubeAnalysis) is left as the incoming
	// one, unmodified by the AllCubeAnalyses bookkeeping.
	if got.DoublingCubeAnalysis.AnalysisEngine != "XG" {
		t.Errorf("DoublingCubeAnalysis.AnalysisEngine = %q, want XG (the incoming engine)", got.DoublingCubeAnalysis.AnalysisEngine)
	}
}

func TestMergeAnalysisUpdatePathSameEngineReplacesInPlace(t *testing.T) {
	existing := &domain.PositionAnalysis{
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{AnalysisEngine: "XG", BestCubeAction: "No Double"},
	}
	incoming := domain.PositionAnalysis{
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{AnalysisEngine: "XG", BestCubeAction: "Double, Take"},
	}
	got := mergeAnalysis(existing, incoming)

	if len(got.AllCubeAnalyses) != 0 {
		t.Errorf("same-engine re-analysis should not fork AllCubeAnalyses, got %+v", got.AllCubeAnalyses)
	}
	if got.DoublingCubeAnalysis.BestCubeAction != "Double, Take" {
		t.Errorf("BestCubeAction = %q, want the incoming (re-)analysis to win", got.DoublingCubeAnalysis.BestCubeAction)
	}
}

func TestMergeAnalysisUpdatePathKeepsExistingCubeAnalysisWhenIncomingHasNone(t *testing.T) {
	existing := &domain.PositionAnalysis{
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{AnalysisEngine: "XG"},
	}
	incoming := domain.PositionAnalysis{
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{{Move: "13/7", Equity: 0.1}}},
	}
	got := mergeAnalysis(existing, incoming)

	if got.DoublingCubeAnalysis == nil || got.DoublingCubeAnalysis.AnalysisEngine != "XG" {
		t.Errorf("existing DoublingCubeAnalysis should survive an incoming checker-only update: got %+v", got.DoublingCubeAnalysis)
	}
}
