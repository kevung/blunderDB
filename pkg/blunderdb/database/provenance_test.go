package database

import "testing"

// TestGetPositionProvenance checks that a position reached from an imported
// match reports that match as its provenance, and that a position no match
// references reports none.
func TestGetPositionProvenance(t *testing.T) {
	db := newTestDBWithXG(t)

	// A position referenced by a move in the imported match, and its match.
	var posID, matchID int64
	if err := db.db.QueryRow(`
		SELECT mv.position_id, g.match_id
		FROM move mv JOIN game g ON mv.game_id = g.id
		WHERE mv.position_id IS NOT NULL
		LIMIT 1`).Scan(&posID, &matchID); err != nil {
		t.Fatalf("finding a match position: %v", err)
	}

	prov, err := db.GetPositionProvenance(posID)
	if err != nil {
		t.Fatalf("GetPositionProvenance: %v", err)
	}
	if len(prov) == 0 {
		t.Fatal("expected at least one match in the provenance")
	}
	found := false
	for _, m := range prov {
		if m.ID == matchID {
			found = true
			if m.Player1Name == "" || m.Player2Name == "" {
				t.Errorf("provenance match %d is missing player names", m.ID)
			}
		}
	}
	if !found {
		t.Errorf("provenance did not include the referencing match %d", matchID)
	}

	// A position no match references has empty provenance.
	var maxID int64
	if err := db.db.QueryRow(`SELECT COALESCE(MAX(id), 0) FROM position`).Scan(&maxID); err != nil {
		t.Fatalf("max position id: %v", err)
	}
	empty, err := db.GetPositionProvenance(maxID + 1000)
	if err != nil {
		t.Fatalf("GetPositionProvenance(non-referenced): %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected empty provenance for a non-referenced position, got %d matches", len(empty))
	}
}

// TestGetMatchByID_IncludesTournamentName checks the tournament LEFT JOIN added
// to GetMatchByID: empty before assignment, the tournament name after.
func TestGetMatchByID_IncludesTournamentName(t *testing.T) {
	db := newTestDBWithXG(t)

	var matchID int64
	if err := db.db.QueryRow(`SELECT id FROM match LIMIT 1`).Scan(&matchID); err != nil {
		t.Fatalf("finding a match: %v", err)
	}

	m, err := db.GetMatchByID(matchID)
	if err != nil {
		t.Fatalf("GetMatchByID: %v", err)
	}
	if m.TournamentName != "" {
		t.Errorf("unassigned match should have an empty tournament name, got %q", m.TournamentName)
	}

	if err := db.SetMatchTournamentByName(matchID, "Provenance Cup"); err != nil {
		t.Fatalf("SetMatchTournamentByName: %v", err)
	}

	m2, err := db.GetMatchByID(matchID)
	if err != nil {
		t.Fatalf("GetMatchByID (after assignment): %v", err)
	}
	if m2.TournamentName != "Provenance Cup" {
		t.Errorf("expected tournament name %q, got %q", "Provenance Cup", m2.TournamentName)
	}
	if m2.TournamentID == nil {
		t.Error("expected tournament_id to be set after assignment")
	}
}
