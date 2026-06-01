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
		{"Analysis/SaveAndCompress", testAnalysisSaveAndCompress},
		{"Match/CreateGameMoveCascade", testMatchCreateGameMove},
		{"Match/DeleteCascade", testMatchDeleteCascade},
		{"Match/FindByHash", testMatchFindByHash},
		{"Tournament/AddRemoveMatch", testTournamentAddRemoveMatch},
		{"Collection/MoveBetweenCollections", testCollectionMoveBetween},
		{"Anki/ReviewUpdatesScheduling", testAnkiReviewUpdatesScheduling},
		{"Filter/SaveAndList", testFilterSaveAndList},
		{"History/SaveLoadClear", testCommandHistory},
		{"SearchHistory/SaveListDelete", testSearchHistory},
		{"Metadata/Counts", testMetadataCounts},
		{"Session/SaveLoadEmpty", nil},
		{"Search/FilterByDecisionType", testSearchFilterByDecisionType},
		{"Stats/AggregateCounts", nil},
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
