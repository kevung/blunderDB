package database

import "testing"

// TestSearchFilterIndexesExist guards the indexes backing search range filters
// that previously fell back to full table scans (Player1 absolute pip count,
// back checkers, no-contact, opponent win/gammon/backgammon rates, player1
// backgammon rate). They are results-neutral; they only let the planner use an
// index for selective filters instead of scanning.
func TestSearchFilterIndexesExist(t *testing.T) {
	db := newTestDB(t)
	want := []string{
		"idx_position_back_checkers_1",
		"idx_position_back_checkers_2",
		"idx_position_pip_1",
		"idx_position_no_contact",
		"idx_analysis_backgammon1",
		"idx_analysis_win2",
		"idx_analysis_gammon2",
		"idx_analysis_backgammon2",
	}
	for _, idx := range want {
		var name string
		if err := db.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='index' AND name=?`, idx).Scan(&name); err != nil {
			t.Errorf("missing index %s: %v", idx, err)
		}
	}
}
