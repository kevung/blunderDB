package storage

import (
	"math"
	"sort"
	"strconv"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
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
	// Positions, PlusPositions and MinusPositions count the distinct
	// positions behind Decisions, Plus and Minus: the lists a "bias"
	// selection loads (BiasPositionIDs).
	Positions      int `json:"Positions"`
	PlusPositions  int `json:"PlusPositions"`
	MinusPositions int `json:"MinusPositions"`
}

// ScoreBias is the doubling bias of one score cell: the away scores of the
// player on roll and of the opponent, post-Crawford read as one away
// (domain.PointsAway), or Money with both at 0.
type ScoreBias struct {
	Money        bool `json:"Money"`
	MoverAway    int  `json:"MoverAway"`
	OpponentAway int  `json:"OpponentAway"`
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
// scores of the normalised position as stored (the money and post-Crawford
// sentinels included) and the cost of the action played in that match.
type BiasCubeRow struct {
	PositionID   int64
	Best, Played string
	MoverAway    int
	OpponentAway int
	ErrorMP      int64
}

// BiasCheckerRow is one counted checker decision with contact: its direction
// against the best play (+1, −1, 0) and its cost, or Unread when the play could
// not be compared.
type BiasCheckerRow struct {
	PositionID int64
	Sign       int
	ErrorMP    int64
	Unread     bool
}

// The axes of the biases, as a "bias" selection names them.
const (
	BiasAxisTakePass = "take_pass"
	BiasAxisDoubles  = "doubles"
	BiasAxisBlots    = "blots"
)

// The sides of a bias a "bias" selection names; the empty side is every
// decision of the bias.
const (
	BiasSidePlus  = "plus"
	BiasSideMinus = "minus"
)

// BiasScoreKey names a score cell of the doubling bias: "money", or
// "<mover away>-<opponent away>".
func BiasScoreKey(b ScoreBias) string {
	if b.Money {
		return "money"
	}
	return strconv.Itoa(b.MoverAway) + "-" + strconv.Itoa(b.OpponentAway)
}

// biasRef names one bias: an axis, and for the doubling bias by score, the
// score cell's key.
type biasRef struct{ axis, score string }

// walkBiases hands each decision on to every bias it counts in, with its sign
// and cost. It is the one reading of the rows, shared by the tally and by the
// selection, so a list holds exactly what its count counts.
// A score cell's decision also carries the cell itself, so the tally never
// reads a cell back from its key.
func walkBiases(cube []BiasCubeRow, checker []BiasCheckerRow, visit func(b biasRef, cell ScoreBias, sign int, errMP, positionID int64)) {
	for _, r := range cube {
		switch cellOf := ClassifyCubeDirection(r.Best, r.Played); cellOf {
		case CubeCellAnswerRight:
			visit(biasRef{axis: BiasAxisTakePass}, ScoreBias{}, 0, 0, r.PositionID)
		case CubeCellAnswerWrongTake:
			visit(biasRef{axis: BiasAxisTakePass}, ScoreBias{}, 1, r.ErrorMP, r.PositionID)
		case CubeCellAnswerWrongPass:
			visit(biasRef{axis: BiasAxisTakePass}, ScoreBias{}, -1, r.ErrorMP, r.PositionID)
		case CubeCellOfferRight, CubeCellOfferPremature, CubeCellOfferMissed:
			sign := 0
			switch cellOf {
			case CubeCellOfferPremature:
				sign = 1
			case CubeCellOfferMissed:
				sign = -1
			}
			visit(biasRef{axis: BiasAxisDoubles}, ScoreBias{}, sign, r.ErrorMP, r.PositionID)
			cell := ScoreBias{MoverAway: domain.PointsAway(r.MoverAway), OpponentAway: domain.PointsAway(r.OpponentAway)}
			if r.MoverAway < 0 && r.OpponentAway < 0 {
				cell = ScoreBias{Money: true}
			}
			visit(biasRef{axis: BiasAxisDoubles, score: BiasScoreKey(cell)}, cell, sign, r.ErrorMP, r.PositionID)
		}
	}
	for _, r := range checker {
		if r.Unread {
			continue
		}
		visit(biasRef{axis: BiasAxisBlots}, ScoreBias{}, r.Sign, r.ErrorMP, r.PositionID)
	}
}

// BuildDirectionalBiases tallies the three biases (ADR-0079). It is pure and
// shared by every backend.
func BuildDirectionalBiases(cube []BiasCubeRow, checker []BiasCheckerRow) *DirectionalBiases {
	out := &DirectionalBiases{MinDecisions: BiasMinDecisions, DoublesByScore: []ScoreBias{}}
	byScore := map[string]*ScoreBias{}
	type positions struct{ all, plus, minus map[int64]bool }
	seen := map[*SignedBias]*positions{}
	biasOf := func(b biasRef) *SignedBias {
		switch {
		case b.axis == BiasAxisTakePass:
			return &out.TakePass
		case b.axis == BiasAxisBlots:
			return &out.Blots
		case b.score == "":
			return &out.Doubles
		}
		return &byScore[b.score].SignedBias
	}
	walkBiases(cube, checker, func(b biasRef, cell ScoreBias, sign int, errMP, positionID int64) {
		if b.score != "" && byScore[b.score] == nil {
			byScore[b.score] = &cell
		}
		sb := biasOf(b)
		sb.add(sign, errMP)
		p := seen[sb]
		if p == nil {
			p = &positions{all: map[int64]bool{}, plus: map[int64]bool{}, minus: map[int64]bool{}}
			seen[sb] = p
		}
		p.all[positionID] = true
		switch {
		case sign > 0:
			p.plus[positionID] = true
		case sign < 0:
			p.minus[positionID] = true
		}
	})
	for _, r := range checker {
		if r.Unread {
			out.BlotsUnread++
		}
	}
	for sb, p := range seen {
		sb.Positions, sb.PlusPositions, sb.MinusPositions = len(p.all), len(p.plus), len(p.minus)
	}
	out.TakePass.measure()
	out.Doubles.measure()
	out.Blots.measure()
	for _, b := range byScore {
		b.measure()
		out.DoublesByScore = append(out.DoublesByScore, *b)
	}
	sort.Slice(out.DoublesByScore, func(i, j int) bool {
		a, b := out.DoublesByScore[i], out.DoublesByScore[j]
		if a.Money != b.Money {
			return a.Money
		}
		if a.MoverAway != b.MoverAway {
			return a.MoverAway < b.MoverAway
		}
		return a.OpponentAway < b.OpponentAway
	})
	return out
}

// BiasPositionIDs lists the distinct positions behind one count of the
// biases: axis is a BiasAxis*, score a BiasScoreKey for a cell of the doubling
// bias by score (empty otherwise), side a BiasSide* or empty for every
// decision. Its length is the matching Positions, PlusPositions or
// MinusPositions of BuildDirectionalBiases over the same rows.
func BiasPositionIDs(cube []BiasCubeRow, checker []BiasCheckerRow, axis, score, side string) []int64 {
	want := biasRef{axis: axis, score: score}
	seen := map[int64]bool{}
	ids := []int64{}
	walkBiases(cube, checker, func(b biasRef, _ ScoreBias, sign int, _, positionID int64) {
		if b != want || seen[positionID] {
			return
		}
		switch side {
		case BiasSidePlus:
			if sign <= 0 {
				return
			}
		case BiasSideMinus:
			if sign >= 0 {
				return
			}
		}
		seen[positionID] = true
		ids = append(ids, positionID)
	})
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// BlotsUnreadShare is the share of the checker plays with contact left out of
// the blots bias because they could not be replayed: the selection the
// measure carries (ADR-0079 rule 11).
func BlotsUnreadShare(b *DirectionalBiases) float64 {
	total := b.BlotsUnread + b.Blots.Decisions
	if total == 0 {
		return 0
	}
	return float64(b.BlotsUnread) / float64(total)
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
