package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// ── Reference JSON types ──────────────────────────────────────────────────────

// refPlayer covers both XG and gnuBG JSON schemas (see SCHEMA.md).
// Pointer fields are nil when the value was not measured / not available.
type refPlayer struct {
	// XG fields
	PR                  *float64 `json:"pr"`
	TotalDecisions      *int     `json:"total_decisions"`
	CheckerUnforced     *int     `json:"checker_unforced"`
	DoubleDecisions     *int     `json:"double_decisions"`
	TakeDecisions       *int     `json:"take_decisions"`
	TotalEquityErrorEMG *float64 `json:"total_equity_error_emg"`
	TotalMWCLossPct     *float64 `json:"total_mwc_loss_pct"`
	CheckerMWCLossPct   *float64 `json:"checker_mwc_loss_pct"`
	CheckerEquityEMG    *float64 `json:"checker_equity_error_emg"`
	// gnuBG-only fields
	CheckerTotal   *int     `json:"checker_total"`
	CheckerForced  *int     `json:"checker_forced"`
	CheckerPRXG500 *float64 `json:"checker_pr_xg_500"`
	CubeEquityEMG  *float64 `json:"cube_equity_error_emg"`
	CubeMWCLossPct *float64 `json:"cube_mwc_loss_pct"`
	TotalCube      *int     `json:"total_cube"`
}

type refMatch struct {
	MatchFile string                `json:"match_file"`
	SGFFile   string                `json:"sgf_file"`
	Players   [2]string             `json:"players"`
	XG        map[string]*refPlayer `json:"xg"`
	GnuBG     map[string]*refPlayer `json:"gnubg"`
	Notes     string                `json:"notes"`
}

func loadRefMatch(t *testing.T, path string) refMatch {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("loadRefMatch %s: %v", path, err)
	}
	var m refMatch
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("loadRefMatch %s unmarshal: %v", path, err)
	}
	return m
}

// ── Tolerances ───────────────────────────────────────────────────────────────

// parityTolerances sets per-metric acceptance thresholds.
// Phase 01 values are intentionally wide (blunderDB counts forced moves and
// all No Double positions); they tighten after fiches 02–04.
type parityTolerances struct {
	TotalDecisions   int     // all decisions combined
	CheckerDecisions int     // checker move count
	DoubleDecisions  int     // cube doubling decisions (very wide pre-fiche 03)
	TakeDecisions    int     // take/pass count
	PR               float64
	MWCPct           float64 // percentage points
	Equity           float64 // EMG
}

// tolPhase01 matches the current blunderDB state (pre forced-moves + pre close-cube).
// These tolerances will tighten progressively in fiches 02–04.
//
// Explanations for wide values:
//   CheckerDecisions=45: bDB counts all checker positions (incl. forced). gnuBG
//     ref stores unforced only; observed deltas up to 40 (38 forced + import gaps).
//   DoubleDecisions=100: bDB counts all No Doubles; XG/gnuBG only close ones
//     (factor 2–6× overcounting pre-fiche 03).
//   PR=3.0: inflated denominator (all decisions vs unforced+close) depresses bDB PR
//     by up to 2+ points vs XG/gnuBG reference.
var tolPhase01 = parityTolerances{
	TotalDecisions:   100, // bDB includes forced + all No Doubles
	CheckerDecisions: 45,  // bDB includes forced moves + SGF import gaps
	DoubleDecisions:  100, // bDB counts all No Doubles
	TakeDecisions:    3,   // takes/passes should align closely
	PR:               3.0, // wide until denominator fixed (fiche 04)
	MWCPct:           3.5, // percentage points
	Equity:           0.35, // EMG
}

// ── Diff helpers ─────────────────────────────────────────────────────────────

func diffFloat(t *testing.T, label string, want *float64, got, tol float64) {
	t.Helper()
	if want == nil {
		t.Logf("  %-40s  ref=n/a   got=%+.4f", label, got)
		return
	}
	diff := math.Abs(got - *want)
	mark := "  "
	if diff > tol {
		mark = "!!"
	}
	t.Logf("%s %-40s  ref=%+.4f  got=%+.4f  diff=%.4f  tol=±%.4f", mark, label, *want, got, diff, tol)
	if diff > tol {
		t.Errorf("%s: got %.4f, ref %.4f (diff %.4f > tol %.4f)", label, got, *want, diff, tol)
	}
}

