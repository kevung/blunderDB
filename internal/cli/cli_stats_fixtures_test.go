package cli

import "testing"

// Stats test fixtures for the package main CLI tests. The database package has
// equivalent helpers for its own tests; these are kept here because cli_stats
// tests live in package main (they drive the CLI type) and cannot reach the
// database package's test helpers.

// insertStatsFixtureRow inserts one decision row: a position (with
// decision_type), an analysis row carrying the error, and a move linking the
// game to the position. Returns the position id.
func insertStatsFixtureRow(t *testing.T, db *Database, matchID int64, gameID int64, errMP int, dt int, player int, moveNum int) int64 {
	t.Helper()
	// Convert blunderDB encoding (0/1) to XG encoding (1/-1).
	xgPlayer := int32(1)
	if player != 0 {
		xgPlayer = -1
	}
	res, err := db.Conn().Exec(
		`INSERT INTO position (decision_type, state) VALUES (?, '')`, dt,
	)
	if err != nil {
		t.Fatalf("insert position: %v", err)
	}
	posID, _ := res.LastInsertId()

	cubeErr := 0
	moveErr := 0
	closeCube := 0
	if dt == 1 {
		cubeErr = errMP
		closeCube = 1
	} else {
		moveErr = errMP
	}
	if _, err = db.Conn().Exec(
		`INSERT INTO analysis (position_id, data, cube_error, best_move_equity_error, is_close_cube) VALUES (?, '{}', ?, ?, ?)`,
		posID, cubeErr, moveErr, closeCube,
	); err != nil {
		t.Fatalf("insert analysis: %v", err)
	}

	if _, err = db.Conn().Exec(
		`INSERT INTO move (game_id, move_number, position_id, player) VALUES (?, ?, ?, ?)`,
		gameID, moveNum, posID, xgPlayer,
	); err != nil {
		t.Fatalf("insert move: %v", err)
	}
	return posID
}

// createMatch creates a match row and returns its id.
func createMatch(t *testing.T, db *Database, p1, p2, date string, matchLength int, tournamentID int64) int64 {
	t.Helper()
	var tidVal any
	if tournamentID > 0 {
		tidVal = tournamentID
	}
	res, err := db.Conn().Exec(
		`INSERT INTO match (player1_name, player2_name, match_date, match_length, tournament_id) VALUES (?,?,?,?,?)`,
		p1, p2, date, matchLength, tidVal,
	)
	if err != nil {
		t.Fatalf("insert match: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

// createGame creates a game row for a match and returns its id.
func createGame(t *testing.T, db *Database, matchID int64) int64 {
	t.Helper()
	res, err := db.Conn().Exec(
		`INSERT INTO game (match_id, game_number) VALUES (?, 1)`, matchID,
	)
	if err != nil {
		t.Fatalf("insert game: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}
