package storage

import (
	"math"
	"sort"
)

// The signed biases of ADR-0079: which way a player errs, not only how much.
// Each decision carries +1 (too much: a wrong take, a premature double, a
// bolder play than the best), −1 (too little) or 0, and the bias is their mean
// with a 95 % interval. The thresholds are fixed there.
const (
	// BiasMinDecisions is the fewest decisions a bias needs before a
	// direction is stated.
	BiasMinDecisions = 20

	// The verdicts of a bias.
	BiasTooMuch      = "too_much"
	BiasTooLittle    = "too_little"
	BiasBalanced     = "balanced"
	BiasInsufficient = "insufficient"
)

// SignedBias is one bias: the decisions behind it, the two directions of
// error with what they cost, and B = (Plus − Minus)/Decisions with its 95 %
// interval.
type SignedBias struct {
	Decisions int     `json:"Decisions"`
	Plus      int     `json:"Plus"`
	PlusMP    int64   `json:"PlusMP"`
	Minus     int     `json:"Minus"`
	MinusMP   int64   `json:"MinusMP"`
	Bias      float64 `json:"Bias"`
	Low       float64 `json:"Low"`
	High      float64 `json:"High"`
	Verdict   string  `json:"Verdict"`
}

// ScoreBias is the doubling bias of one score cell: the away scores of the
// player on roll and of the opponent, (0, 0) for money play.
type ScoreBias struct {
	MoverAway    int `json:"MoverAway"`
	OpponentAway int `json:"OpponentAway"`
	SignedBias
}

// DirectionalBiases are the three signed biases of one filter.
type DirectionalBiases struct {
	MinDecisions int `json:"MinDecisions"`
	// TakePass: Plus the wrong takes, Minus the wrong passes — the player's
	// take rate minus the bot's on the same positions.
	TakePass SignedBias `json:"TakePass"`
	// Doubles: Plus the premature doubles (including doubling a position too
	// good), Minus the missed ones; overall and by score.
	Doubles        SignedBias  `json:"Doubles"`
	DoublesByScore []ScoreBias `json:"DoublesByScore"`
	// Blots: Plus a play leaving more blots than the best, Minus fewer, over
	// the checker decisions with contact. BlotsUnread counts the plays the
	// move generator could not reproduce, left out.
	Blots       SignedBias `json:"Blots"`
	BlotsUnread int        `json:"BlotsUnread"`
}

// BiasCubeRow is one counted cube decision as a backend reads it: the bot's
// ruling and the action played, both as the importer spelt them, the away
// scores of the normalised position and the decision's cost.
type BiasCubeRow struct {
	Best, Played string
	MoverAway    int
	OpponentAway int
	ErrorMP      int64
}

// BiasCheckerRow is one counted checker decision with contact: its direction
// against the best play (+1, −1, 0) and its cost, or Unread when the play could
// not be compared.
type BiasCheckerRow struct {
	Sign    int
	ErrorMP int64
	Unread  bool
}

// BuildDirectionalBiases tallies the three biases (ADR-0079). It is pure and
// shared by every backend.
func BuildDirectionalBiases(cube []BiasCubeRow, checker []BiasCheckerRow) *DirectionalBiases {
	out := &DirectionalBiases{MinDecisions: BiasMinDecisions, DoublesByScore: []ScoreBias{}}
	type cell struct{ mover, opp int }
	byScore := map[cell]*SignedBias{}
	for _, r := range cube {
		switch cellOf := ClassifyCubeDirection(r.Best, r.Played); cellOf {
		case CubeCellAnswerRight:
			out.TakePass.add(0, 0)
		case CubeCellAnswerWrongTake:
			out.TakePass.add(1, r.ErrorMP)
		case CubeCellAnswerWrongPass:
			out.TakePass.add(-1, r.ErrorMP)
		case CubeCellOfferRight, CubeCellOfferPremature, CubeCellOfferMissed:
			sign := 0
			switch cellOf {
			case CubeCellOfferPremature:
				sign = 1
			case CubeCellOfferMissed:
				sign = -1
			}
			out.Doubles.add(sign, r.ErrorMP)
			c := cell{r.MoverAway, r.OpponentAway}
			if byScore[c] == nil {
				byScore[c] = &SignedBias{}
			}
			byScore[c].add(sign, r.ErrorMP)
		}
	}
	for _, r := range checker {
		if r.Unread {
			out.BlotsUnread++
			continue
		}
		out.Blots.add(r.Sign, r.ErrorMP)
	}
	out.TakePass.measure()
	out.Doubles.measure()
	out.Blots.measure()
	for c, b := range byScore {
		b.measure()
		out.DoublesByScore = append(out.DoublesByScore, ScoreBias{MoverAway: c.mover, OpponentAway: c.opp, SignedBias: *b})
	}
	sort.Slice(out.DoublesByScore, func(i, j int) bool {
		a, b := out.DoublesByScore[i], out.DoublesByScore[j]
		if a.MoverAway != b.MoverAway {
			return a.MoverAway < b.MoverAway
		}
		return a.OpponentAway < b.OpponentAway
	})
	return out
}

func (b *SignedBias) add(sign int, errMP int64) {
	b.Decisions++
	switch {
	case sign > 0:
		b.Plus++
		b.PlusMP += errMP
	case sign < 0:
		b.Minus++
		b.MinusMP += errMP
	}
}

// measure computes the mean of the signs, its normal 95 % interval and the
// verdict (ADR-0079 rule 8).
func (b *SignedBias) measure() {
	if b.Decisions == 0 {
		b.Verdict = BiasInsufficient
		return
	}
	n := float64(b.Decisions)
	b.Bias = float64(b.Plus-b.Minus) / n
	variance := float64(b.Plus+b.Minus)/n - b.Bias*b.Bias
	half := StudyPlanZ * math.Sqrt(math.Max(variance, 0)/n)
	b.Low, b.High = b.Bias-half, b.Bias+half
	switch {
	case b.Decisions < BiasMinDecisions:
		b.Verdict = BiasInsufficient
	case b.Low > 0:
		b.Verdict = BiasTooMuch
	case b.High < 0:
		b.Verdict = BiasTooLittle
	default:
		b.Verdict = BiasBalanced
	}
}
