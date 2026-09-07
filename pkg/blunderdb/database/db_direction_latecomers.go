package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// Latecomers, and where they enter (issue #392, fonctionnel.md §4.4).
//
// One turns up at every tournament, almost always while a free bye is still open in the first
// round. The engine NEVER redraws a draw already made — a draw is an event of the journal, and
// redrawing it would move players who have already read their name on the wall. So the
// latecomer takes an empty place, or they enter later; and blunderDB's job is to SAY WHICH,
// before the director validates the entry, rather than leaving someone entered nowhere.

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
// Separate from AddParticipant on purpose: taking a place is a decision of the director's, and
// the engine refuses one that is no longer free rather than picking another. Nothing is
// inferred, and nothing is redrawn.
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
