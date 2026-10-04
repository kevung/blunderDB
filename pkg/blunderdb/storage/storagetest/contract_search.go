// Contract cases for search filters.
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
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
	for pos, err := range s.Search().Find(ctx, "", f, storage.ListOpts{}) {
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
		for pos, err := range s.Search().Find(ctx, "", f, storage.ListOpts{}) {
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
	for pos, err := range s.Search().Find(ctx, "", fc, storage.ListOpts{}) {
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
// decoded the compressed a.data blob. A decode path that skipped decompression,
// or a filter missing from needAnalysis, would silently match nothing.
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

	// Neither filter is pushed to SQL (unlike the rate filters, which read
	// denormalised columns), so they are what exercises the blob decode.
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
	for pos, err := range s.Search().Find(context.Background(), "", f, storage.ListOpts{}) {
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
	for pos, err := range s.Search().Find(ctx, "", domain.SearchFilters{IndividuallyImportedFilter: true}, storage.ListOpts{}) {
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
	for _, err := range s.Search().Find(ctx, "", domain.SearchFilters{}, storage.ListOpts{}) {
		if err != nil {
			t.Fatalf("Find (unfiltered): %v", err)
		}
		all++
	}
	if all != 3 {
		t.Errorf("unfiltered search returned %d positions, want 3", all)
	}
}

// testSearchFilterByCommentPresence pins the comment-presence filter:
// "has" and "none" partition the database, empty text counts as no
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
		for pos, err := range s.Search().Find(ctx, "", f, storage.ListOpts{}) {
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
	// These are also the suite's guard against the cursor deadlock: a Go-phase
	// predicate (SearchText) querying while the cursor is open hangs forever on
	// the single-connection :memory: database.
	if got := find(domain.SearchFilters{CommentFilter: "none", SearchText: `t"blunder"`}); len(got) != 0 {
		t.Errorf("none + content filter returned %v, want nothing", got)
	}
	if got := find(domain.SearchFilters{CommentFilter: "has", SearchText: `t"blunder"`}); len(got) != 1 || got[0] != commented {
		t.Errorf("has + matching content filter returned %v, want exactly [%d]", got, commented)
	}

	// The author filter: whole name, any case, and only a signed comment of
	// that name counts (a second signer on the same position does not hide it).
	signed := save(4)
	if _, err := s.Comments().Add(storage.WithCommentAuthor(ctx, "Alice"), "", signed, "Alice's view"); err != nil {
		t.Fatalf("Add signed comment: %v", err)
	}
	if _, err := s.Comments().Add(storage.WithCommentAuthor(ctx, "Bob"), "", signed, "Bob's view"); err != nil {
		t.Fatalf("Add second signed comment: %v", err)
	}
	for _, q := range []string{`au"alice"`, `au"Bob"`, "Alice"} {
		if got := find(domain.SearchFilters{CommentAuthorFilter: q}); len(got) != 1 || got[0] != signed {
			t.Errorf("CommentAuthorFilter %q returned %v, want exactly [%d]", q, got, signed)
		}
	}
	if got := find(domain.SearchFilters{CommentAuthorFilter: `au"Ali"`}); len(got) != 0 {
		t.Errorf(`CommentAuthorFilter au"Ali" returned %v, want nothing (whole name)`, got)
	}
	if got := find(domain.SearchFilters{CommentAuthorFilter: `au"Carol"`}); len(got) != 0 {
		t.Errorf(`CommentAuthorFilter au"Carol" returned %v, want nothing`, got)
	}
	// Any case means Unicode case, on both backends alike: SQLite's LOWER
	// folds ASCII only, so an accented or Cyrillic name is the real test.
	accented := save(5)
	for _, name := range []string{"Élodie", "Пётр"} {
		if _, err := s.Comments().Add(storage.WithCommentAuthor(ctx, name), "", accented, name+"'s view"); err != nil {
			t.Fatalf("Add comment signed %q: %v", name, err)
		}
	}
	for _, q := range []string{`au"élodie"`, `au"ÉLODIE"`, `au"пётр"`, `au"ПЁТР"`} {
		if got := find(domain.SearchFilters{CommentAuthorFilter: q}); len(got) != 1 || got[0] != accented {
			t.Errorf("CommentAuthorFilter %q returned %v, want exactly [%d]", q, got, accented)
		}
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
		for pos, err := range s.Search().Find(ctx, "", f, storage.ListOpts{}) {
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

// testSearchMoveErrorFilterMaxOverPlays pins the semantics of the move-error
// filter on a position player 1 played more than once: the position is
// scored by the LARGEST error among its plays, on the plain search (SQL
// column plus Go re-check) and on the mirror search (Go re-check alone)
// alike, and the answer is the same on every run.
//
// The fixture is built so the denormalised column holds the SMALLER error:
// PlayedMoves are merged sorted and the column scores the first of them, so
// "13/11 24/23" (50 mp) is what the column sees while "13/11 6/4" (200 mp)
// is what the user wants to find.
func testSearchMoveErrorFilterMaxOverPlays(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	// One game per match; the same positions occur in both matches.
	mkGame := func(n int) int64 {
		m := domain.Match{Player1Name: "me", Player2Name: "them", MatchLength: 7,
			MatchDate: time.Date(2025, 6, n, 0, 0, 0, 0, time.UTC)}
		matchID, err := s.Matches().Save(ctx, "", &m)
		if err != nil {
			t.Fatalf("Save match %d: %v", n, err)
		}
		g := domain.Game{MatchID: matchID, GameNumber: 1, Winner: 1, PointsWon: 1}
		gameID, err := s.Matches().CreateGame(ctx, "", &g)
		if err != nil {
			t.Fatalf("CreateGame %d: %v", n, err)
		}
		return gameID
	}
	gameA, gameB := mkGame(1), mkGame(2)

	save := func(p domain.Position) int64 {
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position: %v", err)
		}
		return id
	}
	play := func(gameID, posID int64, n int32, checkerMove, cubeAction string) {
		moveType := "checker"
		if cubeAction != "" {
			moveType = "cube"
		}
		mv := domain.Move{GameID: gameID, MoveNumber: n, MoveType: moveType,
			PositionID: posID, Player: 1, CheckerMove: checkerMove, CubeAction: cubeAction}
		if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
			t.Fatalf("CreateMove: %v", err)
		}
	}
	analyse := func(posID int64, a domain.PositionAnalysis) {
		if err := s.Analyses().Save(ctx, "", posID, &a); err != nil {
			t.Fatalf("Save analysis %d: %v", posID, err)
		}
	}
	small, big := 0.05, 0.20

	// Checker position played twice, 50 mp then 200 mp.
	twice := save(statsDecisionPos(t, 0))
	play(gameA, twice, 1, "13/11 24/23", "")
	play(gameB, twice, 1, "13/11 6/4", "")
	analyse(twice, domain.PositionAnalysis{
		AnalysisType: "CheckerMove",
		PlayedMoves:  []string{"13/11 24/23", "13/11 6/4"},
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "8/6 6/4", Equity: 0.50},
			{Move: "13/11 24/23", Equity: 0.45, EquityError: &small},
			{Move: "13/11 6/4", Equity: 0.30, EquityError: &big},
		}},
	})

	// Control: the same 50 mp play, made once.
	once := save(statsDecisionPos(t, 1))
	play(gameA, once, 2, "13/11 24/23", "")
	analyse(once, domain.PositionAnalysis{
		AnalysisType: "CheckerMove",
		PlayedMoves:  []string{"13/11 24/23"},
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "8/6 6/4", Equity: 0.50},
			{Move: "13/11 24/23", Equity: 0.45, EquityError: &small},
		}},
	})

	// Cube position: doubled correctly in one match (0 mp), failed to double
	// in the other (150 mp). Sorted, "Double" comes first, so the column
	// holds 0 — the same trap on the cube side.
	cube := save(cubePos())
	play(gameA, cube, 3, "", "Double")
	play(gameB, cube, 3, "", "No double")
	analyse(cube, domain.PositionAnalysis{
		AnalysisType:      "DoublingCube",
		PlayedCubeActions: []string{"Double", "No double"},
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
			BestCubeAction:         "Double/Take",
			CubefulNoDoubleError:   0.15,
			CubefulDoubleTakeError: 0,
			CubefulDoublePassError: 0.30,
		},
	})

	type want struct {
		filter string
		ids    []int64
	}
	cases := []want{
		{"E>100", []int64{twice, cube}}, // ever blundered here?
		{"E<100", []int64{once}},        // never worse than 100 mp
		{"E>250", nil},
		{"E160,250", []int64{twice}},
		{"E100,160", []int64{cube}},
	}
	sorted := func(ids []int64) []int64 {
		out := append([]int64(nil), ids...)
		sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
		return out
	}
	check := func(run int, mirror bool, w want) {
		t.Helper()
		got := sorted(searchIDs(t, s, domain.SearchFilters{MoveErrorFilter: w.filter, MirrorFilter: mirror}))
		if !reflect.DeepEqual(got, sorted(w.ids)) {
			t.Errorf("run %d mirror=%v %s: got %v, want %v", run, mirror, w.filter, got, sorted(w.ids))
		}
	}
	// 20 runs: the answer is a property of the data, not of the run.
	for run := range 20 {
		for _, w := range cases {
			check(run, false, w)
			check(run, true, w)
		}
	}
}

