package storagetest

import (
	"context"
	"strconv"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The batch reads exist for the exporter, which walks a whole selection and
// must not pay one round trip per position for each family. What they promise
// beyond "the same rows as the unit reads": silence about ids that name
// nothing (all of them), oldest first within a position (comments), and id
// order (moves). Position batch loading itself (order, skip-missing) is
// covered by Position/ListIDsAndLoadByIDs in contract_positions.go.

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func testAnalysisLoadMany(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	var ids []int64
	for n := 1; n <= 3; n++ {
		p := provenancePos(n)
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save %d: %v", n, err)
		}
		ids = append(ids, id)
	}
	for _, id := range ids[:2] {
		a := &domain.PositionAnalysis{XGID: "xgid-" + itoa(id), AnalysisType: "XG Roller++"}
		if err := s.Analyses().Save(ctx, "", id, a); err != nil {
			t.Fatalf("Save analysis %d: %v", id, err)
		}
	}

	got, err := s.Analyses().LoadMany(ctx, "", append(ids, 999999))
	if err != nil {
		t.Fatalf("LoadMany: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("LoadMany: %d analyses, want 2 (the third position has none)", len(got))
	}
	for _, id := range ids[:2] {
		a := got[id]
		if a == nil || a.XGID != "xgid-"+itoa(id) || a.PositionID != int(id) {
			t.Errorf("LoadMany[%d] = %+v", id, a)
		}
	}
	if _, ok := got[ids[2]]; ok {
		t.Error("LoadMany reports an analysis for a position that has none")
	}
}

func testCommentByPositions(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	var ids []int64
	for n := 1; n <= 2; n++ {
		p := provenancePos(n)
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save %d: %v", n, err)
		}
		ids = append(ids, id)
	}
	for _, text := range []string{"first", "second", ""} {
		if _, err := s.Comments().Add(ctx, "", ids[0], text); err != nil {
			t.Fatalf("Add %q: %v", text, err)
		}
	}

	got, err := s.Comments().ByPositions(ctx, "", []int64{ids[0], ids[1], 999999})
	if err != nil {
		t.Fatalf("ByPositions: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ByPositions: %d positions with comments, want 1", len(got))
	}
	texts := got[ids[0]]
	if len(texts) != 2 || texts[0].Text != "first" || texts[1].Text != "second" {
		t.Errorf("ByPositions[%d] = %v, want [first second] (oldest first, empty dropped)", ids[0], commentTexts(texts))
	}
}

func commentTexts(es []*domain.CommentEntry) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.Text
	}
	return out
}

// matchWithMoves saves a one-game match whose two moves reach the given
// positions, returning the match id and the move ids.
func matchWithMoves(t *testing.T, s storage.Storage, posIDs [2]int64) (int64, [2]int64) {
	t.Helper()
	ctx := context.Background()
	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 5}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	g := domain.Game{MatchID: matchID, GameNumber: 1}
	gameID, err := s.Matches().CreateGame(ctx, "", &g)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	var moveIDs [2]int64
	for i, pid := range posIDs {
		mv := domain.Move{GameID: gameID, MoveNumber: int32(i + 1), MoveType: "checker", PositionID: pid,
			Player: int32(i), Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5"}
		id, err := s.Matches().CreateMove(ctx, "", &mv)
		if err != nil {
			t.Fatalf("CreateMove %d: %v", i, err)
		}
		moveIDs[i] = id
	}
	return matchID, moveIDs
}

func testMovesByPositions(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	p1, p2 := provenancePos(1), provenancePos(2)
	id1, err := s.Positions().Save(ctx, "", &p1)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	id2, err := s.Positions().Save(ctx, "", &p2)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	matchWithMoves(t, s, [2]int64{id1, id1})

	got, err := s.Matches().MovesByPositions(ctx, "", []int64{id1, id2, 999999})
	if err != nil {
		t.Fatalf("MovesByPositions: %v", err)
	}
	if len(got[id1]) != 2 || len(got[id2]) != 0 {
		t.Fatalf("MovesByPositions: %d moves for %d and %d for %d, want 2 and 0", len(got[id1]), id1, len(got[id2]), id2)
	}
	if got[id1][0].MoveNumber != 1 || got[id1][1].MoveNumber != 2 || got[id1][0].CheckerMove != "8/5 6/5" {
		t.Errorf("MovesByPositions[%d] = %+v, %+v", id1, got[id1][0], got[id1][1])
	}
}

func testMoveAnalysisRoundTrip(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	p := provenancePos(1)
	pid, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	matchID, moveIDs := matchWithMoves(t, s, [2]int64{pid, pid})

	// Whole numbers on purpose: the PostgreSQL columns are BIGINT.
	want := []domain.MoveAnalysis{
		{MoveID: moveIDs[1], AnalysisType: "checker", Depth: "3-ply", Equity: 123, EquityError: 0, WinRate: 55, GammonRate: 10, BackgammonRate: 1, OpponentWinRate: 45, OpponentGammonRate: 8, OpponentBackgammonRate: 1},
		{MoveID: moveIDs[0], AnalysisType: "cube", Depth: "2-ply", Equity: -50, EquityError: 20, WinRate: 48},
	}
	for i := range want {
		id, err := s.Matches().CreateMoveAnalysis(ctx, "", &want[i])
		if err != nil {
			t.Fatalf("CreateMoveAnalysis: %v", err)
		}
		if id == 0 || want[i].ID != id {
			t.Fatalf("CreateMoveAnalysis id: %d / %d", id, want[i].ID)
		}
	}

	var got []domain.MoveAnalysis
	for ma, err := range s.Matches().MoveAnalysesByMatch(ctx, "", matchID) {
		if err != nil {
			t.Fatalf("MoveAnalysesByMatch: %v", err)
		}
		got = append(got, *ma)
	}
	if len(got) != 2 {
		t.Fatalf("MoveAnalysesByMatch: %d rows, want 2", len(got))
	}
	// Streamed in move order, so the cube analysis (move 1) comes first.
	if got[0].ID != want[1].ID || got[1].ID != want[0].ID {
		t.Errorf("MoveAnalysesByMatch order: %d, %d; want %d, %d", got[0].ID, got[1].ID, want[1].ID, want[0].ID)
	}
	if got[1] != want[0] {
		t.Errorf("MoveAnalysesByMatch round trip:\n got %+v\nwant %+v", got[1], want[0])
	}

	var none int
	for _, err := range s.Matches().MoveAnalysesByMatch(ctx, "", 999999) {
		if err != nil {
			t.Fatalf("MoveAnalysesByMatch(unknown): %v", err)
		}
		none++
	}
	if none != 0 {
		t.Errorf("MoveAnalysesByMatch(unknown) streamed %d rows", none)
	}
}

// A match's hashes are what lets a recipient's import recognise a match it
// already holds; they must come back out of Get and List, not only go in.
func testMatchHashesReadBack(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7, MatchHash: "h-format", CanonicalHash: "h-canon"}
	id, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Matches().Get(ctx, "", id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.MatchHash != "h-format" || got.CanonicalHash != "h-canon" {
		t.Errorf("Get hashes = %q / %q, want h-format / h-canon", got.MatchHash, got.CanonicalHash)
	}
	for l, err := range s.Matches().List(ctx, "", storage.MatchListOpts{}) {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if l.MatchHash != "h-format" || l.CanonicalHash != "h-canon" {
			t.Errorf("List hashes = %q / %q", l.MatchHash, l.CanonicalHash)
		}
	}
}
