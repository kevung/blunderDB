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
