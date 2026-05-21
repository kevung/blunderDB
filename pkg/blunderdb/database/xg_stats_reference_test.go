package database

import (
	"path/filepath"
	"testing"
)

// TestXGStatsReference imports the Aachen double-consultation match and logs
// per-player statistics for manual inspection.
//
// Numeric assertions have been moved to TestStatsParity (testdata/stats_reference/
// aachen-double-7pt.json) which uses tighter tolerances (fiche 07 final).
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
// Residual differences (stable, documented in aachen-double-7pt.json notes):
//   - Error/blunder counts differ: bDB uses errMP > 0; XG applies a ~5 millipoint threshold.
//   - Checker unforced: bDB may detect ≤6 more unforced moves (boundary classification).
//   - Equity / MWC: analysis engine rounding produces residuals < 1 pp.
func TestXGStatsReference(t *testing.T) {
	matchFile := filepath.Join(
		"testdata",
		"2024-08-10-Aachen-1x11pt-1x7pt-2x7ptDoubleConsultation",
		"double",
		"Squire-J\u00f8rgensen-Harmand-Unger 7 point match 18-08-2024.xg",
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

	p1 := stats.Player1 // Squire/J\u00f8rgensen
	p2 := stats.Player2 // Harmand/Unger

	t.Logf("=== Squire/J\u00f8rgensen (P1) ===")
	t.Logf("  PR=%.2f  decisions=%d  snowie_er=%.3f  errors=%d  blunders=%d  equity=%.3f  mwc=%.2f%%",
		p1.PR, p1.TotalDecisions, p1.SnowieER, p1.TotalErrors, p1.TotalBlunders, p1.TotalEquityError, p1.MWCLoss*100)
	t.Logf("  Checker: moves=%d  errors=%d  blunders=%d  equity=%.3f  mwc=%.2f%%",
		p1.CheckerDecisions, p1.CheckerErrors, p1.CheckerBlunders, p1.CheckerEquityError, p1.CheckerMWCLoss*100)
	t.Logf("  Doubles: decisions=%d  errors=%d  blunders=%d  equity=%.3f  mwc=%.2f%%",
		p1.DoubleDecisions, p1.DoubleErrors, p1.DoubleBlunders, p1.DoubleEquityError, p1.DoubleMWCLoss*100)
	t.Logf("  Takes:   decisions=%d  errors=%d  blunders=%d  equity=%.3f  mwc=%.2f%%",
		p1.TakeDecisions, p1.TakeErrors, p1.TakeBlunders, p1.TakeEquityError, p1.TakeMWCLoss*100)

	t.Logf("=== Harmand/Unger (P2) ===")
	t.Logf("  PR=%.2f  decisions=%d  snowie_er=%.3f  errors=%d  blunders=%d  equity=%.3f  mwc=%.2f%%",
		p2.PR, p2.TotalDecisions, p2.SnowieER, p2.TotalErrors, p2.TotalBlunders, p2.TotalEquityError, p2.MWCLoss*100)
	t.Logf("  Checker: moves=%d  errors=%d  blunders=%d  equity=%.3f  mwc=%.2f%%",
		p2.CheckerDecisions, p2.CheckerErrors, p2.CheckerBlunders, p2.CheckerEquityError, p2.CheckerMWCLoss*100)
	t.Logf("  Doubles: decisions=%d  errors=%d  blunders=%d  equity=%.3f  mwc=%.2f%%",
		p2.DoubleDecisions, p2.DoubleErrors, p2.DoubleBlunders, p2.DoubleEquityError, p2.DoubleMWCLoss*100)
	t.Logf("  Takes:   decisions=%d  errors=%d  blunders=%d  equity=%.3f  mwc=%.2f%%",
		p2.TakeDecisions, p2.TakeErrors, p2.TakeBlunders, p2.TakeEquityError, p2.TakeMWCLoss*100)
}
