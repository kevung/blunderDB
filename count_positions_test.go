package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
)

func TestCountPositionsAfterImports(t *testing.T) {
	xgFile := filepath.Join("testdata", "test.xg")
	sgfFile := filepath.Join("testdata", "test.sgf")
	matFile := filepath.Join("testdata", "test.mat")

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db := NewDatabase()
	if err := db.SetupDatabase(dbPath); err != nil {
		t.Fatalf("Failed to setup database: %v", err)
	}

	countPositions := func() int {
		var count int
		db.db.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&count)
		return count
	}

	// countSemanticDuplicates checks that no two positions share the same semantic key
	// (board, cube, score, player_on_roll, decision_type, dice)
	countSemanticDuplicates := func() int {
		rows, _ := db.db.Query(`SELECT state FROM position`)
		defer rows.Close()
		seen := make(map[string]int)
		for rows.Next() {
			var stateJSON string
			rows.Scan(&stateJSON)
			var pos Position
			json.Unmarshal([]byte(stateJSON), &pos)
			key := fmt.Sprintf("%v|%v|%v|%d|%d|%v",
				pos.Board, pos.Cube, pos.Score,
				pos.PlayerOnRoll, pos.DecisionType, pos.Dice)
			seen[key]++
		}
		dupes := 0
		for _, count := range seen {
			if count > 1 {
				dupes += count - 1
			}
		}
		return dupes
	}

	// Import MAT first
	_, err := db.ImportGnuBGMatch(matFile)
	if err != nil {
		t.Fatalf("Failed to import MAT: %v", err)
	}
	countAfterMAT := countPositions()
	t.Logf("After MAT import: %d positions", countAfterMAT)

	// Import SGF (canonical dup - may add genuinely new positions but no duplicates)
	_, err = db.ImportGnuBGMatch(sgfFile)
	if err != nil {
		t.Fatalf("Failed to import SGF: %v", err)
	}
	countAfterSGF := countPositions()
	t.Logf("After SGF import: %d positions (added %d genuinely new)", countAfterSGF, countAfterSGF-countAfterMAT)

	// Import XG (canonical dup - may add genuinely new positions but no duplicates)
	_, err = db.ImportXGMatch(xgFile)
	if err != nil {
		t.Fatalf("Failed to import XG: %v", err)
	}
	countAfterXG := countPositions()
	t.Logf("After XG import: %d positions (added %d genuinely new)", countAfterXG, countAfterXG-countAfterSGF)

	// Verify no semantic duplicates exist
	dupes := countSemanticDuplicates()
	if dupes > 0 {
		t.Errorf("Found %d semantic duplicate positions (same board/cube/score/player/decision/dice)", dupes)
	}

	// Verify importing the same file again adds zero positions
	countBefore2ndSGF := countPositions()
	_, err = db.ImportGnuBGMatch(sgfFile)
	if err != nil {
		t.Fatalf("Failed to re-import SGF: %v", err)
	}
	countAfter2ndSGF := countPositions()
	if countAfter2ndSGF != countBefore2ndSGF {
		t.Errorf("Re-importing SGF should not create any positions: had %d, now %d", countBefore2ndSGF, countAfter2ndSGF)
	}

	t.Logf("Final: %d total positions, 0 semantic duplicates", countAfterXG)
}

