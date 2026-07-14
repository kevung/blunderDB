// Package storagetest provides a backend-agnostic contract test suite for
// implementations of storage.Storage. Each backend (SQLite now, PostgreSQL
// later) runs the same suite against a fresh instance:
//
//	func TestContract_SQLite(t *testing.T) {
//	    storagetest.RunContractTests(t, func() storage.Storage {
//	        s, _ := sqlite.Open(context.Background(), ":memory:", nil)
//	        return s
//	    })
//	}
//
// It lives in a regular (non-_test.go) file so it can be imported by the
// backend packages' test binaries — an exported helper in a _test.go file
// would not be visible across packages.
//
// The positions, analyses, search, transaction, match and tournament cases
// are filled in; the remaining ones are skipped until their family is
// implemented on both backends (the suite runs against each).
package storagetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// RunContractTests exercises a storage.Storage implementation. factory must
// return a fresh, empty, migrated Storage on every call.
func RunContractTests(t *testing.T, factory func() storage.Storage) {
	t.Helper()

	cases := []struct {
		name string
		fn   func(t *testing.T, s storage.Storage)
	}{
		{"Position/Save+Load", testPositionSaveLoad},
		{"Position/DedupByZobrist", testPositionDedup},
		{"Position/UpdatePreservesId", testPositionUpdatePreservesId},
		{"Position/ProvenanceIsSticky", testPositionProvenanceSticky},
		{"Search/FilterByIndividuallyImported", testSearchFilterByIndividuallyImported},
		{"Analysis/SaveAndCompress", testAnalysisSaveAndCompress},
		{"Match/CreateGameMoveCascade", testMatchCreateGameMove},
		{"Match/DeleteCascade", testMatchDeleteCascade},
		{"Match/DeleteCascadeRetention", testMatchDeleteCascadeRetention},
		{"Match/FindByHash", testMatchFindByHash},
		{"Match/ListFilterSortPaginate", testMatchListFilterSortPaginate},
		{"Tournament/AddRemoveMatch", testTournamentAddRemoveMatch},
		{"Collection/MoveBetweenCollections", testCollectionMoveBetween},
		{"Anki/ReviewUpdatesScheduling", testAnkiReviewUpdatesScheduling},
		{"Filter/SaveAndList", testFilterSaveAndList},
		{"History/SaveLoadClear", testCommandHistory},
		{"SearchHistory/SaveListDelete", testSearchHistory},
		{"Scope/HistoryAndFilterIsolation", testScopeIsolation},
		{"Metadata/Counts", testMetadataCounts},
		{"Session/SaveLoadEmpty", testSessionSaveLoad},
		{"Session/MultiScopeIsolation", testSessionMultiScope},
		{"Search/FilterByDecisionType", testSearchFilterByDecisionType},
		{"Search/FilterByCubeResponse", testSearchFilterByCubeResponse},
		{"Stats/AggregateCounts", testStatsAggregateCounts},
		{"Tx/RollbackUndoes", testTxRollbackUndoes},
		{"Tx/CommitPersists", testTxCommitPersists},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.fn == nil {
				t.Skip("contract case pending storage implementation (P2 PRs 3-6)")
			}
			s := factory()
			defer s.Close()
			tc.fn(t, s)
		})
	}
}

// checkerPos returns a fresh starting position flagged as a checker decision.
func checkerPos() domain.Position {
	p := domain.InitializePosition()
	p.DecisionType = domain.CheckerAction
	return p
}

// cubePos returns a cube-decision position with a board distinct from
// checkerPos so it hashes (and therefore stores) as its own row.
func cubePos() domain.Position {
	p := domain.InitializePosition()
	p.DecisionType = domain.CubeAction
	p.Board.Points[1] = domain.Point{Checkers: 1, Color: domain.White}
	p.Board.Points[3] = domain.Point{Checkers: 1, Color: domain.White}
	return p
}

func testPositionSaveLoad(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	p := checkerPos()
	id, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if id == 0 {
		t.Fatal("Save returned id 0")
	}
	if p.ID != id {
		t.Errorf("Save did not set p.ID: got %d, want %d", p.ID, id)
	}

	got, err := s.Positions().Load(ctx, "", id)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.ID != id {
		t.Errorf("Load id: got %d, want %d", got.ID, id)
	}
	if got.DecisionType != domain.CheckerAction {
		t.Errorf("Load DecisionType: got %d, want %d", got.DecisionType, domain.CheckerAction)
	}
	if got.Board != p.Board {
		t.Errorf("Load board mismatch:\n got %+v\nwant %+v", got.Board, p.Board)
	}
}

func testPositionDedup(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	p1 := checkerPos()
	id1, err := s.Positions().Save(ctx, "", &p1)
	if err != nil {
		t.Fatalf("first Save: %v", err)
	}
	p2 := checkerPos()
	id2, err := s.Positions().Save(ctx, "", &p2)
	if err != nil {
		t.Fatalf("second Save: %v", err)
	}
	if id1 != id2 {
		t.Errorf("dedup failed: first Save id %d, second Save id %d", id1, id2)
	}

	n := 0
	for _, err := range s.Positions().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		n++
	}
	if n != 1 {
		t.Errorf("after dedup expected 1 stored position, got %d", n)
	}
}

