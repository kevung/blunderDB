//go:build postgres

package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
)

// openMatchStore boots a throwaway PostgreSQL, returns a freshly bootstrapped
// Storage and the DSN (for raw inspection).
func openMatchStore(t *testing.T) (*pg.Storage, string) {
	t.Helper()
	dsn := startPostgres(t)
	s, err := pg.Open(context.Background(), dsn, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s, dsn
}

// savePos stores a fresh position with the given decision type and returns
// its id.
func savePos(t *testing.T, s *pg.Storage, decision int) int64 {
	t.Helper()
	p := domain.InitializePosition()
	p.DecisionType = decision
	id, err := s.Positions().Save(context.Background(), "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	return id
}

// TestMatchSaveGetList exercises the header CRUD path.
func TestMatchSaveGetList(t *testing.T) {
	ctx := context.Background()
	s, _ := openMatchStore(t)

	m := domain.Match{
		Player1Name: "Alice",
		Player2Name: "Bob",
		Event:       "Worlds",
		MatchLength: 7,
		MatchDate:   time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
		Comment:     "good match",
	}
	id, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if id == 0 || m.ID != id {
		t.Fatalf("Save did not set id: id=%d m.ID=%d", id, m.ID)
	}
	if m.ImportDate.IsZero() {
		t.Error("Save did not set ImportDate")
	}

	got, err := s.Matches().Get(ctx, "", id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Player1Name != "Alice" || got.Player2Name != "Bob" {
		t.Errorf("Get players: got %q/%q", got.Player1Name, got.Player2Name)
	}
	if got.MatchLength != 7 || got.Comment != "good match" {
		t.Errorf("Get fields: length=%d comment=%q", got.MatchLength, got.Comment)
	}
	if !got.MatchDate.Equal(m.MatchDate) {
		t.Errorf("Get match date: got %v, want %v", got.MatchDate, m.MatchDate)
	}
	if got.LastVisitedPosition != -1 {
		t.Errorf("Get last visited: got %d, want -1", got.LastVisitedPosition)
	}

	if _, err := s.Matches().Get(ctx, "", 9999); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Get missing: got %v, want ErrNotFound", err)
	}

	// A second match with no date sorts after the dated one.
	m2 := domain.Match{Player1Name: "Carol", Player2Name: "Dave"}
	if _, err := s.Matches().Save(ctx, "", &m2); err != nil {
		t.Fatalf("Save second: %v", err)
	}
	n := 0
	for got, err := range s.Matches().List(ctx, "") {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		_ = got
		n++
	}
	if n != 2 {
		t.Errorf("List count: got %d, want 2", n)
	}
}

// TestMatchUpdate covers Update and UpdateComment.
func TestMatchUpdate(t *testing.T) {
	ctx := context.Background()
	s, _ := openMatchStore(t)

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob"}
	id, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := s.Matches().Update(ctx, "", id, "Anna", "Ben", "2024-12-25"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if err := s.Matches().UpdateComment(ctx, "", id, "edited"); err != nil {
		t.Fatalf("UpdateComment: %v", err)
	}

	got, err := s.Matches().Get(ctx, "", id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Player1Name != "Anna" || got.Player2Name != "Ben" {
		t.Errorf("Update players: got %q/%q", got.Player1Name, got.Player2Name)
	}
	if got.Comment != "edited" {
		t.Errorf("UpdateComment: got %q", got.Comment)
	}
	want := time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC)
	if !got.MatchDate.Equal(want) {
		t.Errorf("Update date: got %v, want %v", got.MatchDate, want)
	}

	if err := s.Matches().Update(ctx, "", id, "x", "y", "not-a-date"); err == nil {
		t.Error("Update with bad date: expected error")
	}
}

// TestMatchGamesAndMoves covers CreateGame/Games and CreateMove/Moves.
func TestMatchGamesAndMoves(t *testing.T) {
	ctx := context.Background()
	s, _ := openMatchStore(t)

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob"}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}

	g := domain.Game{MatchID: matchID, GameNumber: 1, InitialScore: [2]int32{0, 0}, Winner: 1, PointsWon: 2}
	gameID, err := s.Matches().CreateGame(ctx, "", &g)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	if gameID == 0 || g.ID != gameID {
		t.Fatalf("CreateGame id: id=%d g.ID=%d", gameID, g.ID)
	}

	posID := savePos(t, s, domain.CheckerAction)
	mv := domain.Move{
		GameID: gameID, MoveNumber: 1, MoveType: "checker", PositionID: posID,
		Player: 1, Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5",
	}
	moveID, err := s.Matches().CreateMove(ctx, "", &mv)
	if err != nil {
		t.Fatalf("CreateMove: %v", err)
	}
	if moveID == 0 || mv.ID != moveID {
		t.Fatalf("CreateMove id: id=%d mv.ID=%d", moveID, mv.ID)
	}

	games := collect(t, s.Matches().Games(ctx, "", matchID))
	if len(games) != 1 || games[0].Winner != 1 || games[0].PointsWon != 2 {
		t.Fatalf("Games: %+v", games)
	}

	moves := collect(t, s.Matches().Moves(ctx, "", gameID))
	if len(moves) != 1 {
		t.Fatalf("Moves count: got %d, want 1", len(moves))
	}
	if moves[0].CheckerMove != "8/5 6/5" || moves[0].Dice != [2]int32{3, 1} || moves[0].PositionID != posID {
		t.Errorf("Moves content: %+v", moves[0])
	}
}

// TestMatchMovePositions verifies the chronological move-position join and the
// XG→blunderDB player encoding.
func TestMatchMovePositions(t *testing.T) {
	ctx := context.Background()
	s, _ := openMatchStore(t)

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob"}
	matchID, _ := s.Matches().Save(ctx, "", &m)
	g := domain.Game{MatchID: matchID, GameNumber: 1}
	gameID, _ := s.Matches().CreateGame(ctx, "", &g)

	posA := savePos(t, s, domain.CheckerAction)
	posB := savePos(t, s, domain.CubeAction)
	mvA := domain.Move{GameID: gameID, MoveNumber: 1, MoveType: "checker", PositionID: posA, Player: 1}
	mvB := domain.Move{GameID: gameID, MoveNumber: 2, MoveType: "cube", PositionID: posB, Player: -1}
	if _, err := s.Matches().CreateMove(ctx, "", &mvA); err != nil {
		t.Fatalf("CreateMove A: %v", err)
	}
	if _, err := s.Matches().CreateMove(ctx, "", &mvB); err != nil {
		t.Fatalf("CreateMove B: %v", err)
	}

	mps := collectMP(t, s.Matches().MovePositions(ctx, "", matchID))
	if len(mps) != 2 {
		t.Fatalf("MovePositions count: got %d, want 2", len(mps))
	}
	if mps[0].MoveNumber != 1 || mps[1].MoveNumber != 2 {
		t.Errorf("MovePositions order: %d, %d", mps[0].MoveNumber, mps[1].MoveNumber)
	}
	if mps[0].PlayerOnRoll != 0 {
		t.Errorf("player 1 (XG +1) should map to 0, got %d", mps[0].PlayerOnRoll)
	}
	if mps[1].PlayerOnRoll != 1 {
		t.Errorf("player -1 (XG) should map to 1, got %d", mps[1].PlayerOnRoll)
	}
	if mps[0].Player1Name != "Alice" || mps[0].Player2Name != "Bob" {
		t.Errorf("MovePositions names: %q/%q", mps[0].Player1Name, mps[0].Player2Name)
	}

	if _, err := iterErr(s.Matches().MovePositions(ctx, "", 9999)); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("MovePositions missing match: got %v, want ErrNotFound", err)
	}
}

// TestMatchSwapPlayers checks the header / game / move swap.
func TestMatchSwapPlayers(t *testing.T) {
	ctx := context.Background()
	s, _ := openMatchStore(t)

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob"}
	matchID, _ := s.Matches().Save(ctx, "", &m)
	g := domain.Game{MatchID: matchID, GameNumber: 1, InitialScore: [2]int32{2, 5}, Winner: 1}
	gameID, _ := s.Matches().CreateGame(ctx, "", &g)
	mv := domain.Move{GameID: gameID, MoveNumber: 1, MoveType: "checker", Player: 1}
	if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
		t.Fatalf("CreateMove: %v", err)
	}

	if err := s.Matches().SwapPlayers(ctx, "", matchID); err != nil {
		t.Fatalf("SwapPlayers: %v", err)
	}

	got, _ := s.Matches().Get(ctx, "", matchID)
	if got.Player1Name != "Bob" || got.Player2Name != "Alice" {
		t.Errorf("swap names: got %q/%q", got.Player1Name, got.Player2Name)
	}
	games := collect(t, s.Matches().Games(ctx, "", matchID))
	if games[0].InitialScore != [2]int32{5, 2} || games[0].Winner != -1 {
		t.Errorf("swap game: score=%v winner=%d", games[0].InitialScore, games[0].Winner)
	}
	moves := collect(t, s.Matches().Moves(ctx, "", gameID))
	if moves[0].Player != -1 {
		t.Errorf("swap move player: got %d, want -1", moves[0].Player)
	}
}

// TestMatchMergePlayers checks the canonical-name rewrite.
func TestMatchMergePlayers(t *testing.T) {
	ctx := context.Background()
	s, _ := openMatchStore(t)

	for _, p := range []string{"Bob", "Bobby", "Robert"} {
		mm := domain.Match{Player1Name: p, Player2Name: "Opponent"}
		if _, err := s.Matches().Save(ctx, "", &mm); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}
	mm := domain.Match{Player1Name: "Opponent", Player2Name: "Bobby"}
	if _, err := s.Matches().Save(ctx, "", &mm); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := s.Matches().MergePlayers(ctx, "", []string{"Bob", "Bobby", "Robert"}, "Bob R."); err != nil {
		t.Fatalf("MergePlayers: %v", err)
	}

	for got, err := range s.Matches().List(ctx, "") {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if got.Player1Name == "Bob" || got.Player1Name == "Bobby" || got.Player1Name == "Robert" {
			t.Errorf("player1 not merged: %q", got.Player1Name)
		}
		if got.Player2Name == "Bobby" {
			t.Errorf("player2 not merged: %q", got.Player2Name)
		}
	}
}

// TestMatchLastVisited covers SetLastVisitedPosition and LastVisited.
func TestMatchLastVisited(t *testing.T) {
	ctx := context.Background()
	s, _ := openMatchStore(t)

	if _, err := s.Matches().LastVisited(ctx, ""); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("LastVisited on empty db: got %v, want ErrNotFound", err)
	}

	m1 := domain.Match{Player1Name: "A1", Player2Name: "B1"}
	id1, _ := s.Matches().Save(ctx, "", &m1)
	m2 := domain.Match{Player1Name: "A2", Player2Name: "B2"}
	if _, err := s.Matches().Save(ctx, "", &m2); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// No match visited yet → most recent match.
	if _, err := s.Matches().LastVisited(ctx, ""); err != nil {
		t.Fatalf("LastVisited fallback: %v", err)
	}

	if err := s.Matches().SetLastVisitedPosition(ctx, "", id1, 4); err != nil {
		t.Fatalf("SetLastVisitedPosition: %v", err)
	}
	lv, err := s.Matches().LastVisited(ctx, "")
	if err != nil {
		t.Fatalf("LastVisited: %v", err)
	}
	if lv.ID != id1 {
		t.Errorf("LastVisited: got match %d, want %d", lv.ID, id1)
	}
	if lv.LastVisitedPosition != 4 {
		t.Errorf("LastVisited position: got %d, want 4", lv.LastVisitedPosition)
	}
}

