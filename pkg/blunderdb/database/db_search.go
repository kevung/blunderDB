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
	ctx, done := d.beginSearch()
	defer done()
	return d.LoadPositionsByFiltersCtx(ctx, f)
}

// LoadPositionsByFiltersCtx is LoadPositionsByFilters under the caller's context.
func (d *Database) LoadPositionsByFiltersCtx(ctx context.Context, f SearchFilters) ([]Position, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var positions []Position
	for pos, err := range d.store.Search().Find(ctx, "", f, storage.ListOpts{}) {
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
	ctx, done := d.beginSearch()
	defer done()
	return d.SearchPositionIDsCtx(ctx, f, 0, 0)
}

// SearchPositionIDs returns the window [offset, offset+limit) of
// LoadPositionIDsByFilters's ids; limit <= 0 means up to the end. The GUI
// browses a search result through such windows, CountPositionsByFilters and
// IndexOfPositionByFilters, never holding the whole list.
func (d *Database) SearchPositionIDs(f SearchFilters, offset, limit int) ([]int64, error) {
	ctx, done := d.beginSearch()
	defer done()
	return d.SearchPositionIDsCtx(ctx, f, offset, limit)
}

// SearchPositionIDsCtx is SearchPositionIDs under the caller's context: the
// scan stops, chunk by chunk, once it is cancelled.
func (d *Database) SearchPositionIDsCtx(ctx context.Context, f SearchFilters, offset, limit int) ([]int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.store.Search().FindIDs(ctx, "", f, storage.ListOpts{Offset: offset, Limit: limit})
}

// CountPositionsByFilters returns how many positions the search finds.
func (d *Database) CountPositionsByFilters(f SearchFilters) (int, error) {
	ctx, done := d.beginSearch()
	defer done()
	return d.CountPositionsByFiltersCtx(ctx, f)
}

// CountPositionsByFiltersCtx is CountPositionsByFilters under the caller's context.
func (d *Database) CountPositionsByFiltersCtx(ctx context.Context, f SearchFilters) (int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.store.Search().Count(ctx, "", f)
}

// IndexOfPositionByFilters returns the rank of id in SearchPositionIDs's
// order, or -1 when the search does not find it.
func (d *Database) IndexOfPositionByFilters(f SearchFilters, id int64) (int, error) {
	ctx, done := d.beginSearch()
	defer done()
	return d.IndexOfPositionByFiltersCtx(ctx, f, id)
}

// IndexOfPositionByFiltersCtx is IndexOfPositionByFilters under the caller's context.
func (d *Database) IndexOfPositionByFiltersCtx(ctx context.Context, f SearchFilters, id int64) (int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	index, found, err := d.store.Search().IndexOf(ctx, "", f, id)
	if err != nil || !found {
		return -1, err
	}
	return index, nil
}
