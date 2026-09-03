package database

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// LoadPositionsByFiltersCore searches positions and loads every result's
// analysis into a map keyed by position id, so callers can apply
// analysis-based filters without per-row LoadAnalysis round-trips.
//
// Both halves delegate to the Storage backend (D10): SearchStore.Find ports
// the SQL-first search algorithm — opts.Limit/Offset are pushed into its SQL
// scan, a zero ListOpts keeping the previous unbounded behaviour — and
// AnalysisStore.LoadMany decodes every result's analysis in one batched query
// instead of one round trip per position (B.10, #178: on a search returning
// 2 000 positions this used to double the round trips, one per result on top
// of the search query itself). Positions without an analysis are simply
// absent from the map.
func (d *Database) LoadPositionsByFiltersCore(
	f SearchFilters, opts storage.ListOpts,
) ([]Position, map[int64]*PositionAnalysis, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ctx := context.Background()
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
// This is the public Wails-bound method that accepts a single SearchFilters
// struct. It delegates to the SQLite Storage backend's SearchStore.Find with
// an unbounded ListOpts.
//
// Kept for callers that still need whole positions in one round trip (tests,
// scripting); the GUI itself now goes through LoadPositionIDsByFilters (D.8).
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
// This is the GUI-facing counterpart of ListPositionIDs for a search result
// (D.8, #208): a search returning thousands of rows used to ship every one
// of them whole across the Wails bridge before the board ever showed more
// than one of them at a time. The frontend now keeps this id list in
// positionsStore (positionList.js) and fetches the window it is about to
// show through LoadPositionsByIDs — the same lazy path the library already
// uses behind ListPositionIDs.
//
// A single-return-plus-error method, deliberately not LoadPositionsByFiltersCore
// (which also returns a preloaded analysis map): Wails v2's binding dispatcher
// (internal/binding.BoundMethod.Call) only switches on OutputCount() 1 or 2 —
// a bound method with 3 return values silently resolves to (nil, nil) on the
// JS side, so a 3-return method must never be called from the frontend.
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
