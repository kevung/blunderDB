// Contract cases for search filters.
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testSearchFilterByDecisionType(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	chk := checkerPos()
	if _, err := s.Positions().Save(ctx, "", &chk); err != nil {
		t.Fatalf("Save checker position: %v", err)
	}
	cube := cubePos()
	if _, err := s.Positions().Save(ctx, "", &cube); err != nil {
		t.Fatalf("Save cube position: %v", err)
	}

	f := domain.SearchFilters{DecisionTypeFilter: true}
	f.Filter.DecisionType = domain.CheckerAction
	f.Filter.PlayerOnRoll = domain.Black

	var got []domain.Position
	for pos, err := range s.Search().Find(ctx, "", f) {
		if err != nil {
			t.Fatalf("Find: %v", err)
		}
		got = append(got, *pos)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 checker position, got %d", len(got))
	}
	if got[0].DecisionType != domain.CheckerAction {
		t.Errorf("filtered position DecisionType: got %d, want %d", got[0].DecisionType, domain.CheckerAction)
	}
}

func testSearchFilterByCubeResponse(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	// Two distinct cube positions (distinct boards → distinct Zobrist hashes).
	// The take position carries a centered offered cube (owner -1, value 1), as
	// take/pass positions are always stored.
	takePos := cubePos()
	takePos.Cube = domain.Cube{Owner: -1, Value: 1}
	doublePos := cubePos()
	doublePos.Board.Points[5] = domain.Point{Checkers: 1, Color: domain.White}

	takeID, err := s.Positions().Save(ctx, "", &takePos)
	if err != nil {
		t.Fatalf("Save take position: %v", err)
	}
	doubleID, err := s.Positions().Save(ctx, "", &doublePos)
	if err != nil {
		t.Fatalf("Save double position: %v", err)
	}
	if takeID == doubleID {
		t.Fatalf("cube positions deduped to the same id")
	}

	// The take position records a take/pass response → is_cube_response = 1.
	if err := s.Analyses().Save(ctx, "", takeID, &domain.PositionAnalysis{
		PlayedCubeActions: []string{"Take"},
	}); err != nil {
		t.Fatalf("Save take analysis: %v", err)
	}
	// The double position records a doubling decision → stays is_cube_response = 0.
	if err := s.Analyses().Save(ctx, "", doubleID, &domain.PositionAnalysis{
		PlayedCubeActions: []string{"Double"},
	}); err != nil {
		t.Fatalf("Save double analysis: %v", err)
	}

	search := func(sub string) []int64 {
		f := domain.SearchFilters{DecisionTypeFilter: true, CubeResponseFilter: sub}
		f.Filter.DecisionType = domain.CubeAction
		f.Filter.PlayerOnRoll = takePos.PlayerOnRoll
		var ids []int64
		for pos, err := range s.Search().Find(ctx, "", f) {
			if err != nil {
				t.Fatalf("Find(%q): %v", sub, err)
			}
			ids = append(ids, pos.ID)
		}
		return ids
	}

	if got := search("takepass"); len(got) != 1 || got[0] != takeID {
		t.Errorf("takepass filter: got %v, want [%d]", got, takeID)
	}
	if got := search("double"); len(got) != 1 || got[0] != doubleID {
		t.Errorf("double filter: got %v, want [%d]", got, doubleID)
	}
	if got := search(""); len(got) != 2 {
		t.Errorf("all-cube filter: got %d positions, want 2", len(got))
	}

	// IncludeCube + take/pass must match the centered offered cube (owner -1) even
	// though the board filter sends an owned cube — the board can't construct a
	// centered value>1 cube.
	fc := domain.SearchFilters{DecisionTypeFilter: true, CubeResponseFilter: "takepass", IncludeCube: true}
	fc.Filter.DecisionType = domain.CubeAction
	fc.Filter.PlayerOnRoll = takePos.PlayerOnRoll
	fc.Filter.Cube = domain.Cube{Owner: 0, Value: 1} // owned on the board; forced to -1 for take/pass
	var ids []int64
	for pos, err := range s.Search().Find(ctx, "", fc) {
		if err != nil {
			t.Fatalf("Find(includeCube takepass): %v", err)
		}
		ids = append(ids, pos.ID)
	}
	if len(ids) != 1 || ids[0] != takeID {
		t.Errorf("includeCube+takepass filter: got %v, want [%d]", ids, takeID)
	}
}

