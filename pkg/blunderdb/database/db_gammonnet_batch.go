package database

import (
	"context"

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
// that has none, one at a time (never in parallel across positions — a
// single 2-ply k=12 search already uses several cores on its own). It is
// cancellable through ctx, checked before each position (never mid-search:
// gammonnet.Searcher has no internal cancellation checkpoint, the same known
// limit gammonnet_eval.go documents for the live panel). Before starting each
// position, yield is called and must block for as long as the caller wants
// to give priority to something else — internal/gui's wrapper uses it to let
// an interactive live evaluation go first ("le lot cède en une position").
// onProgress is called after every position, successful or not (a single
// unevaluable position must not stall the whole library).
func (d *Database) AnalyzeMissingWithGammonNet(ctx context.Context, ply, pruneK, candidates int, yield func(), onProgress func(done, total int)) error {
	ids, err := d.positionIDsWithoutAnalysis()
	if err != nil {
		return err
	}
	total := len(ids)

	for i, id := range ids {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if yield != nil {
			yield()
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err := d.analyzeOnePositionWithGammonNet(id, ply, pruneK, candidates); err != nil {
			// A single position that fails to convert or evaluate (e.g. an
			// edge case the engine declines) must not stop the batch: log via
			// the same "written as produced" contract, just skip it — it
			// stays without analysis and is picked up again, unchanged, the
			// next time this runs.
			continue
		}

		if onProgress != nil {
			onProgress(i+1, total)
		}
	}
	return nil
}

// analyzeOnePositionWithGammonNet loads, evaluates and writes one position —
// the unit of work the resume/idempotence guarantee is built on.
func (d *Database) analyzeOnePositionWithGammonNet(id int64, ply, pruneK, candidates int) error {
	pos, err := d.LoadPosition(int(id))
	if err != nil {
		return err
	}

	result, err := gammonnet.EvaluatePosition(*pos, ply, pruneK, candidates)
	if err != nil {
		return err
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
		// A dance (no legal move) or an unevaluable cube state (e.g. beyond
		// the MET's range): nothing to write, not an error — the position
		// stays without analysis and is retried on the next run exactly like
		// any other still-missing position.
		return nil
	}

	return d.SaveAnalysis(id, analysis)
}
