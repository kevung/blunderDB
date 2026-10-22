package database

import (
	"context"
	"errors"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/mets"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// SaveDisplayedEvaluation writes a gammonNet verdict the Eval panel computed
// on the board (the moves when dice are set, the cube otherwise) onto a
// stored position, unless that position already carries an evaluation.
//
// The verdict is gammonnet.EvaluatePositionWithMET's own output, the one the
// batch writes, so it is already on the storage scale (ADR-0019); it is
// valued with the library's current table and written by the same rule as
// the batch (SupersedeEntries, with its table id).
//
// An existing evaluation — any engine, any depth, a rollout — is kept
// whole: a 0-ply verdict on screen must never replace an imported XG
// analysis or a deeper gammonNet one. The caller learns which happened from
// the returned bool, and says so. A row holding only an XGID (a scratch
// board saved before) counts as unevaluated and receives the verdict.
func (d *Database) SaveDisplayedEvaluation(positionID int64, verdict PositionAnalysis) (bool, error) {
	hasMoves := verdict.CheckerAnalysis != nil && len(verdict.CheckerAnalysis.Moves) > 0
	if !hasMoves && verdict.DoublingCubeAnalysis == nil {
		return false, errors.New("save displayed evaluation: the verdict holds neither moves nor a cube analysis")
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	ctx := context.Background()

	stored, err := d.store.Analyses().Load(ctx, "", positionID)
	switch {
	case err == nil:
		if analysisHasEvaluation(stored) {
			return false, nil
		}
	case !errors.Is(err, storage.ErrNotFound):
		return false, err
	}

	metID, _, err := mets.Current(ctx, d.store, "")
	if err != nil {
		return false, err
	}

	out := PositionAnalysis{AnalysisEngineVersion: gammonnet.EngineVersion}
	if hasMoves {
		out.AnalysisType = "CheckerMove"
		out.CheckerAnalysis = &CheckerAnalysis{Moves: verdict.CheckerAnalysis.Moves}
	} else {
		out.AnalysisType = "DoublingCube"
		cube := *verdict.DoublingCubeAnalysis
		out.DoublingCubeAnalysis = &cube
	}
	if err := d.saveAnalysisLocked(positionID, out, &metID); err != nil {
		return false, err
	}
	return true, nil
}

// analysisHasEvaluation reports whether a stored analysis says something
// about the position beyond its identity. The scratch board writes an empty
// cube block beside its XGID; a block without depth nor verdict is that one.
func analysisHasEvaluation(a *PositionAnalysis) bool {
	if a == nil {
		return false
	}
	if a.CheckerAnalysis != nil && len(a.CheckerAnalysis.Moves) > 0 {
		return true
	}
	if len(a.AllCubeAnalyses) > 0 || len(a.Rollouts) > 0 {
		return true
	}
	c := a.DoublingCubeAnalysis
	return c != nil && (c.AnalysisDepth != "" || c.BestCubeAction != "")
}
