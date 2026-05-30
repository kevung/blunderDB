package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// setupCLI creates a CLI with an in-memory DB for tests that call internal
// methods directly (listMatches, showStats, etc.).
func setupCLI(t *testing.T) *CLI {
	t.Helper()
	db := NewDatabase()
	if err := db.SetupDatabase(":memory:"); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})
	return &CLI{db: db}
}

// setupCLIWithDB creates a CLI backed by a temp file DB and returns the path.
// The file is cleaned up automatically by t.TempDir().
func setupCLIWithDB(t *testing.T) (*CLI, string) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db := NewDatabase()
	if err := db.SetupDatabase(dbPath); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})
	return &CLI{db: db}, dbPath
}

// captureStdout captures stdout produced by fn and returns it as a string.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	fn()

	w.Close()
	os.Stdout = old
	return <-outC
}

// testdataPath returns the absolute path for a testdata fixture file.
func testdataPath(name string) string {
	return filepath.Join("testdata", name)
}

// ---------------------------------------------------------------------------
// 1. Import round-trip tests
// ---------------------------------------------------------------------------

func TestCLI_ImportXG(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("test.xg")})
	if err != nil {
		t.Fatalf("import XG: %v", err)
	}
	matches, _ := cli.db.GetAllMatches()
	if len(matches) == 0 {
		t.Fatal("expected at least 1 match after XG import")
	}
	positions, _ := cli.db.LoadAllPositions()
	if len(positions) == 0 {
		t.Fatal("expected at least 1 position after XG import")
	}
}

func TestCLI_ImportSGF(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("test.sgf")})
	if err != nil {
		t.Fatalf("import SGF: %v", err)
	}
	matches, _ := cli.db.GetAllMatches()
	if len(matches) == 0 {
		t.Fatal("expected at least 1 match after SGF import")
	}
}

func TestCLI_ImportMAT(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("test.mat")})
	if err != nil {
		t.Fatalf("import MAT: %v", err)
	}
	matches, _ := cli.db.GetAllMatches()
	if len(matches) == 0 {
		t.Fatal("expected at least 1 match after MAT import")
	}
}

func TestCLI_ImportBGF(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("TachiAI_V_player_Nov_2__2025__16_55.bgf")})
	if err != nil {
		t.Fatalf("import BGF: %v", err)
	}
	matches, _ := cli.db.GetAllMatches()
	if len(matches) == 0 {
		t.Fatal("expected at least 1 match after BGF import")
	}
}

func TestCLI_ImportDuplicate(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	// First import should succeed.
	err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("test.xg")})
	if err != nil {
		t.Fatalf("first import: %v", err)
	}

	// Second import of the same file should fail (duplicate).
	err = cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("test.xg")})
	if err == nil {
		t.Fatal("expected duplicate import to return an error")
	}
}

// ---------------------------------------------------------------------------
// 2. List / stats tests
// ---------------------------------------------------------------------------

func TestCLI_ListMatches(t *testing.T) {
	cli := setupCLI(t)
	// Import directly to avoid file-backed DB requirement.
	if _, err := cli.db.ImportXGMatch(testdataPath("test.xg")); err != nil {
		t.Fatalf("ImportXGMatch: %v", err)
	}

	out := captureStdout(t, func() {
		if err := cli.listMatches(10); err != nil {
			t.Fatalf("listMatches: %v", err)
		}
	})

	if len(out) == 0 {
		t.Fatal("expected non-empty listMatches output")
	}
	// Output should contain "Players:" header.
	if !bytes.Contains([]byte(out), []byte("Players:")) {
		t.Errorf("listMatches output missing player info:\n%s", out)
	}
}

func TestCLI_ListPositions(t *testing.T) {
	cli := setupCLI(t)
	if _, err := cli.db.ImportXGMatch(testdataPath("test.xg")); err != nil {
		t.Fatalf("ImportXGMatch: %v", err)
	}

	out := captureStdout(t, func() {
		if err := cli.listPositions(10); err != nil {
			t.Fatalf("listPositions: %v", err)
		}
	})

	if len(out) == 0 {
		t.Fatal("expected non-empty listPositions output")
	}
	if !bytes.Contains([]byte(out), []byte("position(s)")) {
		t.Errorf("listPositions output missing position count:\n%s", out)
	}
}

