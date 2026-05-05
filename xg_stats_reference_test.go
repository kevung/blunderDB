package main

import (
	"math"
	"path/filepath"
	"testing"
)

// TestXGStatsReference imports the Aachen double-consultation match and
// validates per-player statistics. Two sets of values are tracked:
//
//  1. XG reference values from the eXtremeGammon match report — kept in
//     comments for documentation. blunderDB intentionally differs in some
//     areas (see notes below).
//
//  2. blunderDB baseline values — these are the regression targets. Any change
//     to the import or statistics code that alters these numbers should be
//     deliberate.
//
// Known intentional differences vs XG:
//   - Decision counts: blunderDB counts ALL No Double positions (including
//     "dead cube" ones XG skips) and forced checker moves. This inflates
//     decision/error counts and deflates PR relative to XG.
//   - Error threshold: blunderDB counts errMP > 0 as an error; XG may use
//     a small minimum threshold.
//   - MWC: now closely matches XG after fixing cube_value exponent encoding.
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

	// ── Decision counts ──────────────────────────────────────────────────────
	// blunderDB includes all No Double positions and forced checker moves.
	// XG: P1=156 / P2=161 total; blunderDB baseline: P1=212 / P2=252.
	checkCount(t, "P1 total decisions", p1.TotalDecisions, 212, 5)
	checkCount(t, "P2 total decisions", p2.TotalDecisions, 252, 5)

	// XG: 138 / 144 unforced; blunderDB includes forced moves: P1=168 / P2=169.
	checkCount(t, "P1 checker decisions", p1.CheckerDecisions, 168, 5)
	checkCount(t, "P2 checker decisions", p2.CheckerDecisions, 169, 5)

	// XG: 16 / 12 doubling decisions; blunderDB counts all No Doubles: P1=42 / P2=78.
	checkCount(t, "P1 double decisions", p1.DoubleDecisions, 42, 5)
	checkCount(t, "P2 double decisions", p2.DoubleDecisions, 78, 5)

	// Take decisions match XG exactly.
	checkCount(t, "P1 take decisions", p1.TakeDecisions, 2, 1)
	checkCount(t, "P2 take decisions", p2.TakeDecisions, 5, 1)

	// ── Error counts ─────────────────────────────────────────────────────────
	// blunderDB counts errMP > 0; XG likely uses a small min threshold.
	// XG: P1=18 (2) / P2=14 (3); blunderDB: P1=38 (2) / P2=35 (2).
	checkCount(t, "P1 total errors", p1.TotalErrors, 38, 3)
	checkCount(t, "P2 total errors", p2.TotalErrors, 35, 3)
	checkCount(t, "P1 total blunders", p1.TotalBlunders, 2, 1)
	checkCount(t, "P2 total blunders", p2.TotalBlunders, 2, 1)

	// XG: P1=16 (2) / P2=14 (3) checker errors; blunderDB: P1=35 (2) / P2=33 (2).
	checkCount(t, "P1 checker errors", p1.CheckerErrors, 35, 3)
	checkCount(t, "P2 checker errors", p2.CheckerErrors, 33, 3)

	// ── PR ───────────────────────────────────────────────────────────────────
	// PR is lower than XG because blunderDB has more zero-error decisions in denominator.
	// XG: 3.13 / 3.07; blunderDB: 2.28 / 2.08.
	checkFloat(t, "P1 PR", p1.PR, 2.28, 0.3)
	checkFloat(t, "P2 PR", p2.PR, 2.08, 0.3)

	// ── Equity errors (EMG) ──────────────────────────────────────────────────
	// Total equity is close to XG for checker (forced moves add ~0 equity error).
	// The extra No Double positions add small equity errors.
	// XG: P1=-0.976 / P2=-0.989; blunderDB: P1≈0.968 / P2≈1.049.
	checkFloat(t, "P1 total equity error (EMG)", p1.TotalEquityError, 0.968, 0.05)
	checkFloat(t, "P2 total equity error (EMG)", p2.TotalEquityError, 1.049, 0.05)

	// XG checker: P1=-0.894 / P2=-0.977; blunderDB: P1≈0.886 / P2≈0.993.
	checkFloat(t, "P1 checker equity error (EMG)", p1.CheckerEquityError, 0.886, 0.05)
	checkFloat(t, "P2 checker equity error (EMG)", p2.CheckerEquityError, 0.993, 0.05)

	// Double/take equity match XG closely.
	// XG double equity: P1=-0.037 / P2=-0.011.
	checkFloat(t, "P1 double equity error (EMG)", p1.DoubleEquityError, 0.037, 0.03)
	// P2 double equity higher due to extra No Double errors.
	checkFloat(t, "P2 double equity error (EMG)", p2.DoubleEquityError, 0.056, 0.03)

	// XG take equity: P1=-0.045 / P2=0.
	checkFloat(t, "P1 take equity error (EMG)", p1.TakeEquityError, 0.045, 0.02)
	checkFloat(t, "P2 take equity error (EMG)", p2.TakeEquityError, 0.000, 0.02)

	// ── MWC loss (%) ─────────────────────────────────────────────────────────
	// After fixing cube_value exponent encoding, MWC closely matches XG.
	// XG: P1=-19.85% / P2=-14.39%; blunderDB: P1≈20.83% / P2≈15.48%.
	// Tolerance 2pp: blunderDB includes extra No Double MWC contributions.
	checkFloat(t, "P1 total MWC loss (%)", p1.MWCLoss*100, 20.83, 1.5)
	checkFloat(t, "P2 total MWC loss (%)", p2.MWCLoss*100, 15.48, 1.5)

	// Checker MWC very close to XG (forced moves contribute 0 MWC).
	// XG: P1=-18.77% / P2=-14.24%; blunderDB: P1≈18.88% / P2≈14.67%.
	checkFloat(t, "P1 checker MWC loss (%)", p1.CheckerMWCLoss*100, 18.88, 1.5)
	checkFloat(t, "P2 checker MWC loss (%)", p2.CheckerMWCLoss*100, 14.67, 1.5)

	// P1 double MWC matches XG exactly (0.42%); P2 is higher due to extra No Doubles.
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