func testPositionUpdatePreservesId(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	p := checkerPos()
	id, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Mutate the board (15 checkers preserved) and update in place.
	p.Board.Points[19] = domain.Point{Checkers: 4, Color: domain.White}
	p.Board.Points[20] = domain.Point{Checkers: 1, Color: domain.White}
	if err := s.Positions().Update(ctx, "", &p); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := s.Positions().Load(ctx, "", id)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.ID != id {
		t.Errorf("Update changed id: got %d, want %d", got.ID, id)
	}
	if got.Board != p.Board {
		t.Errorf("Update did not persist board change:\n got %+v\nwant %+v", got.Board, p.Board)
	}
}

func testAnalysisSaveAndCompress(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	p := checkerPos()
	posID, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}

	a := domain.PositionAnalysis{
		AnalysisType: "CheckerMove",
		CheckerAnalysis: &domain.CheckerAnalysis{
			Moves: []domain.CheckerMove{
				{Index: 0, Move: "13/11 24/23", Equity: 0.123, PlayerWinChance: 54.32},
			},
		},
	}
	if err := s.Analyses().Save(ctx, "", posID, &a); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}

	// Load round-trips through the zlib-compressed data column.
	got, err := s.Analyses().Load(ctx, "", posID)
	if err != nil {
		t.Fatalf("Load analysis: %v", err)
	}
	if got.AnalysisType != "CheckerMove" {
		t.Errorf("AnalysisType: got %q, want %q", got.AnalysisType, "CheckerMove")
	}
	if got.PositionID != int(posID) {
		t.Errorf("PositionID: got %d, want %d", got.PositionID, posID)
	}
	if got.CheckerAnalysis == nil || len(got.CheckerAnalysis.Moves) != 1 {
		t.Fatalf("CheckerAnalysis not round-tripped: %+v", got.CheckerAnalysis)
	}
	if got.CheckerAnalysis.Moves[0].Move != "13/11 24/23" {
		t.Errorf("move: got %q, want %q", got.CheckerAnalysis.Moves[0].Move, "13/11 24/23")
	}
}

func testMatchFindByHash(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ms := s.Matches()

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7,
		MatchHash: "h1", CanonicalHash: "c1"}
	id, err := ms.Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}

	if got, found, err := ms.FindByHash(ctx, "", "h1", ""); err != nil || !found || got != id {
		t.Fatalf("FindByHash(match_hash): got=%d found=%v err=%v, want id=%d found", got, found, err, id)
	}
	if got, found, err := ms.FindByHash(ctx, "", "", "c1"); err != nil || !found || got != id {
		t.Fatalf("FindByHash(canonical): got=%d found=%v err=%v, want id=%d found", got, found, err, id)
	}
	if _, found, err := ms.FindByHash(ctx, "", "nope", "nope"); err != nil || found {
		t.Fatalf("FindByHash(absent): found=%v err=%v, want not found", found, err)
	}

	// Two hash-less matches must both store (NULL canonical_hash, not '',
	// so the UNIQUE index does not reject the second).
	for i := range 2 {
		hm := domain.Match{Player1Name: "X", Player2Name: "Y", MatchLength: 3}
		if _, err := ms.Save(ctx, "", &hm); err != nil {
			t.Fatalf("Save hash-less match %d: %v", i, err)
		}
	}
}

