package sqlite_test

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestMerge_ConcurrentWritersLoseNoFold: merging saves running at once on one
// position, over pooled connections as under serve, each keep their candidate:
// Merge reads and writes in one transaction, so none overwrites another's
// fold with what it read before it.
func TestMerge_ConcurrentWritersLoseNoFold(t *testing.T) {
	ctx := context.Background()
	st, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "lib.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	p := domain.InitializePosition()
	p.DecisionType = domain.CheckerAction
	p.Dice = [2]int{6, 3}
	pid, err := st.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatal(err)
	}
	const writers = 8
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for i := range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a := domain.PositionAnalysis{
				AnalysisType: "CheckerMove",
				CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
					{Move: fmt.Sprintf("24/%d", 23-i), Equity: float64(i) / 100},
				}},
			}
			errs <- storage.SaveMergedAnalysis(ctx, st.Analyses(), "", pid, a)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("SaveMergedAnalysis: %v", err)
		}
	}
	got, err := st.Analyses().Load(ctx, "", pid)
	if err != nil {
		t.Fatal(err)
	}
	if got.CheckerAnalysis == nil || len(got.CheckerAnalysis.Moves) != writers {
		t.Errorf("after %d concurrent merges the analysis holds %d candidates, want %d", writers, len(got.CheckerAnalysis.Moves), writers)
	}
}
