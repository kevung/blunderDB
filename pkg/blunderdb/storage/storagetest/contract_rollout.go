package storagetest

import (
	"context"
	"sync"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/rollout"
	"github.com/kevung/blunderdb/pkg/blunderdb/rollouts"
	"github.com/kevung/blunderdb/pkg/blunderdb/searchquery"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// movesRollout is a finished checker rollout with s's Configuration, built by
// hand: the contract is about what is stored, not about the games.
func movesRollout(s rollout.Settings, best string, winChance float64) *rollout.Result {
	est := func(eq float64, jsd float64) rollout.Estimate {
		e := rollout.Estimate{Equity: eq, StdErr: 0.01, CI95: 0.0196, Games: s.MaxGames, JSD: jsd}
		e.Chances[0] = winChance
		return e
	}
	return &rollout.Result{
		Kind: rollout.KindMoves, EngineVersion: rollout.EngineVersion, Settings: s, Signature: s.Signature(),
		Candidates: []rollout.Candidate{{Move: best, Estimate: est(0.2, 0)}, {Move: "24/23 13/10", Estimate: est(0.1, 3.5)}},
		Games:      s.MaxGames, Stop: rollout.StopMaxGames, CubefulBias: true,
	}
}

// testRolloutIsASecondAnalysis: a rollout lands beside an imported analysis
// and moves none of it — neither its entries nor the columns the search
// reads (ADR-0013, ADR-0060); a rerun of the same Configuration replaces it,
// another Configuration sits beside it; a cancelled one is refused.
func testRolloutIsASecondAnalysis(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	p := checkerPos()
	id, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	imported := &domain.PositionAnalysis{
		AnalysisType: "CheckerMove", AnalysisEngineVersion: "XG",
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "8/5 6/5", Equity: 0.15, AnalysisEngine: "XG", AnalysisDepth: "XG Roller++", PlayerWinChance: 55},
		}},
	}
	if err := s.Analyses().Save(ctx, "", id, imported); err != nil {
		t.Fatalf("Save imported analysis: %v", err)
	}

	fast := rollout.Fast()
	if err := rollouts.Store(ctx, s, "", id, movesRollout(fast, "13/10 6/5", 0.80)); err != nil {
		t.Fatalf("Store: %v", err)
	}
	got, err := s.Analyses().Load(ctx, "", id)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if m := got.CheckerAnalysis.Moves; len(m) != 1 || m[0].Move != "8/5 6/5" || m[0].AnalysisEngine != "XG" {
		t.Errorf("imported entries moved: %+v", m)
	}
	if len(got.Rollouts) != 1 {
		t.Fatalf("rollouts: got %d, want 1", len(got.Rollouts))
	}
	r := got.Rollouts[0]
	if r.Signature != fast.Signature() || r.AnalysisEngine != rollout.EngineVersion || r.AnalysisDepth != fast.DepthLabel() ||
		r.Settings.MaxGames != fast.MaxGames || r.Kind != domain.RolloutKindMoves || !r.CubefulBias {
		t.Errorf("rollout configuration not kept: %+v", r)
	}
	if c := r.Candidates; len(c) != 2 || c[0].CI95 != 0.0196 || c[1].JSD != 3.5 || c[0].PlayerWinChance != 80 {
		t.Errorf("candidates not kept: %+v", c)
	}

	// The search still reads the imported analysis (55 % to win), not the
	// rollout (80 %).
	if ids := findQuery(t, s, "w>70"); len(ids) != 0 {
		t.Errorf("w>70 found %v: the rollout moved the imported columns", ids)
	}

	if err := rollouts.Store(ctx, s, "", id, movesRollout(fast, "13/10 6/5", 0.80)); err != nil {
		t.Fatalf("Store again: %v", err)
	}
	if err := rollouts.Store(ctx, s, "", id, movesRollout(rollout.Standard(), "13/10 6/5", 0.80)); err != nil {
		t.Fatalf("Store standard: %v", err)
	}
	list, err := rollouts.List(ctx, s, "", id)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("rollouts after a rerun and another Configuration: got %d, want 2", len(list))
	}

	cancelled := movesRollout(fast, "13/10 6/5", 0.8)
	cancelled.Stop = rollout.StopCancelled
	if err := rollouts.Store(ctx, s, "", id, cancelled); err == nil {
		t.Error("a cancelled rollout was stored")
	}

	// Saving the analysis again without its rollouts — what an import or the
	// GUI does — goes through the caller's merge; the plain Save replaces, so
	// the rollouts must ride along with whatever was loaded.
	got, _ = s.Analyses().Load(ctx, "", id)
	if err := s.Analyses().Save(ctx, "", id, got); err != nil {
		t.Fatalf("re-Save: %v", err)
	}
	if list, _ := rollouts.List(ctx, s, "", id); len(list) != 2 {
		t.Errorf("rollouts lost on a re-Save of the loaded analysis: %d", len(list))
	}
}

// testRolloutAloneFeedsTheSearch: a position whose only analysis is a rollout
// is found by the search through it, and the gap-filling sweep counts it as
// analysed, so it is not offered again.
func testRolloutAloneFeedsTheSearch(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	p := checkerPos()
	id, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	fast := rollout.Fast()
	if err := rollouts.Store(ctx, s, "", id, movesRollout(fast, "13/10 6/5", 0.80)); err != nil {
		t.Fatalf("Store: %v", err)
	}
	if ids := findQuery(t, s, "w>70"); len(ids) != 1 || ids[0] != id {
		t.Errorf("w>70: got %v, want [%d]", ids, id)
	}
	todo, err := rollouts.Gather(ctx, s, "", domain.SearchFilters{}, fast)
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}
	if len(todo) != 0 {
		t.Errorf("Gather offers %d position(s) already rolled out with this Configuration", len(todo))
	}
	if todo, _ := rollouts.Gather(ctx, s, "", domain.SearchFilters{}, rollout.Standard()); len(todo) != 1 {
		t.Errorf("Gather with another Configuration: got %d, want 1", len(todo))
	}
}

func findQuery(t *testing.T, s storage.Storage, q string) []int64 {
	t.Helper()
	f, _ := searchquery.Parse(q)
	var ids []int64
	for pos, err := range s.Search().Find(context.Background(), "", f, storage.ListOpts{}) {
		if err != nil {
			t.Fatalf("Find(%q): %v", q, err)
		}
		ids = append(ids, pos.ID)
	}
	return ids
}

// testConcurrentRolloutsAllKept: rollouts of different Configurations stored
// at once on one position all stay — each Store reads and writes the analysis
// in one guarded transaction, so none writes over a row another just changed.
func testConcurrentRolloutsAllKept(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	p := checkerPos()
	id, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	const writers = 8
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for i := range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			set := rollout.Fast()
			set.Seed = uint64(i + 1)
			errs <- rollouts.Store(ctx, s, "", id, movesRollout(set, "13/10 6/5", 0.8))
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("Store: %v", err)
		}
	}
	got, err := s.Analyses().Load(ctx, "", id)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.Rollouts) != writers {
		t.Errorf("%d rollouts kept of %d stored at once", len(got.Rollouts), writers)
	}
}
