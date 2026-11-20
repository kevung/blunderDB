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
)

// A structure sorted by pips reads the positions of the tenant already in
// pip order, the masks from the index: no sort of every matching row, which on
// a full library ranks millions of rows to return one page. A fixture this
// small sorts cheaper than it walks, so sorting is priced out: what is checked
// is that an index can give the order, not the full library's choice.
func TestStructureSortedByPipsReadsAnIndexInOrder(t *testing.T) {
	ctx := context.Background()
	s, rec, dsn := probeFixture(ctx, t)
	board := madePoints(ctx, t, s)
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })
	if _, err := conn.Exec(ctx, `ANALYZE position; ANALYZE analysis; SET enable_seqscan = off; SET enable_bitmapscan = off; SET enable_sort = off; SET enable_incremental_sort = off`); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		sort, index string
	}{
		{"pip_asc", "idx_position_pip_order"},
		{"pip_desc", "idx_position_pip_order_desc"},
	} {
		t.Run(c.sort, func(t *testing.T) {
			f := domain.SearchFilters{Sort: c.sort}
			f.Filter.Board = board
			rec.record(c.sort)
			_, _, err := s.Search().FindIDsPage(ctx, "1", f, storage.PageOpts{Limit: 24})
			rec.record("")
			if err != nil {
				t.Fatal(err)
			}
			rec.mu.Lock()
			stmts := rec.stmts
			rec.stmts = nil
			rec.mu.Unlock()
			explained := 0
			for _, st := range stmts {
				if !strings.Contains(st.sql, "ORDER BY p.pip_1") || strings.HasPrefix(st.sql, "EXPLAIN") {
					continue
				}
				explained++
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
				read, sorted := "", false
				var walk func(planNode)
				walk = func(n planNode) {
					if n.Relation == "position" {
						read = n.NodeType + " " + n.IndexName
					}
					if strings.Contains(n.NodeType, "Sort") {
						sorted = true
					}
					for _, child := range n.Plans {
						walk(child)
					}
				}
				walk(plans[0].Plan)
				if sorted || !strings.HasSuffix(read, " "+c.index) {
					t.Errorf("position read by %q, sort node %v; want %s in order, no sort\n  in: %s\n  plan: %s",
						read, sorted, c.index, strings.Join(strings.Fields(st.sql), " "), out)
				}
			}
			if explained == 0 {
				t.Fatalf("no statement orders by pips; sent: %d", len(stmts))
			}
		})
	}
}
