package database

// Parity gate for the SQLite StatsStore (pkg/blunderdb/storage/sqlite) against
// the legacy Database stats implementation in db_stats.go. For each XG fixture
// it imports a match through the Database wrapper, then reopens the same file
// through the storage.Storage backend and asserts every StatsStore method
// returns byte-identical JSON to its legacy Database counterpart. The two DTO
// sets (database.* and storage.*) share json tags, so JSON equality is a
// field-by-field comparison that also covers slice order and float formatting.
//
// The legacy oracle is FROZEN: it proves the move onto the Storage contract
// changed no number, and nothing new is added to it. Newer storage-only
// figures are dropped by name (frozenOracleGaps).

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// frozenOracleGaps names the StatsResult fields the legacy oracle does not
// compute, with the reason. They are dropped from BOTH sides before the
// comparison — the point is to compare what both were built to answer.
//
// Back-porting them into the legacy SQL would only prove the same query
// agrees with itself.
var frozenOracleGaps = map[string]string{
	"PerPhase":    "#266, added after the migration the oracle guards",
	"PerTag":      "#266, added after the migration the oracle guards",
	"PerScore":    "#266, added after the migration the oracle guards",
	"PerGameType": "#291, added after the migration the oracle guards",
}

func jsonEqual(t *testing.T, label string, legacy, got any) {
	t.Helper()
	jl, err := marshalWithoutGaps(legacy)
	if err != nil {
		t.Fatalf("%s: marshal legacy: %v", label, err)
	}
	jg, err := marshalWithoutGaps(got)
	if err != nil {
		t.Fatalf("%s: marshal storage: %v", label, err)
	}
	if jl != jg {
		t.Errorf("%s mismatch:\n legacy = %s\n storage = %s", label, jl, jg)
	}
}

// marshalWithoutGaps renders v as JSON with the frozenOracleGaps fields
// removed, so the two sides are compared on what both compute.
func marshalWithoutGaps(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	var asMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &asMap); err != nil {
		// Not an object (a slice of ids, a string): nothing to drop.
		return string(raw), nil
	}
	for field := range frozenOracleGaps {
		delete(asMap, field)
	}
	out, err := json.Marshal(asMap)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func TestStatsStorageParity(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fixtures := []string{
		"testdata/charlot1-charlot2_7p_2025-11-08-2305.xg",
		"testdata/HsbtMarseille_main_ronde4_LamourDeCaslouGildas_UngerKevin_7p.xg",
		"testdata/match_with_comment.xg",
		"testdata/test.xg",
	}

	for _, xg := range fixtures {
		t.Run(filepath.Base(xg), func(t *testing.T) {
			t.Parallel()
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

			// ANALYZE before either side computes anything. TopBlunders' ORDER BY
			// is not total (ties on ErrorMP), so the last LIMIT slot depends on
			// the plan; the reopened side gets stats from Close's PRAGMA
			// optimize, so both sides must start with them.
			if _, err := d.db.Exec(`ANALYZE`); err != nil {
				t.Fatalf("ANALYZE: %v", err)
			}

			// The frozen oracle counts any cost above zero as an error: the
			// error threshold at its lowest (ADR-0046). The blunder threshold
			// stays at the default 100 the oracle also hard-codes.
			if err := d.SaveLibrarySettings(storage.LibrarySettings{
				ErrorThresholdMP:   storage.MinThresholdMP,
				BlunderThresholdMP: storage.DefaultBlunderThresholdMP,
			}); err != nil {
				t.Fatalf("SaveLibrarySettings: %v", err)
			}

			// 2. Legacy results. These call the legacy* reference implementations
			// directly (the production Database methods now delegate to storage, so
			// calling them here would compare storage against itself).
			legacyDR := legacyGetStatsDateRange(d)
			legacyAll, err := legacyComputeStats(d, StatsFilter{DecisionType: -1})
			if err != nil {
				t.Fatalf("legacy ComputeStats: %v", err)
			}
			legacyChecker, err := legacyComputeStats(d, StatsFilter{DecisionType: 0})
			if err != nil {
				t.Fatalf("legacy ComputeStats(checker): %v", err)
			}
			legacyPlayers, err := legacyGetAllPlayerNames(d)
			if err != nil {
				t.Fatalf("legacy GetAllPlayerNames: %v", err)
			}
			legacyMatchIDs, err := legacyGetPositionIDsByMatch(d, matchID)
			if err != nil {
				t.Fatalf("legacy GetPositionIDsByMatch: %v", err)
			}
			legacySel, err := legacyGetPositionIDsByStatsSelection(d,
				StatsFilter{DecisionType: -1}, SelectionSpec{Kind: "checker", OnlyWithError: true})
			if err != nil {
				t.Fatalf("legacy GetPositionIDsByStatsSelection: %v", err)
			}
			legacyDetail, err := legacyGetMatchDetailStats(d, matchID)
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
