//go:build postgres

package postgres

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// A structure that asks nothing of a side writes that side's columns as
// filled, not as masks of zero. Both keep the same rows — a row whose column
// is empty passes neither — so the search answers the same, in id order or by
// a sort key, in each tenant.
func TestStructureSearchAnswersTheSameWithoutMasksOfZero(t *testing.T) {
	ctx := context.Background()
	s, rec, dsn := probeFixture(ctx, t)
	oneSide := madePoints(ctx, t, s)
	side, chosen := 0, false
	for i, pt := range oneSide.Points {
		switch {
		case pt.Checkers == 0:
		case !chosen:
			side, chosen = pt.Color, true
		case pt.Color != side:
			oneSide.Points[i] = domain.Point{}
		}
	}
	// Rows with an empty column, the only ones the two writings could tell apart.
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })
	for _, empty := range []string{
		`UPDATE position SET occupancy_1 = NULL WHERE id % 5 = 0`,
		`UPDATE position SET point_mask_2 = NULL WHERE id % 7 = 0`,
		`UPDATE position SET occupancy_2 = NULL, point_mask_1 = NULL WHERE id % 11 = 0`,
	} {
		if _, err := conn.Exec(ctx, empty); err != nil {
			t.Fatal(err)
		}
	}

	corpus, commands := corpusFilters(t)
	var filters []domain.SearchFilters
	var names []string
	for i, f := range corpus {
		for _, v := range []struct {
			except bool
			sort   string
		}{{false, ""}, {false, "pip_asc"}, {true, "error"}, {true, ""}} {
			g := f
			g.Sort = v.sort
			if v.except {
				g.ExcludeFilter.Board = oneSide
			} else {
				g.Filter.Board = oneSide
			}
			filters = append(filters, g)
			names = append(names, fmt.Sprintf("%s (except %v, sort %q)", commands[i], v.except, v.sort))
		}
	}

	restore := sqlshared.SetNeutralMasks(true)
	rec.record("masks")
	want := replay(ctx, s, filters, true)
	rec.record("")
	restore()
	answered := 0
	for _, w := range want {
		if !strings.Contains(w, "find [];") {
			answered++
		}
	}
	if answered < len(want)/8 {
		t.Fatalf("%d answers of %d find a position: the structure is too rare to tell two writings apart", answered, len(want))
	}

	rec.record("filled")
	got := replay(ctx, s, filters, true)
	rec.record("")
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("tenant %d, %s:\n  got  %s\n  want %s", 1+i%2, names[i/2], got[i], want[i])
		}
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	filled := 0
	for _, st := range rec.stmts {
		if !strings.Contains(st.sql, "p.point_mask_2 IS NOT NULL") && !strings.Contains(st.sql, "p.point_mask_1 IS NOT NULL") {
			continue
		}
		if st.phase == "masks" {
			t.Fatalf("a filled column was written with the masks of zero on: %s", oneLine(st.sql))
		}
		filled++
	}
	if filled < len(filters) {
		t.Fatalf("%d statements for %d filters write a side as filled: the masks of zero are back", filled, len(filters))
	}
}
