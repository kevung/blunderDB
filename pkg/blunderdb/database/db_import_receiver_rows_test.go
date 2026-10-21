package database

import (
	"path/filepath"
	"testing"
)

// A receiver row the contract cannot read stops the import instead of being
// left out of the identity index: an incoming copy of that position would
// otherwise be matched by hash alone, or not at all. The import rolls back and
// leaves the receiver as it was.
func TestImportStopsOnUnreadableReceiverRow(t *testing.T) {
	src := newTestDB(t)
	pos := initialPosition()
	if _, err := src.SavePosition(&pos); err != nil {
		t.Fatal(err)
	}
	path := exportTo(t, src, filepath.Join(t.TempDir(), "partage.db"), ExportOptions{AllPositions: true})

	dst := newTestDB(t)
	other := initialPosition()
	other.Dice = [2]int{3, 1}
	if _, err := dst.SavePosition(&other); err != nil {
		t.Fatal(err)
	}
	if _, err := dst.db.Exec(`UPDATE position SET decision_type = 'x'`); err != nil {
		t.Fatal(err)
	}
	if _, err := dst.AnalyzeImportDatabase(path); err == nil {
		t.Error("AnalyzeImportDatabase: no error on an unreadable receiver row")
	}
	if _, err := dst.CommitImportDatabase(path); err == nil {
		t.Error("CommitImportDatabase: no error on an unreadable receiver row")
	}
	var n int
	if err := dst.db.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("receiver holds %d positions after the refused import, want 1", n)
	}
}

// A refused import releases its transaction: the version check runs once the
// transaction is open, and a refusal there must not leave a connection held.
func TestCommitImportDatabaseReleasesTxOnVersionRefusal(t *testing.T) {
	src := newTestDB(t)
	pos := initialPosition()
	if _, err := src.SavePosition(&pos); err != nil {
		t.Fatal(err)
	}
	path := exportTo(t, src, filepath.Join(t.TempDir(), "future.db"), ExportOptions{AllPositions: true})
	raw, err := openExistingSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`UPDATE metadata SET value = '999.0.0' WHERE key = 'database_version'`); err != nil {
		t.Fatal(err)
	}
	raw.Close()

	dst := newTestDB(t)
	before := dst.db.Stats().InUse
	if _, err := dst.CommitImportDatabase(path); err == nil {
		t.Fatal("CommitImportDatabase accepted a newer major version")
	}
	if after := dst.db.Stats().InUse; after != before {
		t.Errorf("connections in use: %d before, %d after the refusal", before, after)
	}
}
