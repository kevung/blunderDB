package main

import (
	"database/sql"
	"encoding/json"
)

func (d *Database) PositionExists(position Position) (map[string]interface{}, error) {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

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

// positionColumns holds the derived scalar columns that will be stored
// alongside the position JSON in the v2 schema. populatePositionColumns
// computes them from a (already-normalized) Position without hitting the DB.
// This helper is intentionally dead code until phase 02 adds the columns.
type positionColumns struct {
	ZobristHash   uint64
	DecisionType  int
	Dice1, Dice2  int
	Pip1, Pip2    int
	PipDiff       int
	Off1, Off2    int
	BackCheckers1 int
	BackCheckers2 int
	NoContact     bool
	Occupancy1    uint32
	Occupancy2    uint32
	PointMask1    uint32
	PointMask2    uint32
	// mirrors of Position fields for indexed columns
	CubeValue int
	CubeOwner int
	Score1    int
	Score2    int
	HasJacoby int
	HasBeaver int
}

// populatePositionColumns computes every derived column value for a Position.
// The input should already be normalized (PlayerOnRoll == 0); if it isn't the
// function normalizes it internally.
func populatePositionColumns(p *Position) positionColumns {
	norm := p.NormalizeForStorage()
	var c positionColumns

	c.ZobristHash = ZobristHash(&norm)
	c.DecisionType = norm.DecisionType
	c.Dice1 = norm.Dice[0]
	c.Dice2 = norm.Dice[1]

	c.Pip1, c.Pip2 = PipCounts(norm.Board)
	c.PipDiff = c.Pip1 - c.Pip2

	c.Off1 = norm.Board.Bearoff[0]
	c.Off2 = norm.Board.Bearoff[1]

	// Back checkers: Black's (player-on-roll's) are in the opponent's home board (points 19-24);
	// White's are in Black's home board (points 1-6).
	for i := 19; i <= 24; i++ {
		if norm.Board.Points[i].Color == Black && norm.Board.Points[i].Checkers > 0 {
			c.BackCheckers1 += norm.Board.Points[i].Checkers
		}
	}
	for i := 1; i <= 6; i++ {
		if norm.Board.Points[i].Color == White && norm.Board.Points[i].Checkers > 0 {
			c.BackCheckers2 += norm.Board.Points[i].Checkers
		}
	}

	c.NoContact = norm.MatchesNoContact()

	c.Occupancy1, c.Occupancy2, c.PointMask1, c.PointMask2 = OccupancyMasks(&norm.Board)

	c.CubeValue = norm.Cube.Value
	c.CubeOwner = norm.Cube.Owner
	c.Score1 = norm.Score[0]
	c.Score2 = norm.Score[1]
	c.HasJacoby = norm.HasJacoby
	c.HasBeaver = norm.HasBeaver

	return c
}

// ---------------------------------------------------------------------------
// Compact board encoding (v2.2.0)
//
// The state column stores ONLY the board layout as a JSON array of 28 signed
// integers. Indices 0-25 correspond to Board.Points: positive values mean
// White checkers, negative mean Black, zero means empty. Indices 26-27 are
// Board.Bearoff[0] and Bearoff[1]. All other Position fields (cube, dice,
// score, decision_type, flags) are reconstructed from the denormalised
// columns on the same row.
//
// Format detection: state[0]=='[' → compact, state[0]=='{' → legacy JSON.
// ---------------------------------------------------------------------------

// encodeBoardCompact encodes a Board as a compact JSON array of 28 signed ints.
func encodeBoardCompact(b Board) string {
	var vals [28]int
	for i := 0; i < NumPoints+2; i++ {
		p := b.Points[i]
		if p.Checkers > 0 {
			if p.Color == White {
				vals[i] = p.Checkers
			} else {
				vals[i] = -p.Checkers
			}
		}
	}
	vals[26] = b.Bearoff[0]
	vals[27] = b.Bearoff[1]
	data, _ := json.Marshal(vals)
	return string(data)
}

// decodeBoardCompact decodes a compact JSON array back into a Board.
func decodeBoardCompact(s string) Board {
	var vals [28]int
	json.Unmarshal([]byte(s), &vals)
	var b Board
	for i := 0; i < NumPoints+2; i++ {
		v := vals[i]
		if v > 0 {
			b.Points[i] = Point{v, White}
		} else if v < 0 {
			b.Points[i] = Point{-v, Black}
		}
	}
	b.Bearoff[0] = vals[26]
	b.Bearoff[1] = vals[27]
	return b
}

// isCompactState returns true if the state string uses the compact array format.
func isCompactState(s string) bool {
	return len(s) > 0 && s[0] == '['
}

// reconstructPosition rebuilds a full Position from the compact state string
// and denormalised column values. For legacy JSON state it unmarshals the full
// Position but still overwrites with the column values (they are authoritative
// in the v2 schema).
func reconstructPosition(id int64, state string, decisionType, playerOnRoll, dice1, dice2, cubeValue, cubeOwner, score1, score2, hasJacoby, hasBeaver int) Position {
	var pos Position
	if isCompactState(state) {
		pos.Board = decodeBoardCompact(state)
	} else {
		json.Unmarshal([]byte(state), &pos)
	}
	pos.ID = id
	pos.DecisionType = decisionType
	pos.PlayerOnRoll = playerOnRoll
	pos.Dice = [2]int{dice1, dice2}
	pos.Cube = Cube{cubeOwner, cubeValue}
	pos.Score = [2]int{score1, score2}
	pos.HasJacoby = hasJacoby
	pos.HasBeaver = hasBeaver
	return pos
}

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

	// Normalize position for storage - always store from player on roll's perspective (player_on_roll = 0)
	normalizedPosition := position.NormalizeForStorage()

	compactState := encodeBoardCompact(normalizedPosition.Board)

	cols := populatePositionColumns(position)
	noContactInt := 0
	if cols.NoContact {
		noContactInt = 1
	}

	result, err := d.db.Exec(`
		INSERT INTO position (
			zobrist_hash, decision_type, player_on_roll, dice_1, dice_2,
			cube_value, cube_owner, score_1, score_2,
			has_jacoby, has_beaver,
			pip_1, pip_2, pip_diff, off_1, off_2,
			back_checkers_1, back_checkers_2, no_contact,
			occupancy_1, occupancy_2, point_mask_1, point_mask_2,
			state
		) VALUES (?,?,?,?,?, ?,?,?,?,  ?,?,  ?,?,?,?,?,  ?,?,?,  ?,?,?,?,  ?)`,
		int64(cols.ZobristHash), cols.DecisionType, normalizedPosition.PlayerOnRoll, cols.Dice1, cols.Dice2,
		cols.CubeValue, cols.CubeOwner, cols.Score1, cols.Score2,
		cols.HasJacoby, cols.HasBeaver,
		cols.Pip1, cols.Pip2, cols.PipDiff, cols.Off1, cols.Off2,
		cols.BackCheckers1, cols.BackCheckers2, noContactInt,
		int64(cols.Occupancy1), int64(cols.Occupancy2), int64(cols.PointMask1), int64(cols.PointMask2),
		compactState)
	if err != nil {
		return 0, err
	}

	positionID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	normalizedPosition.ID = positionID // Update the position ID

	// Update the original position with the saved ID and normalized state
	*position = normalizedPosition

	return positionID, nil
}

