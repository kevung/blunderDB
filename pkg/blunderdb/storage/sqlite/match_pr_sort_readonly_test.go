package sqlite

import (
	"context"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// A reader that cannot fill match_stats orders the match list by PR from the
// decisions, in the order a writer reads from the table, and writes nothing.
func TestMatchListSortByPRReadOnly(t *testing.T) {
	ctx := context.Background()
	path := demoCopy(t)
	w, err := Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	ro := openQueryOnly(t, path)

	ids := func(st *Storage, sort string) []int64 {
		t.Helper()
		var out []int64
		for m, err := range st.Matches().List(ctx, "", storage.MatchListOpts{Sort: sort}) {
			if err != nil {
				t.Fatalf("List(%s): %v", sort, err)
			}
			out = append(out, m.ID)
		}
		return out
	}
	sorts := []string{"pr1", "pr1_desc", "pr2", "pr2_desc"}
	want := make([][]int64, len(sorts))
	for i, sort := range sorts {
		want[i] = ids(w, sort)
	}
	if err := sqlshared.InvalidateAllMatchStats(ctx, w.binder.shared(), ""); err != nil {
		t.Fatal(err)
	}
	for i, sort := range sorts {
		if got := ids(ro, sort); !slices.Equal(got, want[i]) {
			t.Errorf("read-only %s: %v; writer read %v", sort, got, want[i])
		}
	}
	if n := matchStatsRows(t, w); n != 0 {
		t.Errorf("read-only list wrote %d match_stats rows", n)
	}
}
