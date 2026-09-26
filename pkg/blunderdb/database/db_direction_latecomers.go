package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// Latecomers, and where they enter. The engine NEVER redraws a draw (it is a journal event), so
// a latecomer takes an empty bye or enters later; blunderDB says which before the entry is
// validated.

// FreeSlot is an empty bye a latecomer can still take.
type FreeSlot struct {
	Phase   int           `json:"phase"`
	Section string        `json:"section"`
	Key     string        `json:"key"`
	Label   tournoi.Label `json:"label,omitempty"`
}

// DirectionFreeSlots lists the byes still open, in bracket order. The director picks one; the
// engine refuses a place already played, unknown, or taken.
func (d *Database) DirectionFreeSlots(tournamentID int64) ([]FreeSlot, error) {
	dir, err := direction.Open(context.Background(), d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	st := dir.State()
	if st == nil {
		return nil, nil
	}
	slots := st.FreeSlots()
	out := make([]FreeSlot, 0, len(slots))
	for _, s := range slots {
		out = append(out, FreeSlot{Phase: s.Phase, Section: s.Section, Key: s.Key, Label: s.Label})
	}
	return out, nil
}

// AddParticipantAtSlot enters a latecomer on a named free bye.
//
// Separate from AddParticipant on purpose: the place is the director's decision, and one no
// longer free is refused rather than replaced.
func (d *Database) AddParticipantAtSlot(tournamentID int64, name, club string, rating float64, section, key string) (*DirectionView, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("direction: an entry needs a name")
	}
	ctx := context.Background()
	dir, err := direction.Open(ctx, d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	st := dir.State()
	if st == nil {
		return nil, direction.ErrNoDirection
	}
	taken := map[string]bool{}
	for id := range st.Players {
		taken[string(id)] = true
	}
	p := tournoi.Player{ID: tournoi.PlayerID(participantID(name, taken)), Name: name, Club: club, Rating: rating}
	slot := tournoi.Slot{Phase: st.Current, Section: section, Key: key}
	if err := dir.Apply(ctx, tournoi.PlayerAddedAtSlotEvent(p, slot, time.Now())); err != nil {
		return nil, err
	}
	return d.GetDirection(tournamentID)
}
