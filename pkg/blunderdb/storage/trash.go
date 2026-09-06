package storage

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TrashStore persists what was deleted, so it can be put back (issue #285,
// ADR-0036).
//
// A trash entry is a SNAPSHOT written just before the real delete, not a flag
// on the live row. Nothing else in the schema knows this table exists: no
// search filter, no statistic, no retention predicate, no uniqueness index.
// That is the whole point of the choice — a `deleted_at` column would have to
// be honoured in every one of those places, and the one that forgot would
// silently count a deleted position in a PR or hand it to a review session.
//
// Restoring is the caller's business, not this store's: putting a position
// back means re-Saving it through PositionStore (so the Zobrist deduplication
// decides where it lands, and it never creates a duplicate), putting a
// collection back means recreating it and its membership. This store only
// remembers.
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
