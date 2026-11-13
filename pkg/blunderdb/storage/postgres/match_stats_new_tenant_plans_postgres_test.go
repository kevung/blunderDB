//go:build postgres

package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// executedNode is the part of an EXPLAIN (ANALYZE, FORMAT JSON) node the
// check reads.
type executedNode struct {
	NodeType  string         `json:"Node Type"`
	Relation  string         `json:"Relation Name"`
	IndexName string         `json:"Index Name"`
	IndexCond string         `json:"Index Cond"`
	Loops     float64        `json:"Actual Loops"`
	Plans     []executedNode `json:"Plans"`
}

// rlsLibrary is an empty library with RLS applied: admin owns the tables, s
// and pool are bound as a role the policies bind, and rec records the
// statements s sends.
func rlsLibrary(t *testing.T) (ctx context.Context, admin *pgxpool.Pool, pool *pgxpool.Pool, s *Storage, rec *stmtRecorder) {
	t.Helper()
	ctx = context.Background()
	dsn := purgeTestDB(t)

	var err error
	admin, err = pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	owner := &Storage{binder: binder{db: admin}, pool: admin}
	if err := owner.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := owner.ApplyRLS(ctx); err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`CREATE ROLE stats_app LOGIN PASSWORD 'app'`,
		`GRANT USAGE, CREATE ON SCHEMA public TO stats_app`,
		`GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO stats_app`,
		`GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO stats_app`,
	} {
		if _, err := admin.Exec(ctx, stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.User, cfg.ConnConfig.Password = "stats_app", "app"
	configureRLSPool(cfg)
	rec = &stmtRecorder{}
	cfg.ConnConfig.Tracer = rec
	pool, err = pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	s = &Storage{binder: binder{db: pool}, pool: pool}
	return ctx, admin, pool, s, rec
}

// newTenantLibrary builds a library of 39 tenants (2 to 40) with RLS applied
// and analyzed, autovacuum off, and returns a store bound as a role the
// policies bind, whose statements rec records; importAll(1) then makes tenant 1
// the new tenant the statistics do not know.
func newTenantLibrary(t *testing.T) (ctx context.Context, admin *pgxpool.Pool, pool *pgxpool.Pool, s *Storage, rec *stmtRecorder, importAll func(int64), files []string) {
	t.Helper()
	ctx, admin, pool, s, rec = rlsLibrary(t)

	files = []string{"test.xg", "HsbtMarseille_main_ronde4_LamourDeCaslouGildas_UngerKevin_7p.xg",
		"Kev_GammonNetNormal_2026-10-06_7p.xg", "charlot1-charlot2_7p_2025-11-08-2305.xg"}
	importAll = func(tenant int64) {
		tctx := storage.WithTenant(ctx, tenant)
		for _, f := range files {
			src := ingest.Source{Path: filepath.Join("..", "..", "..", "..", "testdata", f)}
			if _, err := (ingest.XGImporter{S: s}).Import(tctx, strconv.FormatInt(tenant, 10), src, nil); err != nil {
				t.Fatal(err)
			}
		}
	}
	importAll(2)
	copyTenant(ctx, t, admin, 2, 3, 40)
	for _, table := range []string{"position", "move", "analysis", "game", "match", "match_stats"} {
		if _, err := admin.Exec(ctx, fmt.Sprintf(`ANALYZE %[1]s; ALTER TABLE %[1]s SET (autovacuum_enabled = false)`, table)); err != nil {
			t.Fatal(err)
		}
	}
	return ctx, admin, pool, s, rec, importAll, files
}

// The first import of a tenant into a library whose statistics predate it, as
// served: under row-level security, as a role the policies bind. Planned for
// its values, a statement of that tenant is estimated at one row per table, and
// the statistics of a match may then reach position, move or game through an
// index that names the tenant and nothing else: every row of the tenant read
// for one match. Each statement the refresh sends is replayed here as the
// engine runs it — prepared, under the settings of its own PinJoinOrder — and
// none may walk the tenant.
func TestMatchStatsOfANewTenantLookUpByKey(t *testing.T) {
	ctx, admin, pool, s, rec, importAll, files := newTenantLibrary(t)
	importAll(1)

	var matchIDs []int64
	rows, err := admin.Query(ctx, `SELECT id FROM match WHERE tenant_id = 1 ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		matchIDs = append(matchIDs, id)
	}
	if rows.Err() != nil || len(matchIDs) != len(files) {
		t.Fatalf("matches of the new tenant: %v, %v", matchIDs, rows.Err())
	}

	tctx := storage.WithTenant(ctx, 1)
	rec.record("refresh")
	for _, id := range matchIDs {
		if err := s.Stats().RefreshMatchStats(tctx, "1", []int64{id}); err != nil {
			t.Fatal(err)
		}
	}
	rec.record("")
	rec.mu.Lock()
	stmts := rec.stmts
	rec.mu.Unlock()

	conn, err := pool.Acquire(tctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	matchReads := 0
	for i, st := range stmts {
		// The reads: a write replayed would collide with the refresh's own.
		if !strings.HasPrefix(strings.TrimSpace(st.sql), "SELECT") {
			continue
		}
		plan, ok := explainAsTheEngineRunsIt(tctx, t, conn.Conn(), fmt.Sprintf("replay_%d", i), st)
		if !ok {
			continue
		}
		if strings.Contains(st.sql, "FROM game g") {
			matchReads++
		}
		for _, bad := range tenantWalks(plan) {
			t.Errorf("%s\n  in: %s", bad, oneLine(st.sql))
		}
	}
	if matchReads < 4*len(matchIDs) {
		t.Fatalf("only %d reads of a match's decisions replayed: the recording missed the refresh", matchReads)
	}
}

// explainAsTheEngineRunsIt executes st, prepared, inside a transaction under
// the engine's PinJoinOrder, and returns its executed plan. A statement on a
// temporary table of the refresh's own transaction cannot be replayed (false).
func explainAsTheEngineRunsIt(ctx context.Context, t *testing.T, conn *pgx.Conn, name string, st recordedStmt) (executedNode, bool) {
	t.Helper()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := (shared{db: tx}).PinJoinOrder(ctx); err != nil {
		t.Fatal(err)
	}
	ps, err := tx.Prepare(ctx, name, st.sql)
	if err != nil {
		return executedNode{}, false
	}
	// A utility statement takes no parameters: the values go in as literals,
	// which EXECUTE casts to the types the statement was prepared with.
	call := "EXPLAIN (ANALYZE, FORMAT JSON) EXECUTE " + name
	if len(ps.ParamOIDs) != len(st.args) {
		return executedNode{}, false
	}
	if len(st.args) > 0 {
		lits := make([]string, len(st.args))
		for i, a := range st.args {
			lit, ok := sqlLiteral(a)
			if !ok {
				return executedNode{}, false
			}
			lits[i] = lit
		}
		call += "(" + strings.Join(lits, ", ") + ")"
	}
	var out []byte
	if err := tx.QueryRow(ctx, call).Scan(&out); err != nil {
		t.Fatalf("replay of %s: %v", oneLine(st.sql), err)
	}
	var plans []struct {
		Plan executedNode `json:"Plan"`
	}
	if err := json.Unmarshal(out, &plans); err != nil || len(plans) == 0 {
		t.Fatalf("plan of %s: %v", oneLine(st.sql), err)
	}
	return plans[0].Plan, true
}

// sqlLiteral renders a recorded scalar argument as SQL (false: not a scalar).
func sqlLiteral(a any) (string, bool) {
	switch v := a.(type) {
	case nil:
		return "NULL", true
	case int, int8, int16, int32, int64, uint8, uint16, uint32, float32, float64, bool:
		return fmt.Sprint(v), true
	case string:
		return "'" + strings.ReplaceAll(v, "'", "''") + "'", true
	}
	return "", false
}

// tenantWalks lists the scans of a match's tables whose index condition names
// the tenant and nothing else: each reads all of the tenant's rows to keep
// those of the matches named.
func tenantWalks(n executedNode) []string {
	var bad []string
	tenantOnly := n.IndexCond != "" && strings.Count(n.IndexCond, " = ") == strings.Count(n.IndexCond, "tenant_id = ")
	switch n.Relation {
	case "position", "move", "analysis", "game":
		if tenantOnly {
			bad = append(bad, fmt.Sprintf("%s of %s via %s (cond: %s), %.0f loops",
				n.NodeType, n.Relation, n.IndexName, n.IndexCond, n.Loops))
		}
	}
	for _, c := range n.Plans {
		bad = append(bad, tenantWalks(c)...)
	}
	return bad
}

// copyTenant copies the library of tenant from under tenants first..last, its
// ids shifted to stay unique: a library large enough for the planner's
// estimates to be those of a served one.
func copyTenant(ctx context.Context, t *testing.T, db *pgxpool.Pool, from, first, last int) {
	t.Helper()
	ids := map[string]bool{"id": true, "match_id": true, "game_id": true, "position_id": true, "tournament_id": true}
	for _, table := range []string{"tournament", "match", "game", "position", "analysis", "move"} {
		var cols, exprs []string
		rows, err := db.Query(ctx, `SELECT column_name FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = $1 AND is_generated = 'NEVER' ORDER BY ordinal_position`, table)
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			var c string
			if err := rows.Scan(&c); err != nil {
				t.Fatal(err)
			}
			cols = append(cols, c)
			switch {
			case c == "tenant_id":
				exprs = append(exprs, "k")
			case ids[c]:
				exprs = append(exprs, c+" + k * 1000000000")
			default:
				exprs = append(exprs, c)
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		q := fmt.Sprintf(`INSERT INTO %s (%s) SELECT %s FROM %s, generate_series($2::bigint, $3::bigint) AS k WHERE tenant_id = $1`,
			table, strings.Join(cols, ", "), strings.Join(exprs, ", "), table)
		if _, err := db.Exec(ctx, q, from, first, last); err != nil {
			t.Fatalf("copy %s: %v", table, err)
		}
	}
}