// testMatchListFilterSortPaginate pins the filter/sort/pagination contract of
// MatchStore.List across backends. Data: three matches with distinct dates,
// lengths and players; Alice plays two of them, one of which is in a tournament.
func testMatchListFilterSortPaginate(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ms := s.Matches()

	date := func(iso string) time.Time {
		tm, err := time.Parse("2006-01-02", iso)
		if err != nil {
			t.Fatalf("parse date %q: %v", iso, err)
		}
		return tm
	}
	// Save oldest→newest so ids and dates disagree with default (date-desc) order.
	old := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7, MatchDate: date("2023-06-15")}
	mid := domain.Match{Player1Name: "Carol", Player2Name: "Dave", MatchLength: 3, MatchDate: date("2024-06-15")}
	// A late-in-the-day timestamp guards the inclusive DateTo boundary.
	recent := domain.Match{Player1Name: "Eve", Player2Name: "Alice", MatchLength: 11, MatchDate: date("2025-06-15").Add(23 * time.Hour)}
	oldID, err := ms.Save(ctx, "", &old)
	if err != nil {
		t.Fatalf("Save old: %v", err)
	}
	midID, err := ms.Save(ctx, "", &mid)
	if err != nil {
		t.Fatalf("Save mid: %v", err)
	}
	recentID, err := ms.Save(ctx, "", &recent)
	if err != nil {
		t.Fatalf("Save recent: %v", err)
	}

	tID, err := s.Tournaments().Create(ctx, "", "Cup", "2024-01-01", "Paris")
	if err != nil {
		t.Fatalf("Create tournament: %v", err)
	}
	if err := s.Tournaments().AddMatch(ctx, "", tID, midID); err != nil {
		t.Fatalf("AddMatch: %v", err)
	}

	ids := func(opts storage.MatchListOpts) []int64 {
		t.Helper()
		var out []int64
		for m, err := range ms.List(ctx, "", opts) {
			if err != nil {
				t.Fatalf("List(%+v): %v", opts, err)
			}
			out = append(out, m.ID)
		}
		return out
	}
	eq := func(name string, got, want []int64) {
		t.Helper()
		if len(got) != len(want) {
			t.Errorf("%s: got %v, want %v", name, got, want)
			return
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s: got %v, want %v", name, got, want)
				return
			}
		}
	}

	// Default: every match, most recent first.
	eq("default order", ids(storage.MatchListOpts{}), []int64{recentID, midID, oldID})
	// Sort keys.
	eq("date_asc", ids(storage.MatchListOpts{Sort: "date_asc"}), []int64{oldID, midID, recentID})
	eq("length_desc", ids(storage.MatchListOpts{Sort: "length_desc"}), []int64{recentID, oldID, midID})
	eq("length_asc", ids(storage.MatchListOpts{Sort: "length_asc"}), []int64{midID, oldID, recentID})
	// Player filter: Alice is player 1 of `old` and player 2 of `recent`.
	eq("player Alice", ids(storage.MatchListOpts{PlayerName: "Alice"}), []int64{recentID, oldID})
	// Date filters, inclusive on the day (recent is at 23:00 on the DateTo day).
	eq("from 2024-01-01", ids(storage.MatchListOpts{DateFrom: "2024-01-01"}), []int64{recentID, midID})
	eq("to 2024-12-31", ids(storage.MatchListOpts{DateTo: "2024-12-31"}), []int64{midID, oldID})
	eq("year 2025", ids(storage.MatchListOpts{DateFrom: "2025-01-01", DateTo: "2025-12-31"}), []int64{recentID})
	// Match length.
	eq("length 3 or 7", ids(storage.MatchListOpts{MatchLength: []int{3, 7}, Sort: "date_asc"}), []int64{oldID, midID})
	// Tournament.
	eq("tournament", ids(storage.MatchListOpts{TournamentIDs: []int64{tID}}), []int64{midID})
	// Pagination over the default order.
	eq("limit 2", ids(storage.MatchListOpts{Limit: 2}), []int64{recentID, midID})
	eq("limit 2 offset 1", ids(storage.MatchListOpts{Limit: 2, Offset: 1}), []int64{midID, oldID})
	eq("offset 2", ids(storage.MatchListOpts{Offset: 2}), []int64{oldID})
	// Combined: Alice's matches, oldest first, first one only.
	eq("combined", ids(storage.MatchListOpts{PlayerName: "Alice", Sort: "date_asc", Limit: 1}), []int64{oldID})
}

func testMatchCreateGameMove(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	if matchID == 0 || m.ID != matchID {
		t.Fatalf("Save match id: id=%d m.ID=%d", matchID, m.ID)
	}

	g := domain.Game{MatchID: matchID, GameNumber: 1, Winner: 1, PointsWon: 2}
	gameID, err := s.Matches().CreateGame(ctx, "", &g)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	p := checkerPos()
	posID, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	mv := domain.Move{
		GameID: gameID, MoveNumber: 1, MoveType: "checker", PositionID: posID,
		Player: 1, Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5",
	}
	moveID, err := s.Matches().CreateMove(ctx, "", &mv)
	if err != nil {
		t.Fatalf("CreateMove: %v", err)
	}
	if moveID == 0 {
		t.Fatal("CreateMove returned id 0")
	}

	got, err := s.Matches().Get(ctx, "", matchID)
	if err != nil {
		t.Fatalf("Get match: %v", err)
	}
	if got.Player1Name != "Alice" || got.MatchLength != 7 {
		t.Errorf("Get match: %+v", got)
	}

	var games []domain.Game
	for g, err := range s.Matches().Games(ctx, "", matchID) {
		if err != nil {
			t.Fatalf("Games: %v", err)
		}
		games = append(games, *g)
	}
	if len(games) != 1 || games[0].Winner != 1 || games[0].PointsWon != 2 {
		t.Fatalf("Games: %+v", games)
	}

	var moves []domain.Move
	for mv, err := range s.Matches().Moves(ctx, "", gameID) {
		if err != nil {
			t.Fatalf("Moves: %v", err)
		}
		moves = append(moves, *mv)
	}
	if len(moves) != 1 || moves[0].CheckerMove != "8/5 6/5" || moves[0].PositionID != posID {
		t.Fatalf("Moves: %+v", moves)
	}

	var mps []domain.MatchMovePosition
	for mp, err := range s.Matches().MovePositions(ctx, "", matchID) {
		if err != nil {
			t.Fatalf("MovePositions: %v", err)
		}
		mps = append(mps, *mp)
	}
	if len(mps) != 1 {
		t.Fatalf("MovePositions count: got %d, want 1", len(mps))
	}
	// Move stored with XG player 1 maps to blunderDB player 0.
	if mps[0].PlayerOnRoll != 0 {
		t.Errorf("MovePositions PlayerOnRoll: got %d, want 0", mps[0].PlayerOnRoll)
	}
	if mps[0].Player1Name != "Alice" || mps[0].CheckerMove != "8/5 6/5" {
		t.Errorf("MovePositions context: %+v", mps[0])
	}
}

