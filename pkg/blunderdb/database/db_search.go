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
// an unbounded ListOpts — GUI pagination is tracked separately (D.8).
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
