package database

import (
	"path/filepath"
	"testing"
)

// The GUI selects positions by identifier so it does not have to serialise the whole set
// across the IPC bridge. Both routes must produce the same file, in the same order —
// otherwise the cheap path would quietly become a different feature.
func TestExportByIDsMatchesExportByPositions(t *testing.T) {
	isolateIdentity(t)
	source := newTestDBWithXG(t)

	positions, err := source.LoadAllPositions()
	if err != nil {
		t.Fatalf("LoadAllPositions: %v", err)
	}
	if len(positions) < 3 {
		t.Fatalf("fixture too small to be meaningful: %d positions", len(positions))
	}

	ids := make([]int64, len(positions))
	for i, p := range positions {
		ids[i] = p.ID
	}

	dir := t.TempDir()
	byValue := filepath.Join(dir, "by-value.db")
	byID := filepath.Join(dir, "by-id.db")

	if err := source.ExportDatabase(ExportOptions{ExportPath: byValue, Positions: positions, Metadata: map[string]string{}}); err != nil {
		t.Fatalf("ExportDatabase (positions): %v", err)
	}
	if err := source.ExportDatabase(ExportOptions{ExportPath: byID, PositionIDs: ids, Metadata: map[string]string{}}); err != nil {
		t.Fatalf("ExportDatabase (identifiers): %v", err)
	}

	expected := readExportedStates(t, byValue)
	got := readExportedStates(t, byID)
	if len(got) != len(expected) {
		t.Fatalf("exported %d positions by id, %d by value", len(got), len(expected))
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("position %d differs between the two routes:\n by value: %s\n by id   : %s", i, expected[i], got[i])
		}
	}
}

// The order the caller asked for is the order written, whatever order the database returns
// the rows in.
func TestExportByIDsPreservesTheRequestedOrder(t *testing.T) {
	isolateIdentity(t)
	source := newTestDBWithXG(t)

	positions, err := source.LoadAllPositions()
	if err != nil {
		t.Fatalf("LoadAllPositions: %v", err)
	}
	reversed := make([]int64, len(positions))
	for i, p := range positions {
		reversed[len(positions)-1-i] = p.ID
	}

	path := filepath.Join(t.TempDir(), "reversed.db")
	if err := source.ExportDatabase(ExportOptions{ExportPath: path, PositionIDs: reversed, Metadata: map[string]string{}}); err != nil {
		t.Fatalf("ExportDatabase: %v", err)
	}

	forward := filepath.Join(t.TempDir(), "forward.db")
	if err := source.ExportDatabase(ExportOptions{ExportPath: forward, Positions: positions, Metadata: map[string]string{}}); err != nil {
		t.Fatalf("ExportDatabase: %v", err)
	}

	got := readExportedStates(t, path)
	expected := readExportedStates(t, forward)
	for i := range expected {
		if got[len(got)-1-i] != expected[i] {
			t.Fatalf("the requested order was not preserved at index %d", i)
		}
	}
}

// An identifier that no longer exists must not cost the rest of the export.
func TestExportByIDsSkipsUnknownIdentifiers(t *testing.T) {
	isolateIdentity(t)
	source := newTestDBWithXG(t)

	positions, err := source.LoadAllPositions()
	if err != nil {
		t.Fatalf("LoadAllPositions: %v", err)
	}
	ids := []int64{positions[0].ID, 999999, positions[1].ID}

	path := filepath.Join(t.TempDir(), "partial.db")
	if err := source.ExportDatabase(ExportOptions{ExportPath: path, PositionIDs: ids, Metadata: map[string]string{}}); err != nil {
		t.Fatalf("ExportDatabase: %v", err)
	}
	if got := readExportedStates(t, path); len(got) != 2 {
		t.Fatalf("expected the two known positions, got %d", len(got))
	}
}

func readExportedStates(t *testing.T, path string) []string {
	t.Helper()
	db := NewDatabase()
	if err := db.OpenDatabase(path); err != nil {
		t.Fatalf("OpenDatabase(%s): %v", path, err)
	}
	defer db.Close()

	rows, err := db.db.Query(`SELECT state FROM position ORDER BY id`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()

	var states []string
	for rows.Next() {
		var state string
		if err := rows.Scan(&state); err != nil {
			t.Fatalf("scan: %v", err)
		}
		states = append(states, state)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return states
}