func testMatchDeleteCascade(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob"}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	g := domain.Game{MatchID: matchID, GameNumber: 1}
	gameID, err := s.Matches().CreateGame(ctx, "", &g)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	p := checkerPos()
	posID, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	mv := domain.Move{GameID: gameID, MoveNumber: 1, MoveType: "checker", PositionID: posID, Player: 1}
	if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
		t.Fatalf("CreateMove: %v", err)
	}

	if err := s.Matches().DeleteCascade(ctx, "", matchID); err != nil {
		t.Fatalf("DeleteCascade: %v", err)
	}

	if _, err := s.Matches().Get(ctx, "", matchID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("after delete Get match: got %v, want ErrNotFound", err)
	}
	for _, err := range s.Matches().Games(ctx, "", matchID) {
		if err != nil {
			t.Fatalf("Games: %v", err)
		}
		t.Error("game not cascade-deleted")
	}
	for _, err := range s.Matches().Moves(ctx, "", gameID) {
		if err != nil {
			t.Fatalf("Moves: %v", err)
		}
		t.Error("move not cascade-deleted")
	}
	if _, err := s.Positions().Load(ctx, "", posID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("orphan position not deleted: got %v, want ErrNotFound", err)
	}
}

func testTournamentAddRemoveMatch(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	tID, err := s.Tournaments().Create(ctx, "", "Cup", "2025-01-01", "Paris")
	if err != nil {
		t.Fatalf("Create tournament: %v", err)
	}

	m1 := domain.Match{Player1Name: "A", Player2Name: "B"}
	id1, err := s.Matches().Save(ctx, "", &m1)
	if err != nil {
		t.Fatalf("Save match 1: %v", err)
	}
	m2 := domain.Match{Player1Name: "C", Player2Name: "D"}
	id2, err := s.Matches().Save(ctx, "", &m2)
	if err != nil {
		t.Fatalf("Save match 2: %v", err)
	}

	if err := s.Tournaments().AddMatch(ctx, "", tID, id1); err != nil {
		t.Fatalf("AddMatch 1: %v", err)
	}
	if err := s.Tournaments().AddMatch(ctx, "", tID, id2); err != nil {
		t.Fatalf("AddMatch 2: %v", err)
	}

	got, err := s.Tournaments().Get(ctx, "", tID)
	if err != nil {
		t.Fatalf("Get tournament: %v", err)
	}
	if got.MatchCount != 2 {
		t.Errorf("MatchCount after AddMatch: got %d, want 2", got.MatchCount)
	}

	of, err := s.Tournaments().TournamentOf(ctx, "", id1)
	if err != nil {
		t.Fatalf("TournamentOf: %v", err)
	}
	if of.ID != tID {
		t.Errorf("TournamentOf: got %d, want %d", of.ID, tID)
	}

	var matchIDs []int64
	for m, err := range s.Tournaments().Matches(ctx, "", tID) {
		if err != nil {
			t.Fatalf("Matches: %v", err)
		}
		matchIDs = append(matchIDs, m.ID)
	}
	if len(matchIDs) != 2 {
		t.Fatalf("Matches count: got %d, want 2", len(matchIDs))
	}

	if err := s.Tournaments().RemoveMatch(ctx, "", id1); err != nil {
		t.Fatalf("RemoveMatch: %v", err)
	}
	if _, err := s.Tournaments().TournamentOf(ctx, "", id1); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("TournamentOf after RemoveMatch: got %v, want ErrNotFound", err)
	}
	got, err = s.Tournaments().Get(ctx, "", tID)
	if err != nil {
		t.Fatalf("Get tournament after remove: %v", err)
	}
	if got.MatchCount != 1 {
		t.Errorf("MatchCount after RemoveMatch: got %d, want 1", got.MatchCount)
	}
}

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

