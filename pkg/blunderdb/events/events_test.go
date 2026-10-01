package events

import (
	"errors"
	"testing"
	"time"
)

func recv(t *testing.T, s *Subscription) (Delivery, bool) {
	t.Helper()
	select {
	case d, ok := <-s.C:
		return d, ok
	case <-time.After(2 * time.Second):
		t.Fatal("no delivery")
		return Delivery{}, false
	}
}

func empty(t *testing.T, s *Subscription) {
	t.Helper()
	select {
	case d, ok := <-s.C:
		if ok {
			t.Fatalf("unexpected delivery %+v", d)
		}
	default:
	}
}

func TestBus_DeliversWithinScopeOnly(t *testing.T) {
	b := NewBus()
	a, _ := b.Subscribe("1", Filter{}, 4)
	other, _ := b.Subscribe("2", Filter{}, 4)
	b.Publish(Event{Scope: "1", Kind: KindDirection, TournamentID: 7, Version: "v"})
	d, ok := recv(t, a)
	if !ok || d.Event.TournamentID != 7 || d.Seq != 1 {
		t.Fatalf("got %+v ok=%v", d, ok)
	}
	empty(t, other)
}

func TestBus_Filter(t *testing.T) {
	b := NewBus()
	byT, _ := b.Subscribe("", Filter{Tournaments: []int64{5}}, 4)
	byR, _ := b.Subscribe("", Filter{Rencontres: []int64{3}}, 4)
	byD, _ := b.Subscribe("", Filter{Transcriptions: []int64{9}}, 4)
	b.Publish(Event{Kind: KindRencontre, RencontreID: 3, TournamentIDs: []int64{4, 5}})
	if _, ok := recv(t, byT); !ok {
		t.Fatal("a member's subscriber misses its room's event")
	}
	if _, ok := recv(t, byR); !ok {
		t.Fatal("the room's subscriber misses its event")
	}
	empty(t, byD)
	b.Publish(Event{Kind: KindTranscription, TranscriptionID: 9, Revision: 2})
	if d, ok := recv(t, byD); !ok || d.Event.Revision != 2 {
		t.Fatalf("draft subscriber got %+v", d)
	}
	empty(t, byT)
	empty(t, byR)
}

// A subscriber that does not read is dropped once its queue is full; the publisher never waits.
func TestBus_SlowSubscriberIsDropped(t *testing.T) {
	b := NewBus()
	slow, _ := b.Subscribe("", Filter{}, 2)
	fast, _ := b.Subscribe("", Filter{}, 16)
	done := make(chan struct{})
	go func() {
		for range 5 {
			b.Publish(Event{Kind: KindDirection, TournamentID: 1})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Publish blocked on a slow subscriber")
	}
	n := 0
	for range slow.C {
		n++
	}
	if n != 2 || !slow.Overflowed() {
		t.Fatalf("slow got %d deliveries, overflowed=%v; want 2, true", n, slow.Overflowed())
	}
	if b.Subscribers() != 1 {
		t.Fatalf("subscribers = %d, want 1", b.Subscribers())
	}
	for range 5 {
		recv(t, fast)
	}
}

func TestBus_CancelAndClose(t *testing.T) {
	b := NewBus()
	s, _ := b.Subscribe("", Filter{}, 1)
	s.Cancel()
	s.Cancel()
	if _, ok := <-s.C; ok {
		t.Fatal("a cancelled subscription stays open")
	}
	s2, _ := b.Subscribe("", Filter{}, 1)
	b.Close()
	if _, ok := <-s2.C; ok {
		t.Fatal("Close leaves a subscription open")
	}
	if s2.Overflowed() {
		t.Fatal("a closed subscription reports an overflow")
	}
	if _, err := b.Subscribe("", Filter{}, 1); !errors.Is(err, ErrClosed) {
		t.Fatalf("Subscribe after Close: %v", err)
	}
	b.Publish(Event{})
	if b.Subscribers() != 0 {
		t.Fatal("subscribers left after Close")
	}
}
