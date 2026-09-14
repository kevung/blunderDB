package gui

import (
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/training"
)

// The Training question generators' desktop face (issues #321 and #322,
// ADR-0041).
//
// One method, one call per question, and nothing kept between two: the
// generator is a pure function of a seed and a dice stream, so there is no
// session state to hold here the way gnCubeMatrix holds a sweep.
//
// It sits on *App rather than on the Database wrapper for the reason
// ComputeCubeMatrix does: a pure function of the ENGINE over one position,
// with no storage behind it (whyEnginePure in internal/cli/parity_test.go).
// The library source draws its candidate through the calls the frontend
// already has — LoadPosition — and hands the position in, so nothing here
// needs a database either.

// GenerateBearoffQuestion makes one Bearoff question: a seed plus k plies
// played out by the engine (ADR-0041 rule 1).
//
// It never returns an error. A seed the exercise cannot use is not a failure
// of the call, it is an answer: the question comes back with Generated false
// and a Refusal code the tab turns into the sentence that names the domain
// (rule 3). An error would have made the frontend show « something went
// wrong » where the user needs to be told what a bear-off is.
func (a *App) GenerateBearoffQuestion(request race.BearoffRequest) race.BearoffQuestion {
	return race.GenerateBearoff(request)
}

// GenerateEvaluationQuestion makes one Evaluation question (#322): a seed plus
// k plies chosen by gammonNet, with its truth — the win chance, the cube
// verdict and the button it makes right, and the EPC when the position has an
// exact one — computed in the same call, so the clock never contains a second
// round trip.
//
// Like its Bearoff sibling it never returns an error: a seed the exercise
// cannot use, or an engine that could not be built, comes back as a Refusal
// the tab names.
//
// The judge is SERIAL. A question is prepared in the background while the
// user answers the previous one; measured, sixteen workers cost what one does
// on these positions (training/cost_test.go), and a search that took every
// core would stack on a batch analysis for nothing.
func (a *App) GenerateEvaluationQuestion(request training.EvaluationRequest) training.EvaluationQuestion {
	a.trainingOnce.Do(func() {
		a.trainingGen, a.trainingErr = training.NewGenerator(1)
	})
	if a.trainingErr != nil {
		return training.EvaluationQuestion{Source: request.Source, Refusal: training.RefusalNotEvaluable}
	}
	return a.trainingGen.Evaluation(request)
}
