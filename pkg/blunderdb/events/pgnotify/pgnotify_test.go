package pgnotify

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/events"
)

// TestTransport_StartFailsWithoutDatabase: a daemon that cannot listen does not start.
func TestTransport_StartFailsWithoutDatabase(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := Start(ctx, "postgres://nobody@127.0.0.1:1/none?sslmode=disable&connect_timeout=1", events.NewBus(0), Options{}); err == nil {
		t.Fatal("Start succeeded without a database")
	}
}

// TestTransport_DeliverChecksWhatItBelieves: any role that may connect may NOTIFY on the
// channel; a payload with a scope that is not a tenant, or a kind no emitter publishes, reaches
// no subscriber.
func TestTransport_DeliverChecksWhatItBelieves(t *testing.T) {
	bus := events.NewBus(0)
	tr := &Transport{bus: bus, instance: "me", log: slog.New(slog.DiscardHandler)}
	sub, err := bus.Subscribe("3", events.Filter{}, 8)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		`not json`,
		`{"i":"me","s":"3","e":{"kind":"direction","tournamentId":1}}`,
		`{"i":"x","s":"alice","e":{"kind":"direction","tournamentId":1}}`,
		`{"i":"x","s":"03","e":{"kind":"direction","tournamentId":1}}`,
		`{"i":"x","s":"3","e":{"kind":"wipe","tournamentId":1}}`,
	} {
		tr.deliver(p)
	}
	select {
	case d := <-sub.C:
		t.Fatalf("a forged or own notification was delivered: %+v", d)
	default:
	}
	tr.deliver(`{"i":"x","s":"3","e":{"kind":"direction","tournamentId":4,"version":"v"}}`)
	if d := <-sub.C; d.Event.Scope != "3" || d.Event.TournamentID != 4 || d.Event.Version != "v" {
		t.Fatalf("got %+v", d)
	}
	tr.deliver(`{"i":"x","s":"3"}`)
	if d := <-sub.C; d.Event.Kind != events.KindResync || d.Event.Reason != ReasonMissed {
		t.Fatalf("got %+v; want a resync", d)
	}
}

// TestTransport_SendOnlyWaitsForNothing: a process that only announces starts without the
// database, delivers locally, and closes within its bound.
func TestTransport_SendOnlyWaitsForNothing(t *testing.T) {
	bus := events.NewBus(0)
	sub, _ := bus.Subscribe("1", events.Filter{}, 2)
	tr, err := Start(context.Background(), "postgres://nobody@127.0.0.1:1/none?sslmode=disable&connect_timeout=1", bus, Options{SendOnly: true})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	tr.Publish(events.Event{Scope: "1", Kind: events.KindDirection, TournamentID: 2})
	if d := <-sub.C; d.Event.TournamentID != 2 {
		t.Fatalf("got %+v", d)
	}
	start := time.Now()
	tr.Close()
	if d := time.Since(start); d > closeTimeout+sendTimeout {
		t.Fatalf("Close took %v", d)
	}
}
