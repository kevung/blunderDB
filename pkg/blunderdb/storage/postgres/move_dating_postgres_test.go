//go:build postgres

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func datingPos(n int) domain.Position {
	p := domain.InitializePosition()
	p.DecisionType = domain.CheckerAction
	p.Score = [2]int{n, 0}
	return p
}

func day(s string) time.Time {
	d, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return d
}

// datingFixture writes, as the app role under RLS, the moves of a new game of
// a match dated date (zero: no date), one per position, through CreateMoves.
type datingFixture struct {
	t     *testing.T
	ctx   context.Context
	admin *pgxpool.Pool
	s     *Storage
}

func (f datingFixture) tenantCtx(scope string) context.Context {
	n, err := storage.ParseTenant(scope)
	if err != nil {
		f.t.Fatal(err)
	}
	return storage.WithTenant(f.ctx, n)
}

func (f datingFixture) position(scope string, n int) int64 {
	f.t.Helper()
	p := datingPos(n)
	id, err := f.s.Positions().Save(f.tenantCtx(scope), scope, &p)
	if err != nil {
		f.t.Fatalf("Save position %d (%s): %v", n, scope, err)
	}
	return id
}

func (f datingFixture) game(scope string, date time.Time) int64 {
	f.t.Helper()
	ctx := f.tenantCtx(scope)
	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7, MatchDate: date}
	matchID, err := f.s.Matches().Save(ctx, scope, &m)
	if err != nil {
		f.t.Fatalf("Save match: %v", err)
	}
	gameID, err := f.s.Matches().CreateGame(ctx, scope, &domain.Game{MatchID: matchID, GameNumber: 1})
	if err != nil {
		f.t.Fatalf("CreateGame: %v", err)
	}
	return gameID
}

// play writes a move per position (0: a move without a position), plus one
// without a position so CreateMoves takes its bulk path.
func (f datingFixture) play(scope string, gameID int64, positions ...int64) error {
	var mvs []*domain.Move
	for i, p := range append(positions, 0) {
		mvs = append(mvs, &domain.Move{GameID: gameID, MoveNumber: int32(i + 1), MoveType: "checker",
			PositionID: p, Player: 1, CheckerMove: "13/11 24/23"})
	}
	return f.s.Matches().CreateMoves(f.tenantCtx(scope), scope, mvs)
}

func (f datingFixture) dateOf(id int64) string {
	f.t.Helper()
	var d *string
	if err := f.admin.QueryRow(f.ctx, `SELECT match_date::date::text FROM position WHERE id = $1`, id).Scan(&d); err != nil {
		f.t.Fatal(err)
	}
	if d == nil {
		return "none"
	}
	return *d
}

// A position takes the earliest date of the matches its moves belong to, in its
// own tenant only, whatever the order of the imports.
func TestMoveDatingResultUnderRLS(t *testing.T) {
	ctx, admin, _, s, _ := rlsLibrary(t)
	f := datingFixture{t, ctx, admin, s}

	shared := f.position("1", 1)   // in two matches
	undated := f.position("1", 2)  // only in a match without a date
	predated := f.position("1", 3) // already dated earlier than its match
	twin := f.position("2", 1)     // tenant 2's copy of shared
	if _, err := admin.Exec(ctx, `UPDATE position SET match_date = '2019-01-01' WHERE id = $1`, predated); err != nil {
		t.Fatal(err)
	}

	for _, step := range []struct {
		scope     string
		date      string
		positions []int64
	}{
		{"1", "2021-03-01", []int64{shared, predated}},
		{"1", "", []int64{undated, shared}},
		{"1", "2021-01-01", []int64{shared}},
		{"1", "2021-01-01", []int64{shared}}, // the same match imported again
		{"1", "2021-02-01", []int64{shared}},
	} {
		var date time.Time
		if step.date != "" {
			date = day(step.date)
		}
		if err := f.play(step.scope, f.game(step.scope, date), step.positions...); err != nil {
			t.Fatalf("CreateMoves(%s, %q): %v", step.scope, step.date, err)
		}
	}
	for _, c := range []struct {
		what string
		id   int64
		want string
	}{
		{"position in two matches", shared, "2021-01-01"},
		{"position only in an undated match", undated, "none"},
		{"position dated before its match", predated, "2019-01-01"},
		{"tenant 2's identical position", twin, "none"},
	} {
		if got := f.dateOf(c.id); got != c.want {
			t.Errorf("%s: match_date = %s, want %s", c.what, got, c.want)
		}
	}

	if err := f.play("2", f.game("2", day("2022-05-01")), twin); err != nil {
		t.Fatal(err)
	}
	if got := f.dateOf(twin); got != "2022-05-01" {
		t.Errorf("tenant 2's position after its own import: %s, want 2022-05-01", got)
	}
	if got := f.dateOf(shared); got != "2021-01-01" {
		t.Errorf("tenant 1's position after tenant 2's import: %s, want 2021-01-01", got)
	}
}

// Another transaction dates the position between the snapshot of the dating
// statement and its update — a second import of the tenant sharing the
// position. Postgres then rechecks the update's condition on the committed
// version of the row: the outcome must still be the earlier of the two dates.
func TestMoveDatingUnderConcurrentWrite(t *testing.T) {
	ctx, admin, _, s, _ := rlsLibrary(t)
	f := datingFixture{t, ctx, admin, s}

	for i, c := range []struct {
		concurrent, ours, want string
	}{
		{"2020-06-01", "2020-01-01", "2020-01-01"},
		{"2019-06-01", "2020-01-01", "2019-06-01"},
	} {
		t.Run(c.concurrent, func(t *testing.T) {
			pos := f.position("1", 10+i)
			gameID := f.game("1", day(c.ours))

			other, err := admin.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = other.Rollback(ctx) }()
			if _, err := other.Exec(ctx, `UPDATE position SET match_date = $1 WHERE id = $2`, day(c.concurrent), pos); err != nil {
				t.Fatal(err)
			}

			done := make(chan error, 1)
			go func() { done <- f.play("1", gameID, pos) }()

			// The dating statement holds its snapshot and waits for the row.
			deadline := time.Now().Add(20 * time.Second)
			for {
				var waiting int
				if err := admin.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity
					WHERE wait_event_type = 'Lock' AND query LIKE 'WITH ins AS%'`).Scan(&waiting); err != nil {
					t.Fatal(err)
				}
				if waiting > 0 {
					break
				}
				select {
				case err := <-done:
					t.Fatalf("the dating statement did not wait for the row: %v", err)
				default:
				}
				if time.Now().After(deadline) {
					t.Fatal("the dating statement never waited for the row")
				}
				time.Sleep(5 * time.Millisecond)
			}
			if err := other.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			if err := <-done; err != nil {
				t.Fatalf("CreateMoves: %v", err)
			}
			if got := f.dateOf(pos); got != c.want {
				t.Errorf("match_date = %s, want %s (concurrent %s, ours %s)", got, c.want, c.concurrent, c.ours)
			}
		})
	}
}
