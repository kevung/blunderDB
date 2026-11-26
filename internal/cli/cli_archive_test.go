package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCLI_ArchiveExportImport archives a library to a page, then imports it
// into another through `archive import` and through the ordinary `import`.
func TestCLI_ArchiveExportImport(t *testing.T) {
	t.Parallel()
	cli, dbPath := setupCLIWithDB(t)
	if err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("test.mat")}); err != nil {
		t.Fatal(err)
	}
	page := filepath.Join(t.TempDir(), "archive.html")
	out := captureOut(t, cli, func() {
		if err := cli.Run([]string{"archive", "export", "--db", dbPath, "--output", page, "--format", "json"}); err != nil {
			t.Error(err)
		}
	})
	var exp struct{ Matches, Codes int }
	if err := json.Unmarshal([]byte(out), &exp); err != nil || exp.Matches != 1 || exp.Codes != 2 {
		t.Fatalf("export: %+v (%v) in %s", exp, err, out)
	}

	other, otherDB := setupCLIWithDB(t)
	out = captureOut(t, other, func() {
		if err := other.Run([]string{"archive", "import", "--db", otherDB, "--format", "json", page}); err != nil {
			t.Error(err)
		}
	})
	var imp struct {
		Imported   []int64
		Duplicates int
	}
	if err := json.Unmarshal([]byte(out), &imp); err != nil || len(imp.Imported) != 1 {
		t.Fatalf("archive import: %+v (%v) in %s", imp, err, out)
	}
	out = captureOut(t, other, func() {
		if err := other.Run([]string{"import", "--db", otherDB, "--type", "match", "--file", page, "--format", "json"}); err != nil {
			t.Error(err)
		}
	})
	if err := json.Unmarshal([]byte(out), &imp); err != nil || len(imp.Imported) != 0 || imp.Duplicates != 1 {
		t.Fatalf("import of the page again: %+v (%v) in %s", imp, err, out)
	}
}

// A batch over a folder where matches sit beside an archive page and an HTML
// file of another program: the archive is imported, the other page passed
// over, and --fail-on-error has nothing to fail on.
func TestCLI_BatchTakesArchivePagesAndPassesOverOtherHTML(t *testing.T) {
	t.Parallel()
	src, srcDB := setupCLIWithDB(t)
	if err := src.Run([]string{"import", "--db", srcDB, "--type", "match", "--file", testdataPath("test.mat")}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := src.Run([]string{"archive", "export", "--db", srcDB, "--output", filepath.Join(dir, "archive.html")}); err != nil {
		t.Fatal(err)
	}
	mat, err := os.ReadFile(testdataPath("charlot1-charlot2_7p_2025-11-08-2305.mat"))
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{
		"charlot.mat":  mat,
		"charlot.html": []byte("<!doctype html><title>gnubg export</title><p>Game 1</p>"),
		"notes.htm":    []byte("<html><body>notes</body></html>"),
	} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	dst, dstDB := setupCLIWithDB(t)
	out := captureOut(t, dst, func() {
		if err := dst.Run([]string{"import", "--db", dstDB, "--type", "batch", "--dir", dir, "--fail-on-error"}); err != nil {
			t.Error(err)
		}
	})
	if strings.Contains(out, "charlot.html") || strings.Contains(out, "notes.htm") || strings.Contains(out, "ERROR") {
		t.Fatalf("the foreign HTML files reached the import:\n%s", out)
	}
	if ms, err := dst.db.GetAllMatches(); err != nil || len(ms) != 2 {
		t.Fatalf("%d matches (%v), want the .mat and the archived one:\n%s", len(ms), err, out)
	}

	// Foreign pages alone are no match file at all, as a folder of PDFs.
	alone := t.TempDir()
	if err := os.WriteFile(filepath.Join(alone, "page.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := dst.Run([]string{"import", "--db", dstDB, "--type", "batch", "--dir", alone}); err == nil || !strings.Contains(err.Error(), "no match files found") {
		t.Fatalf("a folder of foreign pages: %v", err)
	}
}
