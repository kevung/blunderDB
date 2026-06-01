package database

// Parity gate for the SQLite StatsStore (pkg/blunderdb/storage/sqlite) against
// the legacy Database stats implementation in db_stats.go. For each XG fixture
// it imports a match through the Database wrapper, then reopens the same file
// through the storage.Storage backend and asserts every StatsStore method
// returns byte-identical JSON to its legacy Database counterpart. The two DTO
// sets (database.* and storage.*) share json tags, so JSON equality is a
// field-by-field comparison that also covers slice order and float formatting.

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func jsonEqual(t *testing.T, label string, legacy, got any) {
	t.Helper()
	jl, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("%s: marshal legacy: %v", label, err)
	}
	jg, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("%s: marshal storage: %v", label, err)
	}
	if string(jl) != string(jg) {
		t.Errorf("%s mismatch:\n legacy = %s\n storage = %s", label, jl, jg)
	}
}

func TestStatsStorageParity(t *testing.T) {
	ctx := context.Background()
	fixtures := []string{
		"testdata/charlot1-charlot2_7p_2025-11-08-2305.xg",
		"testdata/HsbtMarseille_main_ronde4_LamourDeCaslouGildas_UngerKevin_7p.xg",
		"testdata/match_with_comment.xg",
		"testdata/test.xg",
	}

	for _, xg := range fixtures {
		t.Run(filepath.Base(xg), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "parity.db")

			// 1. Import via the legacy Database wrapper into a file-backed DB.
			d := NewDatabase()
			if err := d.SetupDatabase(path); err != nil {
				t.Fatalf("SetupDatabase: %v", err)
			}
			if _, err := d.ImportXGMatch(xg); err != nil {
				t.Fatalf("ImportXGMatch(%s): %v", xg, err)
			}

			// Grab a match id (and its existence) directly for the per-match methods.
			var matchID int64
			if err := d.db.QueryRow(`SELECT id FROM match ORDER BY id LIMIT 1`).Scan(&matchID); err != nil {
				t.Fatalf("select match id: %v", err)
			}

			// 2. Legacy results.
			legacyDR := d.GetStatsDateRange()
			legacyAll, err := d.ComputeStats(StatsFilter{DecisionType: -1})
			if err != nil {
				t.Fatalf("legacy ComputeStats: %v", err)
			}
			legacyChecker, err := d.ComputeStats(StatsFilter{DecisionType: 0})
			if err != nil {
				t.Fatalf("legacy ComputeStats(checker): %v", err)
			}
			legacyPlayers, err := d.GetAllPlayerNames()
			if err != nil {
				t.Fatalf("legacy GetAllPlayerNames: %v", err)
			}
			legacyMatchIDs, err := d.GetPositionIDsByMatch(matchID)
			if err != nil {
				t.Fatalf("legacy GetPositionIDsByMatch: %v", err)
			}
			legacySel, err := d.GetPositionIDsByStatsSelection(
				StatsFilter{DecisionType: -1}, SelectionSpec{Kind: "checker", OnlyWithError: true})
			if err != nil {
				t.Fatalf("legacy GetPositionIDsByStatsSelection: %v", err)
			}
			legacyDetail, err := d.GetMatchDetailStats(matchID)
			if err != nil {
				t.Fatalf("legacy GetMatchDetailStats: %v", err)
			}

			// Close so WAL is checkpointed into the main db file before reopening.
			if err := d.Close(); err != nil {
				t.Fatalf("Close: %v", err)
			}

			// 3. Storage backend results over the same file.
			st, err := sqlite.Open(ctx, path, nil)
			if err != nil {
				t.Fatalf("sqlite.Open: %v", err)
			}
			defer st.Close()
			ss := st.Stats()

			gotDR, err := ss.DateRange(ctx, "")
			if err != nil {
				t.Fatalf("storage DateRange: %v", err)
			}
			gotAll, err := ss.Compute(ctx, "", storage.StatsFilter{DecisionType: -1})
			if err != nil {
				t.Fatalf("storage Compute: %v", err)
			}
			gotChecker, err := ss.Compute(ctx, "", storage.StatsFilter{DecisionType: 0})
			if err != nil {
				t.Fatalf("storage Compute(checker): %v", err)
			}
			gotPlayers, err := ss.PlayerNames(ctx, "")
			if err != nil {
				t.Fatalf("storage PlayerNames: %v", err)
			}
			gotMatchIDs, err := ss.PositionIDsByMatch(ctx, "", matchID)
			if err != nil {
				t.Fatalf("storage PositionIDsByMatch: %v", err)
			}
			gotSel, err := ss.PositionIDsBySelection(ctx, "",
				storage.StatsFilter{DecisionType: -1},
				storage.SelectionSpec{Kind: "checker", OnlyWithError: true})
			if err != nil {
				t.Fatalf("storage PositionIDsBySelection: %v", err)
			}
			gotDetail, err := ss.MatchDetail(ctx, "", matchID)
			if err != nil {
				t.Fatalf("storage MatchDetail: %v", err)
			}

			// 4. Compare.
			jsonEqual(t, "DateRange", legacyDR, gotDR)
			jsonEqual(t, "Compute(all)", legacyAll, gotAll)
			jsonEqual(t, "Compute(checker)", legacyChecker, gotChecker)
			jsonEqual(t, "PlayerNames", legacyPlayers, gotPlayers)
			jsonEqual(t, "PositionIDsByMatch", legacyMatchIDs, gotMatchIDs)
			jsonEqual(t, "PositionIDsBySelection", legacySel, gotSel)
			jsonEqual(t, "MatchDetail", legacyDetail, gotDetail)
		})
	}
}
