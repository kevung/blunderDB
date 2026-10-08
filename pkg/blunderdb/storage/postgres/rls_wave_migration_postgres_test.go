//go:build postgres

package postgres_test

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// openAsRLSOwner opens a fresh library whose tables belong to a role that is
// neither superuser nor BYPASSRLS — the only owner FORCEd row-level security
// binds, so the only one under which a migration that forgets to lift FORCE
// shows it. RLS is applied before it returns; conn is that owner's own raw
// connection, carrying no tenant, and ownerDSN reaches the same library.
func openAsRLSOwner(t *testing.T) (s *pg.Storage, conn *pgx.Conn, ownerDSN string) {
	t.Helper()
	ctx := context.Background()
	dsn := startPostgres(t)
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("admin connect: %v", err)
	}
	defer admin.Close(ctx)
	for _, stmt := range []string{
		`CREATE ROLE rls_owner LOGIN PASSWORD 'owner' NOSUPERUSER NOBYPASSRLS`,
		`GRANT USAGE, CREATE ON SCHEMA public TO rls_owner`,
	} {
		if _, err := admin.Exec(ctx, stmt); err != nil {
			t.Fatalf("setup owner (%s): %v", stmt, err)
		}
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse dsn: %v", err)
	}
	u.User = url.UserPassword("rls_owner", "owner")
	ownerDSN = u.String()

	s, err = pg.Open(ctx, ownerDSN, nil)
	if err != nil {
		t.Fatalf("Open as rls_owner: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	conn, err = pgx.Connect(ctx, ownerDSN)
	if err != nil {
		t.Fatalf("owner connect: %v", err)
	}
	t.Cleanup(func() { conn.Close(ctx) })
	return s, conn, ownerDSN
}

// asTenant runs fn on conn with app.tenant_id set to tenant, the GUC the
// tenant_isolation policy reads, and clears it afterwards.
func asTenant(t *testing.T, conn *pgx.Conn, tenant string, fn func()) {
	t.Helper()
	ctx := context.Background()
	if _, err := conn.Exec(ctx, `SELECT set_config('app.tenant_id', $1, false)`, tenant); err != nil {
		t.Fatalf("set tenant %s: %v", tenant, err)
	}
	defer func() {
		if _, err := conn.Exec(ctx, `SELECT set_config('app.tenant_id', '', false)`); err != nil {
			t.Fatalf("clear tenant: %v", err)
		}
	}()
	fn()
}

// execUnforced runs stmts on conn with FORCE lifted from tables, as the
// library owner rewinding its schema.
func execUnforced(t *testing.T, conn *pgx.Conn, tables []string, stmts []string) {
	t.Helper()
	ctx := context.Background()
	var all []string
	for _, tbl := range tables {
		all = append(all, `ALTER TABLE `+tbl+` NO FORCE ROW LEVEL SECURITY`)
	}
	all = append(all, stmts...)
	for _, tbl := range tables {
		all = append(all, `ALTER TABLE `+tbl+` FORCE ROW LEVEL SECURITY`)
	}
	for _, stmt := range all {
		if _, err := conn.Exec(ctx, stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
}

func assertForced(t *testing.T, conn *pgx.Conn, tables ...string) {
	t.Helper()
	for _, tbl := range tables {
		var forced bool
		if err := conn.QueryRow(context.Background(),
			`SELECT relforcerowsecurity FROM pg_class WHERE oid = to_regclass($1)`, tbl).Scan(&forced); err != nil {
			t.Fatalf("inspect %s: %v", tbl, err)
		}
		if !forced {
			t.Errorf("%s: RLS no longer FORCEd after the migration", tbl)
		}
	}
}

// TestMigrate_034_UnderRLS replays 034 on a two-tenant library whose RLS is
// FORCEd on its owner: every board and every label of both tenants must come
// through, and FORCE must be back on afterwards.
func TestMigrate_034_UnderRLS(t *testing.T) {
	ctx := context.Background()
	s, conn, ownerDSN := openAsRLSOwner(t)
	ms := s.Matches()

	type seeded struct {
		scope string
		pid   int64
		board domain.Board
		label string
	}
	var rows []seeded
	for i, scope := range []string{"1", "2"} {
		matchID, err := ms.Save(ctx, scope, &domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7})
		if err != nil {
			t.Fatal(err)
		}
		gameID, err := ms.CreateGame(ctx, scope, &domain.Game{MatchID: matchID, GameNumber: 1})
		if err != nil {
			t.Fatal(err)
		}
		// A fixed label and one registered for this tenant alone.
		for j, label := range []string{"Double, Take", "Doppel " + scope} {
			p := domain.InitializePosition()
			p.Board.Points[3+i*5+j].Checkers = 2
			pid, err := s.Positions().Save(ctx, scope, &p)
			if err != nil {
				t.Fatal(err)
			}
			saved, err := s.Positions().Load(ctx, scope, pid)
			if err != nil {
				t.Fatal(err)
			}
			mv := domain.Move{GameID: gameID, MoveNumber: int32(j + 1), MoveType: "cube", PositionID: pid, Player: 1, CubeAction: label}
			if _, err := ms.CreateMove(ctx, scope, &mv); err != nil {
				t.Fatal(err)
			}
			a := domain.PositionAnalysis{AnalysisType: "DoublingCube",
				DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{AnalysisDepth: "4-ply", BestCubeAction: label}}
			if err := s.Analyses().Save(ctx, scope, pid, &a); err != nil {
				t.Fatal(err)
			}
			rows = append(rows, seeded{scope, pid, saved.Board, label})
		}
	}
	if err := s.ApplyRLS(ctx); err != nil {
		t.Fatalf("ApplyRLS: %v", err)
	}
	// The rollback reads the registered labels before it drops action_label.
	execUnforced(t, conn, []string{"position", "move", "analysis"},
		append([]string{`ALTER TABLE action_label NO FORCE ROW LEVEL SECURITY`}, weightWaveRollback()...))

	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate under RLS: %v", err)
	}
	assertForced(t, conn, "position", "move", "analysis", "action_label")
	rls, err := pg.Open(ctx, ownerDSN, &storage.Options{EnableRLS: true})
	if err != nil {
		t.Fatalf("Open with RLS: %v", err)
	}
	defer rls.Close()

	for _, r := range rows {
		asTenant(t, conn, r.scope, func() {
			var cube, best string
			if err := conn.QueryRow(ctx, `SELECT `+sqlshared.TenantActionLabelOrEmptySQL("mv.cube_action")+`,
				        `+sqlshared.TenantActionLabelOrEmptySQL("a.best_cube_action")+`
				   FROM move mv JOIN analysis a ON a.position_id = mv.position_id AND a.tenant_id = mv.tenant_id
				  WHERE mv.position_id = $1`, r.pid).Scan(&cube, &best); err != nil {
				t.Fatalf("tenant %s, position %d: %v", r.scope, r.pid, err)
			}
			if cube != r.label || best != r.label {
				t.Errorf("tenant %s, position %d: labels %q/%q after 034, want %q", r.scope, r.pid, cube, best, r.label)
			}
		})
		n, err := storage.ParseTenant(r.scope)
		if err != nil {
			t.Fatal(err)
		}
		p, err := rls.Positions().Load(storage.WithTenant(ctx, n), r.scope, r.pid)
		if err != nil {
			t.Fatalf("Load %d: %v", r.pid, err)
		}
		if p.Board != r.board {
			t.Errorf("tenant %s, position %d: board after 034 differs from the board before", r.scope, r.pid)
		}
	}
}

// TestMigrate_032_MatchDateUnderRLS replays 032's match_date backfill on a
// library whose RLS is FORCEd on its owner: the position must be dated.
func TestMigrate_032_MatchDateUnderRLS(t *testing.T) {
	ctx := context.Background()
	s, conn, _ := openAsRLSOwner(t)
	ms := s.Matches()
	date := time.Date(2024, 3, 9, 0, 0, 0, 0, time.UTC)
	matchID, err := ms.Save(ctx, "1", &domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7, MatchDate: date})
	if err != nil {
		t.Fatal(err)
	}
	gameID, err := ms.CreateGame(ctx, "1", &domain.Game{MatchID: matchID, GameNumber: 1})
	if err != nil {
		t.Fatal(err)
	}
	p := domain.InitializePosition()
	pid, err := s.Positions().Save(ctx, "1", &p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ms.CreateMove(ctx, "1", &domain.Move{GameID: gameID, MoveNumber: 1, MoveType: "checker", PositionID: pid, Player: 1}); err != nil {
		t.Fatal(err)
	}
	if err := s.ApplyRLS(ctx); err != nil {
		t.Fatalf("ApplyRLS: %v", err)
	}
	execUnforced(t, conn, []string{"position"}, []string{
		`UPDATE position SET match_date = NULL`,
		`DELETE FROM schema_migrations WHERE version = '032_large_library_wave'`,
	})

	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate under RLS: %v", err)
	}
	assertForced(t, conn, "position", "move", "game", "match")
	asTenant(t, conn, "1", func() {
		var got *time.Time
		if err := conn.QueryRow(ctx, `SELECT match_date FROM position WHERE id = $1`, pid).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got == nil || !got.Equal(date) {
			t.Errorf("position.match_date = %v after 032, want %v", got, date)
		}
	})
}

