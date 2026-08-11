package database

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
)

// TestImport_OldFormatExportFixture is fiche-04's "compat lecture" guard:
// blunderDB must go on reading exports produced by every version already in
// the wild, even after the export producer changes (fiche-04 switched the
// three export paths from a hand-rolled two-column position table with full
// JSON state to the current compact-state schema). This file is the format
// EVERY export produced before fiche-04: position(id, state,
// individually_imported) only — no zobrist_hash, no scalar columns — state
// holding the full position JSON, and metadata claiming the CURRENT
// database_version (which is what made the bug so quiet: the migration chain
// never runs its ALTER TABLE steps for a database that already claims to be
// current).
//
// Built by hand here rather than via ExportDatabase: fiche-04's whole point is
// that the producer no longer writes this shape, so there is nothing left in
// the codebase to generate it from — this literal DDL is the only remaining
// source of it, and it must stay frozen to what shipped.
func buildOldFormatExportFixture(t *testing.T, path string, positions []Position) {
	t.Helper()
	fixtureDB, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer fixtureDB.Close()

	for _, stmt := range []string{
		`CREATE TABLE position (id INTEGER PRIMARY KEY AUTOINCREMENT, state TEXT, individually_imported INTEGER NOT NULL DEFAULT 0)`,
		`CREATE TABLE analysis (id INTEGER PRIMARY KEY AUTOINCREMENT, position_id INTEGER UNIQUE, data JSON)`,
		`CREATE TABLE comment (id INTEGER PRIMARY KEY AUTOINCREMENT, position_id INTEGER UNIQUE, text TEXT)`,
		`CREATE TABLE metadata (key TEXT PRIMARY KEY, value TEXT)`,
	} {
		if _, err := fixtureDB.Exec(stmt); err != nil {
			t.Fatalf("fixture DDL %q: %v", stmt, err)
		}
	}
	// The bug: this stamps the CURRENT version onto a database that does not
	// actually have the current schema, so runMigrationChain's version-gated
	// ALTER TABLE steps never fire for it.
	if _, err := fixtureDB.Exec(`INSERT INTO metadata (key, value) VALUES ('database_version', ?)`, DatabaseVersion); err != nil {
		t.Fatalf("fixture metadata: %v", err)
	}

	for _, pos := range positions {
		norm := pos.NormalizeForStorage()
		data, err := json.Marshal(norm)
		if err != nil {
			t.Fatalf("marshal fixture position: %v", err)
		}
		if _, err := fixtureDB.Exec(`INSERT INTO position (state, individually_imported) VALUES (?, 0)`, string(data)); err != nil {
			t.Fatalf("insert fixture position: %v", err)
		}
	}
}

func TestImport_OldFormatExportFixture(t *testing.T) {
	dir := t.TempDir()
	fixturePath := filepath.Join(dir, "old_export.db")

	pos1 := InitializePosition()
	pos2 := InitializePosition()
	pos2.Dice = [2]int{6, 5}
	buildOldFormatExportFixture(t, fixturePath, []Position{pos1, pos2})

	t.Run("OpenDatabase reads it directly", func(t *testing.T) {
		d := NewDatabase()
		if err := d.OpenDatabase(fixturePath); err != nil {
			t.Fatalf("OpenDatabase(old-format export): %v", err)
		}
		defer d.Close()

		positions, err := d.LoadAllPositions()
		if err != nil {
			t.Fatalf("LoadAllPositions: %v", err)
		}
		if len(positions) != 2 {
			t.Fatalf("expected 2 positions, got %d", len(positions))
		}
	})

	t.Run("CommitImportDatabase merges it into an empty database", func(t *testing.T) {
		curPath := filepath.Join(dir, "current.db")
		cur := NewDatabase()
		if err := cur.SetupDatabase(curPath); err != nil {
			t.Fatalf("SetupDatabase: %v", err)
		}
		defer cur.Close()

		result, err := cur.CommitImportDatabase(fixturePath)
		if err != nil {
			t.Fatalf("CommitImportDatabase: %v", err)
		}
		if added, _ := result["added"].(int); added != 2 {
			t.Errorf("added=%v, want 2", result["added"])
		}
		positions, err := cur.LoadAllPositions()
		if err != nil {
			t.Fatalf("LoadAllPositions: %v", err)
		}
		if len(positions) != 2 {
			t.Fatalf("expected 2 positions after import, got %d", len(positions))
		}
	})

	// This is the scenario ADR-0007 exists for: a recipient who already has some
	// of a teacher's course (saved the ordinary way, through SavePosition) is
	// handed an old-format export of the same course and must not end up with
	// duplicates.
	t.Run("CommitImportDatabase does not duplicate positions already held", func(t *testing.T) {
		curPath := filepath.Join(dir, "current_with_positions.db")
		cur := NewDatabase()
		if err := cur.SetupDatabase(curPath); err != nil {
			t.Fatalf("SetupDatabase: %v", err)
		}
		defer cur.Close()

		p1 := InitializePosition()
		if _, err := cur.SavePosition(&p1); err != nil {
			t.Fatalf("SavePosition 1: %v", err)
		}
		p2 := InitializePosition()
		p2.Dice = [2]int{6, 5}
		if _, err := cur.SavePosition(&p2); err != nil {
			t.Fatalf("SavePosition 2: %v", err)
		}

		result, err := cur.CommitImportDatabase(fixturePath)
		if err != nil {
			t.Fatalf("CommitImportDatabase: %v", err)
		}
		if added, _ := result["added"].(int); added != 0 {
			t.Errorf("added=%v, want 0 (both positions already held)", result["added"])
		}
		positions, err := cur.LoadAllPositions()
		if err != nil {
			t.Fatalf("LoadAllPositions: %v", err)
		}
		if len(positions) != 2 {
			t.Errorf("importing the old-format export duplicated positions: got %d, want 2", len(positions))
		}
	})

	t.Run("AnalyzeImportDatabase reports it correctly", func(t *testing.T) {
		curPath := filepath.Join(dir, "current2.db")
		cur := NewDatabase()
		if err := cur.SetupDatabase(curPath); err != nil {
			t.Fatalf("SetupDatabase: %v", err)
		}
		defer cur.Close()

		analysis, err := cur.AnalyzeImportDatabase(fixturePath)
		if err != nil {
			t.Fatalf("AnalyzeImportDatabase: %v", err)
		}
		if toAdd, _ := analysis["toAdd"].(int); toAdd != 2 {
			t.Errorf("toAdd=%v, want 2", analysis["toAdd"])
		}
	})
}
