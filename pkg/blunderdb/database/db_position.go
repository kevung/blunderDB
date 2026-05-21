package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func (d *Database) PositionExists(position Position) (map[string]interface{}, error) {
	d.mu.RLock()         // Lock the mutex
	defer d.mu.RUnlock() // Unlock the mutex when the function returns

	// Create a copy of the position without the ID field inside the state
	positionCopy := position
	positionCopy.ID = 0

	positionJSON, err := json.Marshal(positionCopy)
	if err != nil {
		return nil, err
	}

	rows, err := d.db.Query(`SELECT ` + positionSelectCols + ` FROM position`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		existingPosition, err := scanPositionRow(rows)
		if err != nil {
			return nil, err
		}
		positionID := existingPosition.ID

		// Compare the positions excluding the ID field inside the state
		existingPosition.ID = 0
		existingPositionJSON, err := json.Marshal(existingPosition)
		if err != nil {
			return nil, err
		}

		if string(positionJSON) == string(existingPositionJSON) {
			return map[string]interface{}{"id": positionID, "exists": true}, nil
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return map[string]interface{}{"id": 0, "exists": false}, nil
}

// The position scalar-column codec lives in package engine so the SQLite
// Storage backend can share it (package database imports storage/sqlite, so
// the dependency cannot run the other way). These aliases keep the persistence
// code in this package compiling against the unqualified names.
var (
	populatePositionColumns = engine.PopulatePositionColumns
	encodeBoardCompact      = engine.EncodeBoardCompact
	decodeBoardCompact      = engine.DecodeBoardCompact
	isCompactState          = engine.IsCompactState
	reconstructPosition     = engine.ReconstructPosition
)

// positionSelectCols is the standard column list for reading a position row.
// Use with reconstructPosition after scanning the values.
const positionSelectCols = `id, state, decision_type, player_on_roll, dice_1, dice_2, cube_value, cube_owner, score_1, score_2, has_jacoby, has_beaver`

// positionSelectColsP is positionSelectCols with "p." table prefix for JOINs.
const positionSelectColsP = `p.id, p.state, p.decision_type, p.player_on_roll, p.dice_1, p.dice_2, p.cube_value, p.cube_owner, p.score_1, p.score_2, p.has_jacoby, p.has_beaver`

// scanPositionRow scans a sql.Row / sql.Rows into a Position using the column
// order from positionSelectCols. NULLs are treated as zero (safe for v2+).
func scanPositionRow(scanner interface {
	Scan(dest ...interface{}) error
}) (Position, error) {
	var id int64
	var state string
	var dt, por, d1, d2, cv, co, s1, s2, hj, hb sql.NullInt64
	err := scanner.Scan(&id, &state, &dt, &por, &d1, &d2, &cv, &co, &s1, &s2, &hj, &hb)
	if err != nil {
		return Position{}, err
	}
	return reconstructPosition(id, state,
		int(dt.Int64), int(por.Int64), int(d1.Int64), int(d2.Int64),
		int(cv.Int64), int(co.Int64), int(s1.Int64), int(s2.Int64),
		int(hj.Int64), int(hb.Int64)), nil
}

// fullPositionJSON reconstructs a full Position from DB row values and marshals
// it to JSON. Used by export functions to produce backwards-compatible state blobs.
func fullPositionJSON(pos Position) string {
	data, _ := json.Marshal(pos)
	return string(data)
}

func (d *Database) SavePosition(position *Position) (int64, error) {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	// Save deduplicates by Zobrist hash and updates *position with the
	// normalized board and the resulting id (existing row on hash conflict).
	return d.store.Positions().Save(context.Background(), "", position)
}

func (d *Database) UpdatePosition(position Position) error {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	return d.store.Positions().Update(context.Background(), "", &position)
}

func (d *Database) LoadPosition(id int) (*Position, error) {
	d.mu.RLock()         // Lock the mutex
	defer d.mu.RUnlock() // Unlock the mutex when the function returns

	pos, err := d.store.Positions().Load(context.Background(), "", int64(id))
	if errors.Is(err, storage.ErrNotFound) {
		// Preserve the pre-delegation contract: callers expect sql.ErrNoRows.
		return nil, sql.ErrNoRows
	}
	return pos, err
}

func (d *Database) LoadAllPositions() ([]Position, error) {
	d.mu.RLock()         // Lock the mutex
	defer d.mu.RUnlock() // Unlock the mutex when the function returns

	var positions []Position
	for pos, err := range d.store.Positions().List(context.Background(), "", storage.ListOpts{}) {
		if err != nil {
			return nil, err
		}
		positions = append(positions, *pos)
	}

	return positions, nil
}

func (d *Database) DeletePosition(positionID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// ON DELETE CASCADE handles analysis, comment, and collection_position.
	return d.store.Positions().Delete(context.Background(), "", positionID)
}
