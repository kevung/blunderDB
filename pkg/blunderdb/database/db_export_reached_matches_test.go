package database

import (
	"database/sql"
	"path/filepath"
	"slices"
	"testing"
)

// A filtered list leaves with the matches its positions were played in, not every match of
// the library, and each of those matches with its positions: a move of the file never points
// at a position the file lacks.
func TestExportFilteredListCarriesTheMatchesItReaches(t *testing.T) {
	isolateIdentity(t)
	src := newTestDBWithXG(t)
	xgMatches, err := src.GetAllMatches()
	if err != nil || len(xgMatches) != 1 {
		t.Fatalf("matches after the XG import = %d, %v", len(xgMatches), err)
	}
	reached := xgMatches[0].ID
	other, err := src.ImportXGMatch(filepath.Join("testdata", "HsbtMarseille_main_ronde4_LamourDeCaslouGildas_UngerKevin_7p.xg"))
	if err != nil {
		t.Fatal(err)
	}

	reachedIDs, err := src.GetPositionIDsByMatch(reached)
	if err != nil {
		t.Fatal(err)
	}
	otherIDs, err := src.GetPositionIDsByMatch(other)
	if err != nil {
		t.Fatal(err)
	}
	var picked int64
	for _, id := range reachedIDs {
		if !slices.Contains(otherIDs, id) {
			picked = id
			break
		}
	}
	if picked == 0 {
		t.Fatal("no position belongs to the XG match alone")
	}

	got, err := src.MatchIDsReachedBy([]int64{picked})
	if err != nil || !slices.Equal(got, []int64{reached}) {
		t.Fatalf("MatchIDsReachedBy = %v, %v; want [%d]", got, err, reached)
	}

	path := exportTo(t, src, filepath.Join(t.TempDir(), "liste.db"), ExportOptions{
		PositionIDs: []int64{picked}, IncludeMatches: true, IncludeAnalysis: true,
	})
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	var matches, positions, dangling int
	if err := raw.QueryRow(`SELECT COUNT(*) FROM match`).Scan(&matches); err != nil || matches != 1 {
		t.Fatalf("matches in the file = %d, %v; want the one the list reaches", matches, err)
	}
	if err := raw.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&positions); err != nil || positions < len(reachedIDs) {
		t.Fatalf("positions in the file = %d, %v; want at least the match's %d", positions, err, len(reachedIDs))
	}
	if err := raw.QueryRow(`SELECT COUNT(*) FROM move WHERE position_id IS NOT NULL
		AND position_id NOT IN (SELECT id FROM position)`).Scan(&dangling); err != nil || dangling != 0 {
		t.Fatalf("moves pointing at no position = %d, %v; want 0", dangling, err)
	}
}
