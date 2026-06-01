package database

// Desktop single-user regression guard. Exercises the exact flow the Wails GUI
// uses — create a database with SetupDatabase, write through the GUI-bound
// wrapper methods, reopen with OpenDatabase, and read everything back — against
// the current schema (2.9.0, which added the scope column to command_history /
// search_history / filter_library). The empty scope used by the GUI must round
// trip unchanged.

import (
	"path/filepath"
	"testing"
)

func TestDesktopWrapperRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "desktop.db")

	d := NewDatabase()
	if err := d.SetupDatabase(path); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}

	// Write through the GUI-bound methods.
	if err := d.SaveCommand("position 1"); err != nil {
		t.Fatalf("SaveCommand: %v", err)
	}
	if err := d.SaveCommand("search foo"); err != nil {
		t.Fatalf("SaveCommand: %v", err)
	}
	if err := d.SaveFilter("my filter", "search foo"); err != nil {
		t.Fatalf("SaveFilter: %v", err)
	}
	if err := d.SaveEditPosition("my filter", `{"edit":1}`); err != nil {
		t.Fatalf("SaveEditPosition: %v", err)
	}
	if err := d.SaveExcludePosition("my filter", `{"excl":1}`); err != nil {
		t.Fatalf("SaveExcludePosition: %v", err)
	}
	if err := d.SaveSearchHistory("search foo", `{"p":1}`, `{"x":1}`); err != nil {
		t.Fatalf("SaveSearchHistory: %v", err)
	}
	if err := d.SaveSessionState(SessionState{
		LastSearchCommand: "search foo",
		LastPositionIndex: 2,
		LastPositionIDs:   []int64{3, 4},
		HasActiveSearch:   true,
		ViewsJSON:         "[]",
	}); err != nil {
		t.Fatalf("SaveSessionState: %v", err)
	}
	if err := d.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Reopen as the GUI would (OpenDatabase runs the migration chain; here it is
	// already current, so it only verifies + repairs).
	d2 := NewDatabase()
	if err := d2.OpenDatabase(path); err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	defer d2.Close()

	if v, err := d2.CheckDatabaseVersion(); err != nil || v != DatabaseVersion {
		t.Errorf("version: got %q err=%v, want %q", v, err, DatabaseVersion)
	}

	cmds, err := d2.LoadCommandHistory()
	if err != nil {
		t.Fatalf("LoadCommandHistory: %v", err)
	}
	if len(cmds) != 2 || cmds[0] != "position 1" || cmds[1] != "search foo" {
		t.Errorf("command history: got %v, want [position 1, search foo]", cmds)
	}

	filters, err := d2.LoadFilters()
	if err != nil {
		t.Fatalf("LoadFilters: %v", err)
	}
	if len(filters) != 1 || filters[0]["name"] != "my filter" || filters[0]["command"] != "search foo" {
		t.Errorf("filters: got %v, want one named 'my filter'", filters)
	}

	if ep, err := d2.LoadEditPosition("my filter"); err != nil || ep != `{"edit":1}` {
		t.Errorf("edit position: got %q err=%v, want {\"edit\":1}", ep, err)
	}
	if xp, err := d2.LoadExcludePosition("my filter"); err != nil || xp != `{"excl":1}` {
		t.Errorf("exclude position: got %q err=%v, want {\"excl\":1}", xp, err)
	}

	sh, err := d2.LoadSearchHistory()
	if err != nil {
		t.Fatalf("LoadSearchHistory: %v", err)
	}
	if len(sh) != 1 || sh[0].Command != "search foo" || sh[0].Position != `{"p":1}` {
		t.Errorf("search history: got %+v, want one entry for 'search foo'", sh)
	}

	got, err := d2.LoadSessionState()
	if err != nil {
		t.Fatalf("LoadSessionState: %v", err)
	}
	if got.LastSearchCommand != "search foo" || got.LastPositionIndex != 2 ||
		len(got.LastPositionIDs) != 2 || !got.HasActiveSearch {
		t.Errorf("session state: got %+v", got)
	}
}