// TestMETForeignKeyRejectsCrossTenant: analysis.met_id names a match equity
// table of the analysis's own tenant, and deleting that table clears met_id
// alone.
func TestMETForeignKeyRejectsCrossTenant(t *testing.T) {
	ctx := context.Background()
	dsn := startPostgres(t)
	s, err := pg.Open(ctx, dsn, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	var metID int64
	if err := conn.QueryRow(ctx, `INSERT INTO match_equity_table (tenant_id, name, digest, source)
		VALUES (1, 'Kazaross', 'd1', 'k.met') RETURNING id`).Scan(&metID); err != nil {
		t.Fatal(err)
	}
	analysed := map[string]int64{}
	for _, scope := range []string{"1", "2"} {
		p := domain.InitializePosition()
		pid, err := s.Positions().Save(ctx, scope, &p)
		if err != nil {
			t.Fatal(err)
		}
		a := domain.PositionAnalysis{AnalysisType: "DoublingCube",
			DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{AnalysisDepth: "4-ply", BestCubeAction: "No Double"}}
		if err := s.Analyses().Save(ctx, scope, pid, &a); err != nil {
			t.Fatal(err)
		}
		analysed[scope] = pid
	}

	_, err = conn.Exec(ctx, `UPDATE analysis SET met_id = $1 WHERE tenant_id = 2 AND position_id = $2`, metID, analysed["2"])
	if err == nil || !strings.Contains(err.Error(), "foreign key") {
		t.Fatalf("tenant 2 analysis pointing at tenant 1's MET: err = %v, want a foreign key violation", err)
	}

	if _, err := conn.Exec(ctx, `UPDATE analysis SET met_id = $1 WHERE tenant_id = 1 AND position_id = $2`, metID, analysed["1"]); err != nil {
		t.Fatalf("same-tenant met_id: %v", err)
	}
	if _, err := conn.Exec(ctx, `DELETE FROM match_equity_table WHERE id = $1`, metID); err != nil {
		t.Fatalf("delete MET: %v", err)
	}
	var got *int64
	var tenant int64
	if err := conn.QueryRow(ctx, `SELECT met_id, tenant_id FROM analysis WHERE tenant_id = 1 AND position_id = $1`, analysed["1"]).Scan(&got, &tenant); err != nil {
		t.Fatalf("analysis after the MET's deletion: %v", err)
	}
	if got != nil || tenant != 1 {
		t.Errorf("after deleting the MET: met_id = %v, tenant_id = %d; want NULL, 1", got, tenant)
	}
}

// TestActionLabelReadStaysInTenant: a move whose code names a label another
// tenant registered reads as no label, never as that tenant's text.
func TestActionLabelReadStaysInTenant(t *testing.T) {
	ctx := context.Background()
	dsn := startPostgres(t)
	s, err := pg.Open(ctx, dsn, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	ms := s.Matches()
	pids := map[string]int64{}
	for _, scope := range []string{"1", "2"} {
		matchID, err := ms.Save(ctx, scope, &domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7})
		if err != nil {
			t.Fatal(err)
		}
		gameID, err := ms.CreateGame(ctx, scope, &domain.Game{MatchID: matchID, GameNumber: 1})
		if err != nil {
			t.Fatal(err)
		}
		p := domain.InitializePosition()
		pid, err := s.Positions().Save(ctx, scope, &p)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ms.CreateMove(ctx, scope, &domain.Move{GameID: gameID, MoveNumber: 1, MoveType: "cube",
			PositionID: pid, Player: 1, CubeAction: "Secret of " + scope}); err != nil {
			t.Fatal(err)
		}
		pids[scope] = pid
	}
	var code int64
	if err := conn.QueryRow(ctx, `SELECT code FROM action_label WHERE tenant_id = 1 AND label = 'Secret of 1'`).Scan(&code); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, `UPDATE move SET cube_action = $1 WHERE tenant_id = 2`, code); err != nil {
		t.Fatal(err)
	}

	got, err := ms.MovesByPositions(ctx, "2", []int64{pids["2"]})
	if err != nil {
		t.Fatal(err)
	}
	for _, mvs := range got {
		for _, mv := range mvs {
			if mv.CubeAction != "" {
				t.Errorf("tenant 2 reads cube action %q through tenant 1's code, want none", mv.CubeAction)
			}
		}
	}
	if len(got) != 1 {
		t.Fatalf("read moves for %d positions, want 1", len(got))
	}
}

