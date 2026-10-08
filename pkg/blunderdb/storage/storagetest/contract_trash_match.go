// Contract case for a match deleted through the trash: what the delete purges
// is what a plain DeleteCascade purges, and the restore gives the match back
// as it was, under a new id. The table that runs it lives in contract.go.
package storagetest

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/trash"
)

// matchContent is a match as a reader sees it, stripped of the ids a restore
// cannot keep: games and moves by their order, positions by their content.
type matchContent struct {
	Header   domain.Match
	Games    []domain.Game
	Moves    []domain.Move
	Analyses []domain.MoveAnalysis
	Boards   []domain.Position // the position of each move, in move order
}

func readMatchContent(t *testing.T, s storage.Storage, matchID int64) matchContent {
	t.Helper()
	ctx := context.Background()
	m, err := s.Matches().Get(ctx, "", matchID)
	if err != nil {
		t.Fatalf("Get match %d: %v", matchID, err)
	}
	c := matchContent{Header: *m}
	c.Header.ID, c.Header.ImportDate, c.Header.TournamentSortOrder = 0, time.Time{}, 0
	for g, err := range s.Matches().Games(ctx, "", matchID) {
		if err != nil {
			t.Fatalf("Games: %v", err)
		}
		g.ID, g.MatchID = 0, 0
		c.Games = append(c.Games, *g)
	}
	// Read the moves before loading their positions: a store pinned to one
	// connection cannot run a query while the listing holds it.
	var moves []*domain.Move
	for mv, err := range s.Matches().MovesByMatch(ctx, "", matchID) {
		if err != nil {
			t.Fatalf("MovesByMatch: %v", err)
		}
		moves = append(moves, mv)
	}
	moveIndex := map[int64]int64{}
	for _, mv := range moves {
		moveIndex[mv.ID] = int64(len(c.Moves))
		if mv.PositionID != 0 {
			p, err := s.Positions().Load(ctx, "", mv.PositionID)
			if err != nil {
				t.Fatalf("Load position %d: %v", mv.PositionID, err)
			}
			p.ID = 0
			c.Boards = append(c.Boards, *p)
		} else {
			c.Boards = append(c.Boards, domain.Position{})
		}
		mv.ID, mv.GameID, mv.PositionID = 0, 0, 0
		c.Moves = append(c.Moves, *mv)
	}
	for ma, err := range s.Matches().MoveAnalysesByMatch(ctx, "", matchID) {
		if err != nil {
			t.Fatalf("MoveAnalysesByMatch: %v", err)
		}
		ma.ID, ma.MoveID = 0, moveIndex[ma.MoveID]
		c.Analyses = append(c.Analyses, *ma)
	}
	return c
}

