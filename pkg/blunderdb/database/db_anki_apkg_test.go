package database

import (
	"os"
	"path/filepath"
	"testing"
)

// A failed export leaves neither the .apkg nor its temporary file behind, and
// keeps a file already at the path.
func TestExportAnkiPackageLeavesNothingOnFailure(t *testing.T) {
	d := newTestDB(t)
	dir := t.TempDir()
	out := filepath.Join(dir, "deck.apkg")
	if _, err := d.ExportAnkiPackage(999, 0, "fr", out); err == nil {
		t.Fatal("unknown deck: want an error")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("left behind: %v", entries)
	}
	if err := os.WriteFile(out, []byte("previous"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := d.ExportAnkiPackage(999, 0, "fr", out); err == nil {
		t.Fatal("unknown deck: want an error")
	}
	if b, _ := os.ReadFile(out); string(b) != "previous" {
		t.Errorf("the previous file was overwritten: %q", b)
	}
}
