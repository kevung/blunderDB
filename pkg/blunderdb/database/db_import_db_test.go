package database

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
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
	t.Parallel()
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
	t.Parallel()
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

// TestImport_MajorVersionComparedNumerically (issue #169): a source database
// whose schema major version is newer than the target's is refused, whatever
// the digits. The check used to compare the major components as strings, so
// "10" sorted before "2" and a 10.x.x source passed as older than 2.15.0.
func TestImport_MajorVersionComparedNumerically(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	dst := NewDatabase()
	if err := dst.SetupDatabase(filepath.Join(dir, "target.db")); err != nil {
		t.Fatalf("SetupDatabase(target): %v", err)
	}
	defer dst.Close()

	// A source stamped with an arbitrary version: SetupDatabase builds the
	// current schema, the stamp alone is changed.
	sourceAt := func(t *testing.T, version string) string {
		t.Helper()
		path := filepath.Join(dir, "source_"+version+".db")
		src := NewDatabase()
		if err := src.SetupDatabase(path); err != nil {
			t.Fatalf("SetupDatabase(source %s): %v", version, err)
		}
		src.Close()
		stampVersion(t, path, version)
		return path
	}

	for _, c := range []struct {
		version string
		wantErr string
	}{
		{"10.0.0", "newer major"},
		{"3.0.0", "newer major"},
		{DatabaseVersion, ""},
		{"1.9.0", ""},
		{"two.point.oh", "import database version"},
	} {
		t.Run(c.version, func(t *testing.T) {
			_, err := dst.AnalyzeImportDatabase(sourceAt(t, c.version))
			switch {
			case c.wantErr == "" && err != nil:
				t.Fatalf("AnalyzeImportDatabase(%s): %v", c.version, err)
			case c.wantErr != "" && err == nil:
				t.Fatalf("AnalyzeImportDatabase(%s) accepted the source", c.version)
			case c.wantErr != "" && !strings.Contains(err.Error(), c.wantErr):
				t.Fatalf("AnalyzeImportDatabase(%s) = %v, want %q", c.version, err, c.wantErr)
			}
		})
	}

	// checkImportableVersion is the one place both AnalyzeImportDatabase and
	// CommitImportDatabase ask; the table above covers the first, this covers
	// the helper the second shares with it.
	if err := checkImportableVersion("10.0.0", "2.15.0"); err == nil {
		t.Error("checkImportableVersion(10.0.0, 2.15.0) accepted a newer major")
	}
	if err := checkImportableVersion("2.15.0", "10.0.0"); err != nil {
		t.Errorf("checkImportableVersion(2.15.0, 10.0.0): %v", err)
	}
}