// testSearchFilterByAnalysisDecodesCompressedBlob exercises the analysis-driven
// Go-side filters (move pattern, equity) that only see a match once Find has
// decoded the stored a.data blob. The blob is always written zlib-compressed
// (AnalysisStore.Save), so a decode path that forgot to decompress — as the
// PostgreSQL backend's search once did, unmarshalling the compressed bytes
// directly as JSON — silently produced ana == nil for every row and these
// filters matched nothing, on every search, forever. Guards the fix, and
// guards needAnalysis actually gating the decode on both filters (fiche-05
// T1/T2: a filter missing from needAnalysis leaves a.data unselected, same
// symptom).
func testSearchFilterByAnalysisDecodesCompressedBlob(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	withAnalysis := checkerPos()
	idWith, err := s.Positions().Save(ctx, "", &withAnalysis)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	if err := s.Analyses().Save(ctx, "", idWith, &domain.PositionAnalysis{
		AnalysisType: "CheckerMove",
		CheckerAnalysis: &domain.CheckerAnalysis{
			Moves: []domain.CheckerMove{
				{Index: 0, Move: "13/11 24/23", Equity: 0.123, PlayerWinChance: 62},
			},
		},
	}); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}

	without := cubePos() // distinct board (checkerPos vs cubePos), no analysis saved
	if _, err := s.Positions().Save(ctx, "", &without); err != nil {
		t.Fatalf("Save position without analysis: %v", err)
	}

	// Both filters are applied unconditionally against the decoded ana (never
	// pushed to SQL, unlike WinRateFilter/GammonRateFilter/etc., which read the
	// denormalised a.player1_win_rate column and would pass even with a broken
	// decode) — so they are what actually exercises the blob decode.
	// EquityFilter additionally needs DateFilter/MoveErrorFilter's needAnalysis
	// companion (fiche-05 T2) to be decoded at all; without it ana is nil here
	// too and the filter silently matches nothing, on both backends.
	if got := searchIDs(t, s, domain.SearchFilters{MovePatternFilter: `m"13/11"`}); len(got) != 1 || got[0] != idWith {
		t.Errorf("MovePatternFilter: got %v, want [%d]", got, idWith)
	}
	if got := searchIDs(t, s, domain.SearchFilters{EquityFilter: "e>0"}); len(got) != 1 || got[0] != idWith {
		t.Errorf("EquityFilter: got %v, want [%d]", got, idWith)
	}
}

// searchIDs runs f against s and returns the matched position IDs in result order.
func searchIDs(t *testing.T, s storage.Storage, f domain.SearchFilters) []int64 {
	t.Helper()
	var ids []int64
	for pos, err := range s.Search().Find(context.Background(), "", f) {
		if err != nil {
			t.Fatalf("Find(%+v): %v", f, err)
		}
		ids = append(ids, pos.ID)
	}
	return ids
}

// testSearchFilterByIndividuallyImported is the point of the whole feature: a
// user who saved a position and then imported matches can find it again.
func testSearchFilterByIndividuallyImported(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	save := func(n int, individual bool) int64 {
		p := provenancePos(n)
		p.IndividuallyImported = individual
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position %d: %v", n, err)
		}
		return id
	}
	mine := save(1, true)
	save(2, false)
	save(3, false)

	var got []int64
	for pos, err := range s.Search().Find(ctx, "", domain.SearchFilters{IndividuallyImportedFilter: true}) {
		if err != nil {
			t.Fatalf("Find: %v", err)
		}
		got = append(got, pos.ID)
	}
	if len(got) != 1 || got[0] != mine {
		t.Errorf("filtered search returned %v, want exactly [%d]", got, mine)
	}

	// Without the filter, the match positions are back — and they are the noise
	// the filter exists to cut through.
	var all int
	for _, err := range s.Search().Find(ctx, "", domain.SearchFilters{}) {
		if err != nil {
			t.Fatalf("Find (unfiltered): %v", err)
		}
		all++
	}
	if all != 3 {
		t.Errorf("unfiltered search returned %d positions, want 3", all)
	}
}

