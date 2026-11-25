package gui

import (
	"database/sql"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/jobs"
	"github.com/kevung/blunderdb/pkg/blunderdb/maintenance"
)

type event struct {
	name    string
	payload any
}

// recordEvents routes a's events to a channel, except the registry's
// (jobs:*), which recordJobEvents reads; onEvent, when set, runs on the
// emitting goroutine before the event is queued.
func recordEvents(a *App, onEvent func(name string)) <-chan event {
	ch := make(chan event, 64)
	hook := func(name string, payload ...any) {
		if strings.HasPrefix(name, "jobs:") {
			return
		}
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

// A pass asked for while a batch waits for the library takes its place in
// the library's queue behind it, instead of being refused, and runs once
// the batch is done.
func TestStartMaintenance_WaitsBehindTheQueuedBatch(t *testing.T) {
	a, _, release, stopped := queueBehindRollout(t)
	events := recordEvents(a, nil)
	a.StartMaintenance(string(maintenance.StatsMatches))
	select {
	case e := <-events:
		if e.name != "gammonnet-batch:progress" {
			t.Fatalf("got %s %v while the library was held, want the pass waiting", e.name, e.payload)
		}
	case <-time.After(50 * time.Millisecond):
	}
	if info := a.ListJobs(); len(info) != 2 || info[1].State != jobs.Waiting || info[1].Position != 1 {
		t.Fatalf("tasks %+v, want the pass waiting first in the queue", info)
	}
	release()
	waitStopped(t, stopped)
	for {
		end, seen := lastEvent(t, events)
		if strings.HasPrefix(end.name, "gammonnet-batch:") {
			continue
		}
		if end.name != "maintenance:done" {
			t.Fatalf("events %v ended with %s %v, want maintenance:done", seen, end.name, end.payload)
		}
		return
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
	waitTask(t, a, a.ListJobs()[0].ID, jobs.Done)
}

// A fill ended by the file changing under it ends its own task, cancelled,
// and the next fill is a task of its own.
func TestMatchStatsFill_EachFillIsItsOwnTaskEndedOnEveryPath(t *testing.T) {
	a, d := matchWithoutStats(t)
	_ = recordEvents(a, nil)
	if !d.FillMatchStatsInBackground() {
		t.Fatal("no background fill started")
	}
	first := a.ListJobs()[0]
	if first.Kind != jobs.MatchStatsFill {
		t.Fatalf("task %+v", first)
	}
	if err := a.jobs.Cancel(first.ID); err != nil {
		t.Fatal(err)
	}
	if got := waitTaskEnded(t, a, first.ID); got != jobs.Cancelled && got != jobs.Done {
		t.Fatalf("first fill ended %s", got)
	}
	if d.FillMatchStatsInBackground() {
		list := a.ListJobs()
		if len(list) != 2 || list[1].ID == first.ID {
			t.Fatalf("second fill shares a task: %+v", list)
		}
		waitTaskEnded(t, a, list[1].ID)
	}
}

func waitTaskEnded(t *testing.T, a *App, id int64) jobs.State {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		for _, in := range a.ListJobs() {
			if in.ID == id && in.State.Finished() {
				return in.State
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("task %d never ended: %+v", id, a.ListJobs())
	return ""
}
