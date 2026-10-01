package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction/service"
	"github.com/kevung/blunderdb/pkg/blunderdb/events"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// recorder keeps what was published.
type recorder struct{ got []events.Event }

func (r *recorder) Publish(ev events.Event) { r.got = append(r.got, ev) }

// TestGesturesArePublishedAfterCommit: whoever calls the service — the daemon, the desktop's
// façade — a committed gesture is published once with the version a read now gives, and a
// refused or stale one is not.
func TestGesturesArePublishedAfterCommit(t *testing.T) {
	ctx := context.Background()
	st, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	rec := &recorder{}
	mem := &service.Memory{}
	mem.SetPublisher(rec)
	svc := service.New(st, "", mem)
	tid, err := st.Tournaments().Create(ctx, "", "Open", "2026-09-12", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Open","tables":{"count":4},"phases":[{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous"}]}`
	if err := svc.CreateDirection(ctx, tid, cfg, 7); err != nil {
		t.Fatal(err)
	}
	if len(rec.got) != 1 || rec.got[0].Kind != events.KindDirection || rec.got[0].TournamentID != tid {
		t.Fatalf("after CreateDirection: %+v", rec.got)
	}
	v, _ := svc.DirectionVersion(ctx, tid)
	if rec.got[0].Version != v {
		t.Errorf("published version %q, read %q", rec.got[0].Version, v)
	}

	// Refused by the rules, and stale: nothing committed, nothing published.
	if err := svc.CreateDirection(ctx, tid, cfg, 7); err == nil {
		t.Fatal("a second CreateDirection applied")
	}
	if _, err := svc.AddDirectionNote(service.ExpectVersion(ctx, "stale"), tid, "x"); !errors.Is(err, service.ErrStale) {
		t.Fatalf("stale note: %v", err)
	}
	if len(rec.got) != 1 {
		t.Fatalf("a failed gesture was published: %+v", rec.got[1:])
	}

	// A room: its creation, then a member's gesture names the room and its members.
	room, err := svc.CreateRencontre(ctx, "Salle", "", "", 4)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AttachToRencontre(ctx, tid, room.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddDirectionNote(ctx, tid, "dans la salle"); err != nil {
		t.Fatal(err)
	}
	last := rec.got[len(rec.got)-1]
	if last.Kind != events.KindRencontre || last.RencontreID != room.ID || len(last.TournamentIDs) != 1 || last.TournamentIDs[0] != tid {
		t.Fatalf("member gesture published %+v", last)
	}

	// Detach: the room no longer lists the Direction, but its event still names it, and the
	// Direction is published on its own.
	if err := svc.DetachFromRencontre(ctx, tid); err != nil {
		t.Fatal(err)
	}
	n := len(rec.got)
	room2, dir := rec.got[n-2], rec.got[n-1]
	if room2.Kind != events.KindRencontre || len(room2.TournamentIDs) != 1 || dir.Kind != events.KindDirection || dir.TournamentID != tid {
		t.Fatalf("detach published %+v, %+v", room2, dir)
	}
}

// TestSlowSubscriberNeverBlocksAGesture: a subscriber that never reads is dropped; the gestures
// all apply.
func TestSlowSubscriberNeverBlocksAGesture(t *testing.T) {
	ctx := context.Background()
	st, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	bus := events.NewBus()
	mem := &service.Memory{}
	mem.SetPublisher(bus)
	sub, _ := bus.Subscribe("", events.Filter{}, 1)
	svc := service.New(st, "", mem)
	tid, _ := st.Tournaments().Create(ctx, "", "Open", "2026-09-12", "")
	cfg := `{"name":"Open","tables":{"count":4},"phases":[{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous"}]}`
	if err := svc.CreateDirection(ctx, tid, cfg, 7); err != nil {
		t.Fatal(err)
	}
	for range 5 {
		if _, err := svc.AddDirectionNote(ctx, tid, "n"); err != nil {
			t.Fatal(err)
		}
	}
	got := 0
	for range sub.C {
		got++
	}
	if got != 1 || !sub.Overflowed() {
		t.Fatalf("slow subscriber: %d deliveries, overflowed %v", got, sub.Overflowed())
	}
	evs, _ := st.Directions().LoadEvents(ctx, "", tid)
	if len(evs) < 6 {
		t.Fatalf("%d events recorded", len(evs))
	}
}
