package server

import (
	"context"
	"errors"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// /v1/gammonnet.compare — what gammonNet is worth on this tenant's OWN library
// (issue #270, fiche I.14).
//
// The daemon's half of the sweep: gather the positions carrying an analysis
// somebody else wrote, run the engine on them, and fold the samples. What a
// comparison IS — what counts as the same answer, how a disagreement is priced
// — is gammonnet/compare.go's, shared with the desktop wrapper, exactly as
// IsStaleAnalysis is shared by the two stale sweeps.
//
// Unlike the two sweeps next to it this one is NOT streamed and takes no job
// id: it writes nothing, so there is no partial state to resume or cancel
// into, and its answer is one small object rather than a long tail of
// progress. `limit` is what bounds it — a comparison is a sample question, and
// a client asking it of a sixty-thousand-position tenant should say how many
// it is willing to pay for.

// gammonnetCompareReq asks for a comparison. Limit caps the positions looked
// at (0 = all); the search parameters default to the canonical ones (ADR-0013)
// exactly as the sweeps' do.
type gammonnetCompareReq struct {
	Ply        int `json:"ply"`
	PruneK     int `json:"pruneK"`
	Candidates int `json:"candidates"`
	Limit      int `json:"limit"`
}

func (s *Server) handleGammonNetCompare(w http.ResponseWriter, r *http.Request) {
	var req gammonnetCompareReq
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &req); err != nil {
			writeDecodeError(w, "invalid JSON body", err)
			return
		}
	}
	if req.PruneK <= 0 {
		req.PruneK = 12
	}
	if req.Candidates <= 0 {
		req.Candidates = 10
	}

	ctx := r.Context()
	scope := scopeOf(r)
	positions, stored, err := gammonnetPositionsWithForeignAnalysis(ctx, s.opts.Storage, scope, req.Limit)
	if err != nil {
		writeStorageError(w, err)
		return
	}
	writeJSONResp(w, compareGathered(ctx, positions, stored, req.Ply, req.PruneK, req.Candidates))
}

// gammonnetPositionsWithForeignAnalysis returns the positions whose stored
// analysis is NOT gammonNet's own, with that analysis — the only ones a
// comparison has anything to compare against.
func gammonnetPositionsWithForeignAnalysis(ctx context.Context, s storage.Storage, scope string, limit int) ([]domain.Position, []*domain.PositionAnalysis, error) {
	all, err := drainPositions(ctx, s, scope)
	if err != nil {
		return nil, nil, err
	}
	var positions []domain.Position
	var analyses []*domain.PositionAnalysis
	for _, p := range all {
		if ctx.Err() != nil {
			break
		}
		if limit > 0 && len(positions) >= limit {
			break
		}
		a, err := s.Analyses().Load(ctx, scope, p.ID)
		switch {
		case err == nil:
			if a != nil && !gammonnet.IsOurAnalysis(a) {
				positions = append(positions, p)
				analyses = append(analyses, a)
			}
		case errors.Is(err, storage.ErrNotFound):
			continue
		default:
			return nil, nil, err
		}
	}
	return positions, analyses, nil
}

// compareGathered runs the engine over the gathered positions on NumCPU
// goroutines, each reusing one searcher, and folds the samples.
func compareGathered(ctx context.Context, positions []domain.Position, stored []*domain.PositionAnalysis, ply, pruneK, candidates int) gammonnet.AnalysisComparison {
	total := len(positions)
	if total == 0 {
		return gammonnet.Aggregate(nil)
	}
	jobs := min(runtime.NumCPU(), total)

	var next atomic.Int64
	results := make(chan gammonnet.ComparisonSample, jobs)
	var wg sync.WaitGroup
	wg.Add(jobs)
	for w := 0; w < jobs; w++ {
		go func() {
			defer wg.Done()
			searcher, _ := gammonnet.NewBatchSearcher(ply, pruneK)
			for {
				if ctx.Err() != nil {
					return
				}
				i := next.Add(1) - 1
				if i >= int64(total) {
					return
				}
				results <- gammonnet.CompareOne(&positions[i], stored[i], positions[i].ID, searcher, ply, pruneK, candidates)
			}
		}()
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	samples := make([]gammonnet.ComparisonSample, 0, total)
	for res := range results {
		samples = append(samples, res)
	}
	return gammonnet.Aggregate(samples)
}
