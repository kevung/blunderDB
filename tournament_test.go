package main

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTournamentTestDB(t *testing.T) (*Database, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db := NewDatabase()
	if err := db.SetupDatabase(dbPath); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}

	return db, func() {
		if db.db != nil {
			db.db.Close()
		}
		os.Remove(dbPath)
	}
}

func TestCreateTournament(t *testing.T) {
	db, cleanup := setupTournamentTestDB(t)
	defer cleanup()

	id, err := db.CreateTournament("Grand Prix", "2025-01-15", "Paris")
	if err != nil {
		t.Fatalf("CreateTournament: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive ID, got %d", id)
	}
}

func TestGetAllTournaments(t *testing.T) {
	db, cleanup := setupTournamentTestDB(t)
	defer cleanup()

	for i := 0; i < 3; i++ {
		if _, err := db.CreateTournament("T", "", ""); err != nil {
			t.Fatalf("CreateTournament: %v", err)
		}
	}

	tournaments, err := db.GetAllTournaments()
	if err != nil {
		t.Fatalf("GetAllTournaments: %v", err)
	}
	if len(tournaments) != 3 {
		t.Fatalf("got %d tournaments, want 3", len(tournaments))
	}
	for _, tt := range tournaments {
		if tt.MatchCount != 0 {
			t.Errorf("tournament %d: matchCount = %d, want 0", tt.ID, tt.MatchCount)
		}
	}
}

func TestUpdateTournament(t *testing.T) {
	db, cleanup := setupTournamentTestDB(t)
	defer cleanup()

	id, _ := db.CreateTournament("Old", "2025-01-01", "Old City")
	if err := db.UpdateTournament(id, "New", "2025-06-15", "New City"); err != nil {
		t.Fatalf("UpdateTournament: %v", err)
	}

	tournaments, _ := db.GetAllTournaments()
	var found *Tournament
	for i := range tournaments {
		if tournaments[i].ID == id {
			found = &tournaments[i]
			break
		}
	}
	if found == nil {
		t.Fatal("tournament not found")
	}
	if found.Name != "New" || found.Location != "New City" {
		t.Errorf("got name=%q location=%q", found.Name, found.Location)
	}
}

func TestDeleteTournament(t *testing.T) {
	db, cleanup := setupTournamentTestDB(t)
	defer cleanup()

	importTestMatch(t, db)
	matches, err := db.GetAllMatches()
	if err != nil || len(matches) == 0 {
		t.Fatalf("no matches after import: %v", err)
	}
	matchID := matches[0].ID

	tID, _ := db.CreateTournament("ToDelete", "", "")
	_ = db.AddMatchToTournament(tID, matchID)

	if err := db.DeleteTournament(tID); err != nil {
		t.Fatalf("DeleteTournament: %v", err)
	}

	// Match should still exist (unlinked)
	matches2, _ := db.GetAllMatches()
	if len(matches2) == 0 {
		t.Error("match was deleted when tournament was deleted")
	}
}

func TestAddMatchToTournament(t *testing.T) {
	db, cleanup := setupTournamentTestDB(t)
	defer cleanup()

	importTestMatch(t, db)
	matches, _ := db.GetAllMatches()
	matchID := matches[0].ID

	tID, _ := db.CreateTournament("T", "", "")
	if err := db.AddMatchToTournament(tID, matchID); err != nil {
		t.Fatalf("AddMatchToTournament: %v", err)
	}

	tMatches, err := db.GetTournamentMatches(tID)
	if err != nil {
		t.Fatalf("GetTournamentMatches: %v", err)
	}
	if len(tMatches) != 1 {
		t.Fatalf("got %d matches, want 1", len(tMatches))
	}
}

func TestRemoveMatchFromTournament(t *testing.T) {
	db, cleanup := setupTournamentTestDB(t)
	defer cleanup()

	importTestMatch(t, db)
	matches, _ := db.GetAllMatches()
	matchID := matches[0].ID

	tID, _ := db.CreateTournament("T", "", "")
	_ = db.AddMatchToTournament(tID, matchID)

	if err := db.RemoveMatchFromTournament(matchID); err != nil {
		t.Fatalf("RemoveMatchFromTournament: %v", err)
	}

	tMatches, _ := db.GetTournamentMatches(tID)
	if len(tMatches) != 0 {
		t.Errorf("got %d matches after removal, want 0", len(tMatches))
	}

	// Match should still exist
	m2, _ := db.GetAllMatches()
	if len(m2) == 0 {
		t.Error("match was deleted instead of unlinked")
	}
}

