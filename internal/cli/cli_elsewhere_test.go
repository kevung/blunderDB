package cli

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
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

func TestCLI_ElsewhereSchemaRefusalSaysWhichSideMoves(t *testing.T) {
	if err := schemaReadable("a.db", domain.DatabaseVersion); err != nil {
		t.Errorf("current schema refused: %v", err)
	}
	if err := schemaReadable("a.db", "1.0.0"); err == nil || !strings.Contains(err.Error(), "open it once with blunderDB") {
		t.Errorf("older schema: %v, want the advice to open it once", err)
	}
	if err := schemaReadable("a.db", "999.0.0"); err == nil || !strings.Contains(err.Error(), "update blunderDB") {
		t.Errorf("newer schema: %v, want the advice to update blunderDB", err)
	}
}

// An older file is refused, and left as it was: upgrading it is a write.
func TestCLI_ElsewhereRefusesAnOlderFile(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	old := filepath.Join(tempDir(t), "old?#%.db")
	other := NewDatabase()
	if err := other.SetupDatabase(old); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	other.Close()
	raw, err := sql.Open("sqlite", sqlite.DSN(old))
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if _, err := raw.Exec(`UPDATE metadata SET value = '1.0.0' WHERE key = 'database_version'`); err != nil {
		t.Fatalf("downgrade: %v", err)
	}
	raw.Close()
	err = cli.Run([]string{"elsewhere", "--db", dbPath, "--other", old, "--zobrist", "1"})
	if err == nil || !strings.Contains(err.Error(), "older") {
		t.Errorf("older file: %v, want refused as older", err)
	}
}

func TestCLI_ElsewhereRefusesWhatIsNotADatabase(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	dir := tempDir(t)
	empty := filepath.Join(dir, "empty.db")
	text := filepath.Join(dir, "notes.db")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(text, []byte("not sqlite at all, just some notes"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{empty, text} {
		err := cli.Run([]string{"elsewhere", "--db", dbPath, "--other", f, "--zobrist", "1"})
		if err == nil || !strings.Contains(err.Error(), "not a blunderDB database") {
			t.Errorf("%s: %v, want refused as not a blunderDB database", f, err)
		}
	}
	if fi, _ := os.Stat(empty); fi.Size() != 0 {
		t.Error("the empty file was written to")
	}
}
