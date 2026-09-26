package storage

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TrashStore persists what was deleted, so it can be put back (ADR-0036).
//
// A trash entry is a SNAPSHOT written just before the real delete, not a
// `deleted_at` flag every search filter, statistic and retention predicate
// would have to honour.
//
// Restoring is the caller's business: a position is re-Saved through
// PositionStore (the Zobrist dedup decides where it lands), a collection is
// recreated with its membership. This store only remembers.
type TrashStore interface {
	// Put writes a snapshot and returns its id. payload is the JSON the
	// caller will read back to restore; label is what the trash list shows.
	Put(ctx context.Context, scope string, kind domain.TrashKind, label string, payload []byte) (int64, error)

	// List returns the entries, most recently deleted first, bounded by opts.
	// kind narrows to one kind; empty lists them all.
	List(ctx context.Context, scope string, kind domain.TrashKind, opts ListOpts) ([]*domain.TrashEntry, error)

	// Load returns one entry, or ErrNotFound.
	Load(ctx context.Context, scope string, id int64) (*domain.TrashEntry, error)

	// Discard removes one entry — after a successful restore, or when the user
	// empties one line of the trash. It does NOT restore anything.
	Discard(ctx context.Context, scope string, id int64) error

	// Purge removes every entry deleted more than olderThanDays ago and
	// returns how many it dropped. `blunderdb vacuum` runs it with
	// domain.TrashRetentionDays; 0 empties the trash entirely.
	Purge(ctx context.Context, scope string, olderThanDays int) (int, error)

	// Count is how many entries the trash holds, for a panel that wants to say
	// so before opening.
	Count(ctx context.Context, scope string) (int, error)
}
