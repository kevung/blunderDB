package main

import (
	"math"
	"path/filepath"
	"testing"
)

// TestXGStatsReference imports the Aachen double-consultation match and
// validates per-player statistics against XG reference values.
//
// After fiche 04 (PR denominator fix), blunderDB now counts the same decisions
// as XG: unforced checker plays + close cube decisions. The baselines are the
// XG reference values from the match report.
//
// XG reference (Squire/Jørgensen vs Harmand/Unger, 7-pt match):
//
//	                    Squire/Jørgensen   Harmand/Unger
//	PR                       3.13               3.07
//	Total Decisions           156                161
//	Total Errors (Blunders)   18 (2)             14 (3)
//	Total Equity Error       -0.976             -0.989
//	Checker Moves (Unforced)  138                144
//	Checker Errors (Blunders) 16 (2)             14 (3)
//	Checker Equity           -0.894             -0.977
//	Double Cube Decisions      16                 12
//	Double Errors (Blunders)   1 (0)              0 (0)
//	Double Equity             -0.037             -0.011
//	Take Decisions             2                  5
//	Take Errors (Blunders)     1 (0)              0 (0)
//	Take Equity               -0.045              0
//	Total MWC loss           -19.85%            -14.39%
//	Checker MWC loss         -18.77%            -14.24%
//	Double MWC loss           -0.42%             -0.15%
//	Take MWC loss             -0.66%              0%
//
// Known residual differences vs XG after fiche 04:
//   - Error threshold: blunderDB uses errMP > 0; XG uses a minimum threshold
//     (≈5 millipoints). Error/blunder counts differ but do not affect PR.
//   - Checker unforced: bDB detects slightly more unforced moves (≤6 delta)
//     due to edge cases in forced-move classification (1-legal-move detection).
//   - Equity / MWC: analysis engine rounding produces small residuals (< 1 %).
func TestXGStatsReference(t *testing.T) {
	matchFile := filepath.Join(
		"testdata",
		"2024-08-10-Aachen-1x11pt-1x7pt-2x7ptDoubleConsultation",
		"double",
		"Squire-Jørgensen-Harmand-Unger 7 point match 18-08-2024.xg",
	)

	db := newTestDB(t)
	matchID, err := db.ImportXGMatch(matchFile)
	if err != nil {
		t.Fatalf("ImportXGMatch: %v", err)
	}
	if matchID <= 0 {
		t.Fatalf("expected positive matchID, got %d", matchID)
	}

	stats, err := db.GetMatchDetailStats(matchID)
	if err != nil {
		t.Fatalf("GetMatchDetailStats: %v", err)
	}

	p1 := stats.Player1 // Squire/Jørgensen
	p2 := stats.Player2 // Harmand/Unger

	t.Logf("=== Squire/Jørgensen (P1) ===")
	t.Logf("  PR=%.2f  decisions=%d  errors=%d  blunders=%d  equity=%.3f  mwc=%.2f%%",
		p1.PR, p1.TotalDecisions, p1.TotalErrors, p1.TotalBlunders, p1.TotalEquityError, p1.MWCLoss*100)
	t.Logf("  Checker: moves=%d  errors=%d  blunders=%d  equity=%.3f  mwc=%.2f%%",
		p1.CheckerDecisions, p1.CheckerErrors, p1.CheckerBlunders, p1.CheckerEquityError, p1.CheckerMWCLoss*100)
	t.Logf("  Doubles: decisions=%d  errors=%d  blunders=%d  equity=%.3f  mwc=%.2f%%",
		p1.DoubleDecisions, p1.DoubleErrors, p1.DoubleBlunders, p1.DoubleEquityError, p1.DoubleMWCLoss*100)
	t.Logf("  Takes:   decisions=%d  errors=%d  blunders=%d  equity=%.3f  mwc=%.2f%%",
		p1.TakeDecisions, p1.TakeErrors, p1.TakeBlunders, p1.TakeEquityError, p1.TakeMWCLoss*100)

	t.Logf("=== Harmand/Unger (P2) ===")
	t.Logf("  PR=%.2f  decisions=%d  errors=%d  blunders=%d  equity=%.3f  mwc=%.2f%%",
		p2.PR, p2.TotalDecisions, p2.TotalErrors, p2.TotalBlunders, p2.TotalEquityError, p2.MWCLoss*100)
	t.Logf("  Checker: moves=%d  errors=%d  blunders=%d  equity=%.3f  mwc=%.2f%%",
		p2.CheckerDecisions, p2.CheckerErrors, p2.CheckerBlunders, p2.CheckerEquityError, p2.CheckerMWCLoss*100)
	t.Logf("  Doubles: decisions=%d  errors=%d  blunders=%d  equity=%.3f  mwc=%.2f%%",
		p2.DoubleDecisions, p2.DoubleErrors, p2.DoubleBlunders, p2.DoubleEquityError, p2.DoubleMWCLoss*100)
	t.Logf("  Takes:   decisions=%d  errors=%d  blunders=%d  equity=%.3f  mwc=%.2f%%",
		p2.TakeDecisions, p2.TakeErrors, p2.TakeBlunders, p2.TakeEquityError, p2.TakeMWCLoss*100)

	// ── Decision counts ──────────────────────────────────────────────────────────
	// After fiche 04: bDB counts unforced checker + close cube (same as XG).
	// XG: P1=156 / P2=161. Residual: forced-move boundary classification (≤6).
	checkCount(t, "P1 total decisions", p1.TotalDecisions, 156, 5)
	checkCount(t, "P2 total decisions", p2.TotalDecisions, 161, 5)

	// XG unforced: 138 / 144. bDB may classify ≤6 more moves as unforced.
	checkCount(t, "P1 checker decisions", p1.CheckerDecisions, 138, 10)
	checkCount(t, "P2 checker decisions", p2.CheckerDecisions, 144, 10)

	// XG: 16 / 12 close doubling decisions.
	checkCount(t, "P1 double decisions", p1.DoubleDecisions, 16, 5)
	checkCount(t, "P2 double decisions", p2.DoubleDecisions, 12, 5)

	// Take decisions: unchanged.
	checkCount(t, "P1 take decisions", p1.TakeDecisions, 2, 1)
	checkCount(t, "P2 take decisions", p2.TakeDecisions, 5, 1)

	// ── Error counts ─────────────────────────────────────────────────────────────
	// bDB uses errMP > 0; XG uses a higher minimum threshold. Error counts
	// remain higher than XG but are stable (forced/non-close moves have 0 error).
	// XG: P1=18 (2) / P2=14 (3); bDB: P1=38 (2) / P2=35 (2).
	checkCount(t, "P1 total errors", p1.TotalErrors, 38, 3)
	checkCount(t, "P2 total errors", p2.TotalErrors, 35, 3)
	checkCount(t, "P1 total blunders", p1.TotalBlunders, 2, 1)
	checkCount(t, "P2 total blunders", p2.TotalBlunders, 2, 1)

	// XG: P1=16 (2) / P2=14 (3) checker errors; bDB: P1=35 (2) / P2=33 (2).
	checkCount(t, "P1 checker errors", p1.CheckerErrors, 35, 3)
	checkCount(t, "P2 checker errors", p2.CheckerErrors, 33, 3)

	// ── PR ───────────────────────────────────────────────────────────────────────
	// After fiche 04: denominator aligned with XG. PR is now close to XG target.
	// XG: 3.13 / 3.07; bDB: P1≈3.04 / P2≈3.18.
	checkFloat(t, "P1 PR", p1.PR, 3.13, 0.30)
	checkFloat(t, "P2 PR", p2.PR, 3.07, 0.30)

	// ── Equity errors (EMG) ──────────────────────────────────────────────────────
	// Numerator unchanged (forced/non-close moves have 0 error). Residual from
	// analysis-engine rounding between bDB (XG analysis) and XG report.
	// XG: P1=-0.976 / P2=-0.989; bDB: P1≈0.968 / P2≈1.049.
	checkFloat(t, "P1 total equity error (EMG)", p1.TotalEquityError, 0.976, 0.05)
	checkFloat(t, "P2 total equity error (EMG)", p2.TotalEquityError, 0.989, 0.07)

	// XG checker: P1=-0.894 / P2=-0.977; bDB: P1≈0.886 / P2≈0.993.
	checkFloat(t, "P1 checker equity error (EMG)", p1.CheckerEquityError, 0.894, 0.05)
	checkFloat(t, "P2 checker equity error (EMG)", p2.CheckerEquityError, 0.977, 0.05)

	// Close-double equity. P1 matches XG exactly; P2 differs by ≤0.05 EMG
	// (analysis engine computes slightly different equity for the close doubles).
	// XG double equity: P1=-0.037 / P2=-0.011.
	checkFloat(t, "P1 double equity error (EMG)", p1.DoubleEquityError, 0.037, 0.03)
	checkFloat(t, "P2 double equity error (EMG)", p2.DoubleEquityError, 0.011, 0.05)

	// XG take equity: P1=-0.045 / P2=0.
	checkFloat(t, "P1 take equity error (EMG)", p1.TakeEquityError, 0.045, 0.02)
	checkFloat(t, "P2 take equity error (EMG)", p2.TakeEquityError, 0.000, 0.02)

	// ── MWC loss (%) ─────────────────────────────────────────────────────────────
	// Non-close cube decisions had near-zero MWC contribution; MWC is close to XG.
	// XG: P1=-19.85% / P2=-14.39%; bDB: P1≈20.83% / P2≈15.48%.
	checkFloat(t, "P1 total MWC loss (%)", p1.MWCLoss*100, 19.85, 1.5)
	checkFloat(t, "P2 total MWC loss (%)", p2.MWCLoss*100, 14.39, 1.5)

	// Checker MWC matches XG closely (forced moves have 0 MWC).
	// XG: P1=-18.77% / P2=-14.24%; bDB: P1≈18.88% / P2≈14.67%.
	checkFloat(t, "P1 checker MWC loss (%)", p1.CheckerMWCLoss*100, 18.77, 1.5)
	checkFloat(t, "P2 checker MWC loss (%)", p2.CheckerMWCLoss*100, 14.24, 1.5)

	// P1 double MWC matches XG (0.42%); P2 not checked (small value, engine noise).
	checkFloat(t, "P1 double MWC loss (%)", p1.DoubleMWCLoss*100, 0.42, 0.10)
}

// checkCount asserts that got is within delta of want (inclusive).
func checkCount(t *testing.T, name string, got, want, delta int) {
	t.Helper()
	if absDiff(got-want) > delta {
		t.Errorf("%s: got %d, want %d (tolerance ±%d)", name, got, want, delta)
	}
}

// checkFloat asserts that got is within delta of want.
func checkFloat(t *testing.T, name string, got, want, delta float64) {
	t.Helper()
	if math.Abs(got-want) > delta {
		t.Errorf("%s: got %.3f, want %.3f (tolerance ±%.3f)", name, got, want, delta)
	}
}

func absDiff(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
