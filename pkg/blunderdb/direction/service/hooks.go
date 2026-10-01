package service

import (
	"context"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// BracketSkeleton is the bracket drawn before any match of the phase is known. Exported for the
// façade's tests, which pin its shape.
func BracketSkeleton(st *tournoi.State, ph *tournoi.PhaseState) ([]*tournoi.Section, bool) {
	return bracketSkeleton(st, ph)
}

// ConfirmAt records a proposal at a given instant. Exported for the façade's tests, which replay
// a tournament over several days.
func ConfirmAt(ctx context.Context, dir *direction.Direction, a tournoi.Action, now time.Time) error {
	return confirmAt(ctx, dir, a, now)
}
