package database

import (
	"context"
	"strings"

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
func (d *Database) AnalyzeStaleGammonNet(ctx context.Context, ply, pruneK, candidates int, yield func(), onProgress func(done, total int)) error {
	ids, err := d.positionIDsWithStaleGammonNet()
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
