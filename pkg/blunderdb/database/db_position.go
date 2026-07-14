package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func (d *Database) PositionExists(position Position) (map[string]interface{}, error) {
	d.mu.RLock()         // Lock the mutex
	defer d.mu.RUnlock() // Unlock the mutex when the function returns

	positionJSON, err := positionIdentityJSON(position)
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

		existingPositionJSON, err := positionIdentityJSON(existingPosition)
		if err != nil {
			return nil, err
		}

		if positionJSON == existingPositionJSON {
			return map[string]interface{}{"id": positionID, "exists": true}, nil
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return map[string]interface{}{"id": 0, "exists": false}, nil
}

// IndividualSaveResult reports what SaveIndividualPosition did: the id the
// position is stored under, and whether it was already there.
type IndividualSaveResult struct {
	ID      int64 `json:"id"`
	Existed bool  `json:"existed"`
}

// SaveIndividualPosition stores a position the user brought into the database on
// its own — written from the board, pasted as an XGID — rather than as part of a
// match. It records that provenance (ADR-0001) and reports whether the position
// was already stored, so the caller can merge its analysis and comment onto the
// existing row instead of overwriting someone else's.
//
// It replaces the frontend's former PositionExists + conditional SavePosition
// dance, which had two defects. PositionExists compares marshalled JSON in an
// O(n) scan of every stored position, while the store deduplicates on the
// Zobrist hash — two different notions of "the same position". And when the
// position already existed the frontend skipped SavePosition entirely, so an
// individual import of a position a match had already brought in never reached
// the store and the provenance flag was never raised. That is exactly the case
// the flag exists to serve.
func (d *Database) SaveIndividualPosition(position *Position) (IndividualSaveResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	ctx := context.Background()
	_, existed, err := d.store.Positions().Exists(ctx, "", engine.ZobristHash(position))
	if err != nil {
		return IndividualSaveResult{}, err
	}

	// Sticky: Save ORs this into the stored value, so a match import that later
	// brings in the same position cannot clear it.
	position.IndividuallyImported = true
	id, err := d.store.Positions().Save(ctx, "", position)
	if err != nil {
		return IndividualSaveResult{}, err
	}
	return IndividualSaveResult{ID: id, Existed: existed}, nil
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
// Use with reconstructPosition after scanning the values. individually_imported
// trails the identity columns: it is provenance, applied on top of the
// reconstructed position rather than part of it (ADR-0001).
const positionSelectCols = `id, state, decision_type, player_on_roll, dice_1, dice_2, cube_value, cube_owner, score_1, score_2, has_jacoby, has_beaver, individually_imported`

// positionSelectColsP is positionSelectCols with "p." table prefix for JOINs.
const positionSelectColsP = `p.id, p.state, p.decision_type, p.player_on_roll, p.dice_1, p.dice_2, p.cube_value, p.cube_owner, p.score_1, p.score_2, p.has_jacoby, p.has_beaver, p.individually_imported`

// scanPositionRow scans a sql.Row / sql.Rows into a Position using the column
// order from positionSelectCols. NULLs are treated as zero (safe for v2+).
func scanPositionRow(scanner interface {
	Scan(dest ...interface{}) error
}) (Position, error) {
	var id int64
	var state string
	var dt, por, d1, d2, cv, co, s1, s2, hj, hb sql.NullInt64
	var individual sql.NullBool
	err := scanner.Scan(&id, &state, &dt, &por, &d1, &d2, &cv, &co, &s1, &s2, &hj, &hb, &individual)
	if err != nil {
		return Position{}, err
	}
	pos := reconstructPosition(id, state,
		int(dt.Int64), int(por.Int64), int(d1.Int64), int(d2.Int64),
		int(cv.Int64), int(co.Int64), int(s1.Int64), int(s2.Int64),
		int(hj.Int64), int(hb.Int64))
	pos.IndividuallyImported = individual.Bool
	return pos, nil
}

// fullPositionJSON reconstructs a full Position from DB row values and marshals
// it to JSON. Used by export functions to produce backwards-compatible state blobs.
func fullPositionJSON(pos Position) string {
	data, _ := json.Marshal(pos)
	return string(data)
}

// positionIdentityJSON marshals only what makes two positions the same position.
// The legacy .db importer keys a lookup map on this string, so any Position field
// that is *not* part of the position's identity has to be zeroed here or the two
// sides of the comparison stop matching. Today that is the row id and the
// individually-imported provenance flag (ADR-0001).
func positionIdentityJSON(pos Position) (string, error) {
	pos.ID = 0
	pos.IndividuallyImported = false
	data, err := json.Marshal(pos)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
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
