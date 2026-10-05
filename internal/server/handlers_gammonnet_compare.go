package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// /v1/gammonnet.compare — what gammonNet is worth on this tenant's own
// library: run the engine on positions analysed by someone else and fold the
// samples. What a comparison IS lives in gammonnet/compare.go, shared with the
// desktop. Not streamed, no job id: it writes nothing and answers one small
// object; `limit` bounds the cost.

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
	if s.refuseAnalysis(w, scope) {
		return
	}
	positions, stored, err := gammonnetPositionsWithForeignAnalysis(ctx, s.opts.Storage, scope, req.Limit)
	if err != nil {
		writeStorageError(w, err)
		return
	}
	writeJSONResp(w, compareGathered(ctx, s.analysis, scope, positions, stored, req.Ply, req.PruneK, req.Candidates, s.quota.spender(scope)))
}

// gammonnetCompareResp is the comparison, and whether the tenant's engine
// time ran out before every gathered position was looked at: such a
// comparison folds only some of the Gathered positions, and says so rather
// than pass for the whole library.
type gammonnetCompareResp struct {
	gammonnet.AnalysisComparison
	Gathered      int  `json:"gathered"`
	QuotaExceeded bool `json:"quotaExceeded"`
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

// compareGathered runs the engine over the gathered positions, one unit of
// the shared workers per position, and folds the samples. It stops handing
// out positions once spend says the tenant's engine time is out.
func compareGathered(ctx context.Context, pool *analysisPool, scope string, positions []domain.Position, stored []*domain.PositionAnalysis, ply, pruneK, candidates int, spend func(time.Duration) bool) gammonnetCompareResp {
	total := len(positions)
	if total == 0 {
		return gammonnetCompareResp{AnalysisComparison: gammonnet.Aggregate(nil)}
	}
	results := make([]gammonnet.ComparisonSample, total)
	ran := make([]bool, total)
	err := pool.each(ctx, scope, total, spend, func(i int, get searcherFor) {
		results[i] = gammonnet.CompareOne(&positions[i], stored[i], positions[i].ID, get(ply, pruneK), ply, pruneK, candidates)
		ran[i] = true
	})
	samples := make([]gammonnet.ComparisonSample, 0, total)
	for i, ok := range ran {
		if ok {
			samples = append(samples, results[i])
		}
	}
	return gammonnetCompareResp{
		AnalysisComparison: gammonnet.Aggregate(samples),
		Gathered:           total,
		QuotaExceeded:      errors.Is(err, errAnalysisSpent) && len(samples) < total,
	}
}
