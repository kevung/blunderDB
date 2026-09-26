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

// The gammonNet batch (ADR-0013): "an evaluation only ever fills a gap." A
// Position carrying ANY analysis, from any engine, is never touched: the
// query is "no analysis row at all". It lives on *Database so the CLI shares
// it; gui.App wraps it for progress and cancellation.

// CountPositionsWithoutAnalysis reports how many positions have no analysis
// row at all — the batch's known-in-advance total.
func (d *Database) CountPositionsWithoutAnalysis() (int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var n int
	err := d.db.QueryRow(`SELECT COUNT(*) FROM position p WHERE NOT EXISTS (SELECT 1 FROM analysis a WHERE a.position_id = p.id)`).Scan(&n)
	return n, err
}

// positionIDsWithoutAnalysis snapshots the ids to process — a snapshot, not a
// cursor, so the batch's own writes cannot reappear. A fresh call finds what
// is still missing: ADR-0013's resume, with no journal.
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
// that has none, over jobs goroutines (<= 0 means NumCPU). Each search runs
// serially: WithWorkers is the live panel's regime, and stacking both would
// oversubscribe the cores.
//
// ctx is checked before each position, never mid-search (Searcher has no
// cancellation checkpoint). Each goroutine calls yield before each position,
// which blocks while the caller gives priority elsewhere; the batch thus
// yields within at most jobs positions.
//
// onProgress runs on the single writer goroutine with a monotone count of
// positions PROCESSED (evaluated, refused and failed alike).
func (d *Database) AnalyzeMissingWithGammonNet(ctx context.Context, ply, pruneK, candidates, jobs int, yield func(), onProgress func(done, total int)) (GammonNetBatchSummary, error) {
	ids, err := d.positionIDsWithoutAnalysis()
	if err != nil {
		return GammonNetBatchSummary{}, err
	}
	return d.analyzeIDsWithGammonNet(ctx, ids, ply, pruneK, candidates, jobs, yield, onProgress)
}

// positionIDsWithStaleGammonNet snapshots the ids gammonnet.IsStaleAnalysis
// accepts at targetDepth (gammonnet.DepthLabel(ply)). Engine and depth live
// inside the compressed blob, so every analysed position is decoded once. An
// analysis that cannot be loaded is logged and skipped, never silently counted
// as up to date.
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
// the depth ply would write.
func (d *Database) CountPositionsWithStaleGammonNet(ply int) (int, error) {
	ids, err := d.positionIDsWithStaleGammonNet(gammonnet.DepthLabel(ply))
	if err != nil {
		return 0, err
	}
	return len(ids), nil
}

// AnalyzeStaleGammonNet re-runs gammonNet on every position whose analysis is
// entirely its own, at an older EngineVersion or another depth than ply: an
// older engine can be silently wrong at a match score (ADR-0016). Same
// contract as AnalyzeMissingWithGammonNet; a separate pass so ADR-0013 is never
// read as licensing a general re-analysis switch.
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
	// gnRefused: nothing to write, and not a failure — a dance or
	// gammonnet.ErrNotEvaluable (score beyond the MET, cube state declined).
	// Kept apart from gnFailed so it is not retried as a failure forever.
	gnRefused
	// gnFailed: the position could not be loaded, could not be evaluated for
	// a reason other than ErrNotEvaluable, or its analysis could not be
	// saved. Retried on the next run, unchanged.
	gnFailed
)

// GammonNetBatchSummary is a batch's outcome: Evaluated got a new analysis,
// Refused were legitimately skipped (not an error), Failed are retried next
// run. Their sum is the positions processed, less than the total on cancel.
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
// ids from a shared counter, each reusing ONE searcher (~5.5 MB to allocate;
// its cache is bit-neutral, so keeping it warm cannot change an answer).
//
// Every write goes through a single goroutine: Database.mu would serialise N
// writers anyway, and write order is irrelevant.
//
// A cancelled run starts no further position and drains and writes the
// results in flight.
func (d *Database) analyzeIDsWithGammonNet(ctx context.Context, ids []int64, ply, pruneK, candidates, jobs int, yield func(), onProgress func(done, total int)) (GammonNetBatchSummary, error) {
	total := len(ids)
	if total == 0 {
		return GammonNetBatchSummary{}, ctx.Err()
	}

	// Prefetch every position in batched round trips. An id missing from the
	// result was deleted since the snapshot and is reported as a failure.
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

			// One searcher for this goroutine's whole share. If it cannot be
			// built, EvaluatePositionWith(nil) builds one per position: the
			// batch still runs, slower, and the log says why.
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
			// A write failure is a skip like an evaluation failure: the
			// position is picked up again on the next run.
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

		// Monotone over positions PROCESSED, failures and refusals included,
		// so the progress bar reaches total.
		done++
		if onProgress != nil {
			onProgress(done, total)
		}
	}

	return summary, ctx.Err()
}

// evaluateOnePositionWithGammonNet evaluates one already-loaded position; it
// does not write. A nil pos (deleted since the snapshot) returns
// sql.ErrNoRows. A nil analysis with a nil error means "nothing to write, not
// a failure": a dance or gammonnet.ErrNotEvaluable.
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

// positionIDsWithoutAnalysisForMatch is positionIDsWithoutAnalysis narrowed
// to the positions one match actually walks through — the same "no analysis
// row at all" predicate, reached through the move -> game -> match chain a
// Position is never linked to a Match by any other way (a Position row knows
// nothing of the match it came from; the `move` row is the link).
//
// DISTINCT is load-bearing: positions are deduplicated by Zobrist hash, so
// several moves of one match can reach the same Position, which would
// otherwise be analysed several times. A Position shared with another match is
// analysed all the same — by the gap rule (ADR-0013) it gets the write a full
// sweep would give.
func (d *Database) positionIDsWithoutAnalysisForMatch(matchID int64) ([]int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return queryInt64s(d.db, `
		SELECT DISTINCT p.id
		  FROM position p
		  JOIN move mv ON mv.position_id = p.id
		  JOIN game g ON mv.game_id = g.id
		 WHERE g.match_id = ?
		   AND NOT EXISTS (SELECT 1 FROM analysis a WHERE a.position_id = p.id)
		 ORDER BY p.id`, matchID)
}

// CountMatchPositionsToAnalyze is how many of this match's positions have no
// analysis. An unfinished analysis is this count being nonzero, never a
// stored flag (ADR-0045 §8).
func (d *Database) CountMatchPositionsToAnalyze(matchID int64) (int, error) {
	ids, err := d.positionIDsWithoutAnalysisForMatch(matchID)
	if err != nil {
		return 0, err
	}
	return len(ids), nil
}

// AnalyzeMatchWithGammonNet is AnalyzeMissingWithGammonNet scoped to one
// match (ADR-0045 §8), same contract, so a corrected transcription analyses
// only what changed. Deliberately not an optional match id on the library
// batch: a "0 means everything" sentinel is how a scoped batch silently
// becomes a full one.
func (d *Database) AnalyzeMatchWithGammonNet(ctx context.Context, matchID int64, ply, pruneK, candidates, jobs int, yield func(), onProgress func(done, total int)) (GammonNetBatchSummary, error) {
	ids, err := d.positionIDsWithoutAnalysisForMatch(matchID)
	if err != nil {
		return GammonNetBatchSummary{}, err
	}
	return d.analyzeIDsWithGammonNet(ctx, ids, ply, pruneK, candidates, jobs, yield, onProgress)
}
