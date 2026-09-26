package gui

import (
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/training"
)

// The Training question generators' desktop face (ADR-0041): one stateless
// call per question. On *App, not Database, because it is engine-pure
// (whyEnginePure in internal/cli/parity_test.go); library seeds are passed in.

// GenerateBearoffQuestion makes one Bearoff question: a seed plus k plies
// played out by the engine (ADR-0041 rule 1).
//
// It never returns an error: an unusable seed comes back with Generated false
// and a Refusal code the tab names (rule 3).
func (a *App) GenerateBearoffQuestion(request race.BearoffRequest) race.BearoffQuestion {
	return race.GenerateBearoff(request)
}

// GenerateEvaluationQuestion makes one Evaluation question (a seed plus k
// gammonNet plies) with its truth in the same call. Failures come back as a
// Refusal, never an error. The judge is serial: more workers gain nothing
// here (training/cost_test.go) and would stack on a batch analysis.
func (a *App) GenerateEvaluationQuestion(request training.EvaluationRequest) training.EvaluationQuestion {
	a.trainingOnce.Do(func() {
		a.trainingGen, a.trainingErr = training.NewGenerator(1)
	})
	if a.trainingErr != nil {
		return training.EvaluationQuestion{Source: request.Source, Refusal: training.RefusalNotEvaluable}
	}
	return a.trainingGen.Evaluation(request)
}
