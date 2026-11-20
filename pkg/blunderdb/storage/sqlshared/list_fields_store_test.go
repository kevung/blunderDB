package sqlshared

import (
	"context"
	"testing"
)

// A page of positions costs the same queries whatever its length: the list
// fields are never read row by row.
func TestListFieldsQueriesDoNotGrowWithThePage(t *testing.T) {
	calls := func(n int) (int, int) {
		ids := make([]int64, n)
		for i := range ids {
			ids[i] = int64(i + 1)
		}
		db := &fakeExecer{}
		if _, err := (&ListFieldsStore{DB: db}).ListFields(context.Background(), "", ids); err != nil {
			t.Fatal(err)
		}
		return db.queryCalls, db.queryRowCalls
	}
	q1, r1 := calls(1)
	q24, r24 := calls(24)
	if q1 != q24 || r1 != r24 || q24 > 2 {
		t.Errorf("1 row: %d queries + %d single-row; 24 rows: %d + %d; want the same, at most 2", q1, r1, q24, r24)
	}
}

func TestUUIDV7Day(t *testing.T) {
	for _, c := range []struct{ uuid, want string }{
		{"0193b1e8-1c00-7abc-8def-0123456789ab", "2024-12-10"},
		{"00000000-0000-7abc-8def-0123456789ab", ""}, // derived for a row older than the column
		{"0193b1e8-1c00-4abc-8def-0123456789ab", ""}, // not version 7
		{"nonsense", ""},
	} {
		if got := uuidV7Day(c.uuid); got != c.want {
			t.Errorf("uuidV7Day(%s) = %q, want %q", c.uuid, got, c.want)
		}
	}
}
