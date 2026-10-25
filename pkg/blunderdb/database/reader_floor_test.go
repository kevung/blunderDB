package database

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestOpenRefusesADatabaseAboveItsReaderFloor: the desktop and CLI open path
// refuses a file whose floor is newer than the schema this binary speaks,
// with a message naming both versions, and leaves the file byte for byte as
// it was; a file that predates the marker gets it on its first open.
func TestOpenRefusesADatabaseAboveItsReaderFloor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plus-recente.db")
	d := NewDatabase()
	if err := d.SetupDatabase(path); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.readMeta(storage.MinReaderVersionKey); got != domain.MinReaderVersion {
		t.Fatalf("new database floor = %q, want %q", got, domain.MinReaderVersion)
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}

	setFloor := func(stmt string) {
		t.Helper()
		raw, err := sql.Open("sqlite", path)
		if err != nil {
			t.Fatal(err)
		}
		defer raw.Close()
		if _, err := raw.Exec(stmt); err != nil {
			t.Fatal(err)
		}
		if _, err := raw.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
			t.Fatal(err)
		}
	}

	setFloor(`UPDATE metadata SET value = '99.0.0' WHERE key = 'min_reader_version'`)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	opened := NewDatabase()
	err = opened.OpenDatabase(path)
	if !errors.Is(err, storage.ErrDatabaseTooNew) {
		t.Fatalf("OpenDatabase under floor 99.0.0: err = %v, want ErrDatabaseTooNew", err)
	}
	if msg := err.Error(); !strings.Contains(msg, "99.0.0") || !strings.Contains(msg, DatabaseVersion) {
		t.Errorf("refusal %q names neither the floor nor this binary's schema", msg)
	}
	opened.Close()
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("a refused open changed the file")
	}

	setFloor(`DELETE FROM metadata WHERE key = 'min_reader_version'`)
	reopened := NewDatabase()
	if err := reopened.OpenDatabase(path); err != nil {
		t.Fatalf("OpenDatabase without a floor: %v", err)
	}
	defer reopened.Close()
	if got, _ := reopened.readMeta(storage.MinReaderVersionKey); got != domain.MinReaderVersion {
		t.Errorf("floor after opening a file without one = %q, want %q", got, domain.MinReaderVersion)
	}
}

// TestImportRefusesASourceAboveItsReaderFloor: a .db source a newer blunderDB
// marked above this binary's schema is refused by the preview and by the
// import itself, and the target receives nothing.
func TestImportRefusesASourceAboveItsReaderFloor(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.db")
	s := NewDatabase()
	if err := s.SetupDatabase(src); err != nil {
		t.Fatal(err)
	}
	p := domain.InitializePosition()
	if _, err := s.SavePosition(&p); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := sql.Open("sqlite", src)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`UPDATE metadata SET value = '99.0.0' WHERE key = 'min_reader_version'`); err != nil {
		t.Fatal(err)
	}
	raw.Close()

	target := NewDatabase()
	if err := target.SetupDatabase(filepath.Join(dir, "target.db")); err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	if _, err := target.AnalyzeImportDatabase(src); !errors.Is(err, storage.ErrDatabaseTooNew) {
		t.Errorf("AnalyzeImportDatabase: err = %v, want ErrDatabaseTooNew", err)
	}
	if _, err := target.CommitImportDatabase(src); !errors.Is(err, storage.ErrDatabaseTooNew) {
		t.Errorf("CommitImportDatabase: err = %v, want ErrDatabaseTooNew", err)
	}
	var n int
	if err := target.db.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("target holds %d positions after a refused import, want 0", n)
	}
}
