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

// A library whose analyses hold their blob alone, provenance filled by the
// open-time pass, gets back by repair the columns a transfer derives.
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
	// The state the native import left: each analysis row written with its
	// blob alone, no cube-response flag, then the provenance pass every
	// open runs.
	for _, q := range []string{
		`CREATE TEMP TABLE blob_only AS SELECT position_id, data FROM analysis`,
		`DELETE FROM analysis`,
		`INSERT INTO analysis (position_id, data) SELECT position_id, data FROM blob_only`,
		`UPDATE position SET is_cube_response = 0`,
	} {
		if _, err := broken.db.Exec(q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	if err := broken.backfillAnalysisProvenance(ctx); err != nil {
		t.Fatal(err)
	}
	var unset, responses int
	if err := broken.db.QueryRow(`SELECT count(*) FROM analysis WHERE player1_win_rate IS NULL AND analysis_engine IS NOT NULL`).Scan(&unset); err != nil || unset == 0 {
		t.Fatalf("no blob-only row once provenance is filled (%d, %v)", unset, err)
	}
	if err := transferred.db.QueryRow(`SELECT count(*) FROM position WHERE is_cube_response = 1`).Scan(&responses); err != nil || responses == 0 {
		t.Fatalf("fixture holds no cube response (%d, %v)", responses, err)
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

// A position whose analysis here cannot be read keeps it: the import goes on,
// leaves that analysis as it was and counts it.
func TestImportDatabase_SkipsUnmergeableAnalysis(t *testing.T) {
	src := newTestDBWithXG(t)
	path := exportTo(t, src, filepath.Join(t.TempDir(), "source.db"), ingest.WholeLibraryOptions())
	dst := newTestDB(t)
	held, err := src.store.Positions().Load(context.Background(), "", 1)
	if err != nil {
		t.Fatal(err)
	}
	id, err := dst.SavePosition(held)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dst.db.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, x'00')`, id); err != nil {
		t.Fatal(err)
	}
	res, err := dst.CommitImportDatabase(path)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if res["analysesSkipped"] != 1 {
		t.Errorf("analysesSkipped = %v, want 1", res["analysesSkipped"])
	}
	var data []byte
	if err := dst.db.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, id).Scan(&data); err != nil || string(data) != "\x00" {
		t.Errorf("unreadable analysis rewritten: %x, %v", data, err)
	}
	var n int
	if err := dst.db.QueryRow(`SELECT count(*) FROM analysis`).Scan(&n); err != nil || n != 543 {
		t.Errorf("%d analyses after import, want 543 (%v)", n, err)
	}
}
