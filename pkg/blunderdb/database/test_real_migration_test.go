package database

import (
	"testing"
)

func TestMigrationOnRealDB(t *testing.T) {
	d := NewDatabase()
	if err := d.OpenDatabase("c.db"); err != nil {
		t.Skip("c.db not available or migration failed: " + err.Error())
	}

	// Check how many positions now have non-zero best_move_equity_error
	var total, nonzero int
	d.db.QueryRow(`SELECT COUNT(*) FROM analysis WHERE best_move_equity_error IS NOT NULL`).Scan(&total)
	d.db.QueryRow(`SELECT COUNT(*) FROM analysis WHERE best_move_equity_error != 0`).Scan(&nonzero)
	t.Logf("total=%d, nonzero_error=%d (%.1f%%)", total, nonzero, float64(nonzero)/float64(total)*100)

	// Check version
	ver, _ := d.CheckDatabaseVersion()
	t.Logf("database version: %s", ver)

	// Run stats
	res, err := d.ComputeStats(StatsFilter{DecisionType: -1})
	if err != nil {
		t.Fatalf("ComputeStats: %v", err)
	}
	t.Logf("NumDecisions=%d, PRGlobal=%.1f, PRChecker=%.1f, PRCube=%.1f",
		res.Totals.NumDecisions, res.PRGlobal, res.PRChecker, res.PRCube)
}
