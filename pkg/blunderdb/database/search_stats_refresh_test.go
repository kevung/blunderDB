package database

import (
	"path/filepath"
	"strings"
	"testing"
)

// statRows returns sqlite_stat1 as "tbl|idx|stat" strings, sorted; none when
// no ANALYZE has created the table yet.
func statRows(t *testing.T, d *Database) []string {
	t.Helper()
	var n int
	if err := d.conn().QueryRow(`SELECT count(*) FROM sqlite_master WHERE name = 'sqlite_stat1'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		return nil
	}
	rows, err := d.conn().Query(`SELECT tbl || '|' || COALESCE(idx,'') || '|' || stat FROM sqlite_stat1 ORDER BY tbl, idx`)
	if err != nil {
		t.Fatalf("read sqlite_stat1: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// importedFixture opens a fresh file database and imports testdata/test.xg
// into it: statistics as a first import leaves them, taken on empty tables.
func importedFixture(t *testing.T) *Database {
	t.Helper()
	d := NewDatabase()
	if err := d.SetupDatabase(filepath.Join(t.TempDir(), "stats.db")); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	if _, err := d.ImportXGMatch(filepath.Join("testdata", "test.xg")); err != nil {
		t.Fatalf("ImportXGMatch: %v", err)
	}
	return d
}

// TestRefreshSearchStatistics_SkipsASmallBatch: one match file is below
// statsRefreshMinPositions, so the batch ends without analysing anything.
func TestRefreshSearchStatistics_SkipsASmallBatch(t *testing.T) {
	d := importedFixture(t)
	if d.positionsSinceStats == 0 || d.positionsSinceStats >= statsRefreshMinPositions {
		t.Fatalf("fixture wrote %d positions, want 1..%d", d.positionsSinceStats, statsRefreshMinPositions-1)
	}
	before := statRows(t, d)
	d.RefreshSearchStatistics()
	if got := statRows(t, d); len(got) != len(before) {
		t.Errorf("a one-file batch refreshed the statistics: %d rows, %d before", len(got), len(before))
	}
}

// TestRefreshSearchStatistics_BoundedMatchesFullAnalyze: past the threshold
// the refresh analyses, resets its count, and on the fixture — smaller than
// analysis_limit — writes the very statistics a full ANALYZE writes for every
// non-empty table, so the search plans cannot change. (optimize leaves an
// empty table unanalysed; ANALYZE records it as "0 …".)
func TestRefreshSearchStatistics_BoundedMatchesFullAnalyze(t *testing.T) {
	d := importedFixture(t)
	d.positionsSinceStats = statsRefreshMinPositions
	d.RefreshSearchStatistics()
	bounded := statRows(t, d)
	if len(bounded) == 0 {
		t.Fatal("the refresh wrote no statistics")
	}
	if d.positionsSinceStats != 0 {
		t.Errorf("positionsSinceStats = %d after a refresh, want 0", d.positionsSinceStats)
	}

	if _, err := d.conn().Exec(`DELETE FROM sqlite_stat1; ANALYZE`); err != nil {
		t.Fatal(err)
	}
	full := map[string]string{}
	for _, r := range statRows(t, d) {
		k := r[:strings.LastIndex(r, "|")+1]
		full[k] = r[len(k):]
	}
	got := map[string]bool{}
	for _, r := range bounded {
		k := r[:strings.LastIndex(r, "|")+1]
		got[k] = true
		if want, ok := full[k]; !ok || want != r[len(k):] {
			t.Errorf("bounded refresh %q, full ANALYZE %q", r, want)
		}
	}
	for k, stat := range full {
		if !got[k] && !strings.HasPrefix(stat, "0 ") {
			t.Errorf("bounded refresh left a non-empty index unanalysed: %s%s", k, stat)
		}
	}
}
