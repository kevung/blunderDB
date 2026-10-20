package ingest

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// The exporter scores each move it writes from the analysis it has just
// written, not by reading that analysis back: the per-move columns must come
// out as a rescoring of the exported file computes them. Real matches share
// their opening positions, so moves of several matches land on one row.
func TestExportedMoveScoresMatchStoredAnalyses(t *testing.T) {
	ctx := context.Background()
	src, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	for _, f := range []string{
		"test.xg",
		"charlot1-charlot2_7p_2025-11-08-2305.xg",
		"Kev_GammonNetNormal_2026-10-06_7p.xg",
	} {
		g, err := MapXG(filepath.Join("../../../testdata", f))
		if err != nil {
			t.Fatal(err)
		}
		tx, err := src.BeginTx(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := WriteMatch(ctx, tx, "", g, nil); err != nil {
			_ = tx.Rollback()
			t.Fatal(err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}

	for name, opts := range map[string]ExportOptions{
		"whole tenant":     WholeTenant(FormatSQLite),
		"without played":   func() ExportOptions { o := WholeTenant(FormatSQLite); o.PlayedMoves = false; return o }(),
		"without analysis": func() ExportOptions { o := WholeTenant(FormatSQLite); o.Analysis = false; return o }(),
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "out.db")
			if _, err := ExportSQLite(ctx, src, "", path, opts); err != nil {
				t.Fatal(err)
			}
			db, err := sql.Open("sqlite", sqlite.DSN(path))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			var moves, scored int
			if err := db.QueryRowContext(ctx,
				`SELECT COUNT(*), COUNT(decision_error_mp) FROM move`).Scan(&moves, &scored); err != nil {
				t.Fatal(err)
			}
			if moves == 0 || (opts.Analysis && scored == 0) {
				t.Fatalf("%d moves exported, %d scored: the fixture tests nothing", moves, scored)
			}
			changed, err := sqlite.RescorePlayedDecisions(ctx, db)
			if err != nil {
				t.Fatal(err)
			}
			if changed != 0 {
				t.Errorf("%d of %d exported moves are not scored as their position's stored analysis scores them", changed, moves)
			}
		})
	}
}
