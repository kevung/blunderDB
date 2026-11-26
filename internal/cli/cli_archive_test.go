package cli

import (
	"encoding/json"
	"path/filepath"
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