// testSearchPagination checks that ListOpts.Limit/Offset are genuinely pushed
// into the SQL scan: default order is p.id ascending
// (domain.SearchOrderByClause), so a window is checkable against a plain
// slice of the unbounded id list.
func testSearchPagination(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ps := s.Positions()

	var ids []int64
	for n := 1; n <= 5; n++ {
		p := provenancePos(n)
		id, err := ps.Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save %d: %v", n, err)
		}
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	find := func(opts storage.ListOpts) []int64 {
		t.Helper()
		var got []int64
		for pos, err := range s.Search().Find(ctx, "", domain.SearchFilters{}, opts) {
			if err != nil {
				t.Fatalf("Find(%+v): %v", opts, err)
			}
			got = append(got, pos.ID)
		}
		return got
	}

	if got := find(storage.ListOpts{}); !reflect.DeepEqual(got, ids) {
		t.Fatalf("Find{}: got %v, want %v (unbounded, p.id order)", got, ids)
	}
	if got, want := find(storage.ListOpts{Limit: 2}), ids[:2]; !reflect.DeepEqual(got, want) {
		t.Errorf("Find{Limit:2}: got %v, want %v", got, want)
	}
	if got, want := find(storage.ListOpts{Limit: 2, Offset: 2}), ids[2:4]; !reflect.DeepEqual(got, want) {
		t.Errorf("Find{Limit:2,Offset:2}: got %v, want %v", got, want)
	}
	// Offset with no limit: PostgreSQL accepts a bare OFFSET, SQLite's
	// adapter turns it into "LIMIT -1 OFFSET ?" — both must agree.
	if got, want := find(storage.ListOpts{Offset: 3}), ids[3:]; !reflect.DeepEqual(got, want) {
		t.Errorf("Find{Offset:3}: got %v, want %v", got, want)
	}
	// Past the end: an empty page, not an error.
	if got := find(storage.ListOpts{Limit: 2, Offset: 10}); len(got) != 0 {
		t.Errorf("Find{Limit:2,Offset:10}: got %v, want empty", got)
	}
}

