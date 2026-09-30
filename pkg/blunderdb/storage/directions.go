package storage

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// DirectionStore persists the Direction of a Tournament: its record and its
// append-only event log (ADR-0047). It is direction.Store with a scope, so a
// backend serves every tenant; BindDirection fixes the scope and hands the
// direction package the Store it runs on.
//
// Every method is confined to the scope's tenant: a Direction of another
// tenant reads as absent and cannot be written.
type DirectionStore interface {
	// Get returns the record, or an error matching both direction.ErrNoDirection
	// and ErrNotFound when the Tournament was never directed.
	Get(ctx context.Context, scope string, tournamentID int64) (direction.Record, error)
	// List returns every record of the scope, by Tournament id.
	List(ctx context.Context, scope string) ([]direction.Record, error)
	// Create stores the record of a Tournament of the scope: ErrNotFound when
	// the Tournament does not exist there, ErrConflict when it is already
	// directed.
	Create(ctx context.Context, scope string, rec direction.Record) error
	// Update rewrites the record's own facts; direction.ErrNoDirection when
	// there is none.
	Update(ctx context.Context, scope string, rec direction.Record) error
	// Delete removes the record and its log and frees the Slots the
	// Tournament's Matches filled. It never deletes a Match.
	Delete(ctx context.Context, scope string, tournamentID int64) error
	// AppendEvent adds one event to the log. It REFUSES a sequence number
	// already written (ErrConflict): the log is append-only, and a silent
	// overwrite would lose a decision the director made. It also refuses a
	// log without its record (direction.ErrNoDirection): every writer creates
	// the record first — direction.Create, and the export, which copies the
	// record before the events — and no import writes a log, so an orphan
	// event could only be a bug. Bound to a Tx it is atomic with the rest of
	// that transaction.
	AppendEvent(ctx context.Context, scope string, tournamentID int64, ev direction.StoredEvent) error
	// LoadEvents returns the log in sequence order.
	LoadEvents(ctx context.Context, scope string, tournamentID int64) ([]direction.StoredEvent, error)

	// FilledSlots returns the library Matches of the Tournament that fill a
	// Slot, with the final score their own games give.
	FilledSlots(ctx context.Context, scope string, tournamentID int64) ([]FilledSlot, error)
	// UnattachedMatches returns the Matches of the Tournament that fill no
	// Slot, by id.
	UnattachedMatches(ctx context.Context, scope string, tournamentID int64) ([]SlotCandidate, error)
	// AttachSlot puts a Match in the Tournament and in the Slot, leaving the
	// Slot it held: ErrNotFound when the Match does not exist in the scope.
	AttachSlot(ctx context.Context, scope string, tournamentID int64, slotID string, matchID int64) error
	// DetachSlot empties a Slot. The Match keeps its Tournament.
	DetachSlot(ctx context.Context, scope string, tournamentID int64, slotID string) error
	// SlotOf returns the Tournament and the Slot a Match fills; slotID is ""
	// when it fills none, which is not an error.
	SlotOf(ctx context.Context, scope string, matchID int64) (tournamentID int64, slotID string, err error)
	// Pairs returns the persons behind every doubles Participant of the
	// Tournament, by Participant id, in seat order.
	Pairs(ctx context.Context, scope string, tournamentID int64) (map[string][]PairMember, error)
	// SetPair replaces the persons behind one Participant.
	SetPair(ctx context.Context, scope string, tournamentID int64, participantID string, members []PairMember) error
}

// FilledSlot is a library Match sitting in a Slot, with what its own file
// says happened. HasScore is false when the Match has no game to sum.
type FilledSlot struct {
	SlotID   string
	MatchID  int64
	Player1  string
	Player2  string
	Length   int
	Score1   int
	Score2   int
	HasScore bool
}

// SlotCandidate is a Match of a directed Tournament that fills no Slot.
type SlotCandidate struct {
	MatchID int64
	Player1 string
	Player2 string
	Length  int
	Date    string
}

// PairMember is one of the two persons behind a doubles Participant
// (ADR-0056 §4).
type PairMember struct {
	Name   string  `json:"name"`
	Club   string  `json:"club,omitempty"`
	Rating float64 `json:"rating,omitempty"`
}

// BindDirection adapts a scoped DirectionStore to the direction.Store the
// direction package runs on.
func BindDirection(ds DirectionStore, scope string) direction.Store {
	return boundDirection{ds: ds, scope: scope}
}

type boundDirection struct {
	ds    DirectionStore
	scope string
}

func (b boundDirection) GetDirection(ctx context.Context, tournamentID int64) (direction.Record, error) {
	return b.ds.Get(ctx, b.scope, tournamentID)
}

func (b boundDirection) ListDirections(ctx context.Context) ([]direction.Record, error) {
	return b.ds.List(ctx, b.scope)
}

func (b boundDirection) CreateDirection(ctx context.Context, rec direction.Record) error {
	return b.ds.Create(ctx, b.scope, rec)
}

func (b boundDirection) UpdateDirection(ctx context.Context, rec direction.Record) error {
	return b.ds.Update(ctx, b.scope, rec)
}

func (b boundDirection) DeleteDirection(ctx context.Context, tournamentID int64) error {
	return b.ds.Delete(ctx, b.scope, tournamentID)
}

func (b boundDirection) AppendEvent(ctx context.Context, tournamentID int64, ev direction.StoredEvent) error {
	return b.ds.AppendEvent(ctx, b.scope, tournamentID, ev)
}

func (b boundDirection) LoadEvents(ctx context.Context, tournamentID int64) ([]direction.StoredEvent, error) {
	return b.ds.LoadEvents(ctx, b.scope, tournamentID)
}
