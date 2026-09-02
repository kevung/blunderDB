package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// gammonNet's catch-up sweep (#129/#130, ADR-0013, ADR-0015): "blunderdb
// serve operates on a library. gammonnet serve evaluates a position." This is
// the library operation — analyse every position of the caller's tenant that
// has no analysis at all — never a stateless eval(xgid) endpoint. The same
// gap rule and the same conversion (gammonnet.EvaluatePosition) the GUI
// (internal/gui/gammonnet_batch.go) and the CLI (blunderdb analyze) use;
// composed here from the Storage contract's existing accessors
// (Positions().List, Analyses().Load/Save) rather than a new contract method,
// since those three calls are all this needs and both backends already
// satisfy them identically (storagetest/contract.go).
//
// gammonNetJobs reuses importRegistry's cancel-by-id bookkeeping under a
// separate instance, so cancelling an analysis run can never be confused with
// cancelling an import — the same separation #129 kept between
// Database.importCancel and its own dedicated batch context.

type gammonnetAnalyzeReq struct {
	Ply        int `json:"ply"`
	PruneK     int `json:"pruneK"`
	Candidates int `json:"candidates"`
}

func (s *Server) gammonnetRoutes() []route {
	return []route{
		{http.MethodPost, "/v1/gammonnet.analyzeMissing", s.handleGammonNetAnalyzeMissing},
		{http.MethodPost, "/v1/gammonnet.analyzeMissing.cancel", s.handleGammonNetAnalyzeCancel},
	}
}

