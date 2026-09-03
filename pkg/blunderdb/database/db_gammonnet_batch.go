package database

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

// The gammonNet batch (#129, ADR-0013): "an evaluation only ever fills a
// gap." A Position already carrying ANY analysis — from XG, GNUbg, BGBlitz,
// or a prior gammonNet run — is never touched here, regardless of which
// engine is missing; the query below is "no analysis row at all", not "no
// gammonNet analysis". SaveAnalysis's own per-engine merge is therefore never
// exercised by this batch: it only ever writes the first analysis a Position
// gets.
//
// The batch lives here, on *Database, rather than in internal/gui, so the
// CLI (#130) can call it too without a GUI dependency; gui.App wraps it in
// its own goroutine+context+mutex+EventsEmit shell (gammonnet_batch.go) for
// the progress bar and cancellation, exactly like DownloadBearoffDB wraps a
// plain download function.

// CountPositionsWithoutAnalysis reports how many positions have no analysis
// row at all — the batch's known-in-advance total.
func (d *Database) CountPositionsWithoutAnalysis() (int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var n int
	err := d.db.QueryRow(`SELECT COUNT(*) FROM position p WHERE NOT EXISTS (SELECT 1 FROM analysis a WHERE a.position_id = p.id)`).Scan(&n)
	return n, err
}