func TestCLI_ShowStats(t *testing.T) {
	cli := setupCLI(t)
	if _, err := cli.db.ImportXGMatch(testdataPath("test.xg")); err != nil {
		t.Fatalf("ImportXGMatch: %v", err)
	}

	out := captureStdout(t, func() {
		if err := cli.showStats(StatsFilter{DecisionType: -1}, "pr", "text", 10); err != nil {
			t.Fatalf("showStats: %v", err)
		}
	})

	if !bytes.Contains([]byte(out), []byte("Matches:")) {
		t.Errorf("showStats output missing match count:\n%s", out)
	}
	if !bytes.Contains([]byte(out), []byte("Positions:")) {
		t.Errorf("showStats output missing position count:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// 3. Search tests
// ---------------------------------------------------------------------------

func TestCLI_Search(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	if err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("test.xg")}); err != nil {
		t.Fatalf("import: %v", err)
	}

	// Search for checker positions — should return results.
	out := captureStdout(t, func() {
		err := cli.Run([]string{"search", "--db", dbPath, "--decision", "checker", "--format", "table"})
		if err != nil {
			t.Fatalf("search checker: %v", err)
		}
	})

	if !bytes.Contains([]byte(out), []byte("position(s)")) {
		t.Errorf("search output missing position count:\n%s", out)
	}
}

func TestCLI_SearchDice(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	if err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("test.xg")}); err != nil {
		t.Fatalf("import: %v", err)
	}

	// Both dice (any order)
	out := captureStdout(t, func() {
		if err := cli.Run([]string{"search", "--db", dbPath, "--dice", "6,5", "--format", "table"}); err != nil {
			t.Fatalf("search --dice 6,5: %v", err)
		}
	})
	if !bytes.Contains([]byte(out), []byte("position(s)")) {
		t.Errorf("search --dice 6,5 missing position count:\n%s", out)
	}

	// First die only
	out = captureStdout(t, func() {
		if err := cli.Run([]string{"search", "--db", dbPath, "--dice", "6", "--format", "table"}); err != nil {
			t.Fatalf("search --dice 6: %v", err)
		}
	})
	if !bytes.Contains([]byte(out), []byte("position(s)")) {
		t.Errorf("search --dice 6 missing position count:\n%s", out)
	}

	// Invalid value
	if err := cli.Run([]string{"search", "--db", dbPath, "--dice", "7"}); err == nil {
		t.Errorf("expected error for --dice 7, got nil")
	}
	if err := cli.Run([]string{"search", "--db", dbPath, "--dice", "1,2,3"}); err == nil {
		t.Errorf("expected error for --dice 1,2,3, got nil")
	}
}

func TestCLI_SearchNoResults(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	// Empty DB — search should return 0 positions.
	out := captureStdout(t, func() {
		err := cli.Run([]string{"search", "--db", dbPath, "--decision", "cube"})
		if err != nil {
			t.Fatalf("search: %v", err)
		}
	})

	if !bytes.Contains([]byte(out), []byte("Found 0 position(s)")) {
		t.Errorf("expected 0 positions, got:\n%s", out)
	}
}

func TestCLI_SearchJSON(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	if err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("test.xg")}); err != nil {
		t.Fatalf("import: %v", err)
	}

	out := captureStdout(t, func() {
		err := cli.Run([]string{"search", "--db", dbPath, "--format", "json", "--limit", "2"})
		if err != nil {
			t.Fatalf("search json: %v", err)
		}
	})

	// JSON output should contain the key "id".
	if !bytes.Contains([]byte(out), []byte(`"id"`)) {
		t.Errorf("search JSON output missing id field:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// 4. Export tests
// ---------------------------------------------------------------------------

func TestCLI_Export(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	if err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("test.xg")}); err != nil {
		t.Fatalf("import: %v", err)
	}

	exportPath := filepath.Join(t.TempDir(), "export.db")
	err := cli.Run([]string{"export", "--db", dbPath, "--type", "database", "--file", exportPath})
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	info, err := os.Stat(exportPath)
	if err != nil {
		t.Fatalf("export file does not exist: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("exported file is empty")
	}
}

func TestCLI_ExportRoundTrip(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	if err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("test.xg")}); err != nil {
		t.Fatalf("import: %v", err)
	}

	// Count positions in original DB.
	origPositions, _ := cli.db.LoadAllPositions()

	exportPath := filepath.Join(t.TempDir(), "roundtrip.db")
	if err := cli.Run([]string{"export", "--db", dbPath, "--type", "database", "--file", exportPath}); err != nil {
		t.Fatalf("export: %v", err)
	}

	// Re-import into a fresh DB.
	cli2 := &CLI{db: NewDatabase()}
	if err := cli2.db.SetupDatabase(":memory:"); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	t.Cleanup(func() {
		cli2.db.Close()
	})
	if _, err := cli2.db.CommitImportDatabase(exportPath); err != nil {
		t.Fatalf("CommitImportDatabase: %v", err)
	}

	reimportedPositions, _ := cli2.db.LoadAllPositions()
	if len(reimportedPositions) != len(origPositions) {
		t.Errorf("position count mismatch: original=%d, reimported=%d", len(origPositions), len(reimportedPositions))
	}
}

