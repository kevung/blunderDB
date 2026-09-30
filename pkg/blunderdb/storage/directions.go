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
	// overwrite would lose a decision the director made. Bound to a Tx it is
	// atomic with the rest of that transaction.
	AppendEvent(ctx context.Context, scope string, tournamentID int64, ev direction.StoredEvent) error
	// LoadEvents returns the log in sequence order.
	LoadEvents(ctx context.Context, scope string, tournamentID int64) ([]direction.StoredEvent, error)
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