// testSearchWindowsAgree checks that FindIDs, Count and IndexOf describe the
// same list Find returns, with and without a Go-side predicate, in id order
// and in sorts that cannot resume on the id, whose keys tie and are NULL for
// a position without analysis: a window of survivors is a slice of the whole
// result, never a short page. The Go-filtered scans run once more in chunks
// of two rows, so resuming after the last id and by OFFSET is exercised on
// every backend, not only on a library larger than a chunk.
func testSearchWindowsAgree(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	kept := map[int64]bool{}
	analysed := map[int64]bool{}
	for n := 1; n <= 9; n++ {
		p := provenancePos(n)
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save %d: %v", n, err)
		}
		if n%3 != 0 {
			played := "13/11 24/23"
			equityError := float64(n%3) * 0.02
			a := domain.PositionAnalysis{
				PlayedMoves: []string{played},
				CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
					{Move: "8/6 6/4", Equity: 0.5, PlayerWinChance: float64(n%4) * 10},
					{Move: played, Equity: 0.5 - equityError, EquityError: &equityError},
				}},
			}
			if err := s.Analyses().Save(ctx, "", id, &a); err != nil {
				t.Fatalf("Save analysis %d: %v", n, err)
			}
			analysed[id] = true
		}
		if n%2 == 1 {
			if _, err := s.Comments().Add(ctx, "", id, "keep this one"); err != nil {
				t.Fatalf("Add comment on %d: %v", id, err)
			}
			kept[id] = true
		}
	}
	outsider := searchIDs(t, s, domain.SearchFilters{})[1]

	type searchCase struct {
		f     domain.SearchFilters
		chunk int // 0 keeps the backend's chunk size
	}
	cases := map[string]searchCase{}
	for _, order := range []string{"", "error", "winrate", "close"} {
		by := ", by " + order
		if order == "" {
			by = ", by id"
		}
		cases["sql only"+by] = searchCase{f: domain.SearchFilters{Sort: order}}
		cases["go phase"+by] = searchCase{f: domain.SearchFilters{SearchText: "keep", Sort: order}}
		cases["go phase in chunks"+by] = searchCase{f: domain.SearchFilters{SearchText: "keep", Sort: order}, chunk: 2}
		cases["move error"+by] = searchCase{f: domain.SearchFilters{MoveErrorFilter: "E>10", Sort: order}}
		cases["move error in chunks"+by] = searchCase{f: domain.SearchFilters{MoveErrorFilter: "E>10", Sort: order}, chunk: 2}
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := tc.f
			// The SQL-only order is the reference: the Go phase only drops
			// rows from it, so its survivors are that order filtered.
			want := searchIDs(t, s, domain.SearchFilters{Sort: f.Sort})
			if f.SearchText != "" {
				var survivors []int64
				for _, id := range want {
					if kept[id] {
						survivors = append(survivors, id)
					}
				}
				want = survivors
			}
			if f.MoveErrorFilter != "" {
				var survivors []int64
				for _, id := range want {
					if analysed[id] {
						survivors = append(survivors, id)
					}
				}
				want = survivors
			}
			if len(want) == 0 {
				t.Fatalf("no result to check against")
			}
			if tc.chunk > 0 {
				defer sqlshared.SetSearchChunk(tc.chunk)()
			}
			all, err := s.Search().FindIDs(ctx, "", f, storage.ListOpts{})
			if err != nil {
				t.Fatalf("FindIDs: %v", err)
			}
			if !reflect.DeepEqual(all, want) {
				t.Fatalf("FindIDs: got %v, want %v", all, want)
			}
			var paged []int64
			for off := 0; off < len(want)+2; off += 2 {
				page, err := s.Search().FindIDs(ctx, "", f, storage.ListOpts{Offset: off, Limit: 2})
				if err != nil {
					t.Fatalf("FindIDs window %d: %v", off, err)
				}
				if off < len(want) && len(page) != min(2, len(want)-off) {
					t.Errorf("window at %d: %v is short", off, page)
				}
				paged = append(paged, page...)
			}
			if !reflect.DeepEqual(paged, want) {
				t.Errorf("windows: got %v, want %v", paged, want)
			}
			var found []int64
			for pos, err := range s.Search().Find(ctx, "", f, storage.ListOpts{Offset: 1, Limit: 3}) {
				if err != nil {
					t.Fatalf("Find window: %v", err)
				}
				found = append(found, pos.ID)
			}
			if !reflect.DeepEqual(found, want[1:min(4, len(want))]) {
				t.Errorf("Find{Offset:1,Limit:3}: got %v, want %v", found, want[1:min(4, len(want))])
			}
			if n, err := s.Search().Count(ctx, "", f); err != nil || n != len(want) {
				t.Errorf("Count: got %d, %v; want %d", n, err, len(want))
			}
			for i, id := range want {
				if at, ok, err := s.Search().IndexOf(ctx, "", f, id); err != nil || !ok || at != i {
					t.Errorf("IndexOf(%d): got %d, %v, %v; want %d", id, at, ok, err, i)
				}
			}
			if f.SearchText != "" {
				if _, ok, err := s.Search().IndexOf(ctx, "", f, outsider); err != nil || ok {
					t.Errorf("IndexOf(outsider %d): found %v, %v; want not found", outsider, ok, err)
				}
			}
		})
	}
}

