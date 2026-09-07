package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// « Des positions comme celle-ci » (#293, fiche J.3 ; ADR-0043), côté base.

// SimilarPositionsLimit is how many neighbours the interface asks for by
// default. Ten: enough to see whether the metric agrees with the eye, few
// enough to read without scrolling.
const SimilarPositionsLimit = 10

// SimilarPositions returns the neighbours of the position with this id,
// nearest first and excluding it. The distance is in checker-pips — how much
// checker movement separates the two — so the number shown beside a neighbour
// can be read rather than merely compared.
//
// The ranking is taken inside the position's own class (ADR-0043): same kind
// of decision, same regime for a cube decision, and another match. Ranking the
// whole library instead answered a question nobody asked — the plies around
// the position being looked at.
func (d *Database) SimilarPositions(positionID int, limit int) ([]storage.SimilarPosition, error) {
	pos, err := d.LoadPosition(positionID)
	if err != nil {
		return nil, fmt.Errorf("similar: position %d: %w", positionID, err)
	}
	opts := storage.ClassOf(pos)
	opts.Limit = limit
	if opts.Limit <= 0 {
		opts.Limit = SimilarPositionsLimit
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.store.Positions().Similar(context.Background(), "", pos, opts)
}
