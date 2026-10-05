package sqlite

import (
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/report"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/statsequal"
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

// sameStatsJSON is statsequal.JSON for a test.
func sameStatsJSON(t *testing.T, a, b string) bool {
	t.Helper()
	ok, err := statsequal.JSON(a, b)
	if err != nil {
		t.Fatal(err)
	}
	return ok
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
		{DecisionType: -1, MatchLength: []int{5, 7}},
		{DecisionType: 1, DateFrom: "2000-01-01", DateTo: "2100-01-01", AnalysisEngine: "XG"},
		{DecisionType: -1, TournamentIDs: []int64{3}},
		{DecisionType: -1, DateFrom: "2025-05-18", DateTo: "2025-12-31"},
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
		if got := asJSON(t, res); !sameStatsJSON(t, got, want[i]) {
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

// The HTML report, the progression and breakdown views (all read from
// Compute) and the players table give a read-only reader the figures a writer
// sees, with match_stats emptied and nothing written back.
func TestReportAndPlayerTableReadOnlyMatchWritable(t *testing.T) {
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
		{DecisionType: -1, PlayerName: "Ada Fairweather"},
		{DecisionType: 1, MinAnalysisDepth: 2},
	}
	wantHTML := make([]string, len(filters))
	wantRows := make([]string, len(filters))
	for i, f := range filters {
		if wantHTML[i], err = report.Build(ctx, w, "", f, "fr", nil); err != nil {
			t.Fatalf("writable report %+v: %v", f, err)
		}
		rows, err := w.Stats().PlayerTable(ctx, "", f)
		if err != nil {
			t.Fatalf("writable PlayerTable %+v: %v", f, err)
		}
		wantRows[i] = asJSON(t, rows)
	}
	if err := sqlshared.InvalidateAllMatchStats(ctx, w.binder.shared(), ""); err != nil {
		t.Fatal(err)
	}
	for i, f := range filters {
		html, err := report.Build(ctx, ro, "", f, "fr", nil)
		if err != nil {
			t.Fatalf("read-only report %+v: %v", f, err)
		}
		if html != wantHTML[i] {
			t.Errorf("filter %+v: read-only report differs from the writable one", f)
		}
		rows, err := ro.Stats().PlayerTable(ctx, "", f)
		if err != nil {
			t.Fatalf("read-only PlayerTable %+v: %v", f, err)
		}
		if got := asJSON(t, rows); got != wantRows[i] {
			t.Errorf("filter %+v:\n read-only %s\n writable  %s", f, got, wantRows[i])
		}
	}
	if n := matchStatsRows(t, w); n != 0 {
		t.Errorf("read-only report wrote %d match_stats rows", n)
	}
}

// An aliased player is one person in the head-to-head and the window views:
// a match signed with the alias counts, and the alias against its canonical
// name is not two players.
func TestHeadToHeadAndWindowsFollowPlayerAliases(t *testing.T) {
	ctx := context.Background()
	w, err := Open(ctx, demoCopy(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	all := storage.StatsFilter{DecisionType: -1}
	before, err := w.Stats().HeadToHead(ctx, "", "Ada Fairweather", "Bram Vesterholt", all)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Matches) != 1 {
		t.Fatalf("demo head to head: %d matches, want 1", len(before.Matches))
	}
	winBefore, err := w.Stats().PRByWindow(ctx, "", storage.StatsFilter{DecisionType: -1, PlayerName: "Ada Fairweather"}, 1)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := w.sqlDB.Exec(`UPDATE match SET player2_name = 'A. Fairweather' WHERE player1_name = 'Bram Vesterholt'`); err != nil {
		t.Fatal(err)
	}
	if err := w.Aliases().Set(ctx, "", storage.AliasPlayer, "A. Fairweather", "Ada Fairweather"); err != nil {
		t.Fatal(err)
	}
	after, err := w.Stats().HeadToHead(ctx, "", "Ada Fairweather", "Bram Vesterholt", all)
	if err != nil {
		t.Fatal(err)
	}
	if asJSON(t, after.Matches) != asJSON(t, before.Matches) || after.PRA != before.PRA || after.WinsA != before.WinsA {
		t.Errorf("aliased head to head:\n got  %s\n want %s", asJSON(t, after), asJSON(t, before))
	}
	winAfter, err := w.Stats().PRByWindow(ctx, "", storage.StatsFilter{DecisionType: -1, PlayerName: "Ada Fairweather"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if asJSON(t, winAfter) != asJSON(t, winBefore) {
		t.Errorf("aliased windows:\n got  %s\n want %s", asJSON(t, winAfter), asJSON(t, winBefore))
	}
	if _, err := w.Stats().HeadToHead(ctx, "", "A. Fairweather", "Ada Fairweather", all); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("alias against its canonical: err = %v, want ErrInvalid", err)
	}
}

// The head-to-head, the sliding windows and the per-match series read
// match_stats; a read-only reader whose table is empty computes the same
// seat rows from the decisions instead of refusing, and writes nothing.
func TestHeadToHeadWindowsAndSeriesReadOnlyMatchWritable(t *testing.T) {
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
		{DecisionType: 1, PlayerName: "Ada Fairweather"},
	}
	read := func(st *Storage) []string {
		var out []string
		for _, f := range filters {
			h2h, err := st.Stats().HeadToHead(ctx, "", "Ada Fairweather", "Bram Vesterholt", f)
			if err != nil {
				t.Fatalf("HeadToHead %+v: %v", f, err)
			}
			win, err := st.Stats().PRByWindow(ctx, "", f, 3)
			if err != nil {
				t.Fatalf("PRByWindow %+v: %v", f, err)
			}
			series, err := st.Stats().MatchSeries(ctx, "", f)
			if err != nil {
				t.Fatalf("MatchSeries %+v: %v", f, err)
			}
			out = append(out, asJSON(t, h2h), asJSON(t, win), asJSON(t, series))
		}
		return out
	}
	want := read(w)
	if err := sqlshared.InvalidateAllMatchStats(ctx, w.binder.shared(), ""); err != nil {
		t.Fatal(err)
	}
	got := read(ro)
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("read %d:\n read-only %s\n writable  %s", i, got[i], want[i])
		}
	}
	if n := matchStatsRows(t, w); n != 0 {
		t.Errorf("read-only reads wrote %d match_stats rows", n)
	}
}

// TestComputeReadOnlyReadsCompleteCells: once a writer has filled match_stats
// for every match, a read-only reader sums its cells instead of reading
// every decision; a cell altered behind the table's back shows through.
func TestComputeReadOnlyReadsCompleteCells(t *testing.T) {
	ctx := context.Background()
	path := demoCopy(t)
	w, err := Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	f := storage.StatsFilter{DecisionType: -1}
	want, err := w.Stats().Compute(ctx, "", f)
	if err != nil {
		t.Fatal(err)
	}
	ro := openQueryOnly(t, path)
	got, err := ro.Stats().Compute(ctx, "", f)
	if err != nil {
		t.Fatal(err)
	}
	if !sameStatsJSON(t, asJSON(t, want), asJSON(t, got)) {
		t.Fatal("read-only Compute over complete cells differs from the writer's")
	}
	raw, err := sql.Open("sqlite", DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	if _, err := raw.ExecContext(ctx, `UPDATE match_stats_cell SET decisions = decisions + 1000
		WHERE (match_id, seat, decision_type, met_id, kind, k1, k2) =
		      (SELECT match_id, seat, decision_type, met_id, kind, k1, k2 FROM match_stats_cell WHERE kind = 1 LIMIT 1)`); err != nil {
		t.Fatal(err)
	}
	got, err = ro.Stats().Compute(ctx, "", f)
	if err != nil {
		t.Fatal(err)
	}
	if got.Totals.NumDecisions != want.Totals.NumDecisions+1000 {
		t.Errorf("read-only Compute did not read the cells: %d decisions, writer %d", got.Totals.NumDecisions, want.Totals.NumDecisions)
	}
}
