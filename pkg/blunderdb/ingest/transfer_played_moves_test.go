package ingest

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// derivedColumns snapshots, position by position (keyed by Zobrist hash), the
// columns derived from an analysis and the moves played at it, then the
// per-seat tallies of the matches.
func derivedColumns(t *testing.T, path string) string {
	t.Helper()
	ctx := context.Background()
	db, err := sql.Open("sqlite", sqlite.DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var b strings.Builder
	dump := func(q string) {
		rows, err := db.QueryContext(ctx, q)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		cols, _ := rows.Columns()
		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				t.Fatal(err)
			}
			fmt.Fprintln(&b, vals...)
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
	}
	dump(`SELECT p.zobrist_hash, p.is_cube_response, a.best_move_equity_error, a.cube_error, a.is_forced, a.is_close_cube
	      FROM position p LEFT JOIN analysis a ON a.position_id = p.id ORDER BY p.zobrist_hash`)
	dump(`SELECT p.zobrist_hash, mv.move_number, mv.error_mp, mv.decision_error_mp, mv.is_close_cube
	      FROM move mv JOIN position p ON p.id = mv.position_id ORDER BY p.zobrist_hash, mv.move_number, mv.checker_move`)
	dump(`SELECT seat, SUM(blunders), SUM(error_mp), SUM(decisions) FROM match_stats GROUP BY seat ORDER BY seat`)
	return b.String()
}

// TestTransfer_WithoutPlayedMovesDerivesLikeWithThem: a transfer that carries
// the matches but does not fold the played moves into the analyses ends with
// the same derived columns as one that does — the moves arrive after the
// analyses and must still count.
func TestTransfer_WithoutPlayedMovesDerivesLikeWithThem(t *testing.T) {
	ctx := context.Background()
	src := seedLibrary(t, "test.xg", "src")
	snap := map[bool]string{}
	for _, folded := range []bool{true, false} {
		path := filepath.Join(t.TempDir(), "dst.db")
		dst, err := sqlite.Open(ctx, path, nil)
		if err != nil {
			t.Fatal(err)
		}
		opts := WholeLibraryOptions()
		opts.IncludePlayedMoves = folded
		if _, err := Transfer(ctx, TransferEnd{S: src}, TransferEnd{S: dst}, opts, nil); err != nil {
			t.Fatalf("transfer (played moves %v): %v", folded, err)
		}
		if err := dst.Close(); err != nil {
			t.Fatal(err)
		}
		snap[folded] = derivedColumns(t, path)
	}
	if !strings.Contains(snap[true], "\n") {
		t.Fatal("empty snapshot")
	}
	if snap[true] != snap[false] {
		t.Errorf("derived columns differ without the played moves:\nwith:\n%s\nwithout:\n%s", snap[true], snap[false])
	}
}