// positionIDsWithoutAnalysis snapshots the ids to process. A snapshot, not a
// live cursor: positions the batch itself writes during the run must not
// reappear in the same pass (SaveAnalysis fills the gap this query looks
// for), and a fresh call after a cancelled or completed run simply finds
// whatever is still missing — the resume mechanism ADR-0013 asks for, with
// no journal.
func (d *Database) positionIDsWithoutAnalysis() ([]int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query(`SELECT p.id FROM position p WHERE NOT EXISTS (SELECT 1 FROM analysis a WHERE a.position_id = p.id) ORDER BY p.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// AnalyzeMissingWithGammonNet writes a gammonNet analysis for every position
// that has none, spreading the positions over jobs goroutines (#147). The
// positions of a batch are independent by nature — one search never informs
// the next — so the batch is where the parallelism belongs; each search runs
// in series on purpose (WithWorkers is the live panel's regime, and stacking
// the two would oversubscribe the cores). jobs <= 0 means runtime.NumCPU().
//
// It is cancellable through ctx, checked by every goroutine before each
// position (never mid-search: gammonnet.Searcher has no internal
// cancellation checkpoint, the same known limit gammonnet_eval.go documents
// for the live panel). Before starting each position, every goroutine calls
// yield, which must block for as long as the caller wants to give priority
// to something else — internal/gui's wrapper uses it to let an interactive
// live evaluation go first. With jobs goroutines the batch therefore yields
// within at most jobs positions, not one; that is the contract now.
//
// onProgress is called from the single writer goroutine, with a monotone
// counter over positions PROCESSED — evaluated, refused and failed alike
// (#191) — never a loop index, which parallel goroutines would report out
// of order. The returned GammonNetBatchSummary is the caller's end-of-run
// figure: how many of the positions the snapshot named actually got a new
// analysis, how many were legitimately skipped, and how many failed outright
// and are picked up again, unchanged, the next time this runs.
func (d *Database) AnalyzeMissingWithGammonNet(ctx context.Context, ply, pruneK, candidates, jobs int, yield func(), onProgress func(done, total int)) (GammonNetBatchSummary, error) {
	ids, err := d.positionIDsWithoutAnalysis()
	if err != nil {
		return GammonNetBatchSummary{}, err
	}
	return d.analyzeIDsWithGammonNet(ctx, ids, ply, pruneK, candidates, jobs, yield, onProgress)
}

// positionIDsWithStaleGammonNet snapshots the ids gammonnet.IsStaleAnalysis
// accepts at targetDepth (gammonnet.DepthLabel(ply) — the exact string a run
// at that ply will write). Unlike positionIDsWithoutAnalysis this cannot be
// a single SQL WHERE clause — AnalysisEngine and AnalysisDepth live inside
// the compressed JSON blob, not a column — so every analysed position is
// loaded and decoded once. Run deliberately (a version bump, not
// automatically), the same posture as the existing gap-fill batch.
//
// A position whose stored analysis cannot be loaded at all is logged and
// skipped rather than silently counted as "up to date" (B.6, #174): the
// distinction matters because the two look identical to a caller that only
// reads the count that comes back, and an operator investigating why a
// position never gets swept needs the log line, not a guess.
func (d *Database) positionIDsWithStaleGammonNet(targetDepth string) ([]int64, error) {
	d.mu.RLock()
	ids, err := queryInt64s(d.db, `SELECT position_id FROM analysis ORDER BY position_id`)
	d.mu.RUnlock()
	if err != nil {
		return nil, err
	}

	var stale []int64
	for _, id := range ids {
		a, err := d.LoadAnalysis(id)
		if err != nil {
			slog.Warn("gammonnet stale sweep: loading stored analysis failed, position left out of the sweep", "position_id", id, "error", err)
			continue
		}
		if a == nil {
			continue
		}
		if gammonnet.IsStaleAnalysis(a, targetDepth) {
			stale = append(stale, id)
		}
	}
	return stale, nil
}

// CountPositionsWithStaleGammonNet is len(positionIDsWithStaleGammonNet) at
// the depth ply would write — the number a "re-analyse stale positions (N)"
// button needs before committing to the full decode/scan, the same shape as
// CountPositionsWithoutAnalysis.
func (d *Database) CountPositionsWithStaleGammonNet(ply int) (int, error) {
	ids, err := d.positionIDsWithStaleGammonNet(gammonnet.DepthLabel(ply))
	if err != nil {
		return 0, err
	}
	return len(ids), nil
}

// AnalyzeStaleGammonNet re-runs gammonNet on every position whose stored
// analysis is entirely its own, at an EngineVersion older than the running
// build's or a different depth than ply now asks for (#191) — ADR-0016's
// use_match changed what a money-only number MEANS at a match score, so a
// v1.0.1 row is not merely outdated, it can be silently wrong, and a 0-ply
// row is not what a 2-ply canonical depth promises either. Same shape and
// cancellation contract as AnalyzeMissingWithGammonNet; kept as a separate
// pass because the two query different things (no analysis at all, vs. an
// entirely-ours but stale one) and ADR-0013 must never be read as licensing
// a general re-analysis switch.
func (d *Database) AnalyzeStaleGammonNet(ctx context.Context, ply, pruneK, candidates, jobs int, yield func(), onProgress func(done, total int)) (GammonNetBatchSummary, error) {
	ids, err := d.positionIDsWithStaleGammonNet(gammonnet.DepthLabel(ply))
	if err != nil {
		return GammonNetBatchSummary{}, err
	}
	return d.analyzeIDsWithGammonNet(ctx, ids, ply, pruneK, candidates, jobs, yield, onProgress)
}

// gammonNetOutcome is what evaluateOnePositionWithGammonNet's caller decided
// happened to one position, once a write (if any) has been attempted.
type gammonNetOutcome int

const (
	// gnEvaluated: a new analysis was computed and written.
	gnEvaluated gammonNetOutcome = iota
	// gnRefused: nothing to write, and that is not a failure — a dance (no
	// legal move) or gammonnet.ErrNotEvaluable (a match score beyond the
	// MET's horizon, a cube state the model declines). Before #191 this was
	// indistinguishable from gnFailed: ErrNotEvaluable arrived as a plain
	// error, so a position this build will NEVER be able to answer was
	// retried on every single pass, forever, and counted as a failure each
	// time.
	gnRefused
	// gnFailed: the position could not be loaded, could not be evaluated for
	// a reason other than ErrNotEvaluable, or its analysis could not be
	// saved. Retried on the next run, unchanged.
	gnFailed
)

// GammonNetBatchSummary is a batch's outcome, split three ways (#191):
// Evaluated is how many positions got a new analysis written; Refused is how
// many were legitimately skipped (gnRefused above) — not failures, and a
// caller should not read a nonzero Refused as anything being wrong; Failed
// is how many could not be loaded, evaluated or saved, and are retried the
// next time this runs. Evaluated+Refused+Failed is the number of positions
// actually processed, which can be less than the snapshot's total when the
// run is cancelled partway through.
type GammonNetBatchSummary struct {
	Evaluated int
	Refused   int
	Failed    int
}

// Processed is Evaluated+Refused+Failed — the positions this run actually
// looked at, the number progress reporting counts up to.
func (s GammonNetBatchSummary) Processed() int {
	return s.Evaluated + s.Refused + s.Failed
}

// gammonNetBatchResult is one position's outcome on its way from a worker
// goroutine to the single goroutine that writes.
type gammonNetBatchResult struct {
	id       int64
	analysis *PositionAnalysis
	outcome  gammonNetOutcome
}

// analyzeIDsWithGammonNet is the batch both passes run: jobs goroutines take
// ids from a shared counter, each owning ONE searcher it reuses from one
// position to the next (a fresh one costs about 5.5 MB to allocate and zero,
// and its evaluation cache is worth keeping warm — cache.go: a hit returns
// exactly what a miss would have computed, so carrying it across positions
// cannot move a bit of the answer).
//
// Every write goes through a single goroutine. Database.mu is an RWMutex
// over the legacy wrapper (CLAUDE.md, "Concurrency"): N goroutines writing
// would serialise on it anyway, and the order of the writes changes nothing
// — each position's analysis stands on its own.
//
// Cancellation keeps the sequential contract exactly (CLI_USAGE.md): a
// cancelled run starts no further position, loses nothing already computed
// (the results still in flight are drained and written), and re-running
// simply finds whatever is still missing.
func (d *Database) analyzeIDsWithGammonNet(ctx context.Context, ids []int64, ply, pruneK, candidates, jobs int, yield func(), onProgress func(done, total int)) (GammonNetBatchSummary, error) {
	total := len(ids)
	if total == 0 {
		return GammonNetBatchSummary{}, ctx.Err()
	}

	// Every position is loaded once, in as few round trips as LoadByIDs's own
	// batching needs, instead of once per position inside a worker's loop —
	// evaluateOnePositionWithGammonNet used to call LoadPosition itself, one
	// RLock and one query per id (B.11, #179). An id missing from the result
	// (deleted between positionIDsWithoutAnalysis's snapshot and this
	// prefetch — the only way this map can lack an id it was given) is
	// reported as a failure below, exactly like a LoadPosition error used to
	// be.
	loaded, err := d.LoadPositionsByIDs(ids)
	if err != nil {
		return GammonNetBatchSummary{}, err
	}
	positionsByID := make(map[int64]*Position, len(loaded))
	for i := range loaded {
		positionsByID[loaded[i].ID] = &loaded[i]
	}

	if jobs <= 0 {
		jobs = runtime.NumCPU()
	}
	if jobs > total {
		jobs = total
	}

	var next atomic.Int64
	results := make(chan gammonNetBatchResult, jobs)

	var wg sync.WaitGroup
	wg.Add(jobs)
	for w := 0; w < jobs; w++ {
		go func() {
			defer wg.Done()

			// One searcher for this goroutine's whole share of the batch.
			// A searcher this build cannot create at all (no embedded
			// network) leaves the goroutine idle rather than falling back to
			// one-per-position: EvaluatePositionWith takes nil and builds
			// its own, so the batch still runs, just without the saving.
			// Logged (B.6, #174) — this failure used to be entirely silent,
			// so a build missing its embedded network ran the whole batch
			// at a quiet, permanent slowdown with nothing in the log to
			// explain it.
			searcher, err := gammonnet.NewBatchSearcher(ply, pruneK)
			if err != nil {
				slog.Warn("gammonnet batch: building the shared searcher failed; this worker falls back to one searcher per position", "error", err)
				searcher = nil
			}

			for {
				if ctx.Err() != nil {
					return
				}
				if yield != nil {
					yield()
				}
				if ctx.Err() != nil {
					return
				}

				i := next.Add(1) - 1
				if i >= int64(total) {
					return
				}
				id := ids[i]

				analysis, err := evaluateOnePositionWithGammonNet(positionsByID[id], id, searcher, ply, pruneK, candidates)
				outcome := gnEvaluated
				switch {
				case err != nil:
					outcome = gnFailed
					slog.Warn("gammonnet batch: evaluating a position failed", "position_id", id, "error", err)
				case analysis == nil:
					outcome = gnRefused
				}
				results <- gammonNetBatchResult{id: id, analysis: analysis, outcome: outcome}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var summary GammonNetBatchSummary
	done := 0
	for res := range results {
		outcome := res.outcome
		if outcome == gnEvaluated {
			// A write failure is the same kind of skip as an evaluation
			// failure: the position stays as it was and is picked up again
			// on the next run. Logged (B.6, #174) — silent before, and the
			// only signal a caller had was a lower count than expected.
			if err := d.SaveAnalysis(res.id, *res.analysis); err != nil {
				outcome = gnFailed
				slog.Warn("gammonnet batch: saving the computed analysis failed", "position_id", res.id, "error", err)
			}
		}

		switch outcome {
		case gnEvaluated:
			summary.Evaluated++
		case gnRefused:
			summary.Refused++
		case gnFailed:
			summary.Failed++
		}

		// Monotone over positions PROCESSED, failures and refusals included
		// (#191) — before this, a failed or refused position reported no
		// progress at all, so a batch with any unevaluable positions in it
		// never visibly finished: the progress bar stalled short of total
		// even though every position had in fact been looked at.
		done++
		if onProgress != nil {
			onProgress(done, total)
		}
	}

	return summary, ctx.Err()
}

// evaluateOnePositionWithGammonNet evaluates one already-loaded position —
// the unit of work the resume/idempotence guarantee is built on. pos is nil
// when analyzeIDsWithGammonNet's batched prefetch (LoadPositionsByIDs) did
// not return this id — deleted between positionIDsWithoutAnalysis's snapshot
// and the prefetch — reported as sql.ErrNoRows, the same error LoadPosition
// itself used to return for exactly that case back when this function loaded
// the position on its own, one query per call (B.11, #179).
//
// It does not write: the caller's single writer goroutine does. A nil
// analysis with a nil error means "nothing to write, and that is not a
// failure": a dance (no legal move) or gammonnet.ErrNotEvaluable (a match
// score beyond the MET's horizon, a cube state the model declines) — before
// #191 ErrNotEvaluable came back as a plain non-nil error, so the caller
// counted it as a failure and retried it on every single pass, forever,
// exactly like a position that genuinely could not be read. Either way the
// position stays without an analysis and is picked up again on the next run
// — but only ErrNotEvaluable and a dance are expected to keep coming back
// unchanged; a real error is not.
func evaluateOnePositionWithGammonNet(pos *Position, id int64, searcher *gammonnet.Searcher, ply, pruneK, candidates int) (*PositionAnalysis, error) {
	if pos == nil {
		return nil, sql.ErrNoRows
	}

	result, err := gammonnet.EvaluatePositionWith(searcher, *pos, ply, pruneK, candidates)
	if err != nil {
		if errors.Is(err, gammonnet.ErrNotEvaluable) {
			return nil, nil
		}
		return nil, err
	}

	analysis := PositionAnalysis{
		PositionID:            int(id),
		AnalysisEngineVersion: gammonnet.EngineVersion,
	}
	switch {
	case len(result.Moves) > 0:
		analysis.AnalysisType = "CheckerMove"
		analysis.CheckerAnalysis = &CheckerAnalysis{Moves: result.Moves}
	case result.Cube != nil:
		analysis.AnalysisType = "DoublingCube"
		analysis.DoublingCubeAnalysis = result.Cube
	default:
		return nil, nil
	}

	return &analysis, nil
}
