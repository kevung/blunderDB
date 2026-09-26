package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// RepairGamePhases reclassifies every position whose stored phase disagrees
// with engine.ClassifyGamePhase, and returns how many rows changed
// (ADR-0035). The phase is a DERIVED label, so after a classifier change this
// realigns every row; on an up-to-date database it costs one scan.
func (d *Database) RepairGamePhases() (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return 0, fmt.Errorf("no database is currently open")
	}
	return d.store.Positions().ReclassifyDerived(context.Background(), "")
}

// recomputeGamePhases is the migration's entry point: same work, without
// taking d.mu, which the caller already holds.
//
// It binds its own Storage: d.store is still nil during the chain.
// sqlite.New borrows the handle and closes nothing.
func (d *Database) recomputeGamePhases(ctx context.Context) (int, error) {
	store := d.store
	if store == nil {
		store = sqlite.New(d.db)
	}
	return store.Positions().ReclassifyDerived(ctx, "")
}
