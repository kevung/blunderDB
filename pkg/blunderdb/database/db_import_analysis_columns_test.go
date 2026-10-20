package database

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// analysisColumnsByHash reads the columns derived from each analysis, keyed
// by the position's Zobrist hash so two libraries compare whatever their ids.
func analysisColumnsByHash(t *testing.T, d *Database) map[string]string {
	t.Helper()
	rows, err := d.db.Query(`SELECT p.zobrist_hash, a.best_cube_action, a.cube_error, a.best_move_equity_error,
		a.player1_win_rate, a.player2_gammon_rate, a.is_forced, a.is_close_cube, a.analysis_engine, a.analysis_depth,
		p.is_cube_response
		FROM analysis a JOIN position p ON p.id = a.position_id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var hash int64
		var cols [10]sql.NullString
		dest := []any{&hash}
		for i := range cols {
			dest = append(dest, &cols[i])
		}
		if err := rows.Scan(dest...); err != nil {
			t.Fatal(err)
		}
		var b strings.Builder
		for _, c := range cols {
			fmt.Fprintf(&b, "%v|", c)
		}
		out[fmt.Sprint(hash)] = b.String()
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// Importing an exported library and transferring the same selection bring
// the same library: the analysis columns blunders and statistics are counted
// from are derived on both paths, for a position the receiver did not hold
// and for one it held without an analysis.
func TestImportDatabase_DerivesAnalysisColumnsAsTransfer(t *testing.T) {
	ctx := context.Background()
	src := newTestDBWithXG(t)
	opts := ingest.WholeLibraryOptions()
	path := exportTo(t, src, filepath.Join(t.TempDir(), "source.db"), opts)

	held, err := src.store.Positions().Load(ctx, "", 1)
	if err != nil {
		t.Fatal(err)
	}
	receiver := func() *Database {
		d := newTestDB(t)
		p := *held
		if _, err := d.SavePosition(&p); err != nil {
			t.Fatal(err)
		}
		return d
	}
	imported, transferred := receiver(), receiver()
	if _, err := imported.CommitImportDatabase(path); err != nil {
		t.Fatalf("import: %v", err)
	}
	if _, err := src.TransferTo(ctx, transferred, opts); err != nil {
		t.Fatalf("transfer: %v", err)
	}

	var unset int
	if err := imported.db.QueryRow(`SELECT count(*) FROM analysis
		WHERE analysis_engine IS NULL`).Scan(&unset); err != nil {
		t.Fatal(err)
	}
	if unset != 0 {
		t.Errorf("import left %d analyses without their derived columns", unset)
	}
	got, want := analysisColumnsByHash(t, imported), analysisColumnsByHash(t, transferred)
	if len(want) == 0 {
		t.Fatal("transfer brought no analysis")
	}
	if len(got) != len(want) {
		t.Errorf("import brought %d analyses, transfer %d", len(got), len(want))
	}
	for h, w := range want {
		if got[h] != w {
			t.Errorf("position %s: imported columns %q, transferred %q", h, got[h], w)
		}
	}
	gotStats, err := imported.GetDatabaseStats()
	if err != nil {
		t.Fatal(err)
	}
	wantStats, err := transferred.GetDatabaseStats()
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(gotStats) != fmt.Sprint(wantStats) {
		t.Errorf("imported library counts %v, transferred %v", gotStats, wantStats)
	}
	srcStats, err := src.GetDatabaseStats()
	if err != nil {
		t.Fatal(err)
	}
	if gotStats["blunder_count"] != srcStats["blunder_count"] {
		t.Errorf("imported library counts %v blunders, source %v", gotStats["blunder_count"], srcStats["blunder_count"])
	}
}

// A library whose analyses hold their blob alone, as the native import once
// wrote them, gets back by repair the columns a transfer derives.
func TestRepairAnalyses_FillsBlobOnlyRows(t *testing.T) {
	ctx := context.Background()
	src := newTestDBWithXG(t)
	opts := ingest.WholeLibraryOptions()
	broken, transferred := newTestDB(t), newTestDB(t)
	for _, d := range []*Database{broken, transferred} {
		if _, err := src.TransferTo(ctx, d, opts); err != nil {
			t.Fatalf("transfer: %v", err)
		}
	}
	if _, err := broken.db.Exec(`UPDATE analysis SET best_cube_action = NULL, cube_error = NULL,
		best_move_equity_error = NULL, player1_win_rate = NULL, player1_gammon_rate = NULL,
		player1_backgammon_rate = NULL, player2_win_rate = NULL, player2_gammon_rate = NULL,
		player2_backgammon_rate = NULL, is_forced = 0, is_close_cube = 0,
		analysis_engine = NULL, analysis_depth = NULL, creation_date = NULL`); err != nil {
		t.Fatal(err)
	}
	if _, err := broken.db.Exec(`UPDATE position SET is_cube_response = 0`); err != nil {
		t.Fatal(err)
	}
	if n, err := broken.RepairAnalyses(); err != nil || n == 0 {
		t.Fatalf("repair: %d rows, %v", n, err)
	}
	got, want := analysisColumnsByHash(t, broken), analysisColumnsByHash(t, transferred)
	for h, w := range want {
		if got[h] != w {
			t.Errorf("position %s: repaired columns %q, transferred %q", h, got[h], w)
		}
	}
	if n, err := broken.RepairAnalyses(); err != nil || n != 0 {
		t.Errorf("second repair: %d rows, %v; want 0", n, err)
	}
}
