package gui

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/rollout"
	"github.com/kevung/blunderdb/pkg/blunderdb/jobs"
)

// StartRollout refuses before starting anything: a request it cannot honour
// must not surface later as an error event.
func TestStartRollout_RefusesUpFront(t *testing.T) {
	a := NewApp(nil)
	pos := domain.InitializePosition()
	bad := rollout.Fast()
	bad.MaxGames = 100 // not a multiple of 36
	cases := map[string]RolloutRequest{
		"invalid settings":     {Position: &pos, Settings: bad},
		"no position":          {Settings: rollout.Fast()},
		"store an unsaved one": {Position: &pos, Settings: rollout.Fast(), Store: true},
		"id without database":  {PositionID: 1, Settings: rollout.Fast()},
	}
	for name, req := range cases {
		if _, err := a.StartRollout(req); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if presets := a.RolloutPresets(); len(presets) != 2 || presets[0].Settings != rollout.Fast() || presets[1].Settings != rollout.Standard() {
		t.Errorf("presets: %+v", presets)
	}

	// An unsaved board rolls out and stops cleanly when cancelled.
	if _, err := a.StartRollout(RolloutRequest{Position: &pos, Settings: rollout.Fast()}); err != nil {
		t.Fatal(err)
	}
	if done := a.cancelRollout(); done != nil {
		<-done
	}
}

// StartRolloutFiltered refuses what it cannot honour before starting.
func TestStartRolloutFiltered_RefusesUpFront(t *testing.T) {
	bad := rollout.Fast()
	bad.MaxGames = 100 // not a multiple of 36
	if _, err := (&App{}).StartRolloutFiltered("", bad); err == nil {
		t.Error("invalid settings accepted")
	}
	if _, err := (&App{}).StartRolloutFiltered("", rollout.Fast()); err == nil {
		t.Error("no database accepted")
	}

	d := database.NewDatabase()
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	a := NewApp(d)
	if _, err := a.StartRolloutFiltered("nosuchtoken:1", rollout.Fast()); err == nil {
		t.Error("unknown query token accepted")
	}

	// An empty library has nothing to roll out: the job ends by itself.
	if _, err := a.StartRolloutFiltered("", rollout.Fast()); err != nil {
		t.Fatal(err)
	}
	if done := a.cancelRollout(); done != nil {
		<-done
	}
}

// The rollout batch and the gammonNet batch exclude each other, as on the
// daemon: each takes every core. The one asked second waits its turn in the
// library's queue instead of being refused, and runs once the first ends.
func TestRolloutBatchWaitsBehindTheGammonNetBatch(t *testing.T) {
	d := database.NewDatabase()
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	a := NewApp(d)

	hold := a.jobs.Start(jobs.Spec{Kind: jobs.GammonNetMissing, Library: a.shownGroup(), Exclusive: true})
	if err := hold.Turn(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := a.StartRolloutFiltered("", rollout.Fast()); err != nil {
		t.Fatalf("a rollout batch asked beside a gammonNet batch: %v, want it queued", err)
	}
	if st := a.RolloutStatus(); !st.Running || st.Kind != "batch" {
		t.Errorf("RolloutStatus of the waiting batch: %+v", st)
	}
	list := a.ListJobs()
	if len(list) != 2 || list[1].Kind != jobs.RolloutBatch || list[1].State != jobs.Waiting {
		t.Fatalf("tasks %+v, want the rollout batch waiting", list)
	}
	hold.Finish(nil)
	deadline := time.Now().Add(10 * time.Second)
	for a.RolloutStatus().Running {
		if time.Now().After(deadline) {
			t.Fatal("the rollout batch never ran once the gammonNet batch ended")
		}
		time.Sleep(time.Millisecond)
	}
	if got := a.ListJobs()[1].State; got != jobs.Done {
		t.Errorf("rollout batch ended %s, want done (nothing to roll out)", got)
	}
}

// Opening another file stops the rollout in flight and waits for it, before
// the Database takes its lock.
func TestOpeningAnotherDatabaseStopsTheRollout(t *testing.T) {
	d := database.NewDatabase()
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	a := NewApp(d)
	pos := domain.InitializePosition()
	s := rollout.Standard()
	s.Truncation = 0
	if _, err := a.StartRollout(RolloutRequest{Position: &pos, Settings: s}); err != nil {
		t.Fatal(err)
	}
	a.roMu.Lock()
	stopped := a.roDone
	a.roMu.Unlock()
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-stopped:
	default:
		t.Error("the rollout still runs after the database changed")
	}
}

// On a library another instance holds, a rollout that would store is refused
// when asked, with database.ErrReadOnly, and nothing starts.
func TestStartRollout_RefusedOnReadOnlyDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shared.db")
	writer := database.NewDatabase()
	if err := writer.SetupDatabase(path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = writer.Close() })
	reader := database.NewDatabase()
	if err := reader.OpenDatabase(path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reader.Close() })
	a := NewApp(reader)

	if _, err := a.StartRollout(RolloutRequest{PositionID: 1, Settings: rollout.Fast(), Store: true}); !errors.Is(err, database.ErrReadOnly) {
		t.Errorf("StartRollout: err = %v, want ErrReadOnly", err)
	}
	if _, err := a.StartRolloutFiltered("", rollout.Fast()); !errors.Is(err, database.ErrReadOnly) {
		t.Errorf("StartRolloutFiltered: err = %v, want ErrReadOnly", err)
	}
	if _, err := a.StartRolloutIDs([]int64{1}, rollout.Fast(), a.activeGroup()); !errors.Is(err, database.ErrReadOnly) {
		t.Errorf("StartRolloutIDs: err = %v, want ErrReadOnly", err)
	}
	if st := a.RolloutStatus(); st.Running {
		t.Errorf("a rollout started: %+v", st)
	}

	// A board rolled out in memory (the EVAL scratch board) writes nothing: read-only is no bar.
	pos := domain.InitializePosition()
	if _, err := a.StartRollout(RolloutRequest{Position: &pos, Settings: rollout.Fast()}); err != nil {
		t.Errorf("unsaved board on a read-only database: %v", err)
	}
	if done := a.cancelRollout(); done != nil {
		<-done
	}
}