func TestSetMatchTournamentByName(t *testing.T) {
	db, cleanup := setupTournamentTestDB(t)
	defer cleanup()

	importTestMatch(t, db)
	matches, _ := db.GetAllMatches()
	matchID := matches[0].ID

	// Should create a new tournament
	if err := db.SetMatchTournamentByName(matchID, "Auto Created"); err != nil {
		t.Fatalf("SetMatchTournamentByName: %v", err)
	}

	tt, err := db.GetMatchTournament(matchID)
	if err != nil {
		t.Fatalf("GetMatchTournament: %v", err)
	}
	if tt == nil || tt.Name != "Auto Created" {
		t.Errorf("tournament = %+v, want name 'Auto Created'", tt)
	}
}

func TestSetMatchTournamentByName_Existing(t *testing.T) {
	db, cleanup := setupTournamentTestDB(t)
	defer cleanup()

	importTestMatch(t, db)
	matches, _ := db.GetAllMatches()
	matchID := matches[0].ID

	db.CreateTournament("Existing", "", "")
	if err := db.SetMatchTournamentByName(matchID, "Existing"); err != nil {
		t.Fatalf("SetMatchTournamentByName: %v", err)
	}

	tournaments, _ := db.GetAllTournaments()
	count := 0
	for _, tt := range tournaments {
		if tt.Name == "Existing" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("found %d tournaments named 'Existing', want 1", count)
	}
}

func TestReorderTournamentMatches(t *testing.T) {
	db, cleanup := setupTournamentTestDB(t)
	defer cleanup()

	// Import two different matches
	m1, err := db.ImportGnuBGMatch(filepath.Join("testdata", "test.sgf"))
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	m2, err := db.ImportXGMatch(filepath.Join("testdata", "test.xg"))
	if err != nil {
		t.Fatalf("second import: %v", err)
	}

	tID, _ := db.CreateTournament("T", "", "")
	_ = db.AddMatchToTournament(tID, m1)
	_ = db.AddMatchToTournament(tID, m2)

	// Reorder: second match first
	if err := db.ReorderTournamentMatches(tID, []int64{m2, m1}); err != nil {
		t.Fatalf("ReorderTournamentMatches: %v", err)
	}

	tMatches, _ := db.GetTournamentMatches(tID)
	if len(tMatches) >= 2 && tMatches[0].ID != m2 {
		t.Errorf("first match ID = %d, want %d", tMatches[0].ID, m2)
	}
}

func TestGetMatchTournament(t *testing.T) {
	db, cleanup := setupTournamentTestDB(t)
	defer cleanup()

	importTestMatch(t, db)
	matches, _ := db.GetAllMatches()
	matchID := matches[0].ID

	tID, _ := db.CreateTournament("MyT", "", "")
	_ = db.AddMatchToTournament(tID, matchID)

	tt, err := db.GetMatchTournament(matchID)
	if err != nil {
		t.Fatalf("GetMatchTournament: %v", err)
	}
	if tt == nil || tt.ID != tID {
		t.Errorf("tournament = %+v, want ID=%d", tt, tID)
	}
}

func TestUpdateTournamentComment(t *testing.T) {
	db, cleanup := setupTournamentTestDB(t)
	defer cleanup()

	tID, _ := db.CreateTournament("T", "", "")
	if err := db.UpdateTournamentComment(tID, "Great event"); err != nil {
		t.Fatalf("UpdateTournamentComment: %v", err)
	}

	tournaments, _ := db.GetAllTournaments()
	for _, tt := range tournaments {
		if tt.ID == tID {
			if tt.Comment != "Great event" {
				t.Errorf("comment = %q, want %q", tt.Comment, "Great event")
			}
			return
		}
	}
	t.Error("tournament not found")
}

func TestUpdateMatchComment(t *testing.T) {
	db, cleanup := setupTournamentTestDB(t)
	defer cleanup()

	importTestMatch(t, db)
	matches, _ := db.GetAllMatches()
	matchID := matches[0].ID

	if err := db.UpdateMatchComment(matchID, "Nice match"); err != nil {
		t.Fatalf("UpdateMatchComment: %v", err)
	}

	matches2, _ := db.GetAllMatches()
	for _, m := range matches2 {
		if m.ID == matchID {
			if m.Comment != "Nice match" {
				t.Errorf("comment = %q, want %q", m.Comment, "Nice match")
			}
			return
		}
	}
	t.Error("match not found")
}

func TestExportTournaments(t *testing.T) {
	db, cleanup := setupTournamentTestDB(t)
	defer cleanup()

	importTestMatch(t, db)
	matches, _ := db.GetAllMatches()
	matchID := matches[0].ID

	tID, _ := db.CreateTournament("Export", "", "")
	_ = db.AddMatchToTournament(tID, matchID)

	exportPath := filepath.Join(t.TempDir(), "export.db")
	err := db.ExportTournaments(exportPath, []int64{tID}, map[string]string{}, true, true)
	if err != nil {
		t.Fatalf("ExportTournaments: %v", err)
	}

	info, err := os.Stat(exportPath)
	if err != nil {
		t.Fatalf("export file missing: %v", err)
	}
	if info.Size() == 0 {
		t.Error("export file is empty")
	}
}
