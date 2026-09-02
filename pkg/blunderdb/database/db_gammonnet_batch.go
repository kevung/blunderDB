package database

import (
	"context"
	"runtime"
	"strings"
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
	err := d.db.QueryRow(`SELECT COUNT(*) FROM position WHERE id NOT IN (SELECT position_id FROM analysis)`).Scan(&n)
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

	rows, err := d.db.Query(`SELECT id FROM position WHERE id NOT IN (SELECT position_id FROM analysis) ORDER BY id`)
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
// counter — never a loop index, which parallel goroutines would report out
// of order. A position that fails to load or evaluate reports nothing and
// stops nothing: it stays without analysis and is picked up again, unchanged,
// the next time this runs.
func (d *Database) AnalyzeMissingWithGammonNet(ctx context.Context, ply, pruneK, candidates, jobs int, yield func(), onProgress func(done, total int)) error {
	ids, err := d.positionIDsWithoutAnalysis()
	if err != nil {
		return err
	}
	return d.analyzeIDsWithGammonNet(ctx, ids, ply, pruneK, candidates, jobs, yield, onProgress)
}

// gammonNetEngineVersionPrefix identifies an analysis entry as gammonNet's
// own, whatever its exact version — "gammonNet v1.0.1", "gammonNet v1.1.0",
// … — the prefix that separates "our own past opinion, safe to recompute"
// from "someone else's, permanently hands-off" below.
const gammonNetEngineVersionPrefix = "gammonNet "

// isStaleGammonNetOnly reports whether a's every entry is gammonNet's own
// and at least one is older than the running build's EngineVersion —
// ADR-0016's narrow exception to ADR-0013: a position that also carries an
// XG, GNUbg or BGBlitz entry is never reported stale here, whatever its
// gammonNet entries say, because ADR-0013 protects an imported analysis
// unconditionally and this batch only ever touches its own past output.
func isStaleGammonNetOnly(a *PositionAnalysis) bool {
	allOurs, anyStale, sawAny := true, false, false
	check := func(engine string) {
		sawAny = true
		if !strings.HasPrefix(engine, gammonNetEngineVersionPrefix) {
			allOurs = false
			return
		}
		if engine != gammonnet.EngineVersion {
			anyStale = true
		}
	}
	if a.CheckerAnalysis != nil {
		for _, m := range a.CheckerAnalysis.Moves {
			check(m.AnalysisEngine)
		}
	}
	if a.DoublingCubeAnalysis != nil {
		check(a.DoublingCubeAnalysis.AnalysisEngine)
	}
	return sawAny && allOurs && anyStale
}

// positionIDsWithStaleGammonNet snapshots the ids isStaleGammonNetOnly
// accepts. Unlike positionIDsWithoutAnalysis this cannot be a single SQL
// WHERE clause — AnalysisEngine lives inside the compressed JSON blob, not a
// column — so every analysed position is loaded and decoded once. Run
// deliberately (a version bump, not automatically), the same posture as the
// existing gap-fill batch.
func (d *Database) positionIDsWithStaleGammonNet() ([]int64, error) {
	d.mu.RLock()
	ids, err := queryInt64s(d.db, `SELECT position_id FROM analysis ORDER BY position_id`)
	d.mu.RUnlock()
	if err != nil {
		return nil, err
	}

	var stale []int64
	for _, id := range ids {
		a, err := d.LoadAnalysis(id)
		if err != nil || a == nil {
			continue
		}
		if isStaleGammonNetOnly(a) {
			stale = append(stale, id)
		}
	}
	return stale, nil
}

// AnalyzeStaleGammonNet re-runs gammonNet on every position whose stored
// analysis is entirely its own, at an EngineVersion older than the running
// build's — ADR-0016's use_match changed what a money-only number MEANS at a
// match score, so a v1.0.1 row is not merely outdated, it can be silently
// wrong. Same shape and cancellation contract as AnalyzeMissingWithGammonNet;
// kept as a separate pass because the two query different things (no
// analysis at all, vs. an entirely-ours but stale one) and ADR-0013 must
// never be read as licensing a general re-analysis switch.
func (d *Database) AnalyzeStaleGammonNet(ctx context.Context, ply, pruneK, candidates, jobs int, yield func(), onProgress func(done, total int)) error {
	ids, err := d.positionIDsWithStaleGammonNet()
	if err != nil {
		return err
	}
	return d.analyzeIDsWithGammonNet(ctx, ids, ply, pruneK, candidates, jobs, yield, onProgress)
}

// gammonNetBatchResult is one position's outcome on its way from a worker
// goroutine to the single goroutine that writes. analysis is nil when there
// is nothing to write (a dance, an unevaluable cube state) — which is not a
// failure; failed marks a position that could not be loaded or evaluated at
// all, the only case that reports no progress, exactly as the sequential
// batch did by skipping its onProgress call.
type gammonNetBatchResult struct {
	id       int64
	analysis *PositionAnalysis
	failed   bool
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
func (d *Database) analyzeIDsWithGammonNet(ctx context.Context, ids []int64, ply, pruneK, candidates, jobs int, yield func(), onProgress func(done, total int)) error {
	total := len(ids)
	if total == 0 {
		return ctx.Err()
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
			searcher, err := gammonnet.NewBatchSearcher(ply, pruneK)
			if err != nil {
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

				analysis, err := d.evaluateOnePositionWithGammonNet(searcher, id, ply, pruneK, candidates)
				results <- gammonNetBatchResult{id: id, analysis: analysis, failed: err != nil}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	done := 0
	for res := range results {
		ok := !res.failed
		if ok && res.analysis != nil {
			// A write failure is the same kind of skip as an evaluation
			// failure: the position stays as it was and is picked up again
			// on the next run.
			if err := d.SaveAnalysis(res.id, *res.analysis); err != nil {
				ok = false
			}
		}
		done++
		if ok && onProgress != nil {
			onProgress(done, total)
		}
	}

	return ctx.Err()
}

// evaluateOnePositionWithGammonNet loads and evaluates one position — the
// unit of work the resume/idempotence guarantee is built on. It does not
// write: the caller's single writer goroutine does. A nil analysis with a nil
// error means "nothing to write": a dance (no legal move) or an unevaluable
// cube state (e.g. beyond the MET's range) — not an error, the position stays
// without analysis and is retried on the next run exactly like any other
// still-missing position.
func (d *Database) evaluateOnePositionWithGammonNet(searcher *gammonnet.Searcher, id int64, ply, pruneK, candidates int) (*PositionAnalysis, error) {
	pos, err := d.LoadPosition(int(id))
	if err != nil {
		return nil, err
	}

	result, err := gammonnet.EvaluatePositionWith(searcher, *pos, ply, pruneK, candidates)
	if err != nil {
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
