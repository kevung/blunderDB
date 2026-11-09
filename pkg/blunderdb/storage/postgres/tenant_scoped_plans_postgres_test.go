//go:build postgres

package postgres

import (
	"context"
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type recordedStmt struct {
	phase string
	sql   string
	args  []any
}

// stmtRecorder keeps the statements sent while phase is set, each with the
// phase that sent it.
type stmtRecorder struct {
	mu    sync.Mutex
	phase string
	stmts []recordedStmt
}

func (r *stmtRecorder) record(phase string) {
	r.mu.Lock()
	r.phase = phase
	r.mu.Unlock()
}

func (r *stmtRecorder) TraceQueryStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.phase != "" {
		r.stmts = append(r.stmts, recordedStmt{r.phase, d.SQL, d.Args})
	}
	return ctx
}

func (r *stmtRecorder) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

var explainable = regexp.MustCompile(`(?is)^\s*(SELECT|WITH|INSERT|UPDATE|DELETE)\b`)

// planNode is the part of an EXPLAIN (FORMAT JSON) node the check reads.
type planNode struct {
	NodeType  string     `json:"Node Type"`
	Relation  string     `json:"Relation Name"`
	IndexName string     `json:"Index Name"`
	IndexCond string     `json:"Index Cond"`
	Plans     []planNode `json:"Plans"`
}

// The keys of position, move and analysis lead with tenant_id and none is on
// id or position_id alone: a read of one tenant outside row-level security
// reaches its rows in a time bounded by that tenant only if each lookup names
// the tenant. A lookup that does not walks every tenant's entries of the
// index, once per row of the outer side — the import then slows down with the
// library, quadratically; a search, a training session or an Anki review
// slows down with the other tenants' libraries. Every statement an import,
// the statistics, the search (every token of the shared query corpus), the
// training, the Anki decks and their export, the counts and the maintenance
// of one tenant send is planned here with another tenant present, under the
// settings of withJoinOrderPlan; no lookup into those tables may
// go through a tenant-leading index without the tenant in its condition, nor
// fall back to a sequential scan.
func TestTenantReadsNameTheirTenantWithoutRLS(t *testing.T) {
	ctx := context.Background()
	dsn := purgeTestDB(t)

	rec := &stmtRecorder{}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.Tracer = rec
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	s := &Storage{binder: binder{db: pool}, pool: pool}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	file := filepath.Join("..", "..", "..", "..", "testdata", "test.xg")
	if _, err := (ingest.XGImporter{S: s}).Import(ctx, "2", ingest.Source{Path: file}, nil); err != nil {
		t.Fatal(err)
	}

	const scope = "1"
	rec.record("import")
	if _, err := (ingest.XGImporter{S: s}).Import(ctx, scope, ingest.Source{Path: file}, nil); err != nil {
		t.Fatal(err)
	}
	rec.record("stats")
	if _, err := s.Stats().Compute(ctx, scope, storage.StatsFilter{}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Stats().RebuildMatchStats(ctx, scope, nil); err != nil {
		t.Fatal(err)
	}
	batches, err := s.ImportBatches().List(ctx, scope, storage.ListOpts{})
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range batches {
		if _, err := s.ImportBatches().Report(ctx, scope, b.ID, nil); err != nil {
			t.Fatal(err)
		}
	}
	ids, err := s.Search().FindIDs(ctx, scope, domain.SearchFilters{}, storage.ListOpts{})
	if err != nil || len(ids) < 10 {
		t.Fatalf("positions of the tenant: %d, %v", len(ids), err)
	}
	rec.record("search")
	exerciseSearch(ctx, t, s, scope, ids)
	rec.record("training")
	exerciseTraining(ctx, t, s, scope, ids)
	rec.record("anki")
	exerciseAnki(ctx, t, s, scope, ids)
	rec.record("metadata")
	exerciseMetadata(ctx, t, s, scope)
	rec.record("maintenance")
	exerciseMaintenance(ctx, t, s, scope, ids)
	rec.record("")
	rec.mu.Lock()
	stmts := rec.stmts
	rec.mu.Unlock()

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })
	tenantLeading := map[string]bool{}
	rows, err := conn.Query(ctx, `SELECT i.relname FROM pg_index x
		JOIN pg_class i ON i.oid = x.indexrelid JOIN pg_class c ON c.oid = x.indrelid
		JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = x.indkey[0]
		WHERE a.attname = 'tenant_id'`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		tenantLeading[name] = true
	}
	if rows.Err() != nil || len(tenantLeading) == 0 {
		t.Fatalf("no tenant-leading index found: %v", rows.Err())
	}

	// The settings PinJoinOrder gives the statistics' reads: each next table
	// reached through its index, in the order written.
	if _, err := conn.Exec(ctx, `SET join_collapse_limit = 1; SET from_collapse_limit = 1; SET enable_hashjoin = off;
		SET enable_mergejoin = off; SET enable_seqscan = off; SET enable_bitmapscan = off`); err != nil {
		t.Fatal(err)
	}
	explained, seen := 0, map[string]bool{}
	for _, st := range stmts {
		if !explainable.MatchString(st.sql) || seen[st.sql] {
			continue
		}
		seen[st.sql] = true
		var out []byte
		// A statement on a temporary table of its own transaction cannot be
		// planned from here; the tables it reads were planned when it was filled.
		if err := conn.QueryRow(ctx, "EXPLAIN (FORMAT JSON) "+st.sql, st.args...).Scan(&out); err != nil {
			continue
		}
		explained++
		var plans []struct {
			Plan planNode `json:"Plan"`
		}
		if err := json.Unmarshal(out, &plans); err != nil {
			t.Fatal(err)
		}
		for _, p := range plans {
			for _, bad := range unboundedLookups(p.Plan, tenantLeading) {
				t.Errorf("%s: %s\n  in: %s", st.phase, bad, oneLine(st.sql))
			}
		}
	}
	if explained < 100 {
		t.Fatalf("only %d statements planned: the recording missed the import", explained)
	}
}

func unboundedLookups(n planNode, tenantLeading map[string]bool) []string {
	var bad []string
	switch {
	case n.NodeType == "Seq Scan" && (n.Relation == "position" || n.Relation == "move" || n.Relation == "analysis"):
		bad = append(bad, "sequential scan of "+n.Relation)
	case tenantLeading[n.IndexName] && !strings.Contains(n.IndexCond, "tenant_id"):
		bad = append(bad, n.NodeType+" of "+n.IndexName+" without its tenant (cond: "+n.IndexCond+")")
	}
	for _, c := range n.Plans {
		bad = append(bad, unboundedLookups(c, tenantLeading)...)
	}
	return bad
}

func oneLine(sql string) string {
	sql = strings.Join(strings.Fields(sql), " ")
	if len(sql) > 220 {
		sql = sql[:220] + "…"
	}
	return sql
}
