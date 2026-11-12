//go:build postgres

package postgres

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// A page read by cursor must start at its anchor: the sort key bounds an index
// range (Index Cond), and no node sorts the whole result. Otherwise a deep page
// rereads every page before it, which is what the cursor exists to avoid.
func TestSearchCursorPageStartsAtItsAnchor(t *testing.T) {
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
	for _, scope := range []string{"2", "1"} {
		if _, err := (ingest.XGImporter{S: s}).Import(ctx, scope, ingest.Source{Path: file}, nil); err != nil {
			t.Fatal(err)
		}
	}

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })
	if _, err := conn.Exec(ctx, `SET enable_seqscan = off; SET enable_bitmapscan = off; SET enable_sort = off`); err != nil {
		t.Fatal(err)
	}

	for sort, key := range map[string]string{
		"":          "id",
		"error":     "best_move_equity_error",
		"winrate":   "player1_win_rate",
		"pip_asc":   "pip_1",
		"pip_desc":  "pip_1",
		"date_asc":  "match_date",
		"date_desc": "match_date",
	} {
		t.Run("sort="+sort, func(t *testing.T) {
			f := domain.SearchFilters{Sort: sort}
			_, next, err := s.Search().FindIDsPage(ctx, "1", f, storage.PageOpts{Limit: 2})
			if err != nil || next == "" {
				t.Fatalf("first page: cursor %q, %v", next, err)
			}
			rec.record("page " + sort)
			ids, _, err := s.Search().FindIDsPage(ctx, "1", f, storage.PageOpts{Limit: 2, Cursor: next})
			rec.record("")
			if err != nil || len(ids) == 0 {
				t.Fatalf("second page: %v, %v", ids, err)
			}
			rec.mu.Lock()
			stmts := rec.stmts
			rec.stmts = nil
			rec.mu.Unlock()
			if len(stmts) == 0 {
				t.Fatal("the second page sent no statement")
			}
			// The page's first read holds the anchor: the keyed rows, or the id
			// order. What follows (the NULL tail, the next cursor's key) is
			// bounded by the id alone.
			st := stmts[0]
			var out []byte
			if err := conn.QueryRow(ctx, "EXPLAIN (FORMAT JSON) "+st.sql, st.args...).Scan(&out); err != nil {
				t.Fatalf("explain: %v\n  in: %s", err, oneLine(st.sql))
			}
			var plans []struct {
				Plan planNode `json:"Plan"`
			}
			if err := json.Unmarshal(out, &plans); err != nil {
				t.Fatal(err)
			}
			bounded, sorted := false, false
			var walk func(planNode)
			walk = func(n planNode) {
				if strings.Contains(n.IndexCond, key+" >") || strings.Contains(n.IndexCond, key+" <") {
					bounded = true
				}
				if n.NodeType == "Sort" {
					sorted = true
				}
				for _, c := range n.Plans {
					walk(c)
				}
			}
			walk(plans[0].Plan)
			if !bounded || sorted {
				t.Errorf("key %s in an Index Cond: %v; full sort: %v\n  in: %s\n  plan: %s", key, bounded, sorted, oneLine(st.sql), out)
			}
		})
	}
}