// handleGammonNetAnalyzeMissing streams NDJSON progress exactly like
// handleImport: {"event":"started",...} once, {"event":"progress",...} as
// positions complete, then {"event":"done",...} or {"event":"error",...}.
func (s *Server) handleGammonNetAnalyzeMissing(w http.ResponseWriter, r *http.Request) {
	var req gammonnetAnalyzeReq
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &req); err != nil {
			writeErrorCode(w, CodeInvalid, "invalid JSON body: "+err.Error())
			return
		}
	}
	// Canonical parameters (ADR-0013) when unset — DefaultConfig/EvaluatePosition
	// already clamp ply, but pruneK/candidates of 0 would mean "no limit" to the
	// searcher/renderer rather than "use the canonical default", so pin them here.
	if req.PruneK <= 0 {
		req.PruneK = 12
	}
	if req.Candidates <= 0 {
		req.Candidates = 10
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	scope := scopeOf(r)
	jobID := s.gammonnetJobs.start(scope, cancel)
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

	missing, err := gammonnetPositionsWithoutAnalysis(ctx, s.opts.Storage, scope)
	if err != nil {
		emit(map[string]any{"event": "error", "error": errorBody{Code: codeForErr(err), Message: err.Error()}})
		return
	}
	total := len(missing)

	// The positions of a sweep are independent, so they are evaluated on
	// NumCPU goroutines (#147), each owning one reused Searcher. Nothing is
	// exposed in the request body: the daemon owns its machine, and a
	// per-caller core budget is a scheduling decision that belongs to
	// whoever runs it, not to the protocol. Known and accepted: two tenants
	// sweeping at the same time ask for NumCPU goroutines each, so they
	// share the cores rather than getting them — the Go scheduler makes that
	// fair, and a sweep is a maintenance operation, not a latency budget.
	//
	// Writing and emitting stay on THIS goroutine. http.ResponseWriter is
	// not safe for concurrent use, and a single writer also keeps the
	// progress count monotone under parallelism.
	jobs := runtime.NumCPU()
	if jobs > total {
		jobs = total
	}

	type analysed struct {
		pos      domain.Position
		analysis *domain.PositionAnalysis
		failed   bool
	}

	var next atomic.Int64
	results := make(chan analysed, jobs)
	var wg sync.WaitGroup
	wg.Add(jobs)
	for w := 0; w < jobs; w++ {
		go func() {
			defer wg.Done()
			searcher, err := gammonnet.NewBatchSearcher(req.Ply, req.PruneK)
			if err != nil {
				searcher = nil // EvaluatePositionWith falls back to a per-position searcher
			}
			for {
				if ctx.Err() != nil {
					return
				}
				i := next.Add(1) - 1
				if i >= int64(total) {
					return
				}
				pos := missing[i]
				analysis, err := gammonnetEvaluateOne(searcher, pos, req.Ply, req.PruneK, req.Candidates)
				results <- analysed{pos: pos, analysis: analysis, failed: err != nil}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	done := 0
	for res := range results {
		// One unevaluable position (a dance, an out-of-range cube state)
		// must not stop the sweep — it stays without analysis and is
		// picked up again, unchanged, next time this runs (ADR-0013's
		// gap rule needs no journal for that).
		if res.failed {
			continue
		}
		if res.analysis != nil {
			if err := s.opts.Storage.Analyses().Save(ctx, scope, res.pos.ID, res.analysis); err != nil {
				continue
			}
		}
		done++
		emit(map[string]any{"event": "progress", "done": done, "total": total})
	}

	if ctx.Err() != nil {
		emit(map[string]any{"event": "cancelled", "done": done, "total": total})
		return
	}
	emit(map[string]any{"event": "done", "done": done, "total": total})
}

type gammonnetCancelReq struct {
	JobID string `json:"jobId"`
}

func (s *Server) handleGammonNetAnalyzeCancel(w http.ResponseWriter, r *http.Request) {
	var req gammonnetCancelReq
	if err := decodeJSON(r, &req); err != nil {
		writeErrorCode(w, CodeInvalid, "invalid JSON body: "+err.Error())
		return
	}
	if req.JobID == "" {
		writeErrorCode(w, CodeInvalid, "jobId is required")
		return
	}
	if !s.gammonnetJobs.cancel(scopeOf(r), req.JobID) {
		writeErrorCode(w, CodeNotFound, "no in-flight analysis with that id")
		return
	}
	writeJSONResp(w, okResp{OK: true})
}

// gammonnetPositionsWithoutAnalysis snapshots every position in scope that has
// no analysis row at all (ADR-0013's gap rule: not "no gammonNet analysis" —
// a Position already analysed by XG/GNUbg/BGBlitz is never touched).
func gammonnetPositionsWithoutAnalysis(ctx context.Context, s storage.Storage, scope string) ([]domain.Position, error) {
	// Two passes, deliberately not one: the SQLite backend pins its pool to a
	// single connection (main.go's ConfigurePool doc comment), so calling
	// Analyses().Load while Positions().List still holds its *sql.Rows open
	// on that one connection deadlocks — List never gets a second connection
	// to lend Load, and Load never runs to let List's Rows advance and close.
	// Draining List into a slice first closes its Rows before any Load runs.
	all, err := drainPositions(ctx, s, scope)
	if err != nil {
		return nil, err
	}

	var missing []domain.Position
	for _, p := range all {
		if ctx.Err() != nil {
			return missing, nil
		}
		_, err := s.Analyses().Load(ctx, scope, p.ID)
		switch {
		case err == nil:
			continue // already analysed by something — the gap rule leaves it alone
		case errors.Is(err, storage.ErrNotFound):
			missing = append(missing, p)
		default:
			return nil, err
		}
	}
	return missing, nil
}

func drainPositions(ctx context.Context, s storage.Storage, scope string) ([]domain.Position, error) {
	var all []domain.Position
	for p, err := range s.Positions().List(ctx, scope, storage.ListOpts{}) {
		if err != nil {
			return nil, err
		}
		all = append(all, *p)
	}
	return all, nil
}

// gammonnetEvaluateOne evaluates one position — the unit of work the
// resume/idempotence guarantee is built on, same as the GUI/CLI batch. It
// does not write: the sweep's single writer goroutine does. searcher is the
// caller's reused searcher (nil is allowed and means "build one for this
// position"). A nil analysis with a nil error means a dance or an unevaluable
// cube state: nothing to write, not an error.
func gammonnetEvaluateOne(searcher *gammonnet.Searcher, pos domain.Position, ply, pruneK, candidates int) (*domain.PositionAnalysis, error) {
	result, err := gammonnet.EvaluatePositionWith(searcher, pos, ply, pruneK, candidates)
	if err != nil {
		return nil, err
	}

	analysis := &domain.PositionAnalysis{
		PositionID:            int(pos.ID),
		AnalysisEngineVersion: gammonnet.EngineVersion,
	}
	switch {
	case len(result.Moves) > 0:
		analysis.AnalysisType = "CheckerMove"
		analysis.CheckerAnalysis = &domain.CheckerAnalysis{Moves: result.Moves}
	case result.Cube != nil:
		analysis.AnalysisType = "DoublingCube"
		analysis.DoublingCubeAnalysis = result.Cube
	default:
		return nil, nil
	}

	return analysis, nil
}
