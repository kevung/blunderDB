package sqlite

import (
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// demoCopy writes the demo library (three analysed matches) to a temporary file.
func demoCopy(t *testing.T) string {
	t.Helper()
	src, err := os.Open(filepath.Join("..", "..", "..", "..", "internal", "gui", "demo.db.gz"))
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	zr, err := gzip.NewReader(src)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "demo.db")
	dst, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(dst, zr); err != nil {
		t.Fatal(err)
	}
	if err := dst.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

// openQueryOnly opens path the way the read-only fallback does: one
// connection with query_only on.
func openQueryOnly(t *testing.T, path string) *Storage {
	t.Helper()
	db, err := sql.Open("sqlite", DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA query_only = ON`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return New(db)
}

func asJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func matchStatsRows(t *testing.T, st *Storage) int {
	t.Helper()
	var n int
	if err := st.sqlDB.QueryRow(`SELECT COUNT(*) FROM match_stats`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// Compute over the selection tables (writable) and over the decisions read
// directly (read-only, which can create no table) give the same result, for
// filters that take the match_stats path and filters that cannot.
func TestComputeSelectionMatchesDirectReadAndReadOnlyWritesNothing(t *testing.T) {
	ctx := context.Background()
	path := demoCopy(t)
	w, err := Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	ro := openQueryOnly(t, path)

	filters := []storage.StatsFilter{
		{DecisionType: -1},
		{DecisionType: 0},
		{DecisionType: 1},
		{DecisionType: -1, PlayerName: "Iris Okonkwo"},
		{DecisionType: -1, AnalysisEngine: "XG"},
		{DecisionType: -1, MinAnalysisDepth: 3},
		{DecisionType: 0, PlayerName: "Iris Okonkwo", MinAnalysisDepth: 2},
	}
	want := make([]string, len(filters))
	for i, f := range filters {
		res, err := w.Stats().Compute(ctx, "", f)
		if err != nil {
			t.Fatalf("writable Compute %+v: %v", f, err)
		}
		if i == 0 && res.Totals.NumDecisions == 0 {
			t.Fatal("demo library has no counted decision")
		}
		want[i] = asJSON(t, res)
	}

	// Read-only, with match_stats emptied by the writer: nothing to lean on,
	// nothing may be written, and the figures do not move.
	if err := sqlshared.InvalidateAllMatchStats(ctx, w.binder.shared(), ""); err != nil {
		t.Fatal(err)
	}
	for i, f := range filters {
		res, err := ro.Stats().Compute(ctx, "", f)
		if err != nil {
			t.Fatalf("read-only Compute %+v: %v", f, err)
		}
		if got := asJSON(t, res); got != want[i] {
			t.Errorf("filter %+v:\n read-only %s\n writable  %s", f, got, want[i])
		}
	}
	if n := matchStatsRows(t, w); n != 0 {
		t.Errorf("read-only Compute wrote %d match_stats rows", n)
	}

	// The `pr` token on a read-only connection measures the matches the table
	// lacks instead of dropping them.
	prAll := domain.SearchFilters{PlayerPRFilter: "pr<1000"}
	roCount, err := ro.Search().Count(ctx, "", prAll)
	if err != nil {
		t.Fatal(err)
	}
	if n := matchStatsRows(t, w); n != 0 {
		t.Errorf("read-only search wrote %d match_stats rows", n)
	}
	wCount, err := w.Search().Count(ctx, "", prAll)
	if err != nil {
		t.Fatal(err)
	}
	if roCount == 0 || roCount != wCount {
		t.Errorf("pr search: read-only %d, writable %d", roCount, wCount)
	}
}