// TestImport_AnalyzeComparesAllCommentRows (B.6, #174): AnalyzeImportDatabase
// used to read a single, arbitrarily-chosen comment row per position (a bare
// `QueryRow` with no ORDER BY) and compare only that one against the
// target's own single row — a position commented on more than once could
// report "nothing to merge" while a second comment sat unmerged, exactly the
// gap loadCommentText (storage/sqlshared) already closed for search. The fix
// joins every comment row on both sides (loadJoinedCommentText) before
// comparing.
func TestImport_AnalyzeComparesAllCommentRows(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	srcPath := filepath.Join(dir, "src.db")
	src := NewDatabase()
	if err := src.SetupDatabase(srcPath); err != nil {
		t.Fatalf("SetupDatabase(src): %v", err)
	}
	p := InitializePosition()
	posID, err := src.SavePosition(&p)
	if err != nil {
		t.Fatalf("SavePosition(src): %v", err)
	}
	// Two comment rows on the same position: the source has more to say than
	// a single row could hold.
	if err := src.AddComment(posID, "Alpha"); err != nil {
		t.Fatalf("AddComment(Alpha): %v", err)
	}
	if err := src.AddComment(posID, "NewNote"); err != nil {
		t.Fatalf("AddComment(NewNote): %v", err)
	}
	src.Close()

	dst := NewDatabase()
	if err := dst.SetupDatabase(filepath.Join(dir, "dst.db")); err != nil {
		t.Fatalf("SetupDatabase(dst): %v", err)
	}
	defer dst.Close()
	p2 := InitializePosition() // same board → same Zobrist hash, recognised as the same position
	dstPosID, err := dst.SavePosition(&p2)
	if err != nil {
		t.Fatalf("SavePosition(dst): %v", err)
	}
	if err := dst.AddComment(dstPosID, "Alpha"); err != nil {
		t.Fatalf("AddComment(dst, Alpha): %v", err)
	}

	analysis, err := dst.AnalyzeImportDatabase(srcPath)
	if err != nil {
		t.Fatalf("AnalyzeImportDatabase: %v", err)
	}
	if toAdd, _ := analysis["toAdd"].(int); toAdd != 0 {
		t.Errorf("toAdd = %v, want 0 — the position is recognised by hash", analysis["toAdd"])
	}
	if toMerge, _ := analysis["toMerge"].(int); toMerge != 1 {
		t.Errorf("toMerge = %v, want 1: the source's second comment row, %q, is not on the target", analysis["toMerge"], "NewNote")
	}
	if toSkip, _ := analysis["toSkip"].(int); toSkip != 0 {
		t.Errorf("toSkip = %v, want 0 — the position was wrongly considered fully up to date", analysis["toSkip"])
	}
}

// TestImport_CommitMergesAllCommentRows is TestImport_AnalyzeComparesAllCommentRows's
// counterpart for the actual write: CommitImportDatabase's own comment merge
// used to read a single arbitrary comment row (no ORDER BY) instead of
// joining every row the position carries, and its UPDATE rewrote every one of
// a multi-row position's comment rows to the same merged text (B.11, #179).
// A target position with an unrelated existing comment must end up holding
// both after the merge: its own comment, untouched, and a new row for the
// source's comment that was missing.
func TestImport_CommitMergesAllCommentRows(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	srcPath := filepath.Join(dir, "src.db")
	src := NewDatabase()
	if err := src.SetupDatabase(srcPath); err != nil {
		t.Fatalf("SetupDatabase(src): %v", err)
	}
	p := InitializePosition()
	posID, err := src.SavePosition(&p)
	if err != nil {
		t.Fatalf("SavePosition(src): %v", err)
	}
	if err := src.AddComment(posID, "Alpha"); err != nil {
		t.Fatalf("AddComment(Alpha): %v", err)
	}
	if err := src.AddComment(posID, "NewNote"); err != nil {
		t.Fatalf("AddComment(NewNote): %v", err)
	}
	src.Close()

	dst := NewDatabase()
	if err := dst.SetupDatabase(filepath.Join(dir, "dst.db")); err != nil {
		t.Fatalf("SetupDatabase(dst): %v", err)
	}
	defer dst.Close()
	p2 := InitializePosition() // same board → same Zobrist hash, recognised as the same position
	dstPosID, err := dst.SavePosition(&p2)
	if err != nil {
		t.Fatalf("SavePosition(dst): %v", err)
	}
	if err := dst.AddComment(dstPosID, "Alpha"); err != nil {
		t.Fatalf("AddComment(dst, Alpha): %v", err)
	}

	if _, err := dst.CommitImportDatabase(srcPath); err != nil {
		t.Fatalf("CommitImportDatabase: %v", err)
	}

	entries, err := dst.GetCommentsByPosition(dstPosID)
	if err != nil {
		t.Fatalf("GetCommentsByPosition: %v", err)
	}
	var texts []string
	for _, e := range entries {
		texts = append(texts, e.Text)
	}
	joined := strings.Join(texts, "\n\n")
	if !strings.Contains(joined, "Alpha") {
		t.Errorf("target's own comment %q was lost across the merge; rows = %q", "Alpha", texts)
	}
	if !strings.Contains(joined, "NewNote") {
		t.Errorf("source's second comment row %q was not merged in; rows = %q", "NewNote", texts)
	}
}
