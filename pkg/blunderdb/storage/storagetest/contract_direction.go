package storagetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func newDirectedTournament(t *testing.T, s storage.Storage, name string) int64 {
	t.Helper()
	ctx := context.Background()
	tid, err := s.Tournaments().Create(ctx, "", name, "2026-10-03", "")
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	rec := direction.Record{TournamentID: tid, FormatVersion: 1, EngineVersion: direction.EngineVersion,
		State: direction.StateDraft, Config: `{"name":"` + name + `"}`}
	if err := s.Directions().Create(ctx, "", rec); err != nil {
		t.Fatalf("create direction: %v", err)
	}
	return tid
}

// testDirectionRecord: a record is created for an existing Tournament only,
// once, reads back as written, and an undirected Tournament reads as
// direction.ErrNoDirection — the error the direction package tests for.
func testDirectionRecord(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ds := s.Directions()

	if err := ds.Create(ctx, "", direction.Record{TournamentID: 999999, State: direction.StateDraft}); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("Create on a missing Tournament = %v, want ErrNotFound", err)
	}
	plain, _ := s.Tournaments().Create(ctx, "", "Importé", "2026-10-01", "")
	if _, err := ds.Get(ctx, "", plain); !errors.Is(err, direction.ErrNoDirection) || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("Get undirected = %v, want ErrNoDirection and ErrNotFound", err)
	}

	tid := newDirectedTournament(t, s, "Principal")
	if err := ds.Create(ctx, "", direction.Record{TournamentID: tid, State: direction.StateDraft}); !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("second Create = %v, want ErrConflict", err)
	}
	rec, err := ds.Get(ctx, "", tid)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if rec.TournamentID != tid || rec.FormatVersion != 1 || rec.EngineVersion != direction.EngineVersion ||
		rec.State != direction.StateDraft || rec.Config != `{"name":"Principal"}` || rec.OutputDir != "" {
		t.Fatalf("Get = %+v", rec)
	}
	if rec.CreatedAt.IsZero() || rec.UpdatedAt.IsZero() {
		t.Errorf("timestamps not read back: %+v", rec)
	}

	rec.State = direction.StateRunning
	rec.OutputDir = "/tmp/salle"
	if err := ds.Update(ctx, "", rec); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := ds.Get(ctx, "", tid)
	if got.State != direction.StateRunning || got.OutputDir != "/tmp/salle" {
		t.Fatalf("after Update = %+v", got)
	}
	if err := ds.Update(ctx, "", direction.Record{TournamentID: plain}); !errors.Is(err, direction.ErrNoDirection) {
		t.Fatalf("Update undirected = %v, want ErrNoDirection", err)
	}

	second := newDirectedTournament(t, s, "Speed")
	list, err := ds.List(ctx, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 || list[0].TournamentID != tid || list[1].TournamentID != second {
		t.Fatalf("List = %+v, want %d then %d", list, tid, second)
	}
}

// testDirectionLogIsAppendOnly: events come back in sequence order with
// their time to the sub-second, and a sequence number already written is
// refused without touching the first event (ADR-0047).
func testDirectionLogIsAppendOnly(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ds := s.Directions()
	tid := newDirectedTournament(t, s, "Principal")

	at := time.Date(2026, 10, 3, 9, 30, 15, 123456000, time.UTC)
	// Written out of order: the log is read by seq, not by insertion.
	for _, ev := range []direction.StoredEvent{
		{Seq: 1, Kind: "result", Time: at.Add(time.Minute), Payload: []byte(`{"m":"M1"}`)},
		{Seq: 0, Kind: "created", Time: at, Payload: []byte(`{}`)},
	} {
		if err := ds.AppendEvent(ctx, "", tid, ev); err != nil {
			t.Fatalf("AppendEvent %d: %v", ev.Seq, err)
		}
	}
	err := ds.AppendEvent(ctx, "", tid, direction.StoredEvent{Seq: 1, Kind: "result", Time: at, Payload: []byte(`{"m":"M2"}`)})
	if !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("AppendEvent at an existing seq = %v, want ErrConflict", err)
	}
	plain, _ := s.Tournaments().Create(ctx, "", "Importé", "2026-10-01", "")
	if err := ds.AppendEvent(ctx, "", plain, direction.StoredEvent{Seq: 0, Kind: "created", Time: at, Payload: []byte(`{}`)}); !errors.Is(err, direction.ErrNoDirection) {
		t.Fatalf("AppendEvent undirected = %v, want ErrNoDirection", err)
	}

	if n, last, err := ds.EventsHead(ctx, "", tid); err != nil || n != 2 || last != 1 {
		t.Errorf("EventsHead = %d, %d, %v; want 2 events, last seq 1", n, last, err)
	}
	if n, last, err := ds.EventsHead(ctx, "", plain); err != nil || n != 0 || last != -1 {
		t.Errorf("EventsHead of an empty log = %d, %d, %v; want 0, -1", n, last, err)
	}

	evs, err := ds.LoadEvents(ctx, "", tid)
	if err != nil {
		t.Fatalf("LoadEvents: %v", err)
	}
	if len(evs) != 2 {
		t.Fatalf("LoadEvents = %d events, want 2", len(evs))
	}
	if evs[0].Seq != 0 || evs[0].Kind != "created" || !evs[0].Time.Equal(at) {
		t.Errorf("event 0 = %+v, want created at %v", evs[0], at)
	}
	if evs[1].Seq != 1 || string(evs[1].Payload) != `{"m":"M1"}` || !evs[1].Time.Equal(at.Add(time.Minute)) {
		t.Errorf("event 1 = %+v, want the first write kept", evs[1])
	}
}

