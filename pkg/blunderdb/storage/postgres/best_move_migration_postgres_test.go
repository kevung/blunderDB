//go:build postgres

package postgres_test

import (
	"context"
	"slices"
	"strconv"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestMigrate_BestMoveBackfillRewritesNothingElse upgrades a two-tenant
// library, FORCEd RLS on an owner without BYPASSRLS, whose generation of Go
// passes is already recorded and whose rows are up to date but for some
// best_move never derived. Migrate must fill those, journal nothing, leave
// every other row as it was (same xmin) and keep the match statistics; then a
// write of the blob alone, as an older binary makes it, clears the column.
func TestMigrate_BestMoveBackfillRewritesNothingElse(t *testing.T) {
	ctx := context.Background()
	s, conn, _ := openAsRLSOwner(t)
	ms := s.Matches()
	analysis := &domain.PositionAnalysis{AnalysisType: "CheckerMove", CheckerAnalysis: &domain.CheckerAnalysis{
		Moves: []domain.CheckerMove{{Index: 0, Move: "24/20 13/11", Equity: 0.1}}}}
	var seed []string
	pending := map[string]int64{}
	for i, scope := range []string{"1", "2"} {
		matchID, err := ms.Save(ctx, scope, &domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7})
		if err != nil {
			t.Fatal(err)
		}
		for j := range 2 {
			p := domain.InitializePosition()
			p.Board.Points[4+i].Checkers = 2 + j
			pid, err := s.Positions().Save(ctx, scope, &p)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Analyses().Save(ctx, scope, pid, analysis); err != nil {
				t.Fatal(err)
			}
			if j == 1 {
				pending[scope] = pid
			}
		}
		seed = append(seed,
			`INSERT INTO match_stats (tenant_id, match_id, seat, checker_moves) VALUES (`+scope+`, `+strconv.FormatInt(matchID, 10)+`, 1, 0)`,
			`UPDATE analysis SET best_move = NULL WHERE tenant_id = `+scope+` AND position_id = `+strconv.FormatInt(pending[scope], 10))
	}
	if err := s.ApplyRLS(ctx); err != nil {
		t.Fatalf("ApplyRLS: %v", err)
	}
	tables := []string{"analysis", "match_stats"}
	execUnforced(t, conn, tables, append(seed, `DELETE FROM metadata WHERE key = 'best_move_backfill'`))

	type state struct {
		xmins          []string
		journal, stats int
		pending        *string
	}
	read := func(scope string) (st state) {
		asTenant(t, conn, scope, func() {
			rows, err := conn.Query(ctx, `SELECT xmin::text FROM analysis WHERE position_id <> $1 ORDER BY id`, pending[scope])
			if err != nil {
				t.Fatal(err)
			}
			for rows.Next() {
				var x string
				if err := rows.Scan(&x); err != nil {
					t.Fatal(err)
				}
				st.xmins = append(st.xmins, x)
			}
			rows.Close()
			if err := conn.QueryRow(ctx, `SELECT (SELECT count(*) FROM change_log), (SELECT count(*) FROM match_stats),
				        (SELECT best_move FROM analysis WHERE position_id = $1)`, pending[scope]).Scan(&st.journal, &st.stats, &st.pending); err != nil {
				t.Fatal(err)
			}
		})
		return st
	}
	before := map[string]state{"1": read("1"), "2": read("2")}
	var seqBefore int64
	if err := conn.QueryRow(ctx, `SELECT last_value FROM change_log_seq`).Scan(&seqBefore); err != nil {
		t.Fatal(err)
	}

	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate under RLS: %v", err)
	}
	assertForced(t, conn, tables...)
	for scope, was := range before {
		now := read(scope)
		if was.pending != nil || now.pending == nil || *now.pending != "24/20 13/11" {
			t.Errorf("tenant %s: best_move %v before Migrate, %v after, want NULL then 24/20 13/11", scope, was.pending, now.pending)
		}
		if !slices.Equal(was.xmins, now.xmins) {
			t.Errorf("tenant %s: Migrate rewrote rows already derived: xmin %v, then %v", scope, was.xmins, now.xmins)
		}
		if was.journal != now.journal || was.stats != now.stats || was.stats == 0 {
			t.Errorf("tenant %s: change_log %d → %d, match_stats %d → %d, want both unchanged", scope, was.journal, now.journal, was.stats, now.stats)
		}
	}
	var seqAfter int64
	if err := conn.QueryRow(ctx, `SELECT last_value FROM change_log_seq`).Scan(&seqAfter); err != nil {
		t.Fatal(err)
	}
	if seqAfter != seqBefore {
		t.Errorf("change_log_seq %d → %d: the backfill journalled", seqBefore, seqAfter)
	}
	var gen, done string
	if err := conn.QueryRow(ctx, `SELECT (SELECT value FROM metadata WHERE key = 'go_backfills'),
	        (SELECT value FROM metadata WHERE key = 'best_move_backfill')`).Scan(&gen, &done); err != nil {
		t.Fatalf("read the backfill marks: %v", err)
	}
	if gen != "5" || done == "" {
		t.Errorf("go_backfills %q, best_move_backfill %q, want 5 and recorded", gen, done)
	}

	// An older binary's write, under FORCEd RLS: data alone. The trigger
	// clears the column and adds no journal entry of its own.
	asTenant(t, conn, "1", func() {
		// The journal entries one write of data takes, the trigger aside:
		// the pending row, whose best_move the write changes, does not fire it.
		var derived int64
		if err := conn.QueryRow(ctx, `SELECT position_id FROM analysis WHERE position_id <> $1 LIMIT 1`, pending["1"]).Scan(&derived); err != nil {
			t.Fatal(err)
		}
		seqTaken := func(stmt string, pid int64) int64 {
			t.Helper()
			var s0, s1 int64
			if err := conn.QueryRow(ctx, `SELECT last_value FROM change_log_seq`).Scan(&s0); err != nil {
				t.Fatal(err)
			}
			tag, err := conn.Exec(ctx, stmt, pid)
			if err != nil || tag.RowsAffected() != 1 {
				t.Fatalf("%s: %v (%d rows)", stmt, err, tag.RowsAffected())
			}
			if err := conn.QueryRow(ctx, `SELECT last_value FROM change_log_seq`).Scan(&s1); err != nil {
				t.Fatal(err)
			}
			return s1 - s0
		}
		plain := seqTaken(`UPDATE analysis SET data = data, best_move = 'x' WHERE position_id = $1`, pending["1"])
		fired := seqTaken(`UPDATE analysis SET data = data WHERE position_id = $1`, derived)
		var bm *string
		if err := conn.QueryRow(ctx, `SELECT best_move FROM analysis WHERE position_id = $1`, derived).Scan(&bm); err != nil {
			t.Fatal(err)
		}
		if bm != nil {
			t.Errorf("best_move after the blob alone was rewritten %q, want NULL", *bm)
		}
		if fired != plain {
			t.Errorf("a write the trigger answers takes %d journal sequence values, one it does not %d", fired, plain)
		}
	})
}
