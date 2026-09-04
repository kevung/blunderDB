package database

// search_player_test.go — regression tests for the `pl"Name"` player filter,
// which keeps positions occurring in any match where the named player sat at
// either seat (SearchFilters.PlayerFilter, resolved by the storage search via a
// position→move→game→match subquery).

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchPlayerFilter(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	db := NewDatabase()
	if err := db.SetupDatabase(filepath.Join(dir, "search_player.db")); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer db.Close()

	importTestMatch(t, db)

	matches, err := db.GetAllMatches()
	if err != nil {
		t.Fatalf("GetAllMatches: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("no match imported")
	}
	p1 := matches[0].Player1Name
	p2 := matches[0].Player2Name
	if p1 == "" {
		t.Skip("imported match carries no player1 name")
	}

	all, err := db.LoadPositionsByFilters(SearchFilters{})
	if err != nil {
		t.Fatalf("LoadPositionsByFilters(all): %v", err)
	}
	if len(all) == 0 {
		t.Fatal("no positions imported")
	}

	// The match's player (player 1) selects the match's positions.
	got, err := db.LoadPositionsByFilters(SearchFilters{PlayerFilter: p1})
	if err != nil {
		t.Fatalf("LoadPositionsByFilters(player1): %v", err)
	}
	if len(got) == 0 {
		t.Errorf("PlayerFilter=%q returned no positions", p1)
	}

	// Either seat matches: player 2 selects the same single-match positions.
	if p2 != "" && p2 != p1 {
		got2, err := db.LoadPositionsByFilters(SearchFilters{PlayerFilter: p2})
		if err != nil {
			t.Fatalf("LoadPositionsByFilters(player2): %v", err)
		}
		if len(got2) == 0 {
			t.Errorf("PlayerFilter=%q (opponent seat) returned no positions", p2)
		}
	}

	// Matching is case-insensitive.
	lower, err := db.LoadPositionsByFilters(SearchFilters{PlayerFilter: strings.ToLower(p1)})
	if err != nil {
		t.Fatalf("LoadPositionsByFilters(player1 lowercased): %v", err)
	}
	if len(lower) != len(got) {
		t.Errorf("case-insensitive mismatch: %q=%d positions, lowercased=%d", p1, len(got), len(lower))
	}

	// A name in no match returns nothing (the filter is exclusive, not a no-op).
	none, err := db.LoadPositionsByFilters(SearchFilters{PlayerFilter: "ZZ_NoSuchPlayer_ZZ"})
	if err != nil {
		t.Fatalf("LoadPositionsByFilters(unknown): %v", err)
	}
	if len(none) != 0 {
		t.Errorf("unknown player returned %d positions, want 0", len(none))
	}
}

// TestSearchPlayerFilterQuotedToken pins what the frontend actually sends: the
// command bar and SearchPanel both hand `pl"Name"` — the token, quotes included
// — straight to LoadPositionIDsByFilters, where the storage search compares it
// to match.player1_name. If nothing strips the wrapper, searching by player
// from the GUI silently returns nothing.
func TestSearchPlayerFilterQuotedToken(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	db := NewDatabase()
	if err := db.SetupDatabase(filepath.Join(dir, "search_player_quoted.db")); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer db.Close()

	importTestMatch(t, db)
	matches, err := db.GetAllMatches()
	if err != nil || len(matches) == 0 {
		t.Fatalf("GetAllMatches: %v (%d matches)", err, len(matches))
	}
	p1 := matches[0].Player1Name
	if p1 == "" {
		t.Skip("imported match carries no player1 name")
	}

	bare, err := db.LoadPositionsByFilters(SearchFilters{PlayerFilter: p1})
	if err != nil {
		t.Fatalf("bare: %v", err)
	}
	quoted, err := db.LoadPositionsByFilters(SearchFilters{PlayerFilter: `pl"` + p1 + `"`})
	if err != nil {
		t.Fatalf("quoted: %v", err)
	}
	if len(quoted) != len(bare) {
		t.Errorf("pl%q returned %d positions, bare %q returned %d: the quoted token is not unwrapped", p1, len(quoted), p1, len(bare))
	}
}
