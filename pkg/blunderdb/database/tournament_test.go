package database

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func TestCreateTournament(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

	id, err := db.CreateTournament("Grand Prix", "2025-01-15", "Paris")
	if err != nil {
		t.Fatalf("CreateTournament: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive ID, got %d", id)
	}
}

func TestGetAllTournaments(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

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
	t.Parallel()
	db := newTestDB(t)

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
	t.Parallel()
	db := newTestDB(t)

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
	t.Parallel()
	db := newTestDB(t)

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
	t.Parallel()
	db := newTestDB(t)

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
	t.Parallel()
	db := newTestDB(t)

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
	t.Parallel()
	db := newTestDB(t)

	importTestMatch(t, db)
	matches, _ := db.GetAllMatches()
	matchID := matches[0].ID

	if _, err := db.CreateTournament("Existing", "", ""); err != nil {
		t.Fatal(err)
	}
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
	t.Parallel()
	db := newTestDB(t)

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
	t.Parallel()
	db := newTestDB(t)

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
	t.Parallel()
	db := newTestDB(t)

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
	t.Parallel()
	db := newTestDB(t)

	importTestMatch(t, db)
	matches, _ := db.GetAllMatches()
	matchID := matches[0].ID

	db.SetCommentAuthor("Alice")
	if err := db.UpdateMatchComment(matchID, "Nice match"); err != nil {
		t.Fatalf("UpdateMatchComment: %v", err)
	}

	matches2, _ := db.GetAllMatches()
	for _, m := range matches2 {
		if m.ID == matchID {
			if m.Comment != "Nice match" || m.CommentAuthor != "Alice" {
				t.Errorf("comment = %q by %q, want %q by %q", m.Comment, m.CommentAuthor, "Nice match", "Alice")
			}
			return
		}
	}
	t.Error("match not found")
}

// A tournament leaves with the aliases of its players: its report reads them
// as one player through those aliases.
func TestExportDatabase_TournamentCarriesAliases(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	importTestMatch(t, db)
	matches, _ := db.GetAllMatches()
	tID, _ := db.CreateTournament("Export", "", "")
	_ = db.AddMatchToTournament(tID, matches[0].ID)
	ctx := context.Background()
	if err := db.store.Aliases().Set(ctx, "", storage.AliasPlayer, "Autre orthographe", matches[0].Player1Name); err != nil {
		t.Fatal(err)
	}

	exportPath := filepath.Join(t.TempDir(), "export.db")
	if _, err := db.ExportDatabase(ExportOptions{
		ExportPath: exportPath, IncludeTournaments: true, TournamentIDs: []int64{tID},
		IncludeMatches: true, MatchIDs: []int64{matches[0].ID}, IncludeAliases: true, IncludeAnalysis: true, IncludeComments: true,
	}); err != nil {
		t.Fatalf("ExportDatabase: %v", err)
	}
	dst, err := sqlite.Open(ctx, exportPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()
	got, err := dst.Aliases().List(ctx, "", storage.AliasPlayer)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Alias != "Autre orthographe" || got[0].Canonical != matches[0].Player1Name {
		t.Errorf("exported player aliases = %+v", got)
	}
}
