package sqlshared

import (
	"slices"
	"testing"
)

// PostgreSQL hands the NULL groups last, SQLite first; the cells path stores
// NULL as cellNull and reads it first. The direct path orders in Go, so every
// backend and path lists a NULL row before the 0 it is shown as.
func TestSortBreakdownKeysPutsNullFirst(t *testing.T) {
	asPostgres := []breakdownKey{
		{k1: 0, k2: 0}, {k1: 0, k2: 3}, {k1: 2, k2: 1},
		{k1: 0, k2: 0, nulls: 2}, {k1: 0, k2: 0, nulls: 3}, {k1: 0, k2: 1, nulls: 1},
	}
	want := []breakdownKey{
		{k1: 0, k2: 0, nulls: 3}, {k1: 0, k2: 1, nulls: 1},
		{k1: 0, k2: 0, nulls: 2}, {k1: 0, k2: 0}, {k1: 0, k2: 3}, {k1: 2, k2: 1},
	}
	sortBreakdownKeys(asPostgres)
	if !slices.Equal(asPostgres, want) {
		t.Errorf("order %v, want %v", asPostgres, want)
	}
}
