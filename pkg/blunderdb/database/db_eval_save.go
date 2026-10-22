package database

import (
	"context"
	"errors"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/mets"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// DisplayedEvaluationOutcome is what SaveDisplayedEvaluation did with the
// verdict, for the caller to say.
type DisplayedEvaluationOutcome string

const (
	// EvaluationWritten: the verdict now stands on the position.
	EvaluationWritten DisplayedEvaluationOutcome = "written"
	// EvaluationKept: the position already carried an evaluation, kept whole.
	EvaluationKept DisplayedEvaluationOutcome = "kept"
	// EvaluationMETChanged: the library's match equity table changed since
	// the verdict was computed; it would be labelled with a table that did
	// not value it (ADR-0068), so nothing is written.
	EvaluationMETChanged DisplayedEvaluationOutcome = "metChanged"
)

// SaveDisplayedEvaluation writes a gammonNet verdict the Eval panel computed
// on the board (the moves when dice are set, the cube otherwise) onto a
// stored position, unless that position already carries an evaluation.
// metID is the table the verdict was valued with (0: the built-in one).
//
// The verdict is gammonnet.EvaluatePositionWithMET's own output, the one the
// batch writes, so it is already on the storage scale (ADR-0019); it is
// written by the batch's rule (SupersedeEntries) with its table id.
//
// An existing evaluation — any engine, any depth, a rollout — is kept
// whole: a 0-ply verdict on screen must never replace an imported XG
// analysis or a deeper gammonNet one. A row holding only an XGID (a scratch
// board saved before) counts as unevaluated and receives the verdict.
func (d *Database) SaveDisplayedEvaluation(positionID int64, verdict PositionAnalysis, metID int64) (DisplayedEvaluationOutcome, error) {
	hasMoves := verdict.CheckerAnalysis != nil && len(verdict.CheckerAnalysis.Moves) > 0
	if !hasMoves && verdict.DoublingCubeAnalysis == nil {
		return "", errors.New("save displayed evaluation: the verdict holds neither moves nor a cube analysis")
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	ctx := context.Background()

	stored, err := d.store.Analyses().Load(ctx, "", positionID)
	switch {
	case err == nil:
		if analysisHasEvaluation(stored) {
			return EvaluationKept, nil
		}
	case !errors.Is(err, storage.ErrNotFound):
		return "", err
	}

	current, _, err := mets.Current(ctx, d.store, "")
	if err != nil {
		return "", err
	}
	if current != metID {
		return EvaluationMETChanged, nil
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
		return "", err
	}
	return EvaluationWritten, nil
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
