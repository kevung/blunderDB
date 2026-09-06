package database

import (
	"context"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

// The desktop/CLI half of the comparison sweep (issue #270, fiche I.14).
//
// What a comparison IS — what counts as the same answer, how a disagreement is
// priced, how the samples aggregate — lives in engine/gammonnet/compare.go, so
// the daemon and this wrapper cannot count differently. What lives here is the
// half that is genuinely this mode's: finding the positions to compare in a
// SQLite file, and running them on this process's cores.
//
// It writes nothing, and that is the point rather than a precaution: ADR-0013
// protects an imported analysis unconditionally, and the value of the sweep is
// precisely that it can be run on a library nobody is willing to have
// rewritten.

// positionIDsWithForeignAnalysis lists the positions carrying an analysis
// somebody else wrote — the only ones a comparison has anything to compare
// against. A position analysed solely by gammonNet would be compared with
// itself.
func (d *Database) positionIDsWithForeignAnalysis() ([]int64, error) {
	d.mu.RLock()
	ids, err := queryInt64s(d.db, `SELECT position_id FROM analysis ORDER BY position_id`)
	d.mu.RUnlock()
	if err != nil {
		return nil, err
	}
	var foreign []int64
	for _, id := range ids {
		a, err := d.LoadAnalysis(id)
		if err != nil {
			slog.Warn("gammonnet comparison: loading stored analysis failed, position left out", "position_id", id, "error", err)
			continue
		}
		if a == nil || gammonnet.IsOurAnalysis(a) {
			continue
		}
		foreign = append(foreign, id)
	}
	return foreign, nil
}

// CountPositionsWithForeignAnalysis is how many positions a comparison sweep
// would look at — the number a "compare against my imported analyses (N)"
// button needs before committing to the run.
func (d *Database) CountPositionsWithForeignAnalysis() (int, error) {
	ids, err := d.positionIDsWithForeignAnalysis()
	if err != nil {
		return 0, err
	}
	return len(ids), nil
}

// CompareWithGammonNet runs gammonNet over every position carrying an analysis
// somebody else wrote, and reports where the two disagree and what the
// disagreement costs on the stored analysis's own scale (#270).
//
// limit caps how many positions are looked at (0 = all): a comparison is a
// sample question — "is my library's imported analysis broadly what this
// engine would say" — and a user should be able to ask it of a thousand
// positions without evaluating sixty thousand.
//
// Same parallelism and cancellation contract as the analysis batch: jobs
// goroutines each own one reused Searcher, and a cancelled run reports what it
// managed to compare. Nothing is left half-written because nothing is written.
func (d *Database) CompareWithGammonNet(ctx context.Context, ply, pruneK, candidates, jobs, limit int, onProgress func(done, total int)) (gammonnet.AnalysisComparison, error) {
	ids, err := d.positionIDsWithForeignAnalysis()
	if err != nil {
		return gammonnet.AnalysisComparison{}, err
	}
	if limit > 0 && limit < len(ids) {
		ids = ids[:limit]
	}
	total := len(ids)
	if total == 0 {
		return gammonnet.Aggregate(nil), ctx.Err()
	}

	loaded, err := d.LoadPositionsByIDs(ids)
	if err != nil {
		return gammonnet.AnalysisComparison{}, err
	}
	positionsByID := make(map[int64]*Position, len(loaded))
	for i := range loaded {
		positionsByID[loaded[i].ID] = &loaded[i]
	}
	storedByID := make(map[int64]*PositionAnalysis, len(ids))
	for _, id := range ids {
		if a, err := d.LoadAnalysis(id); err == nil && a != nil {
			storedByID[id] = a
		}
	}

	if jobs <= 0 {
		jobs = runtime.NumCPU()
	}
	if jobs > total {
		jobs = total
	}

	var next atomic.Int64
	results := make(chan gammonnet.ComparisonSample, jobs)
	var wg sync.WaitGroup
	wg.Add(jobs)
	for w := 0; w < jobs; w++ {
		go func() {
			defer wg.Done()
			searcher, err := gammonnet.NewBatchSearcher(ply, pruneK)
			if err != nil {
				slog.Warn("gammonnet comparison: building the shared searcher failed; this worker falls back to one per position", "error", err)
				searcher = nil
			}
			for {
				if ctx.Err() != nil {
					return
				}
				i := next.Add(1) - 1
				if i >= int64(total) {
					return
				}
				id := ids[i]
				results <- gammonnet.CompareOne(positionsByID[id], storedByID[id], id, searcher, ply, pruneK, candidates)
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
		if onProgress != nil {
			onProgress(len(samples), total)
		}
	}
	return gammonnet.Aggregate(samples), ctx.Err()
}
