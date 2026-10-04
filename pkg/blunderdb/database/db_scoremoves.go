package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ScoreMoves writes move.error_mp for every move whose position carries an
// analysis and that is still unscored, by batches of storage.ScoreBatchSize
// moves, each its own transaction, and returns how many it scored. Explicit
// only (`repair --move-errors`), never run at open: the column is a
// projection of the analyses, so a library without it stays correct. A pass
// interrupted loses at most the batch in flight and a new run skips what is
// written.
//
// d.mu is taken per batch, not for the whole run, so the GUI keeps reading a
// large library while it scores.
func (d *Database) ScoreMoves() (int, error) {
	d.mu.RLock()
	open, store := d.db != nil, d.store
	d.mu.RUnlock()
	if !open {
		return 0, fmt.Errorf("score moves: no database open")
	}
	return storage.ScoreAllMoves(context.Background(), store.Matches(), "", func() func() {
		d.mu.Lock()
		return d.mu.Unlock
	}, nil)
}
