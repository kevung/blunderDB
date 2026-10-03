package gui

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/rollout"
)

// StartRollout refuses before starting anything: a request it cannot honour
// must not surface later as an error event.
func TestStartRollout_RefusesUpFront(t *testing.T) {
	a := &App{}
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
		if err := a.StartRollout(req); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if presets := a.RolloutPresets(); len(presets) != 2 || presets[0].Settings != rollout.Fast() || presets[1].Settings != rollout.Standard() {
		t.Errorf("presets: %+v", presets)
	}

	// An unsaved board rolls out and stops cleanly when cancelled.
	if err := a.StartRollout(RolloutRequest{Position: &pos, Settings: rollout.Fast()}); err != nil {
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
	if err := (&App{}).StartRolloutFiltered("", bad); err == nil {
		t.Error("invalid settings accepted")
	}
	if err := (&App{}).StartRolloutFiltered("", rollout.Fast()); err == nil {
		t.Error("no database accepted")
	}

	d := database.NewDatabase()
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	a := NewApp(d)
	if err := a.StartRolloutFiltered("nosuchtoken:1", rollout.Fast()); err == nil {
		t.Error("unknown query token accepted")
	}

	// An empty library has nothing to roll out: the job ends by itself.
	if err := a.StartRolloutFiltered("", rollout.Fast()); err != nil {
		t.Fatal(err)
	}
	if done := a.cancelRollout(); done != nil {
		<-done
	}
}

// The rollout batch and the gammonNet batch exclude each other, as on the
// daemon: each takes every core.
func TestRolloutBatchAndGammonNetBatchExcludeEachOther(t *testing.T) {
	d := database.NewDatabase()
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	a := NewApp(d)

	gnStopped := make(chan struct{})
	a.gnBatchMu.Lock()
	a.gnBatchCancel, a.gnBatchDone = func() {}, gnStopped
	a.gnBatchMu.Unlock()
	if err := a.StartRolloutFiltered("", rollout.Fast()); err == nil {
		t.Error("a rollout batch started beside a gammonNet batch")
	}
	a.gnBatchMu.Lock()
	a.gnBatchCancel, a.gnBatchDone = nil, nil
	a.gnBatchMu.Unlock()

	_, roStopped := a.beginRollout(RolloutStatus{Running: true, Kind: "batch"})
	a.StartGammonNetBatch(0, 0, 0)
	a.gnBatchMu.Lock()
	started := a.gnBatchDone != nil
	a.gnBatchMu.Unlock()
	if started {
		t.Error("a gammonNet batch started beside a rollout batch")
	}
	if st := a.RolloutStatus(); !st.Running || st.Kind != "batch" {
		t.Errorf("RolloutStatus during a batch: %+v", st)
	}
	a.endRollout(roStopped)
	if st := a.RolloutStatus(); st.Running {
		t.Errorf("RolloutStatus after the batch: %+v", st)
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
	if err := a.StartRollout(RolloutRequest{Position: &pos, Settings: s}); err != nil {
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