// ---------------------------------------------------------------------------
// 5. Delete tests
// ---------------------------------------------------------------------------

func TestCLI_DeleteMatch(t *testing.T) {
	cli := setupCLI(t)
	matchID, err := cli.db.ImportXGMatch(testdataPath("test.xg"))
	if err != nil {
		t.Fatalf("ImportXGMatch: %v", err)
	}

	// Delete with confirm=true to skip stdin prompt.
	if err := cli.deleteMatch(matchID, true); err != nil {
		t.Fatalf("deleteMatch: %v", err)
	}

	matches, _ := cli.db.GetAllMatches()
	if len(matches) != 0 {
		t.Errorf("expected 0 matches after delete, got %d", len(matches))
	}
}

func TestCLI_DeleteMatchNotFound(t *testing.T) {
	cli := setupCLI(t)
	err := cli.deleteMatch(9999, true)
	if err == nil {
		t.Fatal("expected error when deleting nonexistent match")
	}
}

// ---------------------------------------------------------------------------
// 6. Create / Verify tests
// ---------------------------------------------------------------------------

func TestCLI_Create(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "created.db")
	cli := &CLI{db: NewDatabase()}

	err := cli.Run([]string{"create", "--db", dbPath, "--user", "tester", "--description", "test db"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	info, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("created DB does not exist: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("created DB is empty")
	}
}

func TestCLI_CreateExistingNoForce(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "existing.db")
	// Create the file first.
	if err := os.WriteFile(dbPath, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	cli := &CLI{db: NewDatabase()}
	err := cli.Run([]string{"create", "--db", dbPath})
	if err == nil {
		t.Fatal("expected error when creating over existing DB without --force")
	}
}

func TestCLI_CreateForce(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "force.db")
	if err := os.WriteFile(dbPath, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	cli := &CLI{db: NewDatabase()}
	err := cli.Run([]string{"create", "--db", dbPath, "--force"})
	if err != nil {
		t.Fatalf("create --force: %v", err)
	}

	info, _ := os.Stat(dbPath)
	if info.Size() == 0 {
		t.Fatal("forced-create DB is empty")
	}
}

func TestCLI_Verify(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	if err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("test.xg")}); err != nil {
		t.Fatalf("import: %v", err)
	}

	out := captureStdout(t, func() {
		err := cli.Run([]string{"verify", "--db", dbPath})
		if err != nil {
			t.Fatalf("verify: %v", err)
		}
	})

	if !bytes.Contains([]byte(out), []byte("Verification complete")) {
		t.Errorf("verify output missing completion message:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// 7. Info / Edit tests
// ---------------------------------------------------------------------------

func TestCLI_Info(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)

	out := captureStdout(t, func() {
		err := cli.Run([]string{"info", "--db", dbPath})
		if err != nil {
			t.Fatalf("info: %v", err)
		}
	})

	if !bytes.Contains([]byte(out), []byte("Database Information")) {
		t.Errorf("info output missing header:\n%s", out)
	}
}

func TestCLI_InfoJSON(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)

	out := captureStdout(t, func() {
		err := cli.Run([]string{"info", "--db", dbPath, "--format", "json"})
		if err != nil {
			t.Fatalf("info json: %v", err)
		}
	})

	if !bytes.Contains([]byte(out), []byte(`"stats"`)) {
		t.Errorf("info JSON output missing stats key:\n%s", out)
	}
}