func testCollectionMoveBetween(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	cp := checkerPos()
	posID, err := s.Positions().Save(ctx, "", &cp)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}

	src, err := s.Collections().Create(ctx, "", "src", "source")
	if err != nil {
		t.Fatalf("Create src: %v", err)
	}
	dst, err := s.Collections().Create(ctx, "", "dst", "destination")
	if err != nil {
		t.Fatalf("Create dst: %v", err)
	}

	if err := s.Collections().AddPosition(ctx, "", src, posID); err != nil {
		t.Fatalf("AddPosition: %v", err)
	}
	// Adding the same position twice is a no-op, not an error.
	if err := s.Collections().AddPosition(ctx, "", src, posID); err != nil {
		t.Fatalf("AddPosition again: %v", err)
	}
	if c, _ := s.Collections().Get(ctx, "", src); c.PositionCount != 1 {
		t.Errorf("src count after add: got %d, want 1", c.PositionCount)
	}

	if err := s.Collections().MovePosition(ctx, "", src, dst, posID); err != nil {
		t.Fatalf("MovePosition: %v", err)
	}
	if c, _ := s.Collections().Get(ctx, "", src); c.PositionCount != 0 {
		t.Errorf("src count after move: got %d, want 0", c.PositionCount)
	}
	if c, _ := s.Collections().Get(ctx, "", dst); c.PositionCount != 1 {
		t.Errorf("dst count after move: got %d, want 1", c.PositionCount)
	}

	// The moved position is reachable through the destination collection.
	var ids []int64
	for p, err := range s.Collections().Positions(ctx, "", dst) {
		if err != nil {
			t.Fatalf("Positions: %v", err)
		}
		ids = append(ids, p.ID)
	}
	if len(ids) != 1 || ids[0] != posID {
		t.Errorf("dst positions: got %v, want [%d]", ids, posID)
	}

	// CollectionsOf reflects the new membership only.
	var cols []int64
	for c, err := range s.Collections().CollectionsOf(ctx, "", posID) {
		if err != nil {
			t.Fatalf("CollectionsOf: %v", err)
		}
		cols = append(cols, c.ID)
	}
	if len(cols) != 1 || cols[0] != dst {
		t.Errorf("CollectionsOf: got %v, want [%d]", cols, dst)
	}
}

func testAnkiReviewUpdatesScheduling(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	deckID, err := s.Anki().CreateDeck(ctx, "", "deck", "", domain.AnkiSourceSearch, 0, "")
	if err != nil {
		t.Fatalf("CreateDeck: %v", err)
	}
	p := checkerPos()
	posID, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	if err := s.Anki().SyncWithPositions(ctx, "", deckID, []int64{posID}); err != nil {
		t.Fatalf("SyncWithPositions: %v", err)
	}

	next, err := s.Anki().NextCard(ctx, "", deckID)
	if err != nil {
		t.Fatalf("NextCard: %v", err)
	}
	if next.Card.PositionID != posID {
		t.Errorf("NextCard position: got %d, want %d", next.Card.PositionID, posID)
	}
	if next.Card.State != 0 {
		t.Errorf("NextCard state: got %d, want 0 (new)", next.Card.State)
	}

	// Reviewing the only card with Easy schedules it into the future, so it
	// leaves the new state and no card remains due.
	following, err := s.Anki().ReviewCard(ctx, "", next.Card.ID, 4)
	if err != nil {
		t.Fatalf("ReviewCard: %v", err)
	}
	if following != nil {
		t.Errorf("ReviewCard next card: got %+v, want nil", following)
	}

	stats, err := s.Anki().DeckStats(ctx, "", deckID)
	if err != nil {
		t.Fatalf("DeckStats: %v", err)
	}
	if stats.NewCount != 0 {
		t.Errorf("NewCount after review: got %d, want 0", stats.NewCount)
	}
	if _, err := s.Anki().NextCard(ctx, "", deckID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("NextCard after review: got %v, want ErrNotFound", err)
	}

	// Resetting the deck returns every card to the new, due state.
	if err := s.Anki().ResetDeck(ctx, "", deckID); err != nil {
		t.Fatalf("ResetDeck: %v", err)
	}
	stats, _ = s.Anki().DeckStats(ctx, "", deckID)
	if stats.NewCount != 1 || stats.DueCount != 1 {
		t.Errorf("DeckStats after reset: %+v", stats)
	}
}

func testFilterSaveAndList(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	fs := s.Filters()

	id1, err := fs.Save(ctx, "", "f1", "cmd1")
	if err != nil {
		t.Fatalf("Save f1: %v", err)
	}
	if _, err := fs.Save(ctx, "", "f1", "dup"); !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("duplicate name: got %v, want ErrConflict", err)
	}
	id2, err := fs.Save(ctx, "", "f2", "cmd2")
	if err != nil {
		t.Fatalf("Save f2: %v", err)
	}

	var names []string
	for f, err := range fs.List(ctx, "") {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		names = append(names, f.Name)
	}
	if len(names) != 2 || names[0] != "f1" || names[1] != "f2" {
		t.Fatalf("List order: %v, want [f1 f2]", names)
	}

	if err := fs.Update(ctx, "", id1, "f1b", "cmd1b"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if err := fs.SaveEditPosition(ctx, "", "f1b", "editX"); err != nil {
		t.Fatalf("SaveEditPosition: %v", err)
	}
	if got, err := fs.LoadEditPosition(ctx, "", "f1b"); err != nil || got != "editX" {
		t.Fatalf("LoadEditPosition: got %q err %v, want editX", got, err)
	}
	if got, err := fs.LoadEditPosition(ctx, "", "unknown"); err != nil || got != "" {
		t.Fatalf("LoadEditPosition(unknown): got %q err %v, want empty", got, err)
	}

	if err := fs.Delete(ctx, "", id2); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := fs.Delete(ctx, "", id2); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("Delete absent: got %v, want ErrNotFound", err)
	}
	n := 0
	for _, err := range fs.List(ctx, "") {
		if err != nil {
			t.Fatalf("List after delete: %v", err)
		}
		n++
	}
	if n != 1 {
		t.Fatalf("filters after delete: got %d, want 1", n)
	}
}

