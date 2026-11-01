package storage

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// DrawingStore keeps the drawing of a position: the arrows and squares a
// player laid over its board, in the order they were drawn.
type DrawingStore interface {
	// Drawing is the marks of the position, none when it has no drawing;
	// ErrNotFound when the position does not exist in the scope.
	Drawing(ctx context.Context, scope string, positionID int64) ([]domain.DrawingMark, error)
	// SetDrawing replaces the drawing of the position by marks; no marks
	// clears it. A mark that cannot be drawn (domain.DrawingMark.Validate) or
	// too many marks is ErrInvalid and leaves the previous drawing as it was.
	SetDrawing(ctx context.Context, scope string, positionID int64, marks []domain.DrawingMark) error
	// Drawings reads the drawings of several positions at once, keyed by
	// position id; a position without a drawing has no key.
	Drawings(ctx context.Context, scope string, positionIDs []int64) (map[int64][]domain.DrawingMark, error)
}

// DrawingProvider is implemented by the Stores of a backend that keeps
// drawings (the remote client does not).
type DrawingProvider interface {
	PositionDrawings() DrawingStore
}
