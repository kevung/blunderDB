package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/rollout"
	"github.com/kevung/blunderdb/pkg/blunderdb/rollouts"
	"github.com/kevung/blunderdb/pkg/blunderdb/searchquery"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Rollouts (ADR-0060): one position on demand, or every position a search
// query selects, written beside the analysis it carries and never in its
// place (ADR-0013). The gather, the loop and the write are
// pkg/blunderdb/rollouts', shared with the GUI and the CLI.
//
// rollout.filter is a library sweep like gammonnet.analyzeMissing: it shares
// that registry, so a tenant runs one sweep of either kind at a time — both
// write analyses and both take every core.

// rolloutPositionReq names a position of the library: serve operates on a
// library and exposes no evaluator of a bare position (ADR-0015). Rollout is
// the settings line rollout.ParseSpec reads ("fast", "standard",
// "games=648,ply=1", …); empty means fast. Store writes the finished rollout
// beside the position's analysis; without it nothing is written.
type rolloutPositionReq struct {
	PositionID int64    `json:"positionId"`
	Rollout    string   `json:"rollout"`
	Moves      []string `json:"moves,omitempty"`
	Store      bool     `json:"store,omitempty"`
}

// rolloutPositionResp is the finished rollout; Stored says it was written.
type rolloutPositionResp struct {
	Result *rollout.Result `json:"result"`
	Stored bool            `json:"stored"`
}

// rolloutFilterReq selects positions with the search query language.
type rolloutFilterReq struct {
	Query   string `json:"query"`
	Rollout string `json:"rollout"`
}

type rolloutListResp struct {
	Rollouts []domain.RolloutAnalysis `json:"rollouts"`
}

func (s *Server) rolloutRoutes() []route {
	return []route{
		{http.MethodPost, "/v1/rollout.position", s.handleRolloutPosition},
		{http.MethodPost, "/v1/rollout.filter", s.handleRolloutFilter},
		{http.MethodPost, "/v1/rollout.filter.cancel", s.handleGammonNetAnalyzeCancel},
		{http.MethodPost, "/v1/rollout.list", rpc(func(ctx context.Context, scope string, req positionIDReq) (rolloutListResp, error) {
			list, err := rollouts.List(ctx, s.opts.Storage, scope, req.PositionID)
			return rolloutListResp{Rollouts: list}, err
		})},
	}
}

// handleRolloutPosition rolls one position out and answers when it is done;
// a client that goes away cancels it, and nothing is stored then.
func (s *Server) handleRolloutPosition(w http.ResponseWriter, r *http.Request) {
	var req rolloutPositionReq
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, "invalid JSON body", err)
		return
	}
	settings, err := rollout.ParseSpec(req.Rollout)
	if err != nil {
		writeErrorCode(w, CodeInvalid, err.Error())
		return
	}
	if req.PositionID == 0 {
		writeErrorCode(w, CodeInvalid, "positionId is required: a rollout runs on a position of the library")
		return
	}

	ctx, scope := r.Context(), scopeOf(r)
	res, err := rollouts.Position(ctx, s.opts.Storage, scope, req.PositionID, settings, req.Moves, nil)
	switch {
	case errors.Is(err, context.Canceled):
		return // nobody is listening
	case errors.Is(err, storage.ErrNotFound):
		writeStorageError(w, err)
		return
	case err != nil:
		// The engine refused the request: a play that is not legal, a
		// position with nothing to decide.
		writeErrorCode(w, CodeInvalid, err.Error())
		return
	}
	if req.Store {
		if err := rollouts.Store(ctx, s.opts.Storage, scope, req.PositionID, res); err != nil {
			writeStorageError(w, err)
			return
		}
	}
	writeJSONResp(w, rolloutPositionResp{Result: res, Stored: req.Store})
}

// handleRolloutFilter streams NDJSON as the gammonNet sweeps do:
// {"event":"started","job_id":…}, {"event":"progress",…} after every batch
// of games, then {"event":"done"|"cancelled",…summary} or {"event":"error"}.
func (s *Server) handleRolloutFilter(w http.ResponseWriter, r *http.Request) {
	var req rolloutFilterReq
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, "invalid JSON body", err)
		return
	}
	settings, err := rollout.ParseSpec(req.Rollout)
	if err != nil {
		writeErrorCode(w, CodeInvalid, err.Error())
		return
	}
	filters, diags := searchquery.Parse(req.Query)
	for _, d := range diags {
		if d.Kind == searchquery.DiagUnknown {
			writeErrorCode(w, CodeInvalid, "unknown token in query: "+d.Token)
			return
		}
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	scope := scopeOf(r)
	jobID, err := s.gammonnetJobs.startExclusive(scope, cancel)
	if err != nil {
		writeStorageError(w, err)
		return
	}
	defer s.gammonnetJobs.finish(jobID)

	w.Header().Set("Content-Type", ndjsonContentType)
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	fl, _ := w.(http.Flusher)
	emit := func(v any) {
		_ = enc.Encode(v)
		if fl != nil {
			fl.Flush()
		}
	}
	emit(map[string]any{"event": "started", "job_id": jobID})

	positions, err := rollouts.Gather(ctx, s.opts.Storage, scope, filters, settings)
	if err != nil {
		emit(map[string]any{"event": "error", "error": errorBodyFor(w, err)})
		return
	}
	// Progress is emitted from Batch's own goroutine, the only writer.
	sum, err := rollouts.Batch(ctx, positions, settings, func(p rollouts.Progress) {
		emit(map[string]any{"event": "progress", "done": p.Done, "total": p.Total,
			"positionId": p.PositionID, "games": p.Games, "maxGames": p.MaxGames})
	}, func(id int64, res *rollout.Result) error {
		return rollouts.Store(ctx, s.opts.Storage, scope, id, res)
	})
	if err != nil {
		emit(map[string]any{"event": "error", "error": errorBodyFor(w, err)})
		return
	}
	event := "done"
	if sum.Cancelled {
		event = "cancelled"
	}
	emit(map[string]any{"event": event, "total": sum.Total, "rolledOut": sum.RolledOut,
		"refused": sum.Refused, "failed": sum.Failed, "signature": settings.Signature()})
}
