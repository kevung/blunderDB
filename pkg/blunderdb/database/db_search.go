package database

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

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

// parseFilterIDList parses a match/tournament ID filter string.
// Supports: "5" (single), "2,7" (range from 2 to 7), or multiple IDs passed
// as a pre-joined comma-separated list from the frontend.
func parseFilterIDList(s string) ([]int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	if len(parts) == 2 {
		// Could be a range (e.g., "2,7" means IDs 2 through 7)
		start, err1 := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		end, err2 := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		if err1 == nil && err2 == nil && end > start {
			var ids []int64
			for i := start; i <= end; i++ {
				ids = append(ids, i)
			}
			return ids, nil
		}
	}
	// Otherwise treat as explicit list of IDs separated by ";"
	parts = strings.Split(s, ";")
	var ids []int64
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid ID %q: %v", p, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// getPositionIDsForMatch returns all position IDs linked to a given match.
func (d *Database) getPositionIDsForMatch(matchID int64) ([]int64, error) {
	d.mu.RLock()
	rows, err := d.db.Query(`
		SELECT DISTINCT mv.position_id
		FROM move mv
		INNER JOIN game g ON mv.game_id = g.id
		WHERE g.match_id = ?
	`, matchID)
	d.mu.RUnlock()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// getMatchIDsForTournament returns all match IDs belonging to a tournament.
func (d *Database) getMatchIDsForTournament(tournamentID int64) ([]int64, error) {
	d.mu.RLock()
	rows, err := d.db.Query(`SELECT id FROM match WHERE tournament_id = ?`, tournamentID)
	d.mu.RUnlock()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