func testTrashMatchRestores(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ms := s.Matches()

	tourID, err := s.Tournaments().Create(ctx, "", "Cup", "2025-01-01", "Paris")
	if err != nil {
		t.Fatalf("Create tournament: %v", err)
	}
	newMatch := func(p1, p2, hash string) (int64, int64) {
		elo := 1650.0
		video := "https://example.org/v"
		m := domain.Match{
			Player1Name: p1, Player2Name: p2, Event: "Open", Round: "R1", MatchLength: 7,
			MatchDate: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), FilePath: "m.xg", GameCount: 2,
			TournamentID: &tourID, Comment: "a good one", MatchHash: hash, CanonicalHash: "c" + hash,
			Player1Elo: &elo, Transcriber: "T", EngineVersion: "XG 2.19", VideoSource: &video,
		}
		id, err := ms.Save(ctx, "", &m)
		if err != nil {
			t.Fatalf("Save match: %v", err)
		}
		if err := ms.SetVideoSource(ctx, "", id, video); err != nil {
			t.Fatalf("SetVideoSource: %v", err)
		}
		g := domain.Game{MatchID: id, GameNumber: 1, InitialScore: [2]int32{0, 0}, Winner: 1, PointsWon: 2, MoveCount: 3}
		gameID, err := ms.CreateGame(ctx, "", &g)
		if err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
		return id, gameID
	}
	addMove := func(gameID int64, n int32, posID int64) int64 {
		luck := int32(-120)
		mv := domain.Move{GameID: gameID, MoveNumber: n, MoveType: "checker", PositionID: posID, Player: 1,
			Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5", LuckMP: &luck}
		id, err := ms.CreateMove(ctx, "", &mv)
		if err != nil {
			t.Fatalf("CreateMove: %v", err)
		}
		return id
	}
	savePos := func(n int) int64 {
		p := provenancePos(n)
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position: %v", err)
		}
		return id
	}

	before, _ := newMatch("Ann", "Ben", "h-before")
	matchID, gameID := newMatch("Alice", "Bob", "h-target")
	after, _ := newMatch("Cid", "Dan", "h-after")
	if err := s.Tournaments().ReorderMatches(ctx, "", tourID, []int64{before, matchID, after}); err != nil {
		t.Fatalf("ReorderMatches: %v", err)
	}

	orphan := savePos(31) // held by the match alone, with the file's note
	if err := s.Analyses().Save(ctx, "", orphan, verdictBy("XG", time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC))); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}
	if _, err := s.Comments().AddFrom(ctx, "", orphan, "from the file", domain.CommentOriginXG); err != nil {
		t.Fatalf("AddFrom: %v", err)
	}
	held := savePos(32) // held by a collection too
	colID, err := s.Collections().Create(ctx, "", "Keep", "")
	if err != nil {
		t.Fatalf("Create collection: %v", err)
	}
	if err := s.Collections().AddPositions(ctx, "", colID, []int64{held}); err != nil {
		t.Fatalf("AddPositions: %v", err)
	}

	m1 := addMove(gameID, 1, orphan)
	addMove(gameID, 2, held)
	addMove(gameID, 3, 0)
	ma := domain.MoveAnalysis{MoveID: m1, AnalysisType: "checker", Depth: "3-ply", Equity: 0.1, EquityError: -0.05, WinRate: 0.55}
	if _, err := ms.CreateMoveAnalysis(ctx, "", &ma); err != nil {
		t.Fatalf("CreateMoveAnalysis: %v", err)
	}
	if err := ms.SetLastVisitedPosition(ctx, "", matchID, 2); err != nil {
		t.Fatalf("SetLastVisitedPosition: %v", err)
	}
	if err := s.Duels().SetOrigin(ctx, "", &storage.MatchOrigin{MatchID: matchID, DiceSeed: "seed"}); err != nil {
		t.Fatalf("SetOrigin: %v", err)
	}
	want := readMatchContent(t, s, matchID)

	entryID, err := trash.Match(ctx, s, "", matchID)
	if err != nil {
		t.Fatalf("trash.Match: %v", err)
	}
	if _, err := ms.Get(ctx, "", matchID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("match after trash delete: %v, want ErrNotFound", err)
	}
	// The retention rule is DeleteCascade's: the orphan goes, the held stays.
	if _, err := s.Positions().Load(ctx, "", orphan); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("orphan position after trash delete: %v, want ErrNotFound", err)
	}
	if _, err := s.Positions().Load(ctx, "", held); err != nil {
		t.Errorf("held position after trash delete: %v", err)
	}
	entries, err := s.Trash().List(ctx, "", domain.TrashMatch, storage.ListOpts{})
	if err != nil || len(entries) != 1 || entries[0].ID != entryID {
		t.Fatalf("trash list of matches: %v %v, want entry %d", entries, err, entryID)
	}

	restored, err := trash.Restore(ctx, s, "", entryID)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if restored == matchID {
		t.Errorf("restored match kept id %d; a deleted id is never reused", restored)
	}
	got := readMatchContent(t, s, restored)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("restored match differs:\n got %+v\nwant %+v", got, want)
	}

	var order []int64
	for m, err := range s.Tournaments().Matches(ctx, "", tourID) {
		if err != nil {
			t.Fatalf("tournament matches: %v", err)
		}
		order = append(order, m.ID)
	}
	if !reflect.DeepEqual(order, []int64{before, restored, after}) {
		t.Errorf("tournament order after restore = %v, want [%d %d %d]", order, before, restored, after)
	}
	if o, err := s.Duels().Origin(ctx, "", restored); err != nil || o.DiceSeed != "seed" {
		t.Errorf("origin after restore = %+v, %v; want seed", o, err)
	}
	var restoredOrphan int64
	for mv, err := range ms.MovesByMatch(ctx, "", restored) {
		if err != nil {
			t.Fatalf("MovesByMatch: %v", err)
		}
		if mv.MoveNumber == 1 {
			restoredOrphan = mv.PositionID
		}
	}
	if _, err := s.Analyses().Load(ctx, "", restoredOrphan); err != nil {
		t.Errorf("analysis of the purged position after restore: %v", err)
	}
	var notes []string
	for c, err := range s.Comments().ByPosition(ctx, "", restoredOrphan) {
		if err != nil {
			t.Fatalf("ByPosition: %v", err)
		}
		notes = append(notes, string(c.Origin)+":"+c.Text)
	}
	if !reflect.DeepEqual(notes, []string{"xg:from the file"}) {
		t.Errorf("comments of the purged position after restore = %v", notes)
	}
	if _, err := s.Trash().Load(ctx, "", entryID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("trash entry after restore: %v, want ErrNotFound", err)
	}

	// A match imported again meanwhile is not restored over: that would be a
	// duplicate. The entry stays.
	entryID, err = trash.Match(ctx, s, "", restored)
	if err != nil {
		t.Fatalf("trash.Match again: %v", err)
	}
	newMatch("Alice", "Bob", "h-target")
	if _, err := trash.Restore(ctx, s, "", entryID); !errors.Is(err, trash.ErrMatchPresent) {
		t.Errorf("restore over a re-imported match: %v, want ErrMatchPresent", err)
	}
	if _, err := s.Trash().Load(ctx, "", entryID); err != nil {
		t.Errorf("trash entry after a refused restore: %v", err)
	}
}
