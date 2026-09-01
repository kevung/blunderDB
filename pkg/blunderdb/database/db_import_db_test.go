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

// countPositionsMissingScalars counts position rows the SQL search cannot see:
// rows whose Zobrist hash or scalar filter columns were never populated.
func countPositionsMissingScalars(t *testing.T, d *Database) int {
	t.Helper()
	var n int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM position
		WHERE zobrist_hash IS NULL OR dice_1 IS NULL OR pip_1 IS NULL
		   OR back_checkers_1 IS NULL OR no_contact IS NULL`).Scan(&n); err != nil {
		t.Fatalf("count positions missing scalars: %v", err)
	}
	return n
}

// TestImport_CommitFillsScalarColumns guards the "Import database" merge path
// (AnalyzeImportDatabase + CommitImportDatabase) against the bug BACKLOG.md
// recorded after fiche-04: a position the target did not hold yet was inserted
// with its state alone — no Zobrist hash, no scalar columns — so the row was
// invisible to every SQL filter and, because ReconstructPosition trusts the
// columns over the state, it no longer matched itself on the next import.
func TestImport_CommitFillsScalarColumns(t *testing.T) {
	dir := t.TempDir()

	// Source A: two positions saved the ordinary way.
	srcPath := filepath.Join(dir, "a.db")
	src := NewDatabase()
	if err := src.SetupDatabase(srcPath); err != nil {
		t.Fatalf("SetupDatabase(A): %v", err)
	}
	p1 := InitializePosition()
	if _, err := src.SavePosition(&p1); err != nil {
		t.Fatalf("SavePosition 1: %v", err)
	}
	p2 := InitializePosition()
	p2.Dice = [2]int{6, 5}
	if _, err := src.SavePosition(&p2); err != nil {
		t.Fatalf("SavePosition 2: %v", err)
	}
	src.Close()

	// Target B: empty, receives A twice.
	dst := NewDatabase()
	if err := dst.SetupDatabase(filepath.Join(dir, "b.db")); err != nil {
		t.Fatalf("SetupDatabase(B): %v", err)
	}
	defer dst.Close()

	analysis, err := dst.AnalyzeImportDatabase(srcPath)
	if err != nil {
		t.Fatalf("AnalyzeImportDatabase #1: %v", err)
	}
	if toAdd, _ := analysis["toAdd"].(int); toAdd != 2 {
		t.Errorf("first analyze: toAdd=%v, want 2", analysis["toAdd"])
	}
	result, err := dst.CommitImportDatabase(srcPath)
	if err != nil {
		t.Fatalf("CommitImportDatabase #1: %v", err)
	}
	if added, _ := result["added"].(int); added != 2 {
		t.Errorf("first commit: added=%v, want 2", result["added"])
	}

	// 1. The scalar search columns are populated, like any saved position.
	if n := countPositionsMissingScalars(t, dst); n != 0 {
		t.Errorf("%d imported position(s) have NULL scalar columns", n)
	}

	// 2. An SQL filter on those columns finds the imported position.
	found, err := dst.LoadPositionsByFilters(SearchFilters{
		Filter:         Position{Dice: [2]int{6, 5}, PlayerOnRoll: p2.PlayerOnRoll, DecisionType: p2.DecisionType},
		DiceRollFilter: true,
	})
	if err != nil {
		t.Fatalf("LoadPositionsByFilters: %v", err)
	}
	if len(found) != 1 {
		t.Errorf("dice filter 6-5 found %d position(s), want 1", len(found))
	}

	// 3. Importing the same database again is a no-op: the positions recognise
	// themselves both through the identity map and through the Zobrist index.
	analysis, err = dst.AnalyzeImportDatabase(srcPath)
	if err != nil {
		t.Fatalf("AnalyzeImportDatabase #2: %v", err)
	}
	if toAdd, _ := analysis["toAdd"].(int); toAdd != 0 {
		t.Errorf("second analyze: toAdd=%v, want 0", analysis["toAdd"])
	}
	result, err = dst.CommitImportDatabase(srcPath)
	if err != nil {
		t.Fatalf("CommitImportDatabase #2: %v", err)
	}
	if added, _ := result["added"].(int); added != 0 {
		t.Errorf("second commit: added=%v, want 0", result["added"])
	}
	positions, err := dst.LoadAllPositions()
	if err != nil {
		t.Fatalf("LoadAllPositions: %v", err)
	}
	if len(positions) != 2 {
		t.Errorf("second import duplicated positions: got %d, want 2", len(positions))
	}

	// 4. SavePosition (the canonical write path) deduplicates against them too.
	again := InitializePosition()
	again.Dice = [2]int{6, 5}
	res, err := dst.SaveIndividualPosition(&again)
	if err != nil {
		t.Fatalf("SaveIndividualPosition: %v", err)
	}
	if !res.Existed {
		t.Errorf("SaveIndividualPosition did not recognise the imported position by hash")
	}
}
