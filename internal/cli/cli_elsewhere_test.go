package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// The board of a match imported in both files is known in the other one:
// its analysis, its comment, the match where it was played. The other file is
// read, never written.
func TestCLI_ElsewhereReadsAnotherFile(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	if _, err := cli.db.ImportXGMatch(testdataPath("test.xg")); err != nil {
		t.Fatalf("ImportXGMatch here: %v", err)
	}
	positions, err := cli.db.LoadAllPositions()
	if err != nil || len(positions) == 0 {
		t.Fatalf("LoadAllPositions: %d, %v", len(positions), err)
	}

	otherPath := filepath.Join(tempDir(t), "other.db")
	other := NewDatabase()
	if err := other.SetupDatabase(otherPath); err != nil {
		t.Fatalf("SetupDatabase other: %v", err)
	}
	if _, err := other.ImportXGMatch(testdataPath("test.xg")); err != nil {
		t.Fatalf("ImportXGMatch other: %v", err)
	}
	theirs, err := other.LoadAllPositions()
	if err != nil {
		t.Fatalf("LoadAllPositions other: %v", err)
	}
	var theirID int64
	for _, p := range theirs {
		if p.Board == positions[0].Board && p.Dice == positions[0].Dice && p.Score == positions[0].Score && p.Cube == positions[0].Cube && p.DecisionType == positions[0].DecisionType && p.PlayerOnRoll == positions[0].PlayerOnRoll {
			theirID = p.ID
		}
	}
	if theirID == 0 {
		t.Fatal("the other file does not hold the same board")
	}
	if err := other.AddComment(theirID, "seen at the club"); err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	other.Close()
	before, err := os.ReadFile(otherPath)
	if err != nil {
		t.Fatalf("read other: %v", err)
	}

	out := captureStdout(t, func() {
		if err := cli.Run([]string{"elsewhere", "--db", dbPath, "--other", otherPath,
			"--position", strconv.FormatInt(positions[0].ID, 10), "--format", "json"}); err != nil {
			t.Errorf("elsewhere: %v", err)
		}
	})
	var got struct {
		Results []struct {
			File       string `json:"file"`
			PositionID int64  `json:"positionId"`
			Comments   []struct {
				Text string `json:"text"`
			} `json:"comments"`
			Plays []struct {
				Verdict string `json:"verdict"`
			} `json:"plays"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("decode %q: %v", out, err)
	}
	if len(got.Results) != 1 {
		t.Fatalf("results = %+v, want one per --other", got.Results)
	}
	r := got.Results[0]
	if r.File != otherPath || r.PositionID != theirID {
		t.Errorf("result %+v, want %s at position %d", r, otherPath, theirID)
	}
	if len(r.Comments) != 1 || r.Comments[0].Text != "seen at the club" {
		t.Errorf("comments = %+v", r.Comments)
	}
	if len(r.Plays) == 0 {
		t.Error("the match where the board was played is missing")
	}
	after, err := os.ReadFile(otherPath)
	if err != nil {
		t.Fatalf("read other: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Error("reading another file wrote to it")
	}
}

func TestCLI_ElsewhereRefusesAMissingFile(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	missing := filepath.Join(tempDir(t), "missing.db")
	if err := cli.Run([]string{"elsewhere", "--db", dbPath, "--other", missing, "--zobrist", "1"}); err == nil {
		t.Error("a missing --other file was accepted")
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Error("a missing --other file was created")
	}
}
