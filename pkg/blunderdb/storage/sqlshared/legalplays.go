package sqlshared

import (
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// LegalPlaysColumns are the position columns a legal-play count depends on,
// in the order LegalPlays takes them, qualified with the alias p.
const LegalPlaysColumns = "p.state, p.player_on_roll, p.dice_1, p.dice_2"

// LegalPlays counts the legal checker plays of a stored position from the
// columns LegalPlaysColumns selects, engine.LegalPlaysUnknown when the row
// has none (an analysis whose position is gone, a cube decision without
// dice). Both backends derive is_forced through it (engine.IsForcedChecker).
func LegalPlays(state []byte, playerOnRoll, dice1, dice2 *int64) int {
	if len(state) == 0 || playerOnRoll == nil || dice1 == nil || dice2 == nil {
		return engine.LegalPlaysUnknown
	}
	p := engine.ReconstructPosition(0, string(state), 0, int(*playerOnRoll), int(*dice1), int(*dice2), 0, 0, 0, 0, 0, 0)
	return engine.CountLegalPlays(&p)
}
