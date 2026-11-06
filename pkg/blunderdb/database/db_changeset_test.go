package database

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// A journal taken on one file refuses to read or write the one opened after it.
func TestJournalRefusesALibrarySwitchedUnderIt(t *testing.T) {
	d := newTestDB(t)
	j := JournalOf(d)
	if _, err := j.Export(context.Background(), storage.Cursor{}, 10); err != nil {
		t.Fatal(err)
	}
	if err := d.SetupDatabase(filepath.Join(tempDir(t), "other.db")); err != nil {
		t.Fatal(err)
	}
	closeOnCleanup(t, d)
	if _, err := j.Export(context.Background(), storage.Cursor{}, 10); !errors.Is(err, ErrDatabaseChanged) {
		t.Errorf("export after a switch: %v, want ErrDatabaseChanged", err)
	}
	if _, err := j.Import(context.Background(), changeset.Set{}); !errors.Is(err, ErrDatabaseChanged) {
		t.Errorf("import after a switch: %v, want ErrDatabaseChanged", err)
	}
}