// testSearchFilterByCommentPresence pins the comment-presence filter (issue
// #109): "has" and "none" partition the database, empty text counts as no
// comment at all, and the filter combines with the content filter as a plain
// AND rather than as a contradiction the backend has to arbitrate.
func testSearchFilterByCommentPresence(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	save := func(n int) int64 {
		p := provenancePos(n)
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position %d: %v", n, err)
		}
		return id
	}
	comment := func(id int64, text string) {
		if _, err := s.Comments().Add(ctx, "", id, text); err != nil {
			t.Fatalf("Add comment on %d: %v", id, err)
		}
	}

	commented := save(1)
	comment(commented, "big blunder, should have hit")
	bare := save(2)
	// An empty comment row is not a comment: it must land on the "none" side,
	// exactly as the rest of the code treats text = '' as absent.
	blank := save(3)
	comment(blank, "")

	find := func(f domain.SearchFilters) []int64 {
		t.Helper()
		var got []int64
		for pos, err := range s.Search().Find(ctx, "", f) {
			if err != nil {
				t.Fatalf("Find: %v", err)
			}
			got = append(got, pos.ID)
		}
		return got
	}

	if got := find(domain.SearchFilters{CommentFilter: "has"}); len(got) != 1 || got[0] != commented {
		t.Errorf(`CommentFilter "has" returned %v, want exactly [%d]`, got, commented)
	}

	none := find(domain.SearchFilters{CommentFilter: "none"})
	wantNone := map[int64]bool{bare: true, blank: true}
	if len(none) != 2 {
		t.Errorf(`CommentFilter "none" returned %v, want the 2 uncommented positions`, none)
	}
	for _, id := range none {
		if !wantNone[id] {
			t.Errorf(`CommentFilter "none" returned position %d, which carries a comment`, id)
		}
	}

	// "has" and "none" partition the database: nothing is in both, nothing in
	// neither.
	if all := find(domain.SearchFilters{}); len(all) != 3 {
		t.Errorf("unfiltered search returned %d positions, want 3", len(all))
	}

	// The presence and content filters are independent AND clauses, so a
	// contradictory pair is answered with an empty set rather than an error or
	// a precedence rule.
	//
	// These two also stand as the regression guard for the cursor deadlock: they
	// are the only assertions in the suite that reach a Go-phase predicate
	// (SearchText), which used to query the database while the search cursor was
	// still open and hung forever against this suite's single-connection
	// :memory: database.
	if got := find(domain.SearchFilters{CommentFilter: "none", SearchText: `t"blunder"`}); len(got) != 0 {
		t.Errorf("none + content filter returned %v, want nothing", got)
	}
	if got := find(domain.SearchFilters{CommentFilter: "has", SearchText: `t"blunder"`}); len(got) != 1 || got[0] != commented {
		t.Errorf("has + matching content filter returned %v, want exactly [%d]", got, commented)
	}
}

// testSearchFilterByFlagged pins the source-tool study mark (docs/adr/0006):
// the filter selects exactly the marked positions, and the mark is sticky — a
// later save of the same position without it must not clear it, since that is
// what an ordinary match import looks like.
func testSearchFilterByFlagged(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	save := func(n int, flagged bool) int64 {
		p := provenancePos(n)
		p.Flagged = flagged
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position %d: %v", n, err)
		}
		return id
	}
	marked := save(1, true)
	plain := save(2, false)
	save(3, false)

	find := func(f domain.SearchFilters) []int64 {
		t.Helper()
		var got []int64
		for pos, err := range s.Search().Find(ctx, "", f) {
			if err != nil {
				t.Fatalf("Find: %v", err)
			}
			got = append(got, pos.ID)
		}
		return got
	}

	if got := find(domain.SearchFilters{FlaggedFilter: true}); len(got) != 1 || got[0] != marked {
		t.Errorf("flagged search returned %v, want exactly [%d]", got, marked)
	}
	if all := find(domain.SearchFilters{}); len(all) != 3 {
		t.Errorf("unfiltered search returned %d positions, want 3", len(all))
	}

	// Sticky: re-saving the marked position unflagged — exactly what a match
	// import that does not carry the mark does — must not clear it.
	again := provenancePos(1)
	again.Flagged = false
	if _, err := s.Positions().Save(ctx, "", &again); err != nil {
		t.Fatalf("re-save unflagged: %v", err)
	}
	if got := find(domain.SearchFilters{FlaggedFilter: true}); len(got) != 1 || got[0] != marked {
		t.Errorf("an unflagged re-save cleared the mark: flagged search returned %v", got)
	}

	// And the converse: marking a position that was stored unflagged raises it.
	promote := provenancePos(2)
	promote.Flagged = true
	if _, err := s.Positions().Save(ctx, "", &promote); err != nil {
		t.Fatalf("re-save flagged: %v", err)
	}
	got := find(domain.SearchFilters{FlaggedFilter: true})
	if len(got) != 2 {
		t.Errorf("flagged search returned %v, want both %d and %d", got, marked, plain)
	}

	// The mark reads back on the position itself, not just through the filter.
	p, err := s.Positions().Load(ctx, "", marked)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !p.Flagged {
		t.Error("Load returned the marked position with Flagged=false")
	}
}
