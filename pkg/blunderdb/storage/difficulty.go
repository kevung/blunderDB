package storage

import "math"

// The reference player of ADR-0075: the parameters are fixed there, before any
// result was read, and change only with a new ADR.
const (
	// DifficultyTemperature is τ, in normalised equity: the reference player
	// picks option i with a probability proportional to exp(−Δᵢ/τ).
	DifficultyTemperature = 0.025
	// AvoidableShare marks an error avoidable when the difficulty is at most
	// this share of the loss.
	AvoidableShare = 0.1
	// DifficultyRatioFloor is the total difficulty (MWC) below which a match's
	// loss-to-difficulty ratio measures only noise and is not given.
	DifficultyRatioFloor = 0.005
)

// ReferenceExpectedLoss is the loss a reference player of temperature tau
// expects on a decision whose options cost costs (any unit, ≥ 0 once shifted
// so the best costs 0): Σ π(i)·Δᵢ with π(i) ∝ exp(−Δᵢ/τ). A single option, or
// none, costs nothing: a forced decision has no difficulty.
func ReferenceExpectedLoss(costs []float64, tau float64) float64 {
	if len(costs) < 2 || !(tau > 0) {
		return 0
	}
	best := math.Inf(1)
	for _, c := range costs {
		best = math.Min(best, c)
	}
	var weights, expected float64
	for _, c := range costs {
		delta := c - best
		w := math.Exp(-delta / tau)
		weights += w
		expected += w * delta
	}
	return expected / weights
}

// IsAvoidable says whether a decision that cost loss, at least the library's
// error threshold, is one the reference player would rarely make: its
// expected loss there is at most AvoidableShare of it.
func IsAvoidable(loss, difficulty float64, isError bool) bool {
	return isError && loss > 0 && difficulty <= AvoidableShare*loss
}

// DifficultySummary is one player's match-level reading of ADR-0075, over the
// decisions that carry both a loss and a difficulty. Losses are MWC fractions.
type DifficultySummary struct {
	Decisions  int     `json:"decisions"`
	Loss       float64 `json:"loss"`
	Difficulty float64 `json:"difficulty"`
	// Excess is Σ(ℓ − d): the winning chances lost beyond what the reference
	// player would have lost in the same positions.
	Excess float64 `json:"excess"`
	// Ratio is Σℓ/Σd, 1 for play as good as the reference; nil when Σd is
	// under DifficultyRatioFloor.
	Ratio     *float64 `json:"ratio"`
	Avoidable int      `json:"avoidable"`
}

// SummariseDifficulty adds a match's decisions up per player (index 0 is
// player 1), as the Match panel and `match --format summary` show them.
func SummariseDifficulty(losses []DecisionLoss) [2]DifficultySummary {
	var out [2]DifficultySummary
	for _, d := range losses {
		if d.MWCLoss == nil || d.Difficulty == nil || d.Player < 0 || d.Player > 1 {
			continue
		}
		s := &out[d.Player]
		s.Decisions++
		s.Loss += *d.MWCLoss
		s.Difficulty += *d.Difficulty
		if d.Avoidable {
			s.Avoidable++
		}
	}
	for i := range out {
		s := &out[i]
		s.Excess = s.Loss - s.Difficulty
		if s.Difficulty >= DifficultyRatioFloor {
			r := s.Loss / s.Difficulty
			s.Ratio = &r
		}
	}
	return out
}
