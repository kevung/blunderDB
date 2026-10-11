package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/maintenance"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// vacuumer is satisfied only by SQLite (PostgreSQL has no file to compact);
// duck-typed like tenantPurger so PostgreSQL needs no stub.
type vacuumer interface {
	Vacuum(ctx context.Context) (storage.VacuumResult, error)
}

// maintenanceRoutes returns the operator's maintenance routes: currently just
// maintenance.vacuum, the daemon's side of the GUI button and the CLI's
// `vacuum`. The three run one implementation on the backend.
func (s *Server) maintenanceRoutes() []route {
	return []route{
		{http.MethodPost, "/ops/maintenance.vacuum", func(w http.ResponseWriter, r *http.Request) {
			v, ok := s.opts.Storage.(vacuumer)
			if !ok {
				writeErrorCode(w, CodeInvalid, "vacuum not supported on this backend (sqlite only)")
				return
			}
			res, err := v.Vacuum(r.Context())
			if err != nil {
				// Never err.Error() to the client: it names the database file.
				// writeStorageError masks and logs it.
				writeStorageError(w, fmt.Errorf("vacuum: %w", err))
				return
			}
			writeJSONResp(w, res)
		}},
	}
}

// reencodeRoutes returns maintenance.reencode, the daemon's side of the CLI's
// `reencode` (ADR-0070). The pass is expensive but stops at the caller's
// tenant, so it stays outside /ops/ like gammonnet.sweepStale.
func (s *Server) reencodeRoutes() []route {
	return []route{
		{http.MethodPost, "/v1/maintenance.reencode", func(w http.ResponseWriter, r *http.Request) {
			n, err := storage.ReencodeAllAnalyses(r.Context(), s.opts.Storage.Analyses(), scopeOf(r), nil)
			if err != nil {
				writeStorageError(w, fmt.Errorf("reencode: %w", err))
				return
			}
			writeJSONResp(w, map[string]int{"rewritten": n})
		}},
	}
}

// maintenanceKindReq names one line of the state of health.
type maintenanceKindReq struct {
	Kind string `json:"kind"`
}

type maintenanceCountResp struct {
	Kind  string `json:"kind"`
	Count int64  `json:"count"`
}

// healthRoutes returns the state of health of ADR-0088: the daemon's side of
// `blunderdb maintenance` and of the GUI's Maintenance tab, through the same
// maintenance package. status, count and ids read only; run streams NDJSON
// and shares the tenant's one slot with the gammonNet sweeps and the
// rollouts (gammonnetJobs), so a second operation is a plain 409.
func (s *Server) healthRoutes() []route {
	return []route{
		{http.MethodPost, "/v1/maintenance.status", rpc(func(ctx context.Context, scope string, _ struct{}) ([]maintenance.Line, error) {
			return maintenance.Status(ctx, s.opts.Storage, scope)
		})},
		{http.MethodPost, "/v1/maintenance.count", rpc(func(ctx context.Context, scope string, req maintenanceKindReq) (maintenanceCountResp, error) {
			n, err := maintenance.Count(ctx, s.opts.Storage, scope, maintenance.Kind(req.Kind))
			return maintenanceCountResp{Kind: req.Kind, Count: n}, err
		})},
		{http.MethodPost, "/v1/maintenance.ids", rpc(func(ctx context.Context, scope string, req maintenanceKindReq) ([]int64, error) {
			ids, err := maintenance.IDs(ctx, s.opts.Storage, scope, maintenance.Kind(req.Kind))
			if ids == nil && err == nil {
				ids = []int64{}
			}
			return ids, err
		})},
		{http.MethodPost, "/v1/maintenance.run", s.handleMaintenanceRun},
		{http.MethodPost, "/v1/maintenance.cancel", s.handleGammonNetAnalyzeCancel},
	}
}

// handleMaintenanceRun runs one line's action as an NDJSON stream: started
// (with the job id maintenance.cancel takes), progress, then done, cancelled
// or error. An unknown or inapplicable operation, and a tenant whose slot is
// taken, are refused before the stream opens.
func (s *Server) handleMaintenanceRun(w http.ResponseWriter, r *http.Request) {
	var req maintenanceKindReq
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, "invalid JSON body", err)
		return
	}
	kind := maintenance.Kind(req.Kind)
	if err := maintenance.Runnable(s.opts.Storage, kind); err != nil {
		writeStorageError(w, err)
		return
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
	run, err := maintenance.RunKind(ctx, s.opts.Storage, scope, kind, maintenance.Opts{
		Progress: func(done, total int) {
			emit(map[string]any{"event": "progress", "done": done, "total": total})
		},
	})
	switch {
	case run.Outcome == maintenance.OutcomeCancelled:
		emit(map[string]any{"event": "cancelled", "run": run})
	case err != nil:
		emit(map[string]any{"event": "error", "error": errorBodyFor(w, err)})
	default:
		emit(map[string]any{"event": "done", "run": run})
	}
}
