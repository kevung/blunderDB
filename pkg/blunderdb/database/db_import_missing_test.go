package database

import (
	"os"
	"path/filepath"
	"testing"
)

// TestImportDatabase_MissingFileIsNotCreated: sql.Open on a path that does
// not exist creates an empty SQLite file there. Analysing or committing an
// import from a mistyped path used to leave that 0-byte .db on the user's
// disk and then fail on the missing metadata table. Both entry points, and
// the issuance inspector that opens a file the same way, refuse first.
func TestImportDatabase_MissingFileIsNotCreated(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cur := NewDatabase()
	if err := cur.SetupDatabase(filepath.Join(dir, "current.db")); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer cur.Close()

	missing := filepath.Join(dir, "typo.db")
	notCreated := func(step string) {
		t.Helper()
		if _, err := os.Stat(missing); !os.IsNotExist(err) {
			t.Fatalf("%s created %s (stat: %v)", step, missing, err)
		}
	}

	if _, err := cur.AnalyzeImportDatabase(missing); err == nil {
		t.Error("AnalyzeImportDatabase on a missing file succeeded")
	}
	notCreated("AnalyzeImportDatabase")

	if _, err := cur.CommitImportDatabase(missing); err == nil {
		t.Error("CommitImportDatabase on a missing file succeeded")
	}
	notCreated("CommitImportDatabase")

	if _, err := InspectIssuance(missing); err == nil {
		t.Error("InspectIssuance on a missing file succeeded")
	}
	notCreated("InspectIssuance")

	if _, err := cur.AnalyzeImportDatabase(dir); err == nil {
		t.Error("AnalyzeImportDatabase on a directory succeeded")
	}
}
