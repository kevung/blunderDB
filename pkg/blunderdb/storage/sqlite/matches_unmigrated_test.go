package sqlite

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The demo library predates the stored match result. Read as it stands, its
// matches come back without a result, and reading them adds no column.
func TestMatchResultOnUnmigratedLibrary(t *testing.T) {
	ctx := context.Background()
	ro, err := OpenReadOnly(ctx, demoCopy(t))
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	var first int64
	n := 0
	for m, err := range ro.Matches().List(ctx, "", storage.MatchListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		if m.FinalScore != nil || m.Winner != domain.WinnerUnfinished {
			t.Errorf("match %d: result %v won by %d from a library without the columns", m.ID, m.FinalScore, m.Winner)
		}
		if n == 0 {
			first = m.ID
		}
		n++
	}
	if n == 0 {
		t.Fatal("the demo library lists no match")
	}
	if _, err := ro.Matches().Get(ctx, "", first); err != nil {
		t.Fatalf("Get(%d): %v", first, err)
	}
	if (&columnProbe{}).has(ctx, ro.sqlDB, "match", "final_score_1") {
		t.Error("reading the library added the result columns")
	}
}
