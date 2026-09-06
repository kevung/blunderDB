package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// RepairGamePhases reclassifies every position whose stored phase disagrees
// with engine.ClassifyGamePhase, and returns how many rows changed (issue
// #264, ADR-0035).
//
// The phase is a DERIVED label: change the classifier or its threshold, run
// this, and every row agrees with the new rule. `blunderdb repair` runs it
// alongside the analysis-column repair; the 2.19.0 migration runs it once so
// an upgraded database does not open with every position unclassified; and
// the daemon serves it as /v1/positions.reclassifyPhases. Running it on a
// database that is already up to date rewrites nothing and costs one scan.
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
// It binds its own Storage rather than using d.store: rebuildStore runs AFTER
// the migration chain, so d.store is still nil while the chain is walking.
// sqlite.New borrows the handle and owns nothing, so a second one costs
// nothing and closes nothing.
func (d *Database) recomputeGamePhases(ctx context.Context) (int, error) {
	store := d.store
	if store == nil {
		store = sqlite.New(d.db)
	}
	return store.Positions().ReclassifyDerived(ctx, "")
}