func diffInt(t *testing.T, label string, want *int, got, tol int) {
	t.Helper()
	if want == nil {
		t.Logf("  %-40s  ref=n/a  got=%d", label, got)
		return
	}
	diff := got - *want
	if diff < 0 {
		diff = -diff
	}
	mark := "  "
	if diff > tol {
		mark = "!!"
	}
	t.Logf("%s %-40s  ref=%d  got=%d  diff=%d  tol=±%d", mark, label, *want, got, diff, tol)
	if diff > tol {
		t.Errorf("%s: got %d, ref %d (diff %d > tol %d)", label, got, *want, diff, tol)
	}
}

// ── Import helpers ────────────────────────────────────────────────────────────

func importAndStats(t *testing.T, file string) (player1Name, player2Name string, s *MatchDetailStats) {
	t.Helper()
	db := newTestDB(t)
	var matchID int64
	var err error
	switch filepath.Ext(file) {
	case ".xg":
		matchID, err = db.ImportXGMatch(file)
	case ".sgf":
		matchID, err = db.ImportGnuBGMatch(file)
	default:
		t.Skipf("unsupported file type: %s", file)
	}
	if err != nil {
		t.Fatalf("import %s: %v", file, err)
	}
	db.db.QueryRow(`SELECT COALESCE(player1_name,''), COALESCE(player2_name,'') FROM match WHERE id = ?`, matchID).
		Scan(&player1Name, &player2Name)
	s, err = db.GetMatchDetailStats(matchID)
	if err != nil {
		t.Fatalf("GetMatchDetailStats: %v", err)
	}
	return
}

// ── Comparison logic ─────────────────────────────────────────────────────────

// compareXGRef compares blunderDB stats against an XG reference player.
// Uses wide tolerances because bDB counts forced moves and all No Doubles.
func compareXGRef(t *testing.T, prefix string, ref *refPlayer, bdb MatchPlayerDetailStats, tol parityTolerances) {
	t.Helper()
	t.Logf("--- %s vs XG reference ---", prefix)
	diffInt(t, prefix+" total_decisions", ref.TotalDecisions, bdb.TotalDecisions, tol.TotalDecisions)
	diffInt(t, prefix+" checker_unforced", ref.CheckerUnforced, bdb.CheckerDecisions, tol.CheckerDecisions)
	diffInt(t, prefix+" double_decisions", ref.DoubleDecisions, bdb.DoubleDecisions, tol.DoubleDecisions)
	diffInt(t, prefix+" take_decisions", ref.TakeDecisions, bdb.TakeDecisions, tol.TakeDecisions)
	diffFloat(t, prefix+" PR", ref.PR, bdb.PR, tol.PR)
	diffFloat(t, prefix+" total_equity_emg", ref.TotalEquityErrorEMG, bdb.TotalEquityError, tol.Equity)
	diffFloat(t, prefix+" checker_equity_emg", ref.CheckerEquityEMG, bdb.CheckerEquityError, tol.Equity)
	if ref.TotalMWCLossPct != nil {
		mwcRef := *ref.TotalMWCLossPct
		diffFloat(t, prefix+" total_mwc_pct", &mwcRef, bdb.MWCLoss*100, tol.MWCPct)
	}
	if ref.CheckerMWCLossPct != nil {
		mwcRef := *ref.CheckerMWCLossPct
		diffFloat(t, prefix+" checker_mwc_pct", &mwcRef, bdb.CheckerMWCLoss*100, tol.MWCPct)
	}
}

