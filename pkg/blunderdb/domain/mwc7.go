package domain

import "math"

// MWC7 is the 7-point MWC loss (ADR-0075): the match winning chances a player
// gave away over a match, all decisions together (checker, cube, take/pass,
// cubeful), rescaled to a 7-point match. Under the FIBS random-walk model the
// loss of a player of a given strength grows as √N, so a loss L over N points
// reads L7 = L·√(7/N); for a 7-point match it is the MWC loss itself.
//
// It counts no decisions: every error weighs what it cost, so it complements
// the Performance Rating, whose denominator depends on what counts as a
// decision. Its Elo reading is secondary: q = 0.5 − L7 is the win probability
// left against a perfect opponent over seven points, and inverting the FIBS
// rating formula gives D = (2000/√7)·log10(q/(1−q)) ≤ 0, which ranks exactly as
// L7 does. A money game has no match length and no MWC7: Available is false
// and the reader shows that, not a number.
type MWC7 struct {
	Available bool `json:"available"`
	// Loss is L7, a fraction of a match (0.123 = 12.3 %).
	Loss float64 `json:"loss"`
	// Low and High bound L7's 95 % interval; they mean nothing unless
	// HasInterval, which needs two independent units to resample (two games
	// for one match, two matches for a pool).
	HasInterval bool    `json:"has_interval"`
	Low         float64 `json:"low"`
	High        float64 `json:"high"`
	// Elo is D, the Elo difference against the engine; EloLow and EloHigh
	// map High and Low through the same formula.
	Elo     float64 `json:"elo"`
	EloLow  float64 `json:"elo_low"`
	EloHigh float64 `json:"elo_high"`
	// EloFloored says L7 reached the floor of the Elo formula (a win
	// probability under MWC7MinWin): Elo is then an upper bound.
	EloFloored bool `json:"elo_floored"`
	// Matches is the number of (match, player) units behind the value.
	Matches int `json:"matches"`
}

const (
	// MWC7Length is the reference length: the most common tournament length,
	// and the one at which L7 is the MWC loss eXtreme Gammon displays.
	MWC7Length = 7
	// fibsScale is the FIBS rating constant: P(win) = 1/(1+10^(−D·√N/2000)).
	fibsScale = 2000.0
	// MWC7MinWin floors q in the Elo reading: at or past a loss of half a
	// match the inversion has no finite answer, and real players do lose that
	// much, since errors made after a swing of luck compound. Below a 1 %
	// chance against the engine the model says nothing more precise.
	MWC7MinWin = 0.01
	// mwc7Z is the two-sided 95 % normal quantile.
	mwc7Z = 1.96
	// mwc7MaxLength is the longest match the match equity table values; a
	// longer length is the importers' money sentinel.
	mwc7MaxLength = 64
)

// MWC7Defined reports whether a match of this length has an MWC7: a match
// length the match equity table can value.
func MWC7Defined(matchLength int) bool {
	return matchLength >= 1 && matchLength <= mwc7MaxLength
}

// mwc7Scale is √(7/N), the factor from a loss over N points to L7.
func mwc7Scale(matchLength int) float64 {
	return math.Sqrt(MWC7Length / float64(matchLength))
}

// MWC7Elo is D for a 7-point loss L7, and whether the floor was reached. A
// negative loss reads as none.
func MWC7Elo(loss7 float64) (float64, bool) {
	q := 0.5 - math.Max(loss7, 0)
	floored := false
	if q < MWC7MinWin {
		q, floored = MWC7MinWin, true
	}
	return fibsScale / math.Sqrt(MWC7Length) * math.Log10(q/(1-q)), floored
}

// newMWC7 completes an MWC7 from L7 and, when hasInterval, its bounds.
func newMWC7(loss7, low, high float64, hasInterval bool, matches int) MWC7 {
	e := MWC7{Available: true, Loss: loss7, Matches: matches}
	e.Elo, e.EloFloored = MWC7Elo(loss7)
	if hasInterval {
		e.HasInterval = true
		e.Low, e.High = math.Max(low, 0), high
		e.EloHigh, _ = MWC7Elo(e.Low)
		e.EloLow, _ = MWC7Elo(e.High)
	}
	return e
}

// MatchMWC7 is one player's MWC7 over one match: loss is the player's total
// MWC loss, gameLosses the same loss split by game. The interval resamples the
// games, the largest independent unit of a match: its variance is the
// bootstrap's, G/(G−1)·Σ(l_g − l̄)² for the sum of G games, computed rather
// than drawn so the figure is deterministic and costs one pass.
func MatchMWC7(loss float64, matchLength int, gameLosses []float64) MWC7 {
	if !MWC7Defined(matchLength) {
		return MWC7{}
	}
	k := mwc7Scale(matchLength)
	g := len(gameLosses)
	if g < 2 {
		return newMWC7(k*loss, 0, 0, false, 1)
	}
	mean := 0.0
	for _, l := range gameLosses {
		mean += l
	}
	mean /= float64(g)
	ss := 0.0
	for _, l := range gameLosses {
		ss += (l - mean) * (l - mean)
	}
	half := mwc7Z * math.Sqrt(float64(g)/float64(g-1)*ss)
	return newMWC7(k*loss, k*(loss-half), k*(loss+half), true, 1)
}

// MWC7Pool aggregates (match, player) units — a player's stats, a tournament,
// a progression window — into one MWC7. It pools the losses and the √N before
// taking one ratio, L7 = √7·ΣL/Σ√N: additive, never a mean of ratios, so a
// seven-point match weighs more than a one-pointer, and a pool of one unit is
// that unit unchanged. The interval resamples the units: the ratio
// estimator's cluster variance n/(n−1)·Σ(L − R·√N)²/(Σ√N)², from sums kept as
// units arrive.
type MWC7Pool struct {
	sumW, sumL, sumLL, sumLW, sumWW float64
	n                               int
	single                          MWC7
}

// Add pools one unit as MatchMWC7 returned it for a match of matchLength
// points. An unavailable unit (money) is ignored.
func (p *MWC7Pool) Add(e MWC7, matchLength int) {
	if !e.Available || !MWC7Defined(matchLength) {
		return
	}
	w := math.Sqrt(float64(matchLength))
	l := e.Loss / mwc7Scale(matchLength)
	p.sumW += w
	p.sumL += l
	p.sumLL += l * l
	p.sumLW += l * w
	p.sumWW += w * w
	p.n++
	p.single = e
}

// AddLoss pools one unit from its loss alone (no per-game split, so the unit
// brings no interval of its own).
func (p *MWC7Pool) AddLoss(loss float64, matchLength int) {
	p.Add(MatchMWC7(loss, matchLength, nil), matchLength)
}

// Result is the pooled MWC7.
func (p *MWC7Pool) Result() MWC7 {
	switch p.n {
	case 0:
		return MWC7{}
	case 1:
		return p.single
	}
	r := p.sumL / p.sumW
	ss := p.sumLL - 2*r*p.sumLW + r*r*p.sumWW
	n := float64(p.n)
	se := math.Sqrt(math.Max(ss, 0)*n/(n-1)) / p.sumW
	k := math.Sqrt(MWC7Length)
	return newMWC7(k*r, k*(r-mwc7Z*se), k*(r+mwc7Z*se), true, p.n)
}
