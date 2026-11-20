//go:build postgres

package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// A structure sorted by pips reads the positions of the tenant already in
// pip order, the masks from the index: no sort of every matching row, which on
// a full library ranks millions of rows to return one page. A fixture this
// small sorts cheaper than it walks, so sorting is priced out: what is checked
// is that an index can give the order, not the full library's choice. In
// ascending pips the structure's black checkers name the bands it can keep, each
// read in order (sqlshared.pipBandWindow), and merged: the pages are those of the
// read across the tenant.
func TestStructureSortedByPipsReadsAnIndexInOrder(t *testing.T) {
	ctx := context.Background()
	s, rec, dsn := probeFixture(ctx, t)
	board := blackPoints(ctx, t, s)
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
		{"pip_asc", "idx_position_pip_band"},
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
			if c.sort != "pip_asc" {
				return
			}
			pages := func() [][]int64 {
				var out [][]int64
				var cursor string
				for range 50 {
					ids, next, err := s.Search().FindIDsPage(ctx, "1", f, storage.PageOpts{Limit: 3, Cursor: cursor})
					if err != nil {
						t.Fatal(err)
					}
					out = append(out, ids)
					if next == "" {
						break
					}
					cursor = next
				}
				return out
			}
			banded := pages()
			restore := sqlshared.SetPipBands(false)
			whole := pages()
			restore()
			if fmt.Sprint(banded) != fmt.Sprint(whole) {
				t.Errorf("pages read band by band %v, across the tenant %v", banded, whole)
			}
			for _, offset := range []int{0, 2, 5} {
				var got [2][]int64
				for i, on := range []bool{true, false} {
					restore := sqlshared.SetPipBands(on)
					ids, _, err := s.Search().FindIDsPage(ctx, "1", f, storage.PageOpts{Limit: 4, Offset: offset})
					restore()
					if err != nil {
						t.Fatal(err)
					}
					got[i] = ids
				}
				if fmt.Sprint(got[0]) != fmt.Sprint(got[1]) {
					t.Errorf("offset %d: band by band %v, across the tenant %v", offset, got[0], got[1])
				}
			}
			if len(banded) < 2 {
				t.Errorf("one page only (%v): the cursor is not exercised", banded)
			}
		})
	}
}

// blackPoints is a structure of black checkers only, taken from the first
// position: two of its points, at least one checker on each.
func blackPoints(ctx context.Context, t *testing.T, s *Storage) domain.Board {
	t.Helper()
	var board domain.Board
	for p, err := range s.Search().Find(ctx, "1", domain.SearchFilters{}, storage.ListOpts{Limit: 1}) {
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for i := len(p.Board.Points) - 1; i > 0 && n < 2; i-- {
			if pt := p.Board.Points[i]; pt.Checkers > 0 && pt.Color == domain.Black {
				board.Points[i] = domain.Point{Checkers: 1, Color: domain.Black}
				n++
			}
		}
		if n < 2 {
			t.Fatalf("the first position has black checkers on %d points", n)
		}
	}
	return board
}
