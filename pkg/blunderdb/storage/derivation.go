package storage

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// DerivationStore keeps, for a position, the position it derives from and the
// move that led to it. A position has at most one origin; an origin has any
// number of variants.
type DerivationStore interface {
	// DerivedFrom is the derivation of the position, nil when it derives from
	// nothing; ErrNotFound when the position does not exist in the scope.
	DerivedFrom(ctx context.Context, scope string, positionID int64) (*domain.Derivation, error)
	// SetDerivedFrom records that the position derives from d.FromID, replacing
	// any previous origin; nil clears it. ErrNotFound when the position does
	// not exist; ErrInvalid when d is not valid (domain.Derivation.Validate),
	// when its origin is not a position of the scope, or when it would close a
	// loop (an origin that itself derives, directly or not, from the position).
	// The previous derivation then stays as it was.
	SetDerivedFrom(ctx context.Context, scope string, positionID int64, d *domain.Derivation) error
	// Derivations reads several at once, keyed by position id; a position that
	// derives from nothing has no key.
	Derivations(ctx context.Context, scope string, positionIDs []int64) (map[int64]domain.Derivation, error)
	// Variants lists the ids of the positions deriving directly from the
	// origin, in ascending order; ErrNotFound when the origin does not exist.
	Variants(ctx context.Context, scope string, originID int64) ([]int64, error)
}

// DerivationProvider is implemented by the Stores of a backend that keeps
// derivations (the remote client does not).
type DerivationProvider interface {
	PositionDerivations() DerivationStore
}
