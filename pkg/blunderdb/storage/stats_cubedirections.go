package storage

import "github.com/kevung/blunderdb/pkg/blunderdb/engine"

// A cube position holds two decisions taken by two different players: the one
// holding the cube chooses to offer it or not, the one receiving it chooses to
// take or pass. Averaging them together — as a single "cube PR" does — says how
// much a player loses on the cube, never in which direction they get it wrong.
//
// These counts answer the second question, and only that: they are counts, not
// a verdict. What a neutral point is, how wide the uncertainty must be before a
// leaning may be named, and what to name it, are decisions that belong to the
// caller (see gammonGo ADR 0058).

// CubeDirectionRow is one raw (best, played) cell as a backend reads it out of
// SQL, before any label is interpreted. Best is the engine's ruling
// (analysis.best_cube_action), Played what the player did (move.cube_action) —
// both in whatever spelling the importer wrote.
type CubeDirectionRow struct {
	Best    string
	Played  string
	Count   int
	ErrorMP int64
}

// CubeOfferCounts tallies the decision of the player holding the cube.
// Missed: the cube should have been offered and was not. Premature: it was
// offered and should not have been.
type CubeOfferCounts struct {
	Right       int   `json:"Right"`
	Missed      int   `json:"Missed"`
	MissedMP    int64 `json:"MissedMP"`
	Premature   int   `json:"Premature"`
	PrematureMP int64 `json:"PrematureMP"`
}

// CubeAnswerCounts tallies the decision of the player receiving the cube.
// WrongPass: a correct take was passed. WrongTake: a correct pass was taken.
type CubeAnswerCounts struct {
	Right       int   `json:"Right"`
	WrongPass   int   `json:"WrongPass"`
	WrongPassMP int64 `json:"WrongPassMP"`
	WrongTake   int   `json:"WrongTake"`
	WrongTakeMP int64 `json:"WrongTakeMP"`
}

// CubeDirections is the per-axis breakdown of cube decisions.
type CubeDirections struct {
	Offer  CubeOfferCounts  `json:"Offer"`
	Answer CubeAnswerCounts `json:"Answer"`
}

// TallyCubeDirections sorts raw cells onto the two axes. It is pure and shared
// by every backend on purpose: the SQL differs between SQLite and PostgreSQL,
// the classification must not.
//
// A cell whose ruling or action cannot be read is DROPPED, never guessed — an
// unrecognised label counted as "no double" would quietly inflate the most
// common cell of all. The caller can spot the loss by comparing these totals
// with the decision count it already has.
func TallyCubeDirections(rows []CubeDirectionRow) CubeDirections {
	var d CubeDirections
	for _, r := range rows {
		verdict, ok := engine.BestCubeVerdict(r.Best)
		if !ok {
			continue
		}
		switch engine.CanonicalCubeAction(r.Played) {
		case engine.CubeNoDouble:
			if verdict.ShouldDouble {
				d.Offer.Missed += r.Count
				d.Offer.MissedMP += r.ErrorMP
			} else {
				d.Offer.Right += r.Count
			}
		case engine.CubeDouble:
			if verdict.ShouldDouble {
				d.Offer.Right += r.Count
			} else {
				d.Offer.Premature += r.Count
				d.Offer.PrematureMP += r.ErrorMP
			}
		case engine.CubePass:
			if verdict.ShouldPass {
				d.Answer.Right += r.Count
			} else {
				d.Answer.WrongPass += r.Count
				d.Answer.WrongPassMP += r.ErrorMP
			}
		case engine.CubeTake:
			if verdict.ShouldPass {
				d.Answer.WrongTake += r.Count
				d.Answer.WrongTakeMP += r.ErrorMP
			} else {
				d.Answer.Right += r.Count
			}
		}
	}
	return d
}