// TestDiagnosePositionDifferences shows what's different between positions
// created by different parsers for the same physical state
func TestDiagnosePositionDifferences(t *testing.T) {
	matFile := filepath.Join("testdata", "test.mat")
	sgfFile := filepath.Join("testdata", "test.sgf")

	tmpDir := t.TempDir()

	// Import MAT into db1
	dbPath1 := filepath.Join(tmpDir, "mat.db")
	db1 := NewDatabase()
	db1.SetupDatabase(dbPath1)
	db1.ImportGnuBGMatch(matFile)

	// Import SGF into db2
	dbPath2 := filepath.Join(tmpDir, "sgf.db")
	db2 := NewDatabase()
	db2.SetupDatabase(dbPath2)
	db2.ImportGnuBGMatch(sgfFile)

	// Load all positions from both
	loadAllPositions := func(db *Database) []Position {
		rows, _ := db.db.Query(`SELECT state FROM position`)
		defer rows.Close()
		var positions []Position
		for rows.Next() {
			var stateJSON string
			rows.Scan(&stateJSON)
			var pos Position
			json.Unmarshal([]byte(stateJSON), &pos)
			positions = append(positions, pos)
		}
		return positions
	}

	matPositions := loadAllPositions(db1)
	sgfPositions := loadAllPositions(db2)

	t.Logf("MAT positions: %d, SGF positions: %d", len(matPositions), len(sgfPositions))

	// Build fingerprint map for MAT positions (board + score + cube + dice + decision)
	type posFingerprint struct {
		Board        Board
		Cube         Cube
		Dice         [2]int
		Score        [2]int
		DecisionType int
	}

	matFP := make(map[string]int)
	for _, p := range matPositions {
		fp := posFingerprint{p.Board, p.Cube, p.Dice, p.Score, p.DecisionType}
		key, _ := json.Marshal(fp)
		matFP[string(key)]++
	}

	sgfOnlyCount := 0
	for _, p := range sgfPositions {
		fp := posFingerprint{p.Board, p.Cube, p.Dice, p.Score, p.DecisionType}
		key, _ := json.Marshal(fp)
		if _, exists := matFP[string(key)]; !exists {
			sgfOnlyCount++
			if sgfOnlyCount <= 5 {
				t.Logf("SGF-only position: Dice=%v Score=%v Cube=%v DecisionType=%d",
					p.Dice, p.Score, p.Cube, p.DecisionType)
			}
		}
	}
	t.Logf("SGF positions not matching any MAT position (full fingerprint): %d", sgfOnlyCount)

	// Try matching by board-only
	type boardFingerprint struct {
		Board Board
	}
	matBoardFP := make(map[string]int)
	for _, p := range matPositions {
		fp := boardFingerprint{p.Board}
		key, _ := json.Marshal(fp)
		matBoardFP[string(key)]++
	}

	sgfBoardOnlyCount := 0
	for _, p := range sgfPositions {
		fp := boardFingerprint{p.Board}
		key, _ := json.Marshal(fp)
		if _, exists := matBoardFP[string(key)]; !exists {
			sgfBoardOnlyCount++
			if sgfBoardOnlyCount <= 3 {
				t.Logf("SGF-only board: Score=%v Dice=%v Cube=%v DecisionType=%d",
					p.Score, p.Dice, p.Cube, p.DecisionType)
				// Show board briefly
				for i := 0; i < 26; i++ {
					if p.Board.Points[i].Checkers > 0 {
						t.Logf("  Point %d: %d checkers, color %d", i, p.Board.Points[i].Checkers, p.Board.Points[i].Color)
					}
				}
				t.Logf("  Bearoff: %v", p.Board.Bearoff)
			}
		}
	}
	t.Logf("SGF positions not matching any MAT position (board only): %d", sgfBoardOnlyCount)

	// Try matching by board + dice + decisionType (ignoring score/cube)
	type coreFP struct {
		Board        Board
		Dice         [2]int
		DecisionType int
	}
	matCoreFP := make(map[string]int)
	for _, p := range matPositions {
		fp := coreFP{p.Board, p.Dice, p.DecisionType}
		key, _ := json.Marshal(fp)
		matCoreFP[string(key)]++
	}

	sgfCoreOnlyCount := 0
	for _, p := range sgfPositions {
		fp := coreFP{p.Board, p.Dice, p.DecisionType}
		key, _ := json.Marshal(fp)
		if _, exists := matCoreFP[string(key)]; !exists {
			sgfCoreOnlyCount++
		}
	}
	t.Logf("SGF positions not matching any MAT position (board+dice+decisionType): %d", sgfCoreOnlyCount)

	// Check: how many SGF positions have identical board but different score/cube?
	sgfDiffMeta := 0
	for _, p := range sgfPositions {
		boardFP := boardFingerprint{p.Board}
		boardKey, _ := json.Marshal(boardFP)
		if _, exists := matBoardFP[string(boardKey)]; exists {
			// Board matches, check if full fingerprint also matches
			fullFP := posFingerprint{p.Board, p.Cube, p.Dice, p.Score, p.DecisionType}
			fullKey, _ := json.Marshal(fullFP)
			if _, fullExists := matFP[string(fullKey)]; !fullExists {
				sgfDiffMeta++
				if sgfDiffMeta <= 3 {
					// find the MAT position with this board
					for _, mp := range matPositions {
						mpBoardFP := boardFingerprint{mp.Board}
						mpBoardKey, _ := json.Marshal(mpBoardFP)
						if string(mpBoardKey) == string(boardKey) {
							t.Logf("Board match but metadata differs:")
							t.Logf("  MAT: Score=%v Dice=%v Cube=%v DT=%d", mp.Score, mp.Dice, mp.Cube, mp.DecisionType)
							t.Logf("  SGF: Score=%v Dice=%v Cube=%v DT=%d", p.Score, p.Dice, p.Cube, p.DecisionType)
							break
						}
					}
				}
			}
		}
	}
	t.Logf("SGF positions with board match but different metadata: %d", sgfDiffMeta)

	fmt.Println("Note: These positions are the ones that currently get created as duplicates during canonical duplicate import")
}