// TestMatchDeleteCascade verifies a match delete removes its games, moves and
// move analyses, and drops the orphaned positions.
func TestMatchDeleteCascade(t *testing.T) {
	ctx := context.Background()
	s, dsn := openMatchStore(t)

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob"}
	matchID, _ := s.Matches().Save(ctx, "", &m)
	g := domain.Game{MatchID: matchID, GameNumber: 1}
	gameID, _ := s.Matches().CreateGame(ctx, "", &g)
	posID := savePos(t, s, domain.CheckerAction)
	mv := domain.Move{GameID: gameID, MoveNumber: 1, MoveType: "checker", PositionID: posID, Player: 1}
	moveID, _ := s.Matches().CreateMove(ctx, "", &mv)

	conn := rawConn(t, dsn)
	if _, err := conn.Exec(ctx,
		`INSERT INTO move_analysis (tenant_id, move_id, analysis_type, equity)
		 VALUES (0, $1, 'checker', 100)`, moveID); err != nil {
		t.Fatalf("seed move_analysis: %v", err)
	}

	if err := s.Matches().DeleteCascade(ctx, "", matchID); err != nil {
		t.Fatalf("DeleteCascade: %v", err)
	}

	if count(t, conn, `SELECT count(*) FROM match WHERE id=$1`, matchID) != 0 {
		t.Error("match not deleted")
	}
	if count(t, conn, `SELECT count(*) FROM game WHERE id=$1`, gameID) != 0 {
		t.Error("game not cascade-deleted")
	}
	if count(t, conn, `SELECT count(*) FROM move WHERE id=$1`, moveID) != 0 {
		t.Error("move not cascade-deleted")
	}
	if count(t, conn, `SELECT count(*) FROM move_analysis WHERE move_id=$1`, moveID) != 0 {
		t.Error("move_analysis not cascade-deleted")
	}
	if count(t, conn, `SELECT count(*) FROM position WHERE id=$1`, posID) != 0 {
		t.Error("orphaned position not deleted")
	}
}

// --- helpers --------------------------------------------------------------

func collect[T any](t *testing.T, seq func(func(*T, error) bool)) []T {
	t.Helper()
	var out []T
	for v, err := range seq {
		if err != nil {
			t.Fatalf("iterate: %v", err)
		}
		out = append(out, *v)
	}
	return out
}

func collectMP(t *testing.T, seq func(func(*domain.MatchMovePosition, error) bool)) []domain.MatchMovePosition {
	t.Helper()
	return collect[domain.MatchMovePosition](t, seq)
}

// iterErr drains an iterator and returns the first error it yields.
func iterErr[T any](seq func(func(*T, error) bool)) (int, error) {
	n := 0
	for _, err := range seq {
		if err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func rawConn(t *testing.T, dsn string) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		t.Fatalf("raw connect: %v", err)
	}
	t.Cleanup(func() { conn.Close(context.Background()) })
	return conn
}

func count(t *testing.T, conn *pgx.Conn, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := conn.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("count query: %v", err)
	}
	return n
}
