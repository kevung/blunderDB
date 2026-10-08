package storage

import (
	"cmp"
	"slices"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// ReviewCount is how many decisions a MatchReview proposes to revisit per
// player (ADR-0078).
const ReviewCount = 3

// MatchReview is a match's study summary (ADR-0078), per player (index 0 is
// player 1): what the uncertainty of the figures is, which decisions are
// worth revisiting, whether the dice or the play decided the result, and
// whether the errors came from haste or from a gap in knowledge.
type MatchReview struct {
	MatchID int64           `json:"match_id"`
	Players [2]PlayerReview `json:"players"`
}

// PlayerReview is one player's part of a MatchReview.
type PlayerReview struct {
	// PR and its interval resample the games of the match.
	PR         float64         `json:"pr"`
	PRInterval domain.Interval `json:"pr_interval"`
	Decisions  int             `json:"decisions"`
	// MWC7 is the badge's, its interval over the games.
	MWC7 domain.MWC7 `json:"mwc7"`
	// MWCLoss is the total the player gave away, a fraction.
	MWCLoss float64 `json:"mwc_loss"`
	// ToReview holds at most ReviewCount errors, the largest avoidable part
	// of the loss first.
	ToReview []ReviewDecision `json:"to_review"`
	Pace     ErrorPace        `json:"pace"`
	Luck     LuckAdjusted     `json:"luck"`
}

// ReviewDecision is an error proposed for review. AvoidableLoss is MWCLoss −
// Difficulty (ADR-0076), the part of the loss a reference player would have
// kept, an unknown difficulty counting 0.
type ReviewDecision struct {
	MoveID        int64    `json:"move_id"`
	GameNumber    int      `json:"game_number"`
	MoveNumber    int      `json:"move_number"`
	DecisionType  string   `json:"decision_type"`
	MWCLoss       float64  `json:"mwc_loss"`
	Difficulty    *float64 `json:"difficulty"`
	Avoidable     bool     `json:"avoidable"`
	AvoidableLoss float64  `json:"avoidable_loss"`
}

// ErrorPace splits a player's errors by the time taken over them: hasty
// below the median duration of the player's priced decisions of the same type
// in the match, deliberate at or above it, Unknown when the time was not
// recorded. Losses are MWC fractions.
type ErrorPace struct {
	Hasty          int     `json:"hasty"`
	Deliberate     int     `json:"deliberate"`
	Unknown        int     `json:"unknown"`
	HastyLoss      float64 `json:"hasty_loss"`
	DeliberateLoss float64 `json:"deliberate_loss"`
}

// LuckAdjusted is a finished match's result with the dice taken out, in MWC
// fractions from the player's side: Result is the outcome (1 or 0) less the
// winning chances at the start, Luck the player's rolls' luck less the
// opponent's, Adjusted = Result − Luck. ErrorBalance, the opponent's MWC loss
// less the player's, is what Adjusted estimates under perfect valuation. A
// money session, an unfinished match or one whose rolls carry no luck is not
// Available; RollsMeasured of Rolls says how complete the luck is.
type LuckAdjusted struct {
	Available     bool    `json:"available"`
	Result        float64 `json:"result"`
	Luck          float64 `json:"luck"`
	Adjusted      float64 `json:"adjusted"`
	ErrorBalance  float64 `json:"error_balance"`
	RollsMeasured int     `json:"rolls_measured"`
	Rolls         int     `json:"rolls"`
}

// BuildMatchReview assembles a MatchReview from the match's decisions, as
// MatchDecisionLosses lists them, its length (0 for money), the winning
// chances of player 1 at its start, and its outcome (MatchOutcome: +1 for
// player 1, -1 for player 2, 0 unfinished).
func BuildMatchReview(matchID int64, decisions []DecisionLoss, matchLength int, startMWC float64, outcome int) MatchReview {
	r := MatchReview{MatchID: matchID}
	var games []int
	seen := map[int]bool{}
	type sums struct{ errMP, n int64 }
	var errs [2]map[int]*sums
	var losses [2]map[int]float64
	var priced [2]bool
	var luck [2]float64
	measured, rolls := 0, 0
	for seat := range 2 {
		errs[seat], losses[seat] = map[int]*sums{}, map[int]float64{}
	}
	for _, d := range decisions {
		if d.DecisionType == "checker" {
			rolls++
			if d.Luck != nil {
				measured++
				luck[d.Player] += *d.Luck
			}
		}
		if d.ErrorMP == nil {
			continue
		}
		if !seen[d.GameNumber] {
			seen[d.GameNumber] = true
			games = append(games, d.GameNumber)
		}
		s := errs[d.Player][d.GameNumber]
		if s == nil {
			s = &sums{}
			errs[d.Player][d.GameNumber] = s
		}
		s.errMP += *d.ErrorMP
		s.n++
		if d.MWCLoss != nil {
			losses[d.Player][d.GameNumber] += *d.MWCLoss
			r.Players[d.Player].MWCLoss += *d.MWCLoss
			priced[d.Player] = true
		}
	}
	slices.Sort(games)
	for seat := range 2 {
		p := &r.Players[seat]
		var pool domain.RatioPool
		var sumErr, n int64
		perGame := make([]float64, len(games))
		for i, g := range games {
			if s := errs[seat][g]; s != nil {
				pool.Add(float64(s.errMP), float64(s.n))
				sumErr += s.errMP
				n += s.n
			}
			perGame[i] = losses[seat][g]
		}
		p.Decisions = int(n)
		if n > 0 {
			p.PR = 500 * float64(sumErr) / 1000 / float64(n)
		}
		p.PRInterval = pool.Interval(500.0 / 1000.0)
		if priced[seat] {
			p.MWC7 = domain.MatchMWC7(p.MWCLoss, matchLength, perGame)
		}
		p.ToReview = toReview(decisions, seat)
		p.Pace = errorPace(decisions, seat)
	}
	for seat := range 2 {
		p, o := &r.Players[seat], &r.Players[1-seat]
		p.Luck = LuckAdjusted{RollsMeasured: measured, Rolls: rolls}
		if matchLength <= 0 || outcome == 0 || measured == 0 {
			continue
		}
		won, start := 0.0, startMWC
		if (outcome > 0) == (seat == 0) {
			won = 1
		}
		if seat == 1 {
			start = 1 - startMWC
		}
		p.Luck.Available = true
		p.Luck.Result = won - start
		p.Luck.Luck = luck[seat] - luck[1-seat]
		p.Luck.Adjusted = p.Luck.Result - p.Luck.Luck
		p.Luck.ErrorBalance = o.MWCLoss - p.MWCLoss
	}
	return r
}

// toReview ranks seat's priced errors by their avoidable loss (ADR-0078).
func toReview(decisions []DecisionLoss, seat int) []ReviewDecision {
	type ranked struct {
		ReviewDecision
		order int
	}
	var out []ranked
	for i, d := range decisions {
		if d.Player != seat || !d.Error || d.MWCLoss == nil {
			continue
		}
		avoidable := *d.MWCLoss
		if d.Difficulty != nil {
			avoidable -= *d.Difficulty
		}
		if avoidable <= 0 {
			continue
		}
		out = append(out, ranked{ReviewDecision{
			MoveID: d.MoveID, GameNumber: d.GameNumber, MoveNumber: d.MoveNumber, DecisionType: d.DecisionType,
			MWCLoss: *d.MWCLoss, Difficulty: d.Difficulty, Avoidable: d.Avoidable, AvoidableLoss: avoidable,
		}, i})
	}
	slices.SortFunc(out, func(a, b ranked) int {
		return cmp.Or(cmp.Compare(b.AvoidableLoss, a.AvoidableLoss), cmp.Compare(b.MWCLoss, a.MWCLoss), cmp.Compare(a.order, b.order))
	})
	review := make([]ReviewDecision, 0, ReviewCount)
	for _, r := range out[:min(len(out), ReviewCount)] {
		review = append(review, r.ReviewDecision)
	}
	return review
}

// errorPace splits seat's priced errors at the median duration of its priced
// decisions of the same type (ADR-0078).
func errorPace(decisions []DecisionLoss, seat int) ErrorPace {
	durations := map[string][]int64{}
	for _, d := range decisions {
		if d.Player == seat && d.MWCLoss != nil && d.DurationMS != nil {
			durations[d.DecisionType] = append(durations[d.DecisionType], *d.DurationMS)
		}
	}
	median := map[string]float64{}
	for t, ds := range durations {
		slices.Sort(ds)
		m := len(ds) / 2
		if len(ds)%2 == 1 {
			median[t] = float64(ds[m])
		} else {
			median[t] = float64(ds[m-1]+ds[m]) / 2
		}
	}
	var p ErrorPace
	for _, d := range decisions {
		if d.Player != seat || !d.Error || d.MWCLoss == nil {
			continue
		}
		switch {
		case d.DurationMS == nil:
			p.Unknown++
		case float64(*d.DurationMS) < median[d.DecisionType]:
			p.Hasty++
			p.HastyLoss += *d.MWCLoss
		default:
			p.Deliberate++
			p.DeliberateLoss += *d.MWCLoss
		}
	}
	return p
}
