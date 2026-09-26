package database

import (
	"context"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

// The desktop/CLI half of the comparison sweep; what a comparison IS lives in
// engine/gammonnet/compare.go so every mode counts alike. It writes nothing:
// ADR-0013 protects an imported analysis unconditionally.

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
// disagreement costs on the stored analysis's own scale.
//
// limit caps how many positions are looked at (0 = all): a comparison is a
// sample question. Same parallelism and cancellation contract as the analysis
// batch; a cancelled run reports what it compared.
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
