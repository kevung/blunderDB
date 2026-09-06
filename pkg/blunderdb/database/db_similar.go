package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// « Des positions comme celle-ci » (#293, fiche J.3), côté base.

// SimilarPositionsLimit is how many neighbours the interface asks for by
// default. Ten: enough to see whether the metric agrees with the eye, few
// enough to read without scrolling.
const SimilarPositionsLimit = 10

// SimilarPositions returns the positions closest to the one with this id,
// nearest first and excluding it. The distance is in checker-pips — how much
// checker movement separates the two — so the number shown beside a neighbour
// can be read rather than merely compared.
func (d *Database) SimilarPositions(positionID int, limit int) ([]storage.SimilarPosition, error) {
	if limit <= 0 {
		limit = SimilarPositionsLimit
	}
	pos, err := d.LoadPosition(positionID)
	if err != nil {
		return nil, fmt.Errorf("similar: position %d: %w", positionID, err)
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.store.Positions().Similar(context.Background(), "", pos, limit)
}
