package database

import (
	"path/filepath"
	"testing"
)

// move_error_cube_test.go — regression tests for cube-error attribution in the
// E>x "move error" (blunder) filter, from player 1's point of view.
//
// Bug history: the filter mapped player 1's cube action to an error with an
// ad-hoc switch that (a) had no case for a doubling action ("Double"), so a bad
// double by player 1 was silently dropped, and (b) routed "Double/Pass" to the
// pass branch, scoring the doubler's decision with the opponent's pass error.
// Both now go through engine.CubeActionError (min(DoubleTakeError, DoublePassError)).

// saveCubeDecision stores a player-1 cube decision: a CubeAction position, a
// DoublingCube analysis with the given equities/errors, and a player=1 move row
// carrying cubeAction. cubeValue distinguishes positions so dedup keeps them.
func saveCubeDecision(t *testing.T, db *Database, gameID int64, cubeValue int, cubeAction string,
	dtErr, dpErr, ndErr, dtEq, dpEq float64) *Position {
	t.Helper()
	pos := InitializePosition()
	pos.DecisionType = CubeAction
	pos.Dice = [2]int{0, 0}
	pos.Cube.Value = cubeValue
	posID, err := db.SavePosition(&pos)
	if err != nil {
		t.Fatalf("SavePosition: %v", err)
	}
	pos.ID = posID

	ana := PositionAnalysis{
		AnalysisType: "DoublingCube",
		DoublingCubeAnalysis: &DoublingCubeAnalysis{
			BestCubeAction:          "No Double",
			CubefulNoDoubleError:    ndErr,
			CubefulDoubleTakeError:  dtErr,
			CubefulDoublePassError:  dpErr,
			CubefulDoubleTakeEquity: dtEq,
			CubefulDoublePassEquity: dpEq,
		},
	}
	if err := db.SaveAnalysis(posID, ana); err != nil {
		t.Fatalf("SaveAnalysis: %v", err)
	}

	if _, err := db.db.Exec(
		`INSERT INTO move (game_id, move_number, move_type, position_id, player, cube_action)
		 VALUES (?, 0, 'cube', ?, 1, ?)`, gameID, posID, cubeAction); err != nil {
		t.Fatalf("insert move: %v", err)
	}
	return &pos
}

func TestMoveErrorFilter_BadDoubleRetained(t *testing.T) {
	db := NewDatabase()
	if err := db.SetupDatabase(filepath.Join(t.TempDir(), "move_error_cube.db")); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer db.Close()
	gameID := createGame(t, db, createMatch(t, db, "P1", "P2", "2024-01-01", 5, 0))

	// best = No Double; doubling error = min(DT,DP) = -0.5 → 500 millipoints.
	pos := saveCubeDecision(t, db, gameID, 0, "Double", -0.3, -0.5, 0, -0.3, -0.5)

	if !MatchesMoveErrorFilter(pos, "E>100", db) {
		t.Error("E>100: bad double by player 1 should be retained (500mp ≥ 100), got false")
	}
	if MatchesMoveErrorFilter(pos, "E>600", db) {
		t.Error("E>600: 500mp < 600 should not match, got true")
	}
}

func TestMoveErrorFilter_DoublePassUsesDoublingError(t *testing.T) {
	db := NewDatabase()
	if err := db.SetupDatabase(filepath.Join(t.TempDir(), "move_error_cube_dp.db")); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer db.Close()
	gameID := createGame(t, db, createMatch(t, db, "P1", "P2", "2024-01-01", 5, 0))

	// DT error (-0.6) is worse than DP error (-0.2): the doubling error is
	// min(DT,DP) = 0.6 → 600mp. The old code used the pass error only (200mp),
	// so E>400 distinguishes the fix from the bug.
	pos := saveCubeDecision(t, db, gameID, 0, "Double/Pass", -0.6, -0.2, 0, -0.6, -0.2)

	if !MatchesMoveErrorFilter(pos, "E>400", db) {
		t.Error("E>400: Double/Pass must be scored with the doubling error (600mp), got false")
	}
}