// TestMigrate_GoBackfillsUnderRLS runs the Go-side passes of Migrate on a
// two-tenant library whose RLS is FORCEd on an owner without BYPASSRLS: the
// provenance backfill must reach every analysis of both tenants and drop the
// match stats it invalidates, the match_stats repair must drop the rows of
// an older shape, and FORCE must be back on afterwards.
func TestMigrate_GoBackfillsUnderRLS(t *testing.T) {
	ctx := context.Background()
	s, conn, ownerDSN := openAsRLSOwner(t)
	ms := s.Matches()
	var seed []string
	for i, scope := range []string{"1", "2"} {
		matchID, err := ms.Save(ctx, scope, &domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7})
		if err != nil {
			t.Fatal(err)
		}
		gameID, err := ms.CreateGame(ctx, scope, &domain.Game{MatchID: matchID, GameNumber: 1})
		if err != nil {
			t.Fatal(err)
		}
		idle, err := ms.Save(ctx, scope, &domain.Match{Player1Name: "Carol", Player2Name: "Dan", MatchLength: 5})
		if err != nil {
			t.Fatal(err)
		}
		p := domain.InitializePosition()
		p.Board.Points[4+i].Checkers = 2
		pid, err := s.Positions().Save(ctx, scope, &p)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ms.CreateMove(ctx, scope, &domain.Move{GameID: gameID, MoveNumber: 1, MoveType: "cube", PositionID: pid, Player: 1}); err != nil {
			t.Fatal(err)
		}
		a := domain.PositionAnalysis{AnalysisType: "DoublingCube",
			DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{AnalysisDepth: "4-ply", BestCubeAction: "No Double, Take"}}
		if err := s.Analyses().Save(ctx, scope, pid, &a); err != nil {
			t.Fatal(err)
		}
		// A current-shape row on the analysed match, an older-shape row
		// (checker_moves NULL) on the match without analysis.
		seed = append(seed,
			`INSERT INTO match_stats (tenant_id, match_id, seat, checker_moves) VALUES (`+scope+`, `+strconv.FormatInt(matchID, 10)+`, 1, 0)`,
			`INSERT INTO match_stats (tenant_id, match_id, seat) VALUES (`+scope+`, `+strconv.FormatInt(idle, 10)+`, 1)`)
	}
	if err := s.ApplyRLS(ctx); err != nil {
		t.Fatalf("ApplyRLS: %v", err)
	}
	tables := []string{"analysis", "move", "game", "match_stats", "match_stats_cell"}
	execUnforced(t, conn, tables, append(seed,
		`UPDATE analysis SET analysis_engine = NULL, analysis_depth = NULL, creation_date = NULL`,
		`DELETE FROM metadata WHERE key = 'go_backfills'`))

	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate under RLS: %v", err)
	}
	assertForced(t, conn, tables...)
	for _, scope := range []string{"1", "2"} {
		asTenant(t, conn, scope, func() {
			var analyses, unset, stats int
			if err := conn.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE analysis_engine IS NULL),
				        (SELECT count(*) FROM match_stats) FROM analysis`).Scan(&analyses, &unset, &stats); err != nil {
				t.Fatal(err)
			}
			if analyses == 0 || unset != 0 {
				t.Errorf("tenant %s: %d of %d analyses without provenance after Migrate", scope, unset, analyses)
			}
			if stats != 0 {
				t.Errorf("tenant %s: %d match_stats rows survive Migrate, want the invalidated and the older-shape rows dropped", scope, stats)
			}
		})
	}
	var marker string
	if err := conn.QueryRow(ctx, `SELECT value FROM metadata WHERE key = 'go_backfills'`).Scan(&marker); err != nil {
		t.Fatalf("go_backfills not recorded after a complete pass: %v", err)
	}

	// Once recorded, Migrate lifts FORCE on nothing: behind a transaction
	// holding analysis, it returns well before the lock timeout.
	holder, err := pgx.Connect(ctx, ownerDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Close(ctx)
	hold := func() pgx.Tx {
		tx, err := holder.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, `LOCK TABLE analysis IN ACCESS SHARE MODE`); err != nil {
			t.Fatal(err)
		}
		return tx
	}
	tx := hold()
	start := time.Now()
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Errorf("second Migrate took %v behind a reader of analysis: it waited for a lock", d)
	}
	_ = tx.Rollback(ctx)

	// Without the record, a busy table defers the pass instead of failing
	// Migrate, and the record stays unwritten.
	if _, err := conn.Exec(ctx, `DELETE FROM metadata WHERE key = 'go_backfills'`); err != nil {
		t.Fatal(err)
	}
	tx = hold()
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate behind a busy table: %v", err)
	}
	_ = tx.Rollback(ctx)
	var n int
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM metadata WHERE key = 'go_backfills'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Error("go_backfills recorded although the pass gave way to a busy table")
	}
	assertForced(t, conn, tables...)
}

// TestMigrate_DecisionRecountUnderRLS runs the decision recount of Migrate on
// a two-tenant library whose RLS is FORCEd on an owner without BYPASSRLS, as
// an upgrade from generation 1 meets it: the moves of both tenants lost their
// own error, an analysis column holds a value of the old rules, and a
// match_stats row survived the tenant-less DELETE of 042 and 043. Every
// tenant must come out scored, its stale row dropped, FORCE back on.
func TestMigrate_DecisionRecountUnderRLS(t *testing.T) {
	ctx := context.Background()
	s, conn, _ := openAsRLSOwner(t)
	ms := s.Matches()
	var seed []string
	for i, scope := range []string{"1", "2"} {
		matchID, err := ms.Save(ctx, scope, &domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7})
		if err != nil {
			t.Fatal(err)
		}
		gameID, err := ms.CreateGame(ctx, scope, &domain.Game{MatchID: matchID, GameNumber: 1})
		if err != nil {
			t.Fatal(err)
		}
		p := domain.InitializePosition()
		p.DecisionType = domain.CubeAction
		p.Board.Points[4+i].Checkers = 2
		pid, err := s.Positions().Save(ctx, scope, &p)
		if err != nil {
			t.Fatal(err)
		}
		a := domain.PositionAnalysis{AnalysisType: "DoublingCube", DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
			CubefulNoDoubleEquity: 0.500, CubefulNoDoubleError: 0.300,
			CubefulDoubleTakeEquity: 0.800, CubefulDoublePassEquity: 1.000, CubefulDoublePassError: 0.200}}
		if err := s.Analyses().Save(ctx, scope, pid, &a); err != nil {
			t.Fatal(err)
		}
		if _, err := ms.CreateMove(ctx, scope, &domain.Move{GameID: gameID, MoveNumber: 1, MoveType: "cube",
			PositionID: pid, Player: 1, CubeAction: "No Double"}); err != nil {
			t.Fatal(err)
		}
		seed = append(seed,
			`INSERT INTO match_stats (tenant_id, match_id, seat, checker_moves) VALUES (`+scope+`, `+strconv.FormatInt(matchID, 10)+`, 1, 0)`)
	}
	if err := s.ApplyRLS(ctx); err != nil {
		t.Fatalf("ApplyRLS: %v", err)
	}
	tables := []string{"analysis", "move", "match_stats"}
	execUnforced(t, conn, tables, append(seed,
		`UPDATE move SET decision_error_mp = NULL, is_close_cube = 0`,
		`UPDATE analysis SET is_close_cube = FALSE`,
		`UPDATE metadata SET value = '1' WHERE key = 'go_backfills'`))

	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate under RLS: %v", err)
	}
	assertForced(t, conn, tables...)
	for _, scope := range []string{"1", "2"} {
		asTenant(t, conn, scope, func() {
			var errMP *int64
			var moveClose int
			var analysisClose bool
			var stats int
			if err := conn.QueryRow(ctx, `SELECT mv.decision_error_mp, mv.is_close_cube, a.is_close_cube,
				        (SELECT count(*) FROM match_stats)
				   FROM move mv JOIN analysis a ON a.position_id = mv.position_id`).Scan(&errMP, &moveClose, &analysisClose, &stats); err != nil {
				t.Fatal(err)
			}
			if errMP == nil || *errMP != 300 || moveClose != 1 {
				t.Errorf("tenant %s: move scored %v, close %d after Migrate; want 300, 1", scope, errMP, moveClose)
			}
			if !analysisClose {
				t.Errorf("tenant %s: analysis.is_close_cube not recomputed", scope)
			}
			if stats != 0 {
				t.Errorf("tenant %s: %d match_stats rows survive Migrate, want them dropped", scope, stats)
			}
		})
	}
}

// TestMigrate_AnsweredDoublesUnderRLS upgrades a two-tenant library of
// generation 2 whose transcribed takes stand on the answerer's own redouble
// row: each tenant's take must land on the ownerless cube, and a row that
// tenant still holds by a comment must survive the purge — a retention
// check that cannot see the tenant's rows would drop it.
func TestMigrate_AnsweredDoublesUnderRLS(t *testing.T) {
	ctx := context.Background()
	s, conn, _ := openAsRLSOwner(t)
	ms := s.Matches()
	takes := map[string]int64{}
	owned := map[string]int64{}
	for i, scope := range []string{"1", "2"} {
		matchID, err := ms.Save(ctx, scope, &domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7})
		if err != nil {
			t.Fatal(err)
		}
		gameID, err := ms.CreateGame(ctx, scope, &domain.Game{MatchID: matchID, GameNumber: 1})
		if err != nil {
			t.Fatal(err)
		}
		p := domain.InitializePosition()
		p.DecisionType = domain.CubeAction
		p.Board.Points[4+i].Checkers = 2
		p.PlayerOnRoll = domain.Black
		p.Cube = domain.Cube{Owner: domain.Black, Value: 1}
		pid, err := s.Positions().Save(ctx, scope, &p)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.Comments().Add(ctx, scope, pid, "redouble"); err != nil {
			t.Fatal(err)
		}
		mv, err := ms.CreateMove(ctx, scope, &domain.Move{GameID: gameID, MoveNumber: 1, MoveType: "cube",
			PositionID: pid, Player: -1, CubeAction: "Take"})
		if err != nil {
			t.Fatal(err)
		}
		takes[scope], owned[scope] = mv, pid
		// Response rows: gammonNet's verdict goes, XG's stays.
		for j, label := range []string{"gammonNet v1.6.0", "XG"} {
			r := p
			r.Board.Points[10+j].Checkers = 1
			r.Cube.Owner = domain.None
			rid, err := s.Positions().Save(ctx, scope, &r)
			if err != nil {
				t.Fatal(err)
			}
			a := domain.PositionAnalysis{AnalysisType: "DoublingCube", AnalysisEngineVersion: label,
				DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{AnalysisDepth: "0-ply", AnalysisEngine: label}}
			if err := s.Analyses().Save(ctx, scope, rid, &a); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := s.ApplyRLS(ctx); err != nil {
		t.Fatalf("ApplyRLS: %v", err)
	}
	execUnforced(t, conn, nil, []string{`UPDATE metadata SET value = '2' WHERE key = 'go_backfills'`})

	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate under RLS: %v", err)
	}
	assertForced(t, conn, "position", "move", "comment")
	for _, scope := range []string{"1", "2"} {
		asTenant(t, conn, scope, func() {
			var owner int
			var held int
			if err := conn.QueryRow(ctx, `SELECT p.cube_owner, (SELECT count(*) FROM position WHERE id = $2)
				   FROM move mv JOIN position p ON p.id = mv.position_id WHERE mv.id = $1`,
				takes[scope], owned[scope]).Scan(&owner, &held); err != nil {
				t.Fatal(err)
			}
			if owner != int(domain.None) {
				t.Errorf("tenant %s: take stands on cube owner %d, want none", scope, owner)
			}
			if held != 1 {
				t.Errorf("tenant %s: the commented redouble row was purged", scope)
			}
			var engines string
			if err := conn.QueryRow(ctx, `SELECT string_agg(a.analysis_engine, ',') FROM analysis a
				JOIN position p ON p.id = a.position_id WHERE p.cube_owner = -1`).Scan(&engines); err != nil {
				t.Fatal(err)
			}
			if engines != "XG" {
				t.Errorf("tenant %s: analyses left on response rows %q, want XG's alone", scope, engines)
			}
		})
	}
}
