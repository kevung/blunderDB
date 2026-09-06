package database

// Parity gate for the SQLite StatsStore (pkg/blunderdb/storage/sqlite) against
// the legacy Database stats implementation in db_stats.go. For each XG fixture
// it imports a match through the Database wrapper, then reopens the same file
// through the storage.Storage backend and asserts every StatsStore method
// returns byte-identical JSON to its legacy Database counterpart. The two DTO
// sets (database.* and storage.*) share json tags, so JSON equality is a
// field-by-field comparison that also covers slice order and float formatting.
//
// # What the legacy oracle is, and what it is not
//
// The legacy implementation is FROZEN. It exists to prove that moving the
// statistics onto the Storage contract did not change a single number the
// application had been showing, and it is not maintained beyond that: nothing
// new is added to it, because adding a figure to both sides would only prove
// that the same code was written twice.
//
// A figure that exists only in the storage implementation is therefore not a
// parity failure — it is a figure that came after the migration. The
// comparison drops those fields by name (frozenOracleGaps below) rather than
// asserting on them, and each one says why it is there.

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
// PerPhase/PerTag/PerScore came with #266, after the migration the oracle
// exists to guard. Back-porting them into the legacy SQL would prove nothing
// (the same query, written twice, agreeing with itself) and would extend a
// body of code the migration exists to have retired.
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
			// is not a total order (ties on ErrorMP — see FOLLOWUPS.md "Rolling
			// stats non-determinism"), so which row lands in the last slot of a
			// LIMIT can depend on the query plan the still-empty vs.
			// already-populated sqlite_stat1 leads the planner to pick. Without
			// this, legacyAll below ran against a database that had never been
			// ANALYZEd, while gotAll (below) ran against the same file reopened
			// after Close — which now also runs PRAGMA optimize (fiche-05 T7) and,
			// on a freshly-imported database with no prior stats, performs a real
			// ANALYZE. That asymmetry alone was enough to flip a tied entry.
			// ANALYZE-ing up front puts both sides on equal footing, which is what
			// a parity test should be comparing in the first place.
			if _, err := d.db.Exec(`ANALYZE`); err != nil {
				t.Fatalf("ANALYZE: %v", err)
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
