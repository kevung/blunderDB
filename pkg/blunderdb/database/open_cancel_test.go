package database

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
)

// staleLibrary writes a library whose recorded schema predates this build, so
// that opening it would run the migration chain.
func staleLibrary(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stale.db")
	d := NewDatabase()
	if err := d.SetupDatabase(path); err != nil {
		t.Fatal(err)
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	setRecordedVersion(t, path, "2.30.0")
	return path
}

func setRecordedVersion(t *testing.T, path, version string) {
	t.Helper()
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	if _, err := raw.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, version); err != nil {
		t.Fatal(err)
	}
}

func recordedVersion(t *testing.T, path string) string {
	t.Helper()
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	var v string
	if err := raw.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

// A cancel asked once the open has begun but before its migration context
// exists stops the open: the file is neither migrated nor left open.
func TestOpenDatabase_CancelBeforeMigrationIsKept(t *testing.T) {
	path := staleLibrary(t)
	d := NewDatabase()
	t.Cleanup(func() { _ = d.Close() })
	d.SetBeforeSwitch(d.CancelImport)

	if err := d.OpenDatabase(path); !errors.Is(err, context.Canceled) {
		t.Fatalf("OpenDatabase after an early cancel = %v, want context.Canceled", err)
	}
	if got := OpenPath(d); got != "" {
		t.Errorf("a cancelled open left %q open", got)
	}
	if v := recordedVersion(t, path); v != "2.30.0" {
		t.Errorf("schema version after a cancelled open = %s, want 2.30.0 untouched", v)
	}
	d.SetBeforeSwitch(nil)
	if err := d.OpenDatabase(path); err != nil {
		t.Fatalf("reopening after the cancelled open: %v", err)
	}
}

// The caller's context stops an open like CancelImport does.
func TestOpenDatabaseContext_CancelledContext(t *testing.T) {
	path := staleLibrary(t)
	d := NewDatabase()
	t.Cleanup(func() { _ = d.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := OpenDatabaseContext(d, ctx, path); !errors.Is(err, context.Canceled) {
		t.Fatalf("OpenDatabaseContext with a cancelled context = %v, want context.Canceled", err)
	}
	if v := recordedVersion(t, path); v != "2.30.0" {
		t.Errorf("schema version = %s, want 2.30.0 untouched", v)
	}
}
