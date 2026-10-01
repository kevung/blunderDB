//go:build postgres

// Needs Docker (testcontainers), like handlers_tenant_test.go:
//
//	go test -tags postgres ./internal/server/... -run TestEvents_TwoInstances -v
package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/internal/server/metrics"
	"github.com/kevung/blunderdb/pkg/blunderdb/events"
	"github.com/kevung/blunderdb/pkg/blunderdb/events/pgnotify"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
)

// instanceOn is a daemon over dsn with its own pool, as a second process would have.
func instanceOn(t *testing.T, dsn string) (*httptest.Server, *Server, storage.Storage) {
	t.Helper()
	st, err := pg.Open(context.Background(), dsn, nil)
	if err != nil {
		t.Fatalf("pg.Open: %v", err)
	}
	srv, err := New(Options{Storage: st, Metrics: metrics.New(), EnableDirection: true, Transcription: true, EventsDSN: dsn})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if srv.notify == nil {
		t.Fatal("no LISTEN/NOTIFY transport under a PostgreSQL EventsDSN")
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(func() {
		ts.Close()
		srv.Close()
		st.Close()
	})
	return ts, srv, st
}

// backends counts the connections of an instance's transport.
func backends(t *testing.T, conn *pgx.Conn, instance string) int {
	t.Helper()
	var n int
	if err := conn.QueryRow(context.Background(),
		`SELECT count(*) FROM pg_stat_activity WHERE application_name LIKE '%' || $1`, instance).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// noFrame fails when the stream delivers anything but heartbeats within d.
func (s *sseStream) noFrame(t *testing.T, d time.Duration) {
	t.Helper()
	deadline := time.After(d)
	for {
		select {
		case f, ok := <-s.frames:
			if !ok {
				t.Fatal("the stream ended")
			}
			if f.comment != "ping" {
				t.Fatalf("unexpected frame %+v", f)
			}
		case <-deadline:
			return
		}
	}
}

// seqOf is the sequence number an SSE id carries.
func seqOf(t *testing.T, id string) int {
	t.Helper()
	n, err := strconv.Atoi(id[strings.LastIndexByte(id, '-')+1:])
	if err != nil {
		t.Fatalf("id %q: %v", id, err)
	}
	return n
}

// TestEvents_TwoInstancesPostgres: two daemons over one PostgreSQL database hear each other's
// gestures, each tenant its own, each instance its own once.
func TestEvents_TwoInstancesPostgres(t *testing.T) {
	dsn := postgresDSN(t)
	tsA, _, stA := instanceOn(t, dsn)
	tsB, srvB, _ := instanceOn(t, dsn)
	mine := seedDirection(t, stA, "1", "Open de Lyon")
	theirs := seedDirection(t, stA, "2", "Open de Nice")
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })

	t.Run("a gesture on A reaches a subscriber of B", func(t *testing.T) {
		s := subscribe(t, tsB.URL, "1", "", nil)
		r := send(t, tsA, "1", "/v1/directions.addNote", mine.body(`"text":"un"`), mine.versionOf(t, tsA), "")
		if r.status != http.StatusOK {
			t.Fatalf("addNote: %d %s", r.status, r.body)
		}
		f := s.next(t)
		if f.event != string(events.KindRencontre) || f.data.RencontreID != mine.rencontreID {
			t.Fatalf("frame %+v", f)
		}
		if quoteVersion(f.data.Version) != r.version {
			t.Errorf("event version %q; the gesture answered %s", f.data.Version, r.version)
		}
	})

	t.Run("a tenant hears its own gestures only", func(t *testing.T) {
		s := subscribe(t, tsB.URL, "2", "", nil)
		send(t, tsA, "1", "/v1/directions.addNote", mine.body(`"text":"deux"`), mine.versionOf(t, tsA), "")
		if r := send(t, tsA, "2", "/v1/directions.addNote", theirs.body(`"text":"nice"`), theirs.versionOf(t, tsA), ""); r.status != http.StatusOK {
			t.Fatalf("tenant 2 addNote: %d", r.status)
		}
		if f := s.next(t); f.data.RencontreID != theirs.rencontreID {
			t.Fatalf("tenant 2 heard %+v", f)
		}
	})

	t.Run("an instance delivers its own gesture once", func(t *testing.T) {
		s := subscribe(t, tsA.URL, "1", "", nil)
		send(t, tsA, "1", "/v1/directions.addNote", mine.body(`"text":"trois"`), mine.versionOf(t, tsA), "")
		first := s.next(t)
		// Its own notification comes back to A within the wait; A must drop it.
		s.noFrame(t, 500*time.Millisecond)
		send(t, tsA, "1", "/v1/directions.addNote", mine.body(`"text":"quatre"`), mine.versionOf(t, tsA), "")
		second := s.next(t)
		if second.data.Version == first.data.Version || seqOf(t, second.id) != seqOf(t, first.id)+1 {
			t.Fatalf("frames %+v then %+v; want two gestures in a row", first, second)
		}
	})

	t.Run("a lost LISTEN connection resyncs, then listens again", func(t *testing.T) {
		s := subscribe(t, tsB.URL, "1", "?tournament=999", nil)
		var killed bool
		if err := conn.QueryRow(context.Background(),
			`SELECT bool_or(pg_terminate_backend(pid)) FROM pg_stat_activity WHERE application_name = $1`,
			"blunderdb-events-"+srvB.notify.Instance()).Scan(&killed); err != nil || !killed {
			t.Fatalf("terminate the listener: %v (killed %v)", err, killed)
		}
		f := s.next(t)
		if f.event != string(events.KindResync) || f.data.Reason != pgnotify.ReasonMissed {
			t.Fatalf("frame %+v; want a resync after the lost connection", f)
		}
		all := subscribe(t, tsB.URL, "1", "", nil)
		send(t, tsA, "1", "/v1/directions.addNote", mine.body(`"text":"cinq"`), mine.versionOf(t, tsA), "")
		if f := all.next(t); f.data.RencontreID != mine.rencontreID {
			t.Fatalf("after the reconnection B heard %+v", f)
		}
	})

	t.Run("a gesture through call reaches the daemons", func(t *testing.T) {
		s := subscribe(t, tsB.URL, "1", "", nil)
		out, err := captureStdout(t, func() error {
			return RunCall([]string{"directions.addNote", "--backend", "postgres", "--dsn", dsn, "--scope", "1",
				"--if-match", mine.versionOf(t, tsA), "--json", mine.body(`"text":"six"`)})
		})
		if err != nil {
			t.Fatalf("call: %v (%s)", err, out)
		}
		if f := s.next(t); f.data.RencontreID != mine.rencontreID {
			t.Fatalf("B heard %+v", f)
		}
	})

	t.Run("Close leaves no connection behind", func(t *testing.T) {
		inst := srvB.notify.Instance()
		if n := backends(t, conn, inst); n == 0 {
			t.Fatal("no transport connection before Close")
		}
		srvB.Close()
		waitFor(t, func() bool { return backends(t, conn, inst) == 0 }, "the transport's connections close")
		if srvB.publisher.Wants("1") {
			t.Error("a closed transport still wants events")
		}
	})
}
