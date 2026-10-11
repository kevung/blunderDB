package database

import (
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/maintenance"
)

// Opening a library computes nothing (ADR-0088, ADR-0007): a match whose
// statistics are missing stays missing across an open, counted on the
// stats.matches line, until that line is run.
func TestOpen_LeavesMissingMatchStatsToMaintenance(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(tempDir(t), "missing_stats.db")
	d := NewDatabase()
	if err := d.SetupDatabase(dbPath); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	if _, err := d.ImportXGMatch(filepath.Join("testdata", "test.xg")); err != nil {
		t.Fatalf("ImportXGMatch: %v", err)
	}
	if _, err := d.db.Exec(`DELETE FROM match_stats`); err != nil {
		t.Fatal(err)
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}

	d = NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	missing := func() int64 {
		t.Helper()
		lines, err := d.MaintenanceStatus()
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range lines {
			if l.Kind == maintenance.StatsMatches && l.Count != nil {
				return *l.Count
			}
		}
		t.Fatal("no free count on stats.matches")
		return 0
	}
	if n := missing(); n != 1 {
		t.Fatalf("after open: %d match(es) without statistics, want 1 (the open filled them)", n)
	}
	r, err := d.RunMaintenance(string(maintenance.StatsMatches))
	if err != nil || r.Changed != 1 {
		t.Fatalf("run: %+v, %v", r, err)
	}
	if n := missing(); n != 0 {
		t.Errorf("after the run: %d match(es) without statistics", n)
	}
}
