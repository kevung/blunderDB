package domain

import "math"

// Interval is a 95 % interval around a figure (ADR-0078). Low and High mean
// nothing unless Available, which needs two independent units to resample.
type Interval struct {
	Available bool    `json:"available"`
	Low       float64 `json:"low"`
	High      float64 `json:"high"`
	// Units is the number of independent units behind the figure: games for
	// one match, matches for an aggregate.
	Units int `json:"units"`
}

// RatioPool estimates a ratio Σnum/Σden — a PR is errors over decisions —
// and its interval from independent units (games of a match, matches of a
// selection), each bringing its own numerator and denominator. The variance
// is the bootstrap's for the ratio estimator, n/(n−1)·Σ(num − R·den)²/(Σden)²,
// computed from sums kept as units arrive rather than drawn: no seed, one
// pass, the same figure on every read and from every path that sums the same
// units.
type RatioPool struct {
	sumN, sumD, sumNN, sumND, sumDD float64
	n                               int
}

// Add pools one unit. A unit with no denominator (no decision) is not a unit.
func (p *RatioPool) Add(num, den float64) {
	if den <= 0 {
		return
	}
	p.sumN += num
	p.sumD += den
	p.sumNN += num * num
	p.sumND += num * den
	p.sumDD += den * den
	p.n++
}

// Interval is the 95 % interval of scale·Σnum/Σden, its low bound floored at
// 0 (every ratio pooled here is a loss).
func (p *RatioPool) Interval(scale float64) Interval {
	if p.n < 2 {
		return Interval{Units: p.n}
	}
	r := p.sumN / p.sumD
	ss := p.sumNN - 2*r*p.sumND + r*r*p.sumDD
	n := float64(p.n)
	se := math.Sqrt(math.Max(ss, 0)*n/(n-1)) / p.sumD
	return Interval{
		Available: true,
		Low:       scale * math.Max(r-mwc7Z*se, 0),
		High:      scale * (r + mwc7Z*se),
		Units:     p.n,
	}
}