func testCommandHistory(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	h := s.History()

	if err := h.Save(ctx, "", "first"); err != nil {
		t.Fatalf("Save first: %v", err)
	}
	if err := h.Save(ctx, "", "second"); err != nil {
		t.Fatalf("Save second: %v", err)
	}
	got, err := h.Load(ctx, "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 2 || got[0] != "first" || got[1] != "second" {
		t.Fatalf("Load order: %v, want [first second]", got)
	}

	if err := h.Clear(ctx, ""); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	got, err = h.Load(ctx, "")
	if err != nil {
		t.Fatalf("Load after clear: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("after clear: %v, want empty", got)
	}
}

func testSearchHistory(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	sh := s.SearchHistory()

	if err := sh.Save(ctx, "", "cmd1", "pos1"); err != nil {
		t.Fatalf("Save 1: %v", err)
	}
	if err := sh.Save(ctx, "", "cmd2", "pos2"); err != nil {
		t.Fatalf("Save 2: %v", err)
	}

	var entries []storage.SearchHistory
	for e, err := range sh.List(ctx, "") {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		entries = append(entries, *e)
	}
	if len(entries) != 2 {
		t.Fatalf("List count: got %d, want 2", len(entries))
	}
	// Most recent first; the id-DESC tiebreak makes this deterministic even
	// when both rows share a millisecond timestamp.
	if entries[0].Command != "cmd2" {
		t.Fatalf("List order: first = %q, want cmd2", entries[0].Command)
	}

	// Delete by timestamp (covers the same-millisecond case: deleting a
	// timestamp removes every entry that carries it).
	for _, e := range entries {
		if err := sh.DeleteEntry(ctx, "", e.Timestamp); err != nil {
			t.Fatalf("DeleteEntry: %v", err)
		}
	}
	n := 0
	for _, err := range sh.List(ctx, "") {
		if err != nil {
			t.Fatalf("List after delete: %v", err)
		}
		n++
	}
	if n != 0 {
		t.Fatalf("search history after delete: got %d, want 0", n)
	}
}

func testSessionSaveLoad(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ss := s.Session()

	// A scope that never stored a session loads an empty (non-nil) state.
	empty, err := ss.Load(ctx, "")
	if err != nil {
		t.Fatalf("Load empty: %v", err)
	}
	if empty == nil || empty.LastSearchCommand != "" || empty.HasActiveSearch || len(empty.LastPositionIDs) != 0 {
		t.Fatalf("fresh session not empty: %+v", empty)
	}

	want := storage.SessionState{
		LastSearchCommand:  "decision_type checker",
		LastSearchPosition: "xgid",
		LastPositionIndex:  5,
		LastPositionIDs:    []int64{1, 2, 3},
		HasActiveSearch:    true,
		ViewsJSON:          `{"a":1}`,
	}
	if err := ss.Save(ctx, "", want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := ss.Load(ctx, "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.LastSearchCommand != want.LastSearchCommand || got.LastSearchPosition != want.LastSearchPosition ||
		got.LastPositionIndex != want.LastPositionIndex || got.HasActiveSearch != want.HasActiveSearch ||
		got.ViewsJSON != want.ViewsJSON || len(got.LastPositionIDs) != 3 {
		t.Fatalf("Load round-trip:\n got %+v\nwant %+v", got, want)
	}
}

func testSessionMultiScope(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ss := s.Session()

	if err := ss.Save(ctx, "1", storage.SessionState{LastSearchCommand: "alpha", LastPositionIndex: 1}); err != nil {
		t.Fatalf("Save scope 1: %v", err)
	}
	if err := ss.Save(ctx, "2", storage.SessionState{LastSearchCommand: "beta", LastPositionIndex: 2}); err != nil {
		t.Fatalf("Save scope 2: %v", err)
	}

	s1, _ := ss.Load(ctx, "1")
	s2, _ := ss.Load(ctx, "2")
	if s1.LastSearchCommand != "alpha" || s1.LastPositionIndex != 1 {
		t.Fatalf("scope 1 leaked: %+v", s1)
	}
	if s2.LastSearchCommand != "beta" || s2.LastPositionIndex != 2 {
		t.Fatalf("scope 2 leaked: %+v", s2)
	}
	// The empty scope is independent of both.
	if e, _ := ss.Load(ctx, ""); e.LastSearchCommand != "" {
		t.Fatalf("empty scope sees other tenants: %+v", e)
	}

	// Clearing one scope leaves the other intact.
	if err := ss.Clear(ctx, "1"); err != nil {
		t.Fatalf("Clear scope 1: %v", err)
	}
	if e, _ := ss.Load(ctx, "1"); e.LastSearchCommand != "" {
		t.Fatalf("scope 1 not cleared: %+v", e)
	}
	if s2, _ := ss.Load(ctx, "2"); s2.LastSearchCommand != "beta" {
		t.Fatalf("scope 2 affected by clearing scope 1: %+v", s2)
	}
}

func testMetadataCounts(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	p := checkerPos()
	posID, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	a := domain.PositionAnalysis{AnalysisType: "CheckerMove"}
	if err := s.Analyses().Save(ctx, "", posID, &a); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}
	m := domain.Match{Player1Name: "A", Player2Name: "B"}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	g := domain.Game{MatchID: matchID, GameNumber: 1}
	gameID, err := s.Matches().CreateGame(ctx, "", &g)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	mv := domain.Move{GameID: gameID, MoveNumber: 1, MoveType: "checker", PositionID: posID, Player: 1}
	if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
		t.Fatalf("CreateMove: %v", err)
	}

	c, err := s.Metadata().Counts(ctx, "")
	if err != nil {
		t.Fatalf("Counts: %v", err)
	}
	want := storage.Counts{Positions: 1, Analyses: 1, Matches: 1, Games: 1, Moves: 1}
	if c != want {
		t.Fatalf("Counts = %+v, want %+v", c, want)
	}
}

func testTxRollbackUndoes(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	tx, err := s.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	p := checkerPos()
	id, err := tx.Positions().Save(ctx, "", &p)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("tx Save: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback: %v", err)
	}

	if _, err := s.Positions().Load(ctx, "", id); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("after rollback expected ErrNotFound, got %v", err)
	}
}