// testDirectionDelete: deleting a Direction takes its log with it and leaves
// the Tournament in place.
func testDirectionDelete(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ds := s.Directions()
	tid := newDirectedTournament(t, s, "Principal")
	if err := ds.AppendEvent(ctx, "", tid, direction.StoredEvent{Seq: 0, Kind: "created", Time: time.Now(), Payload: []byte(`{}`)}); err != nil {
		t.Fatalf("AppendEvent: %v", err)
	}
	if err := ds.Delete(ctx, "", tid); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := ds.Get(ctx, "", tid); !errors.Is(err, direction.ErrNoDirection) {
		t.Fatalf("Get after Delete = %v, want ErrNoDirection", err)
	}
	if evs, err := ds.LoadEvents(ctx, "", tid); err != nil || len(evs) != 0 {
		t.Fatalf("LoadEvents after Delete = %d, %v; want none", len(evs), err)
	}
	if _, err := s.Tournaments().Get(ctx, "", tid); err != nil {
		t.Fatalf("Tournament after Delete: %v", err)
	}
	// Directed again from scratch: the old log does not come back.
	newDirectedTournamentFor(t, s, tid)
	if err := ds.AppendEvent(ctx, "", tid, direction.StoredEvent{Seq: 0, Kind: "created", Time: time.Now(), Payload: []byte(`{}`)}); err != nil {
		t.Fatalf("AppendEvent at seq 0 after Delete: %v", err)
	}
}

func newDirectedTournamentFor(t *testing.T, s storage.Storage, tid int64) {
	t.Helper()
	if err := s.Directions().Create(context.Background(), "", direction.Record{TournamentID: tid, FormatVersion: 1, State: direction.StateDraft}); err != nil {
		t.Fatalf("re-create direction: %v", err)
	}
}

// testDirectionTxRollback: an event appended inside a transaction is gone
// when the transaction rolls back — a gesture that writes several Directions
// lands whole or not at all (ADR-0056 §2).
func testDirectionTxRollback(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	a := newDirectedTournament(t, s, "Principal")
	b := newDirectedTournament(t, s, "Speed")

	tx, err := s.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	for _, tid := range []int64{a, b} {
		if err := tx.Directions().AppendEvent(ctx, "", tid, direction.StoredEvent{Seq: 0, Kind: "created", Time: time.Now(), Payload: []byte(`{}`)}); err != nil {
			_ = tx.Rollback()
			t.Fatalf("tx AppendEvent %d: %v", tid, err)
		}
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	for _, tid := range []int64{a, b} {
		if evs, err := s.Directions().LoadEvents(ctx, "", tid); err != nil || len(evs) != 0 {
			t.Fatalf("LoadEvents(%d) after rollback = %d, %v; want none", tid, len(evs), err)
		}
	}
}

// testDirectionBindsToThePackageStore: BindDirection hands the direction
// package a Store over the backend that keeps the append-only refusal.
func testDirectionBindsToThePackageStore(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	tid := newDirectedTournament(t, s, "Principal")
	st := storage.BindDirection(s.Directions(), "")
	if err := st.AppendEvent(ctx, tid, direction.StoredEvent{Seq: 0, Kind: "created", Time: time.Now(), Payload: []byte(`{}`)}); err != nil {
		t.Fatalf("AppendEvent: %v", err)
	}
	if err := st.AppendEvent(ctx, tid, direction.StoredEvent{Seq: 0, Kind: "created", Time: time.Now(), Payload: []byte(`{}`)}); err == nil {
		t.Fatal("the bound Store accepted a second write at seq 0")
	}
	rec, err := st.GetDirection(ctx, tid)
	if err != nil || rec.TournamentID != tid {
		t.Fatalf("GetDirection = %+v, %v", rec, err)
	}
	if list, err := st.ListDirections(ctx); err != nil || len(list) != 1 {
		t.Fatalf("ListDirections = %d, %v", len(list), err)
	}
	if evs, err := st.LoadEvents(ctx, tid); err != nil || len(evs) != 1 {
		t.Fatalf("LoadEvents = %d, %v", len(evs), err)
	}
}
