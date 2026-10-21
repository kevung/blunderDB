package gui

import (
	"database/sql"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/maintenance"
)

type event struct {
	name    string
	payload any
}

// recordEvents routes a's events to a channel; onEvent, when set, runs on
// the emitting goroutine before the event is queued.
func recordEvents(a *App, onEvent func(name string)) <-chan event {
	ch := make(chan event, 64)
	hook := func(name string, payload ...any) {
		if onEvent != nil {
			onEvent(name)
		}
		var p any
		if len(payload) > 0 {
			p = payload[0]
		}
		ch <- event{name, p}
	}
	a.emitHook.Store(&hook)
	return ch
}

// lastEvent drains ch up to the first event that ends a pass.
func lastEvent(t *testing.T, ch <-chan event) (event, []string) {
	t.Helper()
	var seen []string
	deadline := time.After(30 * time.Second)
	for {
		select {
		case e := <-ch:
			seen = append(seen, e.name)
			if e.name != "maintenance:progress" {
				return e, seen
			}
		case <-deadline:
			t.Fatalf("no end event; saw %v", seen)
		}
	}
}

// matchWithoutStats opens a library holding one match whose statistics are
// missing: the stats.matches line has one unit of work.
func matchWithoutStats(t *testing.T) (*App, *database.Database) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "m.db")
	d := database.NewDatabase()
	if err := d.SetupDatabase(path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if _, err := d.ImportXGMatch(filepath.Join("..", "..", "testdata", "test.xg")); err != nil {
		t.Fatal(err)
	}
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	if _, err := raw.Exec(`DELETE FROM match_stats_cell; DELETE FROM match_stats`); err != nil {
		t.Fatal(err)
	}
	return NewApp(d), d
}

func TestStartMaintenance_ReportsProgressThenTheRun(t *testing.T) {
	a, _ := matchWithoutStats(t)
	events := recordEvents(a, nil)
	a.StartMaintenance(string(maintenance.StatsMatches))
	end, seen := lastEvent(t, events)
	if end.name != "maintenance:done" {
		t.Fatalf("ended with %s %v, want maintenance:done", end.name, end.payload)
	}
	if seen[0] != "maintenance:progress" {
		t.Errorf("events %v: no progress before the end", seen)
	}
	run, ok := end.payload.(maintenance.Run)
	if !ok || run.Changed != 1 || run.Outcome != maintenance.OutcomeDone {
		t.Errorf("done payload = %#v, want the Run of one filled match", end.payload)
	}
}

func TestStartMaintenance_CancelEndsThePass(t *testing.T) {
	a, _ := matchWithoutStats(t)
	var once sync.Once
	events := recordEvents(a, func(name string) {
		if name == "maintenance:progress" {
			once.Do(a.CancelMaintenance)
		}
	})
	a.StartMaintenance(string(maintenance.StatsMatches))
	if end, seen := lastEvent(t, events); end.name != "maintenance:cancelled" {
		t.Fatalf("events %v, last payload %v: want maintenance:cancelled", seen, end.payload)
	}
}

// A pass asked for while another job holds the library is refused with the
// BusyError text the frontend's busyMessage reads.
func TestStartMaintenance_RefusedWhileABatchHoldsTheLibrary(t *testing.T) {
	a, _, release, _ := queueBehindRollout(t)
	defer release()
	events := recordEvents(a, nil)
	a.StartMaintenance(string(maintenance.StatsMatches))
	end, _ := lastEvent(t, events)
	msg, _ := end.payload.(map[string]string)
	if end.name != "maintenance:error" || !strings.Contains(msg["message"], "database is busy: "+database.JobRollout) {
		t.Fatalf("got %s %v, want the rollout's busy refusal", end.name, end.payload)
	}
}

func TestStartMaintenance_NoopWithoutDatabase(t *testing.T) {
	a := NewApp(nil)
	a.StartMaintenance(string(maintenance.StatsMatches))
	a.CancelMaintenance()
}

// The fill an open starts reports as the stats.matches line running, ending
// on maintenance:done with the matches it filled, which the panels waiting
// on that line read.
func TestMatchStatsFill_ReportsAsTheStatsMatchesLine(t *testing.T) {
	a, d := matchWithoutStats(t)
	events := recordEvents(a, nil)
	if !d.FillMatchStatsInBackground() {
		t.Fatal("no background fill started")
	}
	end, seen := lastEvent(t, events)
	run, ok := end.payload.(maintenance.Run)
	if end.name != "maintenance:done" || !ok || run.Kind != maintenance.StatsMatches || run.Changed != 1 {
		t.Fatalf("ended with %s %+v, want maintenance:done for stats.matches, 1 changed", end.name, end.payload)
	}
	if seen[0] != "maintenance:progress" {
		t.Errorf("events %v: no progress before the end", seen)
	}
}
