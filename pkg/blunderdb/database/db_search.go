package database

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// LoadPositionsByFiltersCore searches positions and loads every result's
// analysis into a map keyed by position id, so callers can apply
// analysis-based filters without per-row LoadAnalysis round-trips.
//
// SearchStore.Find takes opts.Limit/Offset into its SQL (zero = unbounded);
// AnalysisStore.LoadMany loads all analyses in one batched query. Positions
// without an analysis are absent from the map. Uses context.Background();
// prefer LoadPositionsByFiltersCoreCtx when the caller can cancel.
func (d *Database) LoadPositionsByFiltersCore(
	f SearchFilters, opts storage.ListOpts,
) ([]Position, map[int64]*PositionAnalysis, error) {
	return d.LoadPositionsByFiltersCoreCtx(context.Background(), f, opts)
}

// LoadPositionsByFiltersCoreCtx is LoadPositionsByFiltersCore with a caller-supplied
// context, threaded into both the search scan and the analysis load.
func (d *Database) LoadPositionsByFiltersCoreCtx(
	ctx context.Context, f SearchFilters, opts storage.ListOpts,
) ([]Position, map[int64]*PositionAnalysis, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var positions []Position
	var ids []int64
	for pos, err := range d.store.Search().Find(ctx, "", f, opts) {
		if err != nil {
			return nil, nil, err
		}
		positions = append(positions, *pos)
		ids = append(ids, pos.ID)
	}
	analysisMap, err := d.store.Analyses().LoadMany(ctx, "", ids)
	if err != nil {
		return nil, nil, err
	}
	return positions, analysisMap, nil
}

// LoadPositionsByFilters returns positions matching the supplied filters.
// Unbounded; for callers wanting whole positions in one round trip (tests,
// scripting). The GUI uses LoadPositionIDsByFilters.
func (d *Database) LoadPositionsByFilters(f SearchFilters) ([]Position, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var positions []Position
	for pos, err := range d.store.Search().Find(context.Background(), "", f, storage.ListOpts{}) {
		if err != nil {
			return nil, err
		}
		positions = append(positions, *pos)
	}
	return positions, nil
}

// LoadPositionIDsByFilters returns the ids of positions matching f, in the
// same order LoadPositionsByFilters would return the positions themselves.
// Only ids cross the Wails bridge; the frontend fetches the visible window
// through LoadPositionsByIDs, as it does behind ListPositionIDs.
//
// Two return values on purpose: Wails v2's dispatcher only handles 1 or 2, so
// a 3-return method (like LoadPositionsByFiltersCore) resolves to (nil, nil)
// in JS and must never be called from the frontend.
func (d *Database) LoadPositionIDsByFilters(f SearchFilters) ([]int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var ids []int64
	for pos, err := range d.store.Search().Find(context.Background(), "", f, storage.ListOpts{}) {
		if err != nil {
			return nil, err
		}
		ids = append(ids, pos.ID)
	}
	return ids, nil
}
