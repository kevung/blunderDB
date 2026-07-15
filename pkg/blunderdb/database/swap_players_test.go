package database

import "testing"

// TestSwapMatchPlayersDelegates checks the legacy Desktop path swaps a match's
// players by delegating to the storage layer (whose SwapPlayers does the #107
// copy-on-write). Importing a real .mat, swapping, and reading back must show
// the names exchanged with no error.
func TestSwapMatchPlayersDelegates(t *testing.T) {
	db := newTestDB(t)

	matchID, err := db.ImportGnuBGMatch("testdata/test.mat")
	if err != nil {
		t.Fatalf("import test.mat: %v", err)
	}

	before := matchByID(t, db, matchID)
	p1, p2 := before.Player1Name, before.Player2Name
	if p1 == "" || p2 == "" || p1 == p2 {
		t.Fatalf("fixture needs two distinct named players, got %q / %q", p1, p2)
	}

	if err := db.SwapMatchPlayers(matchID); err != nil {
		t.Fatalf("SwapMatchPlayers: %v", err)
	}

	after := matchByID(t, db, matchID)
	if after.Player1Name != p2 || after.Player2Name != p1 {
		t.Errorf("names not swapped: got %q / %q, want %q / %q",
			after.Player1Name, after.Player2Name, p2, p1)
	}
}

func matchByID(t *testing.T, db *Database, id int64) Match {
	t.Helper()
	matches, err := db.GetAllMatches()
	if err != nil {
		t.Fatalf("GetAllMatches: %v", err)
	}
	for _, m := range matches {
		if m.ID == id {
			return m
		}
	}
	t.Fatalf("match %d not found", id)
	return Match{}
}