func testTxCommitPersists(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	tx, err := s.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	p := checkerPos()
	id, err := tx.Positions().Save(ctx, "", &p)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("tx Save: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	got, err := s.Positions().Load(ctx, "", id)
	if err != nil {
		t.Fatalf("after commit Load: %v", err)
	}
	if got.ID != id {
		t.Errorf("loaded id: got %d, want %d", got.ID, id)
	}
}

// testScopeIsolation checks that command history, search history and saved
// filters are isolated per scope. PostgreSQL scopes them by tenant_id; SQLite
// scopes them by a `scope` column (added in schema 2.9.0). The same filter name
// may coexist in distinct scopes.
func testScopeIsolation(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	// Command history.
	if err := s.History().Save(ctx, "1", "cmd-a"); err != nil {
		t.Fatalf("History.Save scope 1: %v", err)
	}
	if err := s.History().Save(ctx, "2", "cmd-b"); err != nil {
		t.Fatalf("History.Save scope 2: %v", err)
	}
	if got, _ := s.History().Load(ctx, "1"); len(got) != 1 || got[0] != "cmd-a" {
		t.Errorf("History scope 1: got %v, want [cmd-a]", got)
	}
	if got, _ := s.History().Load(ctx, "2"); len(got) != 1 || got[0] != "cmd-b" {
		t.Errorf("History scope 2: got %v, want [cmd-b]", got)
	}

	// Search history.
	if err := s.SearchHistory().Save(ctx, "1", "search-a", "pos-a"); err != nil {
		t.Fatalf("SearchHistory.Save scope 1: %v", err)
	}
	if err := s.SearchHistory().Save(ctx, "2", "search-b", "pos-b"); err != nil {
		t.Fatalf("SearchHistory.Save scope 2: %v", err)
	}
	if got := drainSearch(t, s, "1"); len(got) != 1 || got[0] != "search-a" {
		t.Errorf("SearchHistory scope 1: got %v, want [search-a]", got)
	}
	if got := drainSearch(t, s, "2"); len(got) != 1 || got[0] != "search-b" {
		t.Errorf("SearchHistory scope 2: got %v, want [search-b]", got)
	}

	// Filters: scope-isolated, and the same name may live in two scopes.
	if _, err := s.Filters().Save(ctx, "1", "f", "cmd1"); err != nil {
		t.Fatalf("Filters.Save scope 1: %v", err)
	}
	if _, err := s.Filters().Save(ctx, "2", "f", "cmd2"); err != nil {
		t.Errorf("same filter name in a different scope should be allowed: %v", err)
	}
	if got := drainFilters(t, s, "1"); len(got) != 1 || got[0] != "cmd1" {
		t.Errorf("Filters scope 1: got %v, want [cmd1]", got)
	}
	if got := drainFilters(t, s, "2"); len(got) != 1 || got[0] != "cmd2" {
		t.Errorf("Filters scope 2: got %v, want [cmd2]", got)
	}
}

func drainSearch(t *testing.T, s storage.Storage, scope string) []string {
	t.Helper()
	var out []string
	for e, err := range s.SearchHistory().List(context.Background(), scope) {
		if err != nil {
			t.Fatalf("SearchHistory.List: %v", err)
		}
		out = append(out, e.Command)
	}
	return out
}

func drainFilters(t *testing.T, s storage.Storage, scope string) []string {
	t.Helper()
	var out []string
	for f, err := range s.Filters().List(context.Background(), scope) {
		if err != nil {
			t.Fatalf("Filters.List: %v", err)
		}
		out = append(out, f.Command)
	}
	return out
}

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

	mb, err := ss.MatchBadges(ctx, "")
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

// provenancePos returns a position unique to n (the score is part of a
// position's identity, so each n hashes to its own row).
func provenancePos(n int) domain.Position {
	p := domain.InitializePosition()
	p.DecisionType = domain.CheckerAction
	p.Score = [2]int{n, 0}
	return p
}

// testPositionProvenanceSticky pins the rule that makes the individually
// imported flag usable: it is ORed into the stored value, never assigned.
// Both orderings below are ordinary user behaviour, and the flag must mean the
// same thing in each — see docs/adr/0001.
func testPositionProvenanceSticky(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	flag := func(id int64) bool {
		p, err := s.Positions().Load(ctx, "", id)
		if err != nil {
			t.Fatalf("Load position %d: %v", id, err)
		}
		return p.IndividuallyImported
	}
	save := func(p domain.Position, individual bool) int64 {
		p.IndividuallyImported = individual
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position: %v", err)
		}
		return id
	}

	// S1 — the user imports a position on its own, then imports the match it
	// came from. The match import must not clear the flag.
	solo := save(provenancePos(1), true)
	if got := save(provenancePos(1), false); got != solo {
		t.Fatalf("dedup failed: match import created id %d, want %d", got, solo)
	}
	if !flag(solo) {
		t.Error("S1: importing the match cleared individually_imported")
	}

	// S2 — the match is imported first, then the user imports one of its
	// positions on its own. The insert is a no-op, so the flag has to be raised
	// on the row that is already there.
	fromMatch := save(provenancePos(2), false)
	if flag(fromMatch) {
		t.Error("a match-sourced position came back individually imported")
	}
	if got := save(provenancePos(2), true); got != fromMatch {
		t.Fatalf("dedup failed: individual import created id %d, want %d", got, fromMatch)
	}
	if !flag(fromMatch) {
		t.Error("S2: individually importing an already-stored position did not mark it")
	}

	// A position only ever seen inside a match stays unmarked.
	if only := save(provenancePos(3), false); flag(only) {
		t.Error("a position seen only in a match was marked individually imported")
	}
}

