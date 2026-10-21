package sqlshared

import (
	"context"
	"strings"
	"testing"
)

// recordingExecer records each executed statement's row count and the
// arguments it carried, so a test can check that insertRows wrote every row
// once and in order, prepared or not.
type recordingExecer struct {
	fakeExecer
	execRows     []int
	args         []any
	prepares     int
	preparedRows int
}

func valuesRows(q string) int { return strings.Count(q, "(?") }

func (r *recordingExecer) Exec(_ context.Context, q string, args ...any) (int64, error) {
	r.execRows = append(r.execRows, valuesRows(q))
	r.args = append(r.args, args...)
	return 0, nil
}

type preparingExecer struct{ *recordingExecer }

func (p preparingExecer) Prepare(_ context.Context, q string) (Stmt, error) {
	p.prepares++
	p.preparedRows = valuesRows(q)
	return recordedStmt{p.recordingExecer}, nil
}

type recordedStmt struct{ r *recordingExecer }

func (s recordedStmt) Exec(_ context.Context, args ...any) error {
	s.r.execRows = append(s.r.execRows, s.r.preparedRows)
	s.r.args = append(s.r.args, args...)
	return nil
}
func (recordedStmt) Close() error { return nil }

func TestInsertRows_EveryRowOnceInOrder(t *testing.T) {
	ctx := context.Background()
	cols := []string{"a", "b"}
	for _, n := range []int{0, 1, rowsPerInsert, 2*rowsPerInsert + 7} {
		for _, prepared := range []bool{false, true} {
			rec := &recordingExecer{}
			var db Execer = rec
			if prepared {
				db = preparingExecer{rec}
			}
			if err := insertRows(ctx, db, "t", cols, n, func(i int) []any { return []any{i, -i} }); err != nil {
				t.Fatal(err)
			}
			if len(rec.args) != 2*n {
				t.Fatalf("n=%d prepared=%v: %d args, want %d", n, prepared, len(rec.args), 2*n)
			}
			for i := range n {
				if rec.args[2*i] != i || rec.args[2*i+1] != -i {
					t.Fatalf("n=%d prepared=%v: row %d out of place", n, prepared, i)
				}
			}
			total := 0
			for _, k := range rec.execRows {
				if k > rowsPerInsert {
					t.Fatalf("n=%d: a statement of %d rows", n, k)
				}
				total += k
			}
			if total != n {
				t.Fatalf("n=%d prepared=%v: statements cover %d rows", n, prepared, total)
			}
			if want := min(1, n/rowsPerInsert); prepared && rec.prepares != want {
				t.Fatalf("n=%d: %d prepares, want %d", n, rec.prepares, want)
			}
		}
	}
}