func (d *Database) UpdatePosition(position Position) error {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	compactState := encodeBoardCompact(position.Board)

	cols := populatePositionColumns(&position)
	noContactInt := 0
	if cols.NoContact {
		noContactInt = 1
	}

	_, err := d.db.Exec(`UPDATE position SET state = ?,
		zobrist_hash=?, decision_type=?, player_on_roll=?, dice_1=?, dice_2=?,
		cube_value=?, cube_owner=?, score_1=?, score_2=?,
		has_jacoby=?, has_beaver=?,
		pip_1=?, pip_2=?, pip_diff=?, off_1=?, off_2=?,
		back_checkers_1=?, back_checkers_2=?, no_contact=?,
		occupancy_1=?, occupancy_2=?, point_mask_1=?, point_mask_2=?
		WHERE id = ?`,
		compactState,
		int64(cols.ZobristHash), cols.DecisionType, position.PlayerOnRoll, cols.Dice1, cols.Dice2,
		cols.CubeValue, cols.CubeOwner, cols.Score1, cols.Score2,
		cols.HasJacoby, cols.HasBeaver,
		cols.Pip1, cols.Pip2, cols.PipDiff, cols.Off1, cols.Off2,
		cols.BackCheckers1, cols.BackCheckers2, noContactInt,
		int64(cols.Occupancy1), int64(cols.Occupancy2), int64(cols.PointMask1), int64(cols.PointMask2),
		position.ID)
	if err != nil {
		return err
	}

	return nil
}

func (d *Database) LoadPosition(id int) (*Position, error) {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	row := d.db.QueryRow(`SELECT `+positionSelectCols+` FROM position WHERE id = ?`, id)
	pos, err := scanPositionRow(row)
	if err != nil {
		return nil, err
	}

	return &pos, nil
}

func (d *Database) LoadAllPositions() ([]Position, error) {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	rows, err := d.db.Query(`SELECT ` + positionSelectCols + ` FROM position`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []Position
	for rows.Next() {
		position, err := scanPositionRow(rows)
		if err != nil {
			return nil, err
		}
		positions = append(positions, position)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return positions, nil
}

func (d *Database) DeletePosition(positionID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Delete the position — ON DELETE CASCADE handles analysis, comment, and collection_position
	_, err := d.db.Exec(`DELETE FROM position WHERE id = ?`, positionID)
	if err != nil {
		return err
	}

	return nil
}

func (p *Position) MatchesMirrorPosition(filter Position) bool {
	mirroredPosition := p.Mirror()
	return p.MatchesCheckerPosition(filter) || mirroredPosition.MatchesCheckerPosition(filter)
}

// Mirror creates a mirrored version of the current Position.
// It reverses the board points, swaps the bearoff positions,
// changes the player on roll, swaps the scores, and changes the cube owner.
// Returns the mirrored Position.
func (p *Position) Mirror() Position {
	mirrored := *p
	for i, point := range p.Board.Points {
		mirrored.Board.Points[25-i] = Point{
			Color:    point.Color,
			Checkers: point.Checkers,
		}
		if point.Color != -1 {
			mirrored.Board.Points[25-i].Color = 1 - point.Color
		}
	}
	mirrored.Board.Bearoff[0], mirrored.Board.Bearoff[1] = p.Board.Bearoff[1], p.Board.Bearoff[0]
	mirrored.PlayerOnRoll = 1 - p.PlayerOnRoll
	mirrored.Score[0], mirrored.Score[1] = p.Score[1], p.Score[0]
	if p.Cube.Owner != -1 {
		mirrored.Cube.Owner = 1 - p.Cube.Owner
	}
	return mirrored
}

// NormalizeForStorage returns a normalized version of the position for storage.
// Positions are always stored from the player on roll's perspective (player_on_roll = 0).
// If player_on_roll is 1, the position is mirrored so that player_on_roll becomes 0.
// This prevents storing duplicate positions that are just mirror images of each other.
func (p *Position) NormalizeForStorage() Position {
	if p.PlayerOnRoll == 1 {
		return p.Mirror()
	}
	return *p
}