// compareGnuBGRef compares blunderDB stats against a gnuBG reference player.
// gnuBG ref uses checker_unforced (bDB will over-count by the forced-moves count).
func compareGnuBGRef(t *testing.T, prefix string, ref *refPlayer, bdb MatchPlayerDetailStats, tol parityTolerances) {
	t.Helper()
	t.Logf("--- %s vs gnuBG reference ---", prefix)
	// Compare checker count: bDB counts all checker positions (incl. forced).
	// gnuBG ref stores both unforced and total; compare against unforced (stricter).
	diffInt(t, prefix+" checker_unforced", ref.CheckerUnforced, bdb.CheckerDecisions, tol.CheckerDecisions)
	// checker_total: bDB matches gnuBG total (incl. forced) in principle, but some
	// SGF positions lack analysis data, so counts may fall short. Use the same
	// CheckerDecisions tolerance; this will tighten once is_forced is tracked.
	if ref.CheckerTotal != nil {
		diffInt(t, prefix+" checker_total", ref.CheckerTotal, bdb.CheckerDecisions, tol.CheckerDecisions)
	}
	diffFloat(t, prefix+" total_equity_emg", ref.TotalEquityErrorEMG, bdb.TotalEquityError, tol.Equity)
	if ref.TotalMWCLossPct != nil {
		mwcRef := *ref.TotalMWCLossPct
		diffFloat(t, prefix+" total_mwc_pct", &mwcRef, bdb.MWCLoss*100, tol.MWCPct)
	}
	// Checker-only equity (gnuBG ref: checker_equity_error_emg)
	diffFloat(t, prefix+" checker_equity_emg", ref.CheckerEquityEMG, bdb.CheckerEquityError, tol.Equity)
	if ref.CheckerMWCLossPct != nil {
		mwcRef := *ref.CheckerMWCLossPct
		diffFloat(t, prefix+" checker_mwc_pct", &mwcRef, bdb.CheckerMWCLoss*100, tol.MWCPct)
	}
	// checker_pr_xg_500: compare against bDB PRChecker (same 500-factor formula)
	diffFloat(t, prefix+" checker_pr_xg_500", ref.CheckerPRXG500, bdb.PRChecker, tol.PR)
}

// ── Main test ─────────────────────────────────────────────────────────────────

func TestStatsParity(t *testing.T) {
	fixtures := []string{
		"testdata/stats_reference/aachen-double-7pt.json",
		"testdata/stats_reference/test.json",
		"testdata/stats_reference/charlot1-charlot2.json",
	}
	tol := tolPhase01

	for _, jsonPath := range fixtures {
		t.Run(filepath.Base(jsonPath), func(t *testing.T) {
			ref := loadRefMatch(t, jsonPath)

			// ── XG import ─────────────────────────────────────────────────
			if ref.MatchFile != "" {
				if _, err := os.Stat(ref.MatchFile); err == nil {
					p1n, _, bdbStats := importAndStats(t, ref.MatchFile)
					t.Logf("XG import: %s  P1=%q", filepath.Base(ref.MatchFile), p1n)

					if ref.XG != nil {
						if rp := ref.XG["player1"]; rp != nil {
							compareXGRef(t, "XG/P1", rp, bdbStats.Player1, tol)
						}
						if rp := ref.XG["player2"]; rp != nil {
							compareXGRef(t, "XG/P2", rp, bdbStats.Player2, tol)
						}
					}
					if ref.GnuBG != nil {
						if rp := ref.GnuBG["player1"]; rp != nil {
							compareGnuBGRef(t, "XG→gnuBGref/P1", rp, bdbStats.Player1, tol)
						}
						if rp := ref.GnuBG["player2"]; rp != nil {
							compareGnuBGRef(t, "XG→gnuBGref/P2", rp, bdbStats.Player2, tol)
						}
					}
				} else {
					t.Logf("SKIP XG import: %s not found", ref.MatchFile)
				}
			}

			// ── SGF import ────────────────────────────────────────────────
			if ref.SGFFile != "" {
				if _, err := os.Stat(ref.SGFFile); err == nil {
					p1n, _, bdbStats := importAndStats(t, ref.SGFFile)
					t.Logf("SGF import: %s  P1=%q", filepath.Base(ref.SGFFile), p1n)

					if ref.GnuBG != nil {
						if rp := ref.GnuBG["player1"]; rp != nil {
							compareGnuBGRef(t, "SGF/P1", rp, bdbStats.Player1, tol)
						}
						if rp := ref.GnuBG["player2"]; rp != nil {
							compareGnuBGRef(t, "SGF/P2", rp, bdbStats.Player2, tol)
						}
					}
				} else {
					t.Logf("SKIP SGF import: %s not found", ref.SGFFile)
				}
			}
		})
	}
}
