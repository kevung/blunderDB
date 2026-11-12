package direction

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"testing"
	"time"

	tournoi "github.com/kevung/nicomaque"
)

// journalFixture is a log written by an earlier engine and the final ranking that engine gave
// it — computed by that engine, so the comparison is not the current engine agreeing with itself.
type journalFixture struct {
	Name    string            `json:"name"`
	Journal []json.RawMessage `json:"journal"`
	Ranking []string          `json:"ranking"`
}

func rankingIDs(d *Direction) []string {
	var ids []string
	for _, r := range d.Ranking() {
		ids = append(ids, string(r.Player))
	}
	return ids
}

// TestAnEarlierJournalStillReads: a Direction stored under journal format 2 (Nicomaque v0.8.0,
// with withdrawals, corrections and cancellations) reopens under the current engine with the
// standings it had, and takes new events written in the current format after the old ones.
func TestAnEarlierJournalStillReads(t *testing.T) {
	raw, err := os.ReadFile("testdata/journal_v2_poules_tableau.json")
	if err != nil {
		t.Fatal(err)
	}
	var fx journalFixture
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	store := newMemStore()
	if err := store.CreateDirection(ctx, Record{TournamentID: 1, FormatVersion: 2, State: StateFinished}); err != nil {
		t.Fatal(err)
	}
	for i, payload := range fx.Journal {
		var ev tournoi.Event
		if err := json.Unmarshal(payload, &ev); err != nil {
			t.Fatal(err)
		}
		if ev.Version != 2 {
			t.Fatalf("fixture event %d has version %d, want 2", i, ev.Version)
		}
		se := StoredEvent{Seq: i, Kind: string(ev.Kind), Time: ev.Time, Payload: payload}
		if err := store.AppendEvent(ctx, 1, se); err != nil {
			t.Fatal(err)
		}
	}

	d, err := Open(ctx, store, 1)
	if err != nil {
		t.Fatalf("an earlier journal must reopen: %v", err)
	}
	if !d.State().Finished {
		t.Fatal("the stored tournament was finished")
	}
	if got := rankingIDs(d); !slices.Equal(got, fx.Ranking) {
		t.Fatalf("ranking %v, want %v (as the engine that wrote it gave it)", got, fx.Ranking)
	}

	now := time.Date(2026, 11, 12, 21, 0, 0, 0, time.UTC)
	if err := d.Reopen(ctx, now); err != nil {
		t.Fatal(err)
	}
	if err := d.Apply(ctx, tournoi.NoteEvent("relu", now)); err != nil {
		t.Fatal(err)
	}
	d, err = Open(ctx, store, 1)
	if err != nil {
		t.Fatalf("a journal mixing formats must reopen: %v", err)
	}
	if last := d.Journal()[len(d.Journal())-1]; last.Version != tournoi.JournalVersion {
		t.Errorf("a new event carries format %d, want %d", last.Version, tournoi.JournalVersion)
	}
	if got := rankingIDs(d); !slices.Equal(got, fx.Ranking) {
		t.Errorf("after reopening, ranking %v, want %v", got, fx.Ranking)
	}
}

// TestAnEarlierJournalContinues: a tournament stored IN PROGRESS under format 2 reopens and
// carries on under the current engine. The rest of its decisions, as the earlier engine recorded
// them, apply one by one and end on the standings that engine gave.
func TestAnEarlierJournalContinues(t *testing.T) {
	raw, err := os.ReadFile("testdata/journal_v2_poules_tableau.json")
	if err != nil {
		t.Fatal(err)
	}
	var fx journalFixture
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, cut := range []int{len(fx.Journal) / 3, len(fx.Journal) / 2, len(fx.Journal) - 2} {
		store := newMemStore()
		if err := store.CreateDirection(ctx, Record{TournamentID: 1, FormatVersion: 2, State: StateRunning}); err != nil {
			t.Fatal(err)
		}
		for i, payload := range fx.Journal[:cut] {
			var ev tournoi.Event
			if err := json.Unmarshal(payload, &ev); err != nil {
				t.Fatal(err)
			}
			if err := store.AppendEvent(ctx, 1, StoredEvent{Seq: i, Kind: string(ev.Kind), Time: ev.Time, Payload: payload}); err != nil {
				t.Fatal(err)
			}
		}
		d, err := Open(ctx, store, 1)
		if err != nil {
			t.Fatalf("cut %d: an earlier journal in progress must reopen: %v", cut, err)
		}
		if d.State().Finished {
			t.Fatalf("cut %d: the stored tournament was still running", cut)
		}
		for i, payload := range fx.Journal[cut:] {
			var ev tournoi.Event
			if err := json.Unmarshal(payload, &ev); err != nil {
				t.Fatal(err)
			}
			if err := d.Apply(ctx, ev); err != nil {
				t.Fatalf("cut %d: decision %d (%s) refused under the current engine: %v", cut, cut+i, ev.Kind, err)
			}
		}
		d, err = Open(ctx, store, 1)
		if err != nil {
			t.Fatal(err)
		}
		if got := rankingIDs(d); !slices.Equal(got, fx.Ranking) {
			t.Errorf("cut %d: ranking %v, want %v", cut, got, fx.Ranking)
		}
	}
}
