//go:build postgres

// Needs Docker (testcontainers):
//
//	go test -tags postgres ./pkg/blunderdb/events/pgnotify/ -v
package pgnotify

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/kevung/blunderdb/pkg/blunderdb/events"
)

func dsnOf(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	c, err := tcpg.Run(ctx, "postgres:16-alpine", tcpg.WithDatabase("blunderdb"),
		tcpg.WithUsername("test"), tcpg.WithPassword("test"), tcpg.BasicWaitStrategies())
	if err != nil {
		if os.Getenv("BLUNDERDB_REQUIRE_PG") == "1" {
			t.Fatalf("postgres container unavailable (BLUNDERDB_REQUIRE_PG=1 requires Docker): %v", err)
		}
		t.Skipf("postgres container unavailable (Docker required): %v", err)
	}
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(c) })
	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	return dsn
}

func started(t *testing.T, dsn string, bus *events.Bus) *Transport {
	t.Helper()
	tr, err := Start(context.Background(), dsn, bus, Options{MinBackoff: 20 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(tr.Close)
	return tr
}

func next(t *testing.T, s *events.Subscription) events.Delivery {
	t.Helper()
	select {
	case d, ok := <-s.C:
		if !ok {
			t.Fatal("subscription closed")
		}
		return d
	case <-time.After(5 * time.Second):
		t.Fatal("no delivery")
		return events.Delivery{}
	}
}

// TestTransport_OversizeIsResync: an event whose payload would not fit reaches the other
// instance as a resync of its scope, never truncated.
func TestTransport_OversizeIsResync(t *testing.T) {
	dsn := dsnOf(t)
	busA, busB := events.NewBus(0), events.NewBus(0)
	a := started(t, dsn, busA)
	started(t, dsn, busB)
	sub, err := busB.Subscribe("7", events.Filter{Tournaments: []int64{1}}, 8)
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]int64, 2000)
	for i := range ids {
		ids[i] = int64(1_000_000 + i)
	}
	a.Publish(events.Event{Scope: "7", Kind: events.KindRencontre, RencontreID: 3, TournamentIDs: ids})
	if d := next(t, sub); d.Event.Kind != events.KindResync || d.Event.Reason != ReasonMissed {
		t.Fatalf("got %+v; want a resync", d)
	}
	a.Publish(events.Event{Scope: "7", Kind: events.KindDirection, TournamentID: 1, Version: "v"})
	if d := next(t, sub); d.Event.TournamentID != 1 || d.Event.Version != "v" || d.Event.Scope != "7" {
		t.Fatalf("got %+v", d)
	}
}
