//go:build postgres

package postgres_test

import (
	"context"
	"net/url"
	"slices"
	"strconv"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
)

// TestMigrate_BestMoveBackfillRewritesNothingElse upgrades a two-tenant
// library at the deployed generation of Go passes, rows up to date but for
// some best_move never derived and one stale column a pass would repair,
// first as a superuser, then as an owner under FORCEd RLS. Each Migrate must
// fill those best_move, journal nothing, leave every other row as it was
// (same xmin) and keep the match statistics; then a write of the blob alone,
// as an older binary makes it, clears the column.
func TestMigrate_BestMoveBackfillRewritesNothingElse(t *testing.T) {
	ctx := context.Background()
	s, conn, ownerDSN := openAsRLSOwner(t)
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
			`UPDATE analysis SET best_move = NULL WHERE tenant_id = `+scope+` AND position_id = `+strconv.FormatInt(pending[scope], 10),
			// A stale column the decision recount would repair: a pass run
			// again rewrites its row.
			`UPDATE analysis SET cube_error = 12345 WHERE tenant_id = `+scope+` AND position_id <> `+strconv.FormatInt(pending[scope], 10))
	}
	if err := s.ApplyRLS(ctx); err != nil {
		t.Fatalf("ApplyRLS: %v", err)
	}
	tables := []string{"analysis", "match_stats"}
	// 077 run again lists the tenants the backfill reads.
	rerun077 := []string{`DELETE FROM schema_migrations WHERE version = '077_analysis_best_move'`}
	execUnforced(t, conn, tables, append(append(seed, rerun077...),
		// The generation the deployed libraries record: a binary that bumped
		// it would run every pass again over them.
		`UPDATE metadata SET value = '5' WHERE key = 'go_backfills'`))

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
	// The owner under FORCE sees no position without a tenant, so the journal
	// trigger would stay silent there anyway; a superuser, as a deployment
	// migrates, sees every row, and a backfill that journals or a pass run
	// again (match_stats invalidated, rows rewritten) shows under it.
	u, err := url.Parse(ownerDSN)
	if err != nil {
		t.Fatal(err)
	}
	u.User = url.UserPassword("test", "test")
	for _, m := range []struct {
		who     string
		migrate func() error
	}{
		{"superuser", func() error {
			super, err := pg.Open(ctx, u.String(), nil)
			if err != nil {
				return err
			}
			defer func() { _ = super.Close() }()
			return super.Migrate(ctx)
		}},
		{"owner under FORCE", func() error { return s.Migrate(ctx) }},
	} {
		if m.who != "superuser" {
			var reset []string
			for scope, pid := range pending {
				reset = append(reset, `UPDATE analysis SET best_move = NULL WHERE tenant_id = `+scope+` AND position_id = `+strconv.FormatInt(pid, 10))
			}
			execUnforced(t, conn, tables, append(reset, rerun077...))
		}
		before := map[string]state{"1": read("1"), "2": read("2")}
		var seqBefore int64
		if err := conn.QueryRow(ctx, `SELECT last_value FROM change_log_seq`).Scan(&seqBefore); err != nil {
			t.Fatal(err)
		}
		if err := m.migrate(); err != nil {
			t.Fatalf("Migrate as %s: %v", m.who, err)
		}
		assertForced(t, conn, tables...)
		for scope, was := range before {
			now := read(scope)
			if was.pending != nil || now.pending == nil || *now.pending != "24/20 13/11" {
				t.Errorf("%s, tenant %s: best_move %v before Migrate, %v after, want NULL then 24/20 13/11", m.who, scope, was.pending, now.pending)
			}
			if !slices.Equal(was.xmins, now.xmins) {
				t.Errorf("%s, tenant %s: Migrate rewrote rows already derived: xmin %v, then %v", m.who, scope, was.xmins, now.xmins)
			}
			if was.journal != now.journal || was.stats != now.stats || was.stats == 0 {
				t.Errorf("%s, tenant %s: change_log %d → %d, match_stats %d → %d, want both unchanged", m.who, scope, was.journal, now.journal, was.stats, now.stats)
			}
		}
		var seqAfter int64
		if err := conn.QueryRow(ctx, `SELECT last_value FROM change_log_seq`).Scan(&seqAfter); err != nil {
			t.Fatal(err)
		}
		if seqAfter != seqBefore {
			t.Errorf("%s: change_log_seq %d → %d: the backfill journalled", m.who, seqBefore, seqAfter)
		}
		var gen string
		var pendingCursor *string
		if err := conn.QueryRow(ctx, `SELECT (SELECT value FROM metadata WHERE key = 'go_backfills'),
		        (SELECT value FROM metadata WHERE key = 'best_move_backfill_pending')`).Scan(&gen, &pendingCursor); err != nil {
			t.Fatalf("read the backfill marks: %v", err)
		}
		if gen != "5" || pendingCursor != nil {
			t.Errorf("%s: go_backfills %q, best_move_backfill_pending %v, want 5 and closed", m.who, gen, pendingCursor)
		}
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

// TestRepairOfBestMoveAloneKeepsMatchStats: a row whose only stale column is
// best_move is corrected and counted, but match_stats summarises none of it,
// so the statistics stay.
func TestRepairOfBestMoveAloneKeepsMatchStats(t *testing.T) {
	ctx := context.Background()
	s, dsn := openMatchStore(t)
	conn := rawConn(t, dsn)
	matchID, err := s.Matches().Save(ctx, "", &domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7})
	if err != nil {
		t.Fatal(err)
	}
	p := domain.InitializePosition()
	pid, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Analyses().Save(ctx, "", pid, &domain.PositionAnalysis{AnalysisType: "CheckerMove", CheckerAnalysis: &domain.CheckerAnalysis{
		Moves: []domain.CheckerMove{{Index: 0, Move: "24/20 13/11", Equity: 0.1}}}}); err != nil {
		t.Fatal(err)
	}
	var tenant int64
	if err := conn.QueryRow(ctx, `UPDATE analysis SET best_move = 'stale' WHERE position_id = $1 RETURNING tenant_id`, pid).Scan(&tenant); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, `INSERT INTO match_stats (tenant_id, match_id, seat, checker_moves) VALUES ($1, $2, 1, 0)`, tenant, matchID); err != nil {
		t.Fatal(err)
	}
	n, err := s.Analyses().RepairDenormalisedColumns(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	var bm string
	if err := conn.QueryRow(ctx, `SELECT best_move FROM analysis WHERE position_id = $1`, pid).Scan(&bm); err != nil {
		t.Fatal(err)
	}
	if bm != "24/20 13/11" {
		t.Errorf("best_move after the repair %q, want 24/20 13/11", bm)
	}
	if stats := count(t, conn, `SELECT count(*) FROM match_stats`); n != 1 || stats != 1 {
		t.Errorf("repair counted %d rows and left %d match_stats rows, want 1 and 1", n, stats)
	}
}
