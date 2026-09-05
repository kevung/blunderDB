package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
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
// /v1/gammonnet.sweepStale (#191) is the same shell run against a different
// gather: every position whose stored analysis is entirely gammonNet's own
// but stale at the requested ply — gammonnet.IsStaleAnalysis is the single
// predicate the GUI/CLI batch (database/db_gammonnet_batch.go) and this
// route both call, never a second copy (CLI/GUI/server parity).
//
// gammonNetJobs reuses importRegistry's cancel-by-id bookkeeping under a
// separate instance, so cancelling an analysis run can never be confused with
// cancelling an import — the same separation #129 kept between
// Database.importCancel and its own dedicated batch context. The same
// registry, and the same /v1/gammonnet.analyzeMissing.cancel route, cancels
// either sweep by job id.

type gammonnetAnalyzeReq struct {
	Ply        int `json:"ply"`
	PruneK     int `json:"pruneK"`
	Candidates int `json:"candidates"`
}

func (s *Server) gammonnetRoutes() []route {
	return []route{
		{http.MethodPost, "/v1/gammonnet.analyzeMissing", s.handleGammonNetAnalyzeMissing},
		{http.MethodPost, "/v1/gammonnet.analyzeMissing.cancel", s.handleGammonNetAnalyzeCancel},
		{http.MethodPost, "/v1/gammonnet.sweepStale", s.handleGammonNetSweepStale},
	}
}

// gammonnetGather snapshots the positions one sweep run should process, from
// the caller's already-decoded request (sweepStale needs req.Ply to compute
// its staleness target; analyzeMissing ignores it).
type gammonnetGather func(ctx context.Context, s storage.Storage, scope string, req gammonnetAnalyzeReq) ([]domain.Position, error)

// handleGammonNetAnalyzeMissing streams NDJSON progress exactly like
// handleImport: {"event":"started",...} once, {"event":"progress",...} as
// positions complete, then {"event":"done",...} or {"event":"error",...}.
func (s *Server) handleGammonNetAnalyzeMissing(w http.ResponseWriter, r *http.Request) {
	s.runGammonNetSweep(w, r, func(ctx context.Context, st storage.Storage, scope string, _ gammonnetAnalyzeReq) ([]domain.Position, error) {
		return gammonnetPositionsWithoutAnalysis(ctx, st, scope)
	})
}

// handleGammonNetSweepStale is analyzeMissing's twin for re-analysis (#191):
// same NDJSON shape, same worker pool, the only difference is which
// positions get gathered — every one whose analysis is entirely gammonNet's
// own but stale at the requested ply, instead of every one with no analysis
// at all.
func (s *Server) handleGammonNetSweepStale(w http.ResponseWriter, r *http.Request) {
	s.runGammonNetSweep(w, r, func(ctx context.Context, st storage.Storage, scope string, req gammonnetAnalyzeReq) ([]domain.Position, error) {
		return gammonnetPositionsWithStaleAnalysis(ctx, st, scope, gammonnet.DepthLabel(req.Ply))
	})
}

