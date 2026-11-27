//go:build postgres

package postgres

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// A move-pattern search reads the tenant's best moves from
// idx_analysis_best_move alone, not the analysis heap with its blobs, and the
// statements that fetch the blobs are narrowed by them. The plans are those
// of a role under RLS, as the engine reads: LIKE is not leakproof there, so it
// is a filter of the index read, never an index condition. Both ways of
// writing the narrowing are explained: the list read first (rare pattern) and
// the clause on the walk (common).
func TestMovePatternReadsTheBestMoveIndexUnderRLS(t *testing.T) {
	ctx := context.Background()
	s, rec, dsn := probeFixture(ctx, t)
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })
	var best string
	if err := conn.QueryRow(ctx, `SELECT best_move FROM analysis WHERE tenant_id = 1 AND best_move <> '' ORDER BY id LIMIT 1`).Scan(&best); err != nil {
		t.Fatalf("no best move written at import: %v", err)
	}
	// The planner prices a small fixture's heap reads as cheap as an index's:
	// a visibility map set and sequential scans priced out leave the choice
	// between indexes, which is what is checked.
	if _, err := conn.Exec(ctx, `VACUUM ANALYZE analysis`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, `ANALYZE position; SET enable_seqscan = off; SET enable_bitmapscan = off`); err != nil {
		t.Fatal(err)
	}
	if err := s.ApplyRLS(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, `DO $$ BEGIN CREATE ROLE move_pattern_rls NOLOGIN; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
		GRANT USAGE ON SCHEMA public TO move_pattern_rls; GRANT SELECT ON ALL TABLES IN SCHEMA public TO move_pattern_rls`); err != nil {
		t.Fatal(err)
	}
	explainUnderRLS := func(sql string, args []any) (planNode, []byte) {
		t.Helper()
		tx, err := conn.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if _, err := tx.Exec(ctx, `SET LOCAL ROLE move_pattern_rls; SELECT set_config('app.tenant_id', '1', true)`); err != nil {
			t.Fatal(err)
		}
		var out []byte
		if err := tx.QueryRow(ctx, "EXPLAIN (FORMAT JSON) "+sql, args...).Scan(&out); err != nil {
			t.Fatalf("explain: %v\n  in: %s", err, oneLine(sql))
		}
		var plans []struct {
			Plan planNode `json:"Plan"`
		}
		if err := json.Unmarshal(out, &plans); err != nil {
			t.Fatal(err)
		}
		return plans[0].Plan, out
	}
	pattern := strings.Fields(best)[0]
	for _, c := range []struct {
		name  string
		limit int
	}{{"list", 1 << 20}, {"walk", -1}} {
		t.Run(c.name, func(t *testing.T) {
			defer sqlshared.SetNarrowLimit(c.limit)()
			f := domain.SearchFilters{MovePatternFilter: `m"` + pattern + `"`}
			rec.record(c.name)
			ids, _, err := s.Search().FindIDsPage(ctx, "1", f, storage.PageOpts{Limit: 24})
			rec.record("")
			if err != nil {
				t.Fatal(err)
			}
			if len(ids) == 0 {
				t.Fatalf("%s found nothing", f.MovePatternFilter)
			}
			rec.mu.Lock()
			stmts := rec.stmts
			rec.stmts = nil
			rec.mu.Unlock()
			byIndex, blobs, narrowed := 0, 0, 0
			for _, st := range stmts {
				if strings.HasPrefix(st.sql, "EXPLAIN") || !strings.Contains(st.sql, "analysis") {
					continue
				}
				if strings.Contains(st.sql, "a.data") {
					blobs++
					if !strings.Contains(st.sql, "best_move") && !strings.Contains(st.sql, "p.id IN (") {
						t.Errorf("the blobs are read unnarrowed: %s", oneLine(st.sql))
					}
				}
				if !strings.Contains(st.sql, "best_move LIKE") {
					continue
				}
				narrowed++
				plan, out := explainUnderRLS(st.sql, st.args)
				var walk func(planNode)
				walk = func(n planNode) {
					if n.Relation == "analysis" && n.IndexName == "idx_analysis_best_move" && n.NodeType == "Index Only Scan" {
						byIndex++
					}
					if n.Relation == "analysis" && !strings.Contains(n.NodeType, "Index") {
						t.Errorf("analysis read by %s\n  in: %s\n  plan: %s", n.NodeType, oneLine(st.sql), out)
					}
					for _, child := range n.Plans {
						walk(child)
					}
				}
				walk(plan)
			}
			if blobs == 0 {
				t.Fatalf("no statement read the blobs; sent: %d", len(stmts))
			}
			if narrowed == 0 {
				t.Errorf("no statement narrows by best_move; sent: %d", len(stmts))
			}
			if c.name == "list" && byIndex == 0 {
				t.Errorf("no statement read idx_analysis_best_move alone; sent: %d", len(stmts))
			}
		})
	}
}
