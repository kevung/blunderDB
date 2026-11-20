//go:build postgres

package postgres

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// pipAscAnswers is everything a search in ascending pips answers: the pages
// chained by cursor, every offset up to past the end in two page sizes, and
// one page large enough that the ceiling of the band read has to rise.
func pipAscAnswers(ctx context.Context, t *testing.T, s *Storage, f domain.SearchFilters) (string, int) {
	t.Helper()
	var b strings.Builder
	var cursor string
	total := 0
	for range 500 {
		ids, next, err := s.Search().FindIDsPage(ctx, "1", f, storage.PageOpts{Limit: 3, Cursor: cursor})
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&b, "c%v", ids)
		total += len(ids)
		if next == "" {
			break
		}
		cursor = next
	}
	for _, limit := range []int{2, 5} {
		for offset := 0; offset <= total+1; offset++ {
			ids, _, err := s.Search().FindIDsPage(ctx, "1", f, storage.PageOpts{Limit: limit, Offset: offset})
			if err != nil {
				t.Fatal(err)
			}
			fmt.Fprintf(&b, " o%d/%d%v", offset, limit, ids)
		}
	}
	ids, _, err := s.Search().FindIDsPage(ctx, "1", f, storage.PageOpts{Limit: 200})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(&b, " l%v", ids)
	return b.String(), total
}

// A structure read band by band answers what the read across the tenant
// answers — pages by cursor, every offset, a page past every ceiling — for
// black points, a point asked with three checkers (masks count two, the board
// decides the third), a white point beside them, and another filter. Rows the
// masks keep below the floor of their band, and rows without a pip count at
// the end of the order, are planted: an offset that ends past the rows in pip
// order must count the same rows the band read ranked.
func TestPipBandsAnswerAsTheTenantRead(t *testing.T) {
	ctx := context.Background()
	s, rec, dsn := probeFixture(ctx, t)
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })

	black := blackPoints(ctx, t, s)
	var three domain.Board
	threeAt := 0
	var white domain.Board
	for p, err := range s.Search().Find(ctx, "1", domain.SearchFilters{}, storage.ListOpts{Limit: 500}) {
		if err != nil {
			t.Fatal(err)
		}
		for i := 7; i < 25 && threeAt == 0; i++ {
			if pt := p.Board.Points[i]; pt.Color == domain.Black && pt.Checkers >= 3 {
				threeAt = i
				three.Points[i] = domain.Point{Checkers: 3, Color: domain.Black}
			}
		}
		if white == (domain.Board{}) {
			for i := 1; i < 25; i++ {
				if pt := p.Board.Points[i]; pt.Color == domain.White && pt.Checkers > 0 && black.Points[i].Checkers == 0 {
					white = black
					white.Points[i] = domain.Point{Checkers: 1, Color: domain.White}
					break
				}
			}
		}
	}
	if threeAt == 0 || white == (domain.Board{}) {
		t.Fatalf("the fixture has no black point of three checkers outside the home board (%d) or no white point", threeAt)
	}
	cases := map[string]domain.SearchFilters{
		"black":    {Filter: domain.Position{Board: black}},
		"three":    {Filter: domain.Position{Board: three}},
		"white":    {Filter: domain.Position{Board: white}},
		"decision": {Filter: domain.Position{Board: black, DecisionType: domain.CheckerAction}, DecisionTypeFilter: true},
	}

	// Plant, for the three-checker structure: two of its rows lose their pip count,
	// and a row it does not keep becomes two black checkers on the point and none
	// elsewhere — what the masks keep and the board refuses, with fewer pips than
	// three checkers there can have.
	f := cases["three"]
	f.Sort = "pip_asc"
	restore := sqlshared.SetPipBands(false)
	kept, _, err := s.Search().FindIDsPage(ctx, "1", f, storage.PageOpts{Limit: 1000})
	restore()
	if err != nil {
		t.Fatal(err)
	}
	if len(kept) < 4 {
		t.Fatalf("the three-checker structure keeps %d rows", len(kept))
	}
	bit := int64(1) << threeAt
	if _, err := conn.Exec(ctx, `UPDATE position SET pip_1 = NULL WHERE tenant_id = 1 AND id = ANY($1)`, kept[len(kept)-2:]); err != nil {
		t.Fatal(err)
	}
	tag, err := conn.Exec(ctx, `UPDATE position SET occupancy_1 = $1, point_mask_1 = $1, pip_1 = $2
		WHERE id = (SELECT id FROM position WHERE tenant_id = 1 AND NOT id = ANY($3) AND occupancy_1 IS NOT NULL ORDER BY id LIMIT 1)`,
		bit, 2*threeAt, kept)
	if err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("plant a row under the floor: %v, %d rows", err, tag.RowsAffected())
	}
	if _, err := conn.Exec(ctx, `ANALYZE position`); err != nil {
		t.Fatal(err)
	}

	reads := 0
	for name, f := range cases {
		t.Run(name, func(t *testing.T) {
			f.Sort = "pip_asc"
			rec.record(name)
			banded, total := pipAscAnswers(ctx, t, s, f)
			rec.record("")
			restore := sqlshared.SetPipBands(false)
			whole, _ := pipAscAnswers(ctx, t, s, f)
			restore()
			if banded != whole {
				t.Errorf("band by band:\n  %s\nacross the tenant:\n  %s", banded, whole)
			}
			if total < 4 {
				t.Errorf("%d rows: too few to page", total)
			}
		})
	}
	rec.mu.Lock()
	for _, st := range rec.stmts {
		if strings.Contains(st.sql, "pg_backend_pid") && strings.Contains(st.sql, "p.pip_1 <= ") {
			reads++
		}
	}
	rec.stmts = nil
	rec.mu.Unlock()
	if reads == 0 {
		t.Error("no band read under a ceiling was sent")
	}
}
