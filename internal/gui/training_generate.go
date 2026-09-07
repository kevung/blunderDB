package gui

import (
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
)

// The Training question generator's desktop face (issue #321, ADR-0041).
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
