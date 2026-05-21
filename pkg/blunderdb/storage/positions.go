package storage

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// PositionStore persists backgammon positions. Positions are deduplicated by
// their Zobrist hash; Save is idempotent for an already-stored position and
// returns the existing id.
type PositionStore interface {
	// Save stores p (or returns the id of an identical existing position).
	Save(ctx context.Context, scope string, p *domain.Position) (int64, error)

	// Update overwrites the stored position with the same id as p.
	Update(ctx context.Context, scope string, p *domain.Position) error

	// Load returns the position with the given id, or ErrNotFound.
	Load(ctx context.Context, scope string, id int64) (*domain.Position, error)

	// Exists reports whether a position with the given Zobrist hash is stored,
	// returning its id when found.
	Exists(ctx context.Context, scope string, zobrist uint64) (id int64, found bool, err error)

	// Delete removes the position with the given id (analysis, comments and
	// collection links cascade).
	Delete(ctx context.Context, scope string, id int64) error

	// List streams stored positions.
	List(ctx context.Context, scope string, opts ListOpts) iter.Seq2[*domain.Position, error]
}
