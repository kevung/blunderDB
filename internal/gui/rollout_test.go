package gui

import (
	"testing"

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