// testCommentRestrictionLosesNothing: a position with no comment that
// satisfies every other filter is left out by a text or a tag filter and kept
// without one, and the commented position that matches is found either way.
func testCommentRestrictionLosesNothing(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	bare, commented := provenancePos(1), provenancePos(2)
	bareID, err := s.Positions().Save(ctx, "", &bare)
	if err != nil {
		t.Fatal(err)
	}
	commentedID, err := s.Positions().Save(ctx, "", &commented)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Comments().Add(ctx, "", commentedID, "an Illegal move #prime"); err != nil {
		t.Fatal(err)
	}
	for name, f := range map[string]domain.SearchFilters{
		"text": {SearchText: `t"illegal"`},
		"tag":  {TagFilter: "#prime"},
	} {
		got := searchIDs(t, s, f)
		if len(got) != 1 || got[0] != commentedID {
			t.Errorf("%s filter returned %v, want only the commented position %d", name, got, commentedID)
		}
		if n, err := s.Search().Count(ctx, "", f); err != nil || n != 1 {
			t.Errorf("%s Count = %d, %v; want 1", name, n, err)
		}
		if at, ok, err := s.Search().IndexOf(ctx, "", f, commentedID); err != nil || !ok || at != 0 {
			t.Errorf("%s IndexOf = %d, %v, %v; want 0", name, at, ok, err)
		}
		if _, ok, err := s.Search().IndexOf(ctx, "", f, bareID); err != nil || ok {
			t.Errorf("%s IndexOf(bare) found %v, %v; want not found", name, ok, err)
		}
	}
	if got := searchIDs(t, s, domain.SearchFilters{}); len(got) != 2 {
		t.Errorf("unfiltered search returned %v, want both positions", got)
	}
}
