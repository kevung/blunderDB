package database

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
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
	dir, err := direction.Create(ctx, d.DirectionStore(), tID, cfg, 7, time.Now())
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
	now := time.Now()
	for _, p := range players {
		if err := dir.Enter(ctx, p, now); err != nil {
			t.Fatalf("entry %s: %v", p.ID, err)
		}
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
	// Entering players does not start the tournament: only a launched match does.
	if recs[0].State != direction.StateDraft {
		t.Errorf("state %q, want %q", recs[0].State, direction.StateDraft)
	}
	if recs[0].EngineVersion != direction.EngineVersion {
		t.Errorf("engine version %q, want %q", recs[0].EngineVersion, direction.EngineVersion)
	}
}

// TestDirectionAPI_PreparationThenFirstMatch covers the shape the panel drives: direct a
// Tournament, edit the configuration and enter players while in preparation, then launch the
// first match — which is what ends preparation (issue #368, #369).
func TestDirectionAPI_PreparationThenFirstMatch(t *testing.T) {
	d := newTestDB(t)
	tID, err := d.CreateTournament("Open de Lyon", "2026-09-12", "Lyon")
	if err != nil {
		t.Fatal(err)
	}

	// A Tournament assembled from files is not directed, and saying so must not cost a replay.
	directed, err := d.HasDirection(tID)
	if err != nil || directed {
		t.Fatalf("a fresh Tournament is not directed: %v %v", directed, err)
	}

	cfg := `{"name":"Open de Lyon","tables":{"count":8},"phases":[
		{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous","target":16},
		{"kind":"lives_bracket","length":9,"final_length":11}]}`
	if err := d.CreateDirection(tID, cfg, 7); err != nil {
		t.Fatalf("create: %v", err)
	}
	if directed, err = d.HasDirection(tID); err != nil || !directed {
		t.Fatalf("the Tournament should now be directed: %v %v", directed, err)
	}

	v, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	if v.State != "draft" {
		t.Errorf("a new Direction is in preparation, not %q", v.State)
	}
	if v.EventCount != 1 {
		t.Errorf("the log starts with the created event, %d event(s)", v.EventCount)
	}
	if len(v.Config.Phases) != 2 {
		t.Fatalf("configuration not kept: %+v", v.Config)
	}

	// The director changes their mind while still in preparation.
	edited := `{"name":"Open de Lyon","tables":{"count":6},"phases":[
		{"kind":"swiss_lives","length":7,"lives":3,"mode":"continuous"},
		{"kind":"lives_bracket","length":9}]}`
	if err := d.SetDirectionConfig(tID, edited); err != nil {
		t.Fatalf("editing in preparation: %v", err)
	}
	if v, err = d.GetDirection(tID); err != nil {
		t.Fatal(err)
	}
	if v.Config.Phases[0].Lives != 3 || v.Config.Tables.Count != 6 {
		t.Errorf("the edited configuration was not kept: %+v", v.Config)
	}
	if v.State != "draft" {
		t.Error("changing the configuration does not start the tournament")
	}

	players := `[{"id":"alice","name":"Alice"},{"id":"bob","name":"Bob"},
	             {"id":"chloe","name":"Chloé"},{"id":"dan","name":"Dan"}]`
	if err := d.EnterParticipants(tID, players); err != nil {
		t.Fatalf("entries: %v", err)
	}
	if v, err = d.GetDirection(tID); err != nil {
		t.Fatal(err)
	}
	if len(v.Players) != 4 {
		t.Errorf("%d entrants written to the log, want 4", len(v.Players))
	}
	if v.State != "draft" {
		t.Error("entering players does not start the tournament")
	}
	if len(v.Proposals) == 0 {
		t.Error("with four entrants the engine already proposes something")
	}

	// Launching the first match is what ends preparation.
	after, err := d.ConfirmAllProposals(tID)
	if err != nil {
		t.Fatal(err)
	}
	if after.State != "running" {
		t.Errorf("the first launched match makes it running, state %q", after.State)
	}

	// And a listing sees it without replaying it.
	list, err := d.ListDirections()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].State != "running" {
		t.Fatalf("listing: %+v", list)
	}
}

// TestDirectionAPI_NoTranslatedText: nothing the panel receives is a sentence. Labels, notes,
// warnings and wait reasons are the engine's codes, rendered by the frontend — blunderDB
// speaks nine languages and these values live in the database.
func TestDirectionAPI_NoTranslatedText(t *testing.T) {
	d := newTestDB(t)
	tID, err := d.CreateTournament("T", "2026-09-12", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"T","tables":{"count":4},"phases":[{"kind":"swiss_lives","length":7,"lives":2}]}`
	if err := d.CreateDirection(tID, cfg, 7); err != nil {
		t.Fatal(err)
	}
	var players []string
	for i := 0; i < 8; i++ {
		id := string(rune('a' + i))
		players = append(players, `{"id":"`+id+`","name":"`+id+`"}`)
	}
	if err := d.EnterParticipants(tID, "["+strings.Join(players, ",")+"]"); err != nil {
		t.Fatal(err)
	}
	v, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	blob, err := json.Marshal(struct {
		P any
		W any
		R any
	}{v.Proposals, v.Warnings, v.Ranking})
	if err != nil {
		t.Fatal(err)
	}
	var raw any
	if err := json.Unmarshal(blob, &raw); err != nil {
		t.Fatal(err)
	}
	var walk func(any, string)
	walk = func(x any, path string) {
		switch val := x.(type) {
		case map[string]any:
			for k, e := range val {
				if k == "Text" || k == "text" || k == "name" || k == "club" {
					continue
				}
				walk(e, path+"."+k)
			}
		case []any:
			for _, e := range val {
				walk(e, path)
			}
		case string:
			for _, r := range val {
				if r > 127 || r == ' ' {
					t.Errorf("%s is %q — a sentence reached the frontend instead of a code", path, val)
					return
				}
			}
		}
	}
	walk(raw, "view")
}