func TestCLI_Edit(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)

	err := cli.Run([]string{"edit", "--db", dbPath, "--user", "Alice", "--description", "My games"})
	if err != nil {
		t.Fatalf("edit: %v", err)
	}

	metadata, err := cli.db.LoadMetadata()
	if err != nil {
		t.Fatalf("LoadMetadata: %v", err)
	}
	if metadata["user"] != "Alice" {
		t.Errorf("expected user=Alice, got %q", metadata["user"])
	}
	if metadata["description"] != "My games" {
		t.Errorf("expected description='My games', got %q", metadata["description"])
	}
}

// ---------------------------------------------------------------------------
// 8. Match command tests
// ---------------------------------------------------------------------------

func TestCLI_MatchJSON(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	if err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("test.xg")}); err != nil {
		t.Fatalf("import: %v", err)
	}
	matches, _ := cli.db.GetAllMatches()
	if len(matches) == 0 {
		t.Fatal("no matches after import")
	}

	out := captureStdout(t, func() {
		err := cli.Run([]string{"match", "--db", dbPath, "--id", fmt.Sprintf("%d", matches[0].ID), "--format", "json"})
		if err != nil {
			t.Fatalf("match json: %v", err)
		}
	})

	if !bytes.Contains([]byte(out), []byte(`"match"`)) && !bytes.Contains([]byte(out), []byte(`"player1_name"`)) {
		t.Errorf("match JSON output looks wrong:\n%s", out[:min(len(out), 500)])
	}
}

// ---------------------------------------------------------------------------
// 9. Batch import test
// ---------------------------------------------------------------------------

func TestCLI_ImportBatch(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	// Use the testdata directory itself — it contains .xg, .sgf, .mat, .bgf files.
	err := cli.Run([]string{"import", "--db", dbPath, "--type", "batch", "--dir", "testdata", "--recursive=false"})
	if err != nil {
		t.Fatalf("batch import: %v", err)
	}

	matches, _ := cli.db.GetAllMatches()
	if len(matches) == 0 {
		t.Fatal("expected at least 1 match after batch import")
	}
}

// ---------------------------------------------------------------------------
// 10. Edge cases
// ---------------------------------------------------------------------------

func TestCLI_ImportNonexistentFile(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", "testdata/nonexistent.xg"})
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestCLI_ImportCorruptFile(t *testing.T) {
	// Create a temp file with garbage content and a .xg extension.
	tmp := filepath.Join(t.TempDir(), "corrupt.xg")
	if err := os.WriteFile(tmp, []byte("not a valid xg file"), 0644); err != nil {
		t.Fatal(err)
	}

	cli, dbPath := setupCLIWithDB(t)
	err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", tmp})
	if err == nil {
		t.Fatal("expected error importing corrupt file")
	}
}

func TestCLI_ExportNoData(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	exportPath := filepath.Join(t.TempDir(), "empty_export.db")
	// Export from empty DB should not panic.
	err := cli.Run([]string{"export", "--db", dbPath, "--type", "database", "--file", exportPath})
	if err != nil {
		// Some implementations may error on empty export; just verify no panic.
		t.Logf("export empty DB returned: %v (acceptable)", err)
	}
}

func TestCLI_UnknownCommand(t *testing.T) {
	cli := &CLI{db: NewDatabase()}
	err := cli.Run([]string{"bogus"})
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestCLI_MissingRequiredFlags(t *testing.T) {
	cli := &CLI{db: NewDatabase()}

	// import without --db
	err := cli.Run([]string{"import", "--type", "match", "--file", "x.xg"})
	if err == nil {
		t.Error("expected error for import without --db")
	}

	// list without --type
	err = cli.Run([]string{"list", "--db", "x.db"})
	if err == nil {
		t.Error("expected error for list without --type")
	}
}

func TestCLI_Version(t *testing.T) {
	cli := &CLI{db: NewDatabase()}
	out := captureStdout(t, func() {
		cli.Run([]string{"version"})
	})
	if len(out) == 0 {
		t.Error("expected non-empty version output")
	}
}

func TestCLI_Help(t *testing.T) {
	cli := &CLI{db: NewDatabase()}
	out := captureStdout(t, func() {
		cli.Run([]string{"help"})
	})
	if !bytes.Contains([]byte(out), []byte("blunderDB")) {
		t.Errorf("help output missing tool name:\n%s", out)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