// runGammonNetSweep is the shell both gammonNet sweeps share: decode the
// request, register a cancellable job, gather the positions to process
// (the only thing that differs between the two routes), evaluate them on
// NumCPU goroutines, and stream NDJSON progress. Evaluated/refused/failed
// (#191) are counted and reported in the final event — a nil analysis with a
// nil error (a dance, or gammonnet.ErrNotEvaluable — a match score beyond
// the MET's horizon) is "refused", not "failed": before this a refused
// position and a genuinely broken one both just failed to advance done,
// indistinguishable from each other or from a stall.
func (s *Server) runGammonNetSweep(w http.ResponseWriter, r *http.Request, gather gammonnetGather) {
	var req gammonnetAnalyzeReq
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &req); err != nil {
			writeDecodeError(w, "invalid JSON body", err)
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
	// One sweep per tenant (G.11, #239). Two of them do not go twice as fast:
	// they halve each other's cores while both writing analyses into the rows
	// the other is reading as missing. The refusal comes BEFORE the NDJSON
	// stream opens, so the caller gets an ordinary 409 rather than an error
	// event inside a 200.
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

	positions, err := gather(ctx, s.opts.Storage, scope, req)
	if err != nil {
		emit(map[string]any{"event": "error", "error": errorBodyFor(w, err)})
		return
	}
	total := len(positions)

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

	type outcome int
	const (
		outcomeEvaluated outcome = iota
		outcomeRefused
		outcomeFailed
	)

	type analysed struct {
		pos      domain.Position
		analysis *domain.PositionAnalysis
		outcome  outcome
	}

	var next atomic.Int64
	results := make(chan analysed, jobs)
	var wg sync.WaitGroup
	wg.Add(jobs)
	for wk := 0; wk < jobs; wk++ {
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
				pos := positions[i]
				analysis, err := gammonnetEvaluateOne(searcher, pos, req.Ply, req.PruneK, req.Candidates)
				oc := outcomeEvaluated
				switch {
				case err != nil:
					oc = outcomeFailed
					slog.Warn("gammonnet sweep: evaluating a position failed", "position_id", pos.ID, "error", err)
				case analysis == nil:
					oc = outcomeRefused
				}
				results <- analysed{pos: pos, analysis: analysis, outcome: oc}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	var done, evaluated, refused, failed int
	for res := range results {
		oc := res.outcome
		if oc == outcomeEvaluated {
			if err := s.opts.Storage.Analyses().Save(ctx, scope, res.pos.ID, res.analysis); err != nil {
				oc = outcomeFailed
				slog.Warn("gammonnet sweep: saving the computed analysis failed", "position_id", res.pos.ID, "error", err)
			}
		}
		switch oc {
		case outcomeEvaluated:
			evaluated++
		case outcomeRefused:
			refused++
		case outcomeFailed:
			failed++
		}
		// Monotone over positions PROCESSED, refused and failed included
		// (#191) — a refused or failed position used to report no progress
		// at all, so a sweep with any unevaluable positions in it never
		// visibly reached total.
		done++
		emit(map[string]any{"event": "progress", "done": done, "total": total})
	}

	if ctx.Err() != nil {
		emit(map[string]any{"event": "cancelled", "done": done, "total": total, "evaluated": evaluated, "refused": refused, "failed": failed})
		return
	}
	emit(map[string]any{"event": "done", "done": done, "total": total, "evaluated": evaluated, "refused": refused, "failed": failed})
}

type gammonnetCancelReq struct {
	JobID string `json:"jobId"`
}

func (s *Server) handleGammonNetAnalyzeCancel(w http.ResponseWriter, r *http.Request) {
	var req gammonnetCancelReq
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, "invalid JSON body", err)
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
//
// One query, not one per position. It used to list every position and then ask
// Analyses().Load about each one, keeping the ErrNotFound ones: a round trip
// per row to learn something the database states in a join, and the whole
// library materialised first because the SQLite pool is a single connection
// and Load could not run while List still held its rows (G.11, #239).
//
// Still a snapshot rather than a live cursor, and deliberately so: the sweep
// writes the analyses it computes, so a position it has just filled must not
// be a row the cursor has yet to reach. Draining first is also the resume
// mechanism ADR-0013 asks for — a fresh call finds whatever is still missing,
// with no journal.
func gammonnetPositionsWithoutAnalysis(ctx context.Context, s storage.Storage, scope string) ([]domain.Position, error) {
	var missing []domain.Position
	for p, err := range s.Analyses().WithoutAnalysis(ctx, scope, storage.ListOpts{}) {
		if err != nil {
			return nil, err
		}
		if ctx.Err() != nil {
			return missing, nil
		}
		missing = append(missing, *p)
	}
	return missing, nil
}

// gammonnetPositionsWithStaleAnalysis snapshots every position in scope
// whose stored analysis is entirely gammonNet's own but stale at
// targetDepth (#191) — the server-side twin of
// database.positionIDsWithStaleGammonNet, sharing its predicate
// (gammonnet.IsStaleAnalysis) rather than a second copy. A position with no
// analysis at all is left to analyzeMissing, not this sweep; one analysed by
// XG/GNUbg/BGBlitz is never touched, whatever IsStaleAnalysis would say
// about a hypothetical gammonNet entry, because it never gets past
// storage.ErrNotFound here in the first place — IsStaleAnalysis itself
// enforces the same rule for a position that carries BOTH.
func gammonnetPositionsWithStaleAnalysis(ctx context.Context, s storage.Storage, scope, targetDepth string) ([]domain.Position, error) {
	all, err := drainPositions(ctx, s, scope)
	if err != nil {
		return nil, err
	}

	var stale []domain.Position
	for _, p := range all {
		if ctx.Err() != nil {
			return stale, nil
		}
		a, err := s.Analyses().Load(ctx, scope, p.ID)
		switch {
		case err == nil:
			if gammonnet.IsStaleAnalysis(a, targetDepth) {
				stale = append(stale, p)
			}
		case errors.Is(err, storage.ErrNotFound):
			continue
		default:
			return nil, err
		}
	}
	return stale, nil
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
// position"). A nil analysis with a nil error means "nothing to write, and
// that is not a failure": a dance (no legal move) or gammonnet.ErrNotEvaluable
// (a match score beyond the MET's horizon, a cube state the model declines).
// Before #191 ErrNotEvaluable came back as a plain non-nil error here, so a
// position this build can never answer was reported "failed" and left out of
// "done" forever, on every sweep run.
func gammonnetEvaluateOne(searcher *gammonnet.Searcher, pos domain.Position, ply, pruneK, candidates int) (*domain.PositionAnalysis, error) {
	result, err := gammonnet.EvaluatePositionWith(searcher, pos, ply, pruneK, candidates)
	if err != nil {
		if errors.Is(err, gammonnet.ErrNotEvaluable) {
			return nil, nil
		}
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
