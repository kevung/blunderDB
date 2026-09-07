package cli

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestCLIRepairReportsTheCrawfordCounter (#338): `repair` now has a third pass
// — the Crawford sentinel — and the counter it reports must say how many
// positions were actually rehashed, not merely that the pass ran. The fixture
// is a 7-point match whose second game is the Crawford game and whose third is
// post-Crawford, with one position wrongly stored at away 1 in each of the two.
func TestCLIRepairReportsTheCrawfordCounter(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "repair.db")

	db := NewDatabase()
	if err := db.SetupDatabase(dbPath); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	raw := RawConn(db)
	matchID := createMatch(t, db, "Alice", "Bob", "2025-01-01", 7, 0)
	var games [2]int64
	for i, initial := range [2][2]int{{6, 2}, {6, 3}} {
		res, err := raw.Exec(
			`INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2) VALUES (?, ?, ?, ?)`,
			matchID, i+1, initial[0], initial[1])
		if err != nil {
			t.Fatalf("insert game: %v", err)
		}
		games[i], _ = res.LastInsertId()
	}
	for i, dice := range [2][2]int{{4, 2}, {6, 5}} {
		pos := InitializePosition()
		pos.Score = [2]int{domain.Crawford, 4 - i}
		pos.Dice = dice
		id, err := db.SavePosition(&pos)
		if err != nil {
			t.Fatalf("SavePosition: %v", err)
		}
		if _, err := raw.Exec(
			`INSERT INTO move (game_id, move_number, move_type, position_id, player, dice_1, dice_2)
			 VALUES (?, 1, 'checker', ?, 0, ?, ?)`, games[i], id, dice[0], dice[1]); err != nil {
			t.Fatalf("insert move: %v", err)
		}
	}
	db.Close()

	cli := NewCLI()
	t.Cleanup(func() {
		if cli.db != nil {
			cli.db.Close()
		}
	})
	out := captureStdout(t, func() {
		if err := cli.runRepair([]string{"--db", dbPath, "--format", "json"}); err != nil {
			t.Fatalf("runRepair: %v", err)
		}
	})

	var report struct {
		Repaired int `json:"repaired"`
		Phases   int `json:"phases"`
		Crawford int `json:"crawford"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("output is not the JSON report: %v\n%s", err, out)
	}
	// Only the post-Crawford one: the Crawford game's own position is right.
	if report.Crawford != 1 {
		t.Errorf("crawford counter = %d, want 1\n%s", report.Crawford, out)
	}

	// And the text format says it in words.
	text := captureStdout(t, func() {
		if err := cli.runRepair([]string{"--db", dbPath}); err != nil {
			t.Fatalf("runRepair (text): %v", err)
		}
	})
	if !strings.Contains(text, "Crawford") {
		t.Errorf("the text report says nothing about the Crawford sentinel:\n%s", text)
	}
}
