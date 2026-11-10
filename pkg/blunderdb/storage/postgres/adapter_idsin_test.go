package postgres

import (
	"reflect"
	"testing"
)

// The id list must reach Postgres as a scalar sub-select: a bare
// `= ANY($n::bigint[])` is planned with the array as a constant and a
// custom plan retains it per backend (tens of MB for 200k ids).
func TestIDsInWrapsArrayInSubselect(t *testing.T) {
	cond, args := shared{}.IDsIn("p.id", []string{"1", "22", "333"})
	if want := "p.id = ANY((SELECT ?::bigint[])::bigint[])"; cond != want {
		t.Errorf("cond = %q, want %q", cond, want)
	}
	if want := []any{"{1,22,333}"}; !reflect.DeepEqual(args, want) {
		t.Errorf("args = %v, want %v", args, want)
	}
}
