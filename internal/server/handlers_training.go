package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The Training journal over HTTP: a client that drills a person in front of
// its own board records the sessions here and reads back the same summaries
// the desktop tab shows. The questions themselves are drawn by the client;
// the daemon keeps the journal, scoped to the tenant like every other row.

type trainingSessionsReq struct {
	// Exercise narrows to one exercise; empty is every exercise.
	Exercise string `json:"exercise"`
	// Limit bounds the sessions returned, most recent first; 0 is no bound.
	Limit int `json:"limit"`
}

type trainingStatsReq struct {
	Exercise string `json:"exercise"`
}

type trainingSaveResp struct {
	ID int64 `json:"id"`
}

func (s *Server) trainingRoutes() []route {
	ts := func() storage.TrainingStore { return s.opts.Storage.Training() }
	return []route{
		{http.MethodPost, "/v1/training.sessions", rpc(func(ctx context.Context, scope string, req trainingSessionsReq) ([]storage.TrainingSession, error) {
			if req.Limit < 0 {
				return nil, fmt.Errorf("%w: limit must be 0 (no bound) or more", storage.ErrInvalid)
			}
			out, err := ts().Sessions(ctx, scope, req.Exercise, req.Limit)
			if out == nil {
				out = []storage.TrainingSession{}
			}
			return out, err
		})},
		{http.MethodPost, "/v1/training.numberStats", rpc(func(ctx context.Context, scope string, req trainingStatsReq) ([]storage.TrainingNumberStat, error) {
			out, err := ts().NumberStats(ctx, scope, req.Exercise)
			if out == nil {
				out = []storage.TrainingNumberStat{}
			}
			return out, err
		})},
		{http.MethodPost, "/v1/training.missed", rpc(func(ctx context.Context, scope string, req storage.TrainingMissedFilter) ([]int64, error) {
			if req.Limit < 0 || req.SessionID < 0 {
				return nil, fmt.Errorf("%w: limit and sessionId must be 0 (no bound) or more", storage.ErrInvalid)
			}
			out, err := ts().Missed(ctx, scope, req)
			if out == nil {
				out = []int64{}
			}
			return out, err
		})},
		{http.MethodPost, "/v1/training.save", s.withIdempotency(rpc(func(ctx context.Context, scope string, req storage.TrainingSession) (trainingSaveResp, error) {
			if req.Exercise == "" {
				return trainingSaveResp{}, fmt.Errorf("%w: a session names its exercise", storage.ErrInvalid)
			}
			req.ID = 0
			id, err := ts().Save(ctx, scope, req)
			return trainingSaveResp{ID: id}, err
		}))},
	}
}
