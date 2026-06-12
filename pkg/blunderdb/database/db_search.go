package database

import (
	"context"
	"errors"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// LoadPositionsByFiltersCore searches positions and, for every result, loads
// its analysis into a map keyed by position id, so callers can apply
// analysis-based filters without per-row LoadAnalysis round-trips.
//
// Both halves delegate to the Storage backend (D10): SearchStore.Find ports
// the SQL-first search algorithm, and AnalysisStore.Load returns the
// decompressed analysis. Positions without an analysis are simply absent from
// the map.
func (d *Database) LoadPositionsByFiltersCore(
	f SearchFilters,
) ([]Position, map[int64]*PositionAnalysis, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ctx := context.Background()
	var positions []Position
	analysisMap := make(map[int64]*PositionAnalysis)
	for pos, err := range d.store.Search().Find(ctx, "", f) {
		if err != nil {
			return nil, nil, err
		}
		positions = append(positions, *pos)
		ana, err := d.store.Analyses().Load(ctx, "", pos.ID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				continue
			}
			return nil, nil, err
		}
		analysisMap[pos.ID] = ana
	}
	return positions, analysisMap, nil
}

// LoadPositionsByFilters returns positions matching the supplied filters.
// This is the public Wails-bound method that accepts a single SearchFilters
// struct. It delegates to the SQLite Storage backend's SearchStore.Find.
func (d *Database) LoadPositionsByFilters(f SearchFilters) ([]Position, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var positions []Position
	for pos, err := range d.store.Search().Find(context.Background(), "", f) {
		if err != nil {
			return nil, err
		}
		positions = append(positions, *pos)
	}
	return positions, nil
}
