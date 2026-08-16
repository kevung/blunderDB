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

// The six cells of the matrix, as returned by ClassifyCubeDirection. They are
// also the values a SelectionSpec carries to drill from a cell down to the
// positions behind it — so a cell displayed and a cell clicked cannot drift
// apart ("ce qu'on clique = ce qu'on voit").
const (
	CubeCellNone            = ""
	CubeCellOfferRight      = "offer_right"
	CubeCellOfferMissed     = "offer_missed"
	CubeCellOfferPremature  = "offer_premature"
	CubeCellAnswerRight     = "answer_right"
	CubeCellAnswerWrongPass = "answer_wrong_pass"
	CubeCellAnswerWrongTake = "answer_wrong_take"
)

// ClassifyCubeDirection places one (ruling, action) pair in its cell, or
// returns CubeCellNone when either label cannot be read.
//
// An unreadable label is DROPPED, never guessed: counted as "no double" — by
// far the most common action — it would quietly inflate the busiest cell of the
// matrix, and nothing downstream would look wrong.
func ClassifyCubeDirection(best, played string) string {
	verdict, ok := engine.BestCubeVerdict(best)
	if !ok {
		return CubeCellNone
	}
	switch engine.CanonicalCubeAction(played) {
	case engine.CubeNoDouble:
		if verdict.ShouldDouble {
			return CubeCellOfferMissed
		}
		return CubeCellOfferRight
	case engine.CubeDouble:
		if verdict.ShouldDouble {
			return CubeCellOfferRight
		}
		return CubeCellOfferPremature
	case engine.CubePass:
		if verdict.ShouldPass {
			return CubeCellAnswerRight
		}
		return CubeCellAnswerWrongPass
	case engine.CubeTake:
		if verdict.ShouldPass {
			return CubeCellAnswerWrongTake
		}
		return CubeCellAnswerRight
	}
	return CubeCellNone
}

// TallyCubeDirections sorts raw cells onto the two axes. It is pure and shared
// by every backend on purpose: the SQL differs between SQLite and PostgreSQL,
// the classification must not.
func TallyCubeDirections(rows []CubeDirectionRow) CubeDirections {
	var d CubeDirections
	for _, r := range rows {
		switch ClassifyCubeDirection(r.Best, r.Played) {
		case CubeCellOfferRight:
			d.Offer.Right += r.Count
		case CubeCellOfferMissed:
			d.Offer.Missed += r.Count
			d.Offer.MissedMP += r.ErrorMP
		case CubeCellOfferPremature:
			d.Offer.Premature += r.Count
			d.Offer.PrematureMP += r.ErrorMP
		case CubeCellAnswerRight:
			d.Answer.Right += r.Count
		case CubeCellAnswerWrongPass:
			d.Answer.WrongPass += r.Count
			d.Answer.WrongPassMP += r.ErrorMP
		case CubeCellAnswerWrongTake:
			d.Answer.WrongTake += r.Count
			d.Answer.WrongTakeMP += r.ErrorMP
		}
	}
	return d
}
