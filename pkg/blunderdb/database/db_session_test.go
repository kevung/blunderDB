package database

import (
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The session/history/filter adapters hand the backend's errors through
// unchanged, so a caller can tell a duplicate name from an unknown filter with
// errors.Is rather than by matching message text.
func TestFilterLibraryErrorsAreStorageSentinels(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

	if err := db.SaveFilter("blunders", "s e>0.1"); err != nil {
		t.Fatalf("SaveFilter: %v", err)
	}
	if err := db.SaveFilter("blunders", "s e>0.2"); !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("duplicate name: got %v, want ErrConflict", err)
	}
	if err := db.UpdateFilter(9999, "x", "y"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("UpdateFilter(unknown id): got %v, want ErrNotFound", err)
	}
	if err := db.DeleteFilter(9999); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("DeleteFilter(unknown id): got %v, want ErrNotFound", err)
	}
	if err := db.SaveEditPosition("nope", "{}"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("SaveEditPosition(unknown name): got %v, want ErrNotFound", err)
	}
	if err := db.SaveExcludePosition("nope", "{}"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("SaveExcludePosition(unknown name): got %v, want ErrNotFound", err)
	}

	// A filter saved without an edit position carries a NULL column: that is
	// "no edit position", not a scan failure.
	if got, err := db.LoadEditPosition("blunders"); err != nil || got != "" {
		t.Fatalf("LoadEditPosition(NULL): got %q err %v, want empty", got, err)
	}
	if got, err := db.LoadExcludePosition("blunders"); err != nil || got != "" {
		t.Fatalf("LoadExcludePosition(NULL): got %q err %v, want empty", got, err)
	}
}

// Every method of the family refuses cleanly before a database is open instead
// of dereferencing a nil handle.
func TestSessionFamilyRefusesWhenNotOpened(t *testing.T) {
	t.Parallel()
	db := NewDatabase()
	calls := map[string]func() error{
		"SaveCommand":              func() error { return db.SaveCommand("s") },
		"LoadCommandHistory":       func() error { _, err := db.LoadCommandHistory(); return err },
		"ClearCommandHistory":      db.ClearCommandHistory,
		"SaveSearchHistory":        func() error { return db.SaveSearchHistory("s", "", "") },
		"LoadSearchHistory":        func() error { _, err := db.LoadSearchHistory(); return err },
		"DeleteSearchHistoryEntry": func() error { return db.DeleteSearchHistoryEntry(1) },
		"SaveSessionState":         func() error { return db.SaveSessionState(SessionState{}) },
		"LoadSessionState":         func() error { _, err := db.LoadSessionState(); return err },
		"ClearSessionState":        db.ClearSessionState,
		"SaveFilter":               func() error { return db.SaveFilter("f", "s") },
		"UpdateFilter":             func() error { return db.UpdateFilter(1, "f", "s") },
		"DeleteFilter":             func() error { return db.DeleteFilter(1) },
		"LoadFilters":              func() error { _, err := db.LoadFilters(); return err },
		"SaveEditPosition":         func() error { return db.SaveEditPosition("f", "{}") },
		"LoadEditPosition":         func() error { _, err := db.LoadEditPosition("f"); return err },
		"SaveExcludePosition":      func() error { return db.SaveExcludePosition("f", "{}") },
		"LoadExcludePosition":      func() error { _, err := db.LoadExcludePosition("f"); return err },
	}
	for name, call := range calls {
		if err := call(); !errors.Is(err, errNotOpened) {
			t.Errorf("%s on a closed wrapper: got %v, want errNotOpened", name, err)
		}
	}
}