// testMatchDeleteCascadeRetention pins what survives deleting a match. Every
// position below occurs in the match; they differ only in what else holds them.
// Before the individually-imported flag existed, a position the user had
// imported on its own was purged here as an orphan, silently, along with its
// Anki card.
//
// A comment does NOT hold a position, and that is deliberate: match importers
// attach the source file's per-move notes as comments (ingest/xg.go), so a
// comment is not evidence the user did anything — holding on it would keep a
// whole annotated match alive after the user deleted it.
func testMatchDeleteCascadeRetention(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob"}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	g := domain.Game{MatchID: matchID, GameNumber: 1}
	gameID, err := s.Matches().CreateGame(ctx, "", &g)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	// Every position is reached by one of the match's moves.
	inMatch := func(n int, individual bool) int64 {
		p := provenancePos(n)
		p.IndividuallyImported = individual
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position %d: %v", n, err)
		}
		mv := domain.Move{GameID: gameID, MoveNumber: int32(n), MoveType: "checker", PositionID: id, Player: 1}
		if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
			t.Fatalf("CreateMove %d: %v", n, err)
		}
		return id
	}

	purged := inMatch(1, false)    // held by nothing but the match…
	individual := inMatch(2, true) // …the user brought this one in themselves
	inCollection := inMatch(3, false)
	commented := inMatch(4, false)
	inDeck := inMatch(5, false)

	// Neither an analysis nor a comment holds a position: both can arrive with
	// the match, so holding on them would mean never purging anything.
	if err := s.Analyses().Save(ctx, "", purged, &domain.PositionAnalysis{}); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}

	coll, err := s.Collections().Create(ctx, "", "keep", "")
	if err != nil {
		t.Fatalf("Create collection: %v", err)
	}
	if err := s.Collections().AddPosition(ctx, "", coll, inCollection); err != nil {
		t.Fatalf("AddPosition: %v", err)
	}
	if _, err := s.Comments().Add(ctx, "", commented, "note that came in with the match"); err != nil {
		t.Fatalf("Add comment: %v", err)
	}
	deck, err := s.Anki().CreateDeck(ctx, "", "deck", "", domain.AnkiSourceSearch, 0, "")
	if err != nil {
		t.Fatalf("CreateDeck: %v", err)
	}
	if err := s.Anki().SyncWithPositions(ctx, "", deck, []int64{inDeck}); err != nil {
		t.Fatalf("SyncWithPositions: %v", err)
	}

	if err := s.Matches().DeleteCascade(ctx, "", matchID); err != nil {
		t.Fatalf("DeleteCascade: %v", err)
	}

	for _, tc := range []struct {
		name string
		id   int64
		kept bool
	}{
		{"held by nothing (analysis only)", purged, false},
		{"commented (the note may have come from the match)", commented, false},
		{"individually imported", individual, true},
		{"in a collection", inCollection, true},
		{"in an Anki deck", inDeck, true},
	} {
		_, err := s.Positions().Load(ctx, "", tc.id)
		switch {
		case tc.kept && err != nil:
			t.Errorf("position %s was purged with the match: %v", tc.name, err)
		case !tc.kept && !errors.Is(err, storage.ErrNotFound):
			t.Errorf("position %s survived the match: got %v, want ErrNotFound", tc.name, err)
		}
	}
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
