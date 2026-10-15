package sqlshared

import (
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// LegalPlaysColumns are the position columns a legal-play count depends on,
// in the order PlayPosition takes them, qualified with the alias p.
const LegalPlaysColumns = "p.state, p.player_on_roll, p.dice_1, p.dice_2"

// PlayPosition rebuilds, from the columns LegalPlaysColumns selects, what of a
// stored position its checker plays depend on (board, player on roll, dice);
// nil when the row has none (an analysis whose position is gone, a cube
// decision without dice). Both backends derive is_forced and the played
// move's candidate through it (engine.PopulateAnalysisColumns).
func PlayPosition(state []byte, playerOnRoll, dice1, dice2 *int64) *domain.Position {
	if len(state) == 0 || playerOnRoll == nil || dice1 == nil || dice2 == nil {
		return nil
	}
	p := engine.ReconstructPosition(0, string(state), 0, int(*playerOnRoll), int(*dice1), int(*dice2), 0, 0, 0, 0, 0, 0)
	return &p
}
