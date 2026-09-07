package database

import (
	"context"
	"errors"
	"testing"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// directedTournament creates a Tournament and starts a Direction on it.
func directedTournament(t *testing.T, d *Database, n int) (*direction.Direction, int64) {
	t.Helper()
	ctx := context.Background()
	tID, err := d.CreateTournament("Open de Lyon", "2026-09-12", "Lyon")
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	cfg := tournoi.Config{Name: "Open de Lyon", Tables: tournoi.Tables{Count: 8},
		Phases: []tournoi.PhaseConfig{
			{Kind: tournoi.KindSwissLives, Length: 7, Target: 16},
			{Kind: tournoi.KindLivesBracket, Length: 9},
		}}
	dir, err := direction.Create(ctx, d.DirectionStore(), tID, cfg)
	if err != nil {
		t.Fatalf("create direction: %v", err)
	}
	players := make([]tournoi.Player, n)
	for i := range players {
		id := tournoi.PlayerID(string(rune('a' + i%26)))
		if i >= 26 {
			id += tournoi.PlayerID(string(rune('a' + i/26)))
		}
		players[i] = tournoi.Player{ID: id, Name: string(id)}
	}
	if err := dir.Start(ctx, 7, time.Now(), players); err != nil {
		t.Fatalf("start: %v", err)
	}
	return dir, tID
}

// TestDirectionStore_AppendOnly: the log refuses a second write at the same sequence number.
// The rule is enforced by the composite primary key, which is why the test lives here and not
// in the direction package: only SQL can guarantee it.
func TestDirectionStore_AppendOnly(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	_, tID := directedTournament(t, d, 24)
	store := d.DirectionStore()

	events, err := store.LoadEvents(ctx, tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 {
		t.Fatal("a started Direction has a log")
	}
	dup := direction.StoredEvent{Seq: events[0].Seq, Kind: "result", Time: time.Now(), Payload: []byte(`{}`)}
	if err := store.AppendEvent(ctx, tID, dup); err == nil {
		t.Error("rewriting a sequence number must fail: the log is append-only")
	}
}

// TestDirectionStore_ReplayAfterReopen: the state is replayed from the database, never read
// from a stored copy.
func TestDirectionStore_ReplayAfterReopen(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	dir, tID := directedTournament(t, d, 24)
	// Play a few matches.
	for i := 0; i < 3; i++ {
		for _, a := range dir.Propose() {
			if a.Kind != tournoi.ActStartMatch {
				continue
			}
			ev, err := dir.EventFor(a, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if err := dir.Apply(ctx, ev); err != nil {
				t.Fatal(err)
			}
			if err := dir.Apply(ctx, tournoi.ResultEvent(ev.MatchID, ev.A, 7, 3, time.Now())); err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	want := len(dir.Journal())

	again, err := direction.Open(ctx, d.DirectionStore(), tID)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if got := len(again.Journal()); got != want {
		t.Errorf("reopened journal has %d events, want %d", got, want)
	}
	if len(again.Warnings()) != 0 {
		t.Errorf("reopening raised warnings: %v", again.Warnings())
	}
	if len(again.Ranking()) != len(dir.Ranking()) {
		t.Error("the replayed ranking should match the live one")
	}
}

// TestDirectionStore_DeleteKeepsMatches: deleting a Direction empties the Slots and keeps every
// Match and the Tournament itself (ADR-0047).
func TestDirectionStore_DeleteKeepsMatches(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	dir, tID := directedTournament(t, d, 24)

	res, err := RawConn(d).Exec(`INSERT INTO match (player1_name, player2_name, match_length, tournament_id, direction_match_id) VALUES ('a','b',7,?, 'M1')`, tID)
	if err != nil {
		t.Fatal(err)
	}
	mID, _ := res.LastInsertId()

	if err := dir.Delete(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := direction.Open(ctx, d.DirectionStore(), tID); !errors.Is(err, direction.ErrNoDirection) {
		t.Errorf("after deletion: %v, want ErrNoDirection", err)
	}
	var slot string
	var tid *int64
	if err := RawConn(d).QueryRow(`SELECT direction_match_id, tournament_id FROM match WHERE id = ?`, mID).Scan(&slot, &tid); err != nil {
		t.Fatalf("the Match must still be there: %v", err)
	}
	if slot != "" {
		t.Errorf("the Slot should be empty, it is %q", slot)
	}
	if tid == nil || *tid != tID {
		t.Error("the Match keeps its Tournament")
	}
	var n int
	if err := RawConn(d).QueryRow(`SELECT COUNT(*) FROM tournament WHERE id = ?`, tID).Scan(&n); err != nil || n != 1 {
		t.Errorf("the Tournament must still be there (count %d): %v", n, err)
	}
}

// TestDirectionStore_UndirectedTournament: a Tournament assembled from files has no Direction,
// and that is not an error condition to hide.
func TestDirectionStore_UndirectedTournament(t *testing.T) {
	d := newTestDB(t)
	tID, err := d.CreateTournament("Imported", "2025-01-01", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := direction.Open(context.Background(), d.DirectionStore(), tID); !errors.Is(err, direction.ErrNoDirection) {
		t.Errorf("%v, want ErrNoDirection", err)
	}
}

// TestDirectionStore_List: directed tournaments are listable without replaying them.
func TestDirectionStore_List(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	_, tID := directedTournament(t, d, 24)
	recs, err := d.DirectionStore().ListDirections(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].TournamentID != tID {
		t.Fatalf("expected one direction on tournament %d, got %+v", tID, recs)
	}
	if recs[0].State != direction.StateRunning {
		t.Errorf("state %q, want %q", recs[0].State, direction.StateRunning)
	}
	if recs[0].EngineVersion != direction.EngineVersion {
		t.Errorf("engine version %q, want %q", recs[0].EngineVersion, direction.EngineVersion)
	}
}
