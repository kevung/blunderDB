package gammonnet

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// TestCountLegalPlaysRegistered: linking gammonnet is what lets the stores
// tell a forced play from a decision, through engine.CountLegalPlays.
func TestCountLegalPlaysRegistered(t *testing.T) {
	t.Parallel()
	p := domain.InitializePosition()
	p.Dice = [2]int{6, 6}
	if got, want := engine.CountLegalPlays(&p), CountLegalPlays(&p); got != want || got < 2 {
		t.Errorf("engine.CountLegalPlays = %d, gammonnet = %d, want the same, several", got, want)
	}
	p.Dice = [2]int{0, 0}
	if got := engine.CountLegalPlays(&p); got != engine.LegalPlaysUnknown {
		t.Errorf("no dice: %d, want unknown", got)
	}
}
