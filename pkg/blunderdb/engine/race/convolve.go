package race

import (
	"math"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// Win-probability estimation for pure-bearoff positions outside the exact
// two-sided domain. Method (ADR-0009, tasks/ts-bearoff/hypotheses.md):
//
//  1. Convolve the two one-sided roll distributions (embedded OS-06-15): the
//     player on roll wins iff their roll count ≤ the opponent's. Exact under
//     two hypotheses — independence of the two bear-off processes
//     (structural in a race) and one-sided optimal play by both sides.
//  2. Apply a frozen polynomial correction (correction_coeffs.go) absorbing
//     the systematic bias of hypothesis 2 (the trailer plays for variance);
//     calibrated offline against the TS-06-11 oracle with cmd/calibrace.
//
// Measured residual after correction, on the oracle domain (7–11 checkers):
// see the constants in correction_coeffs.go. Beyond 11 checkers per side the
// bound is an extrapolation (monotone trend, no oracle).

// moments summarises a roll distribution for the correction features.
type moments struct {
	mean, std, skew, kurt float64
}

func distMoments(p []float64) moments {
	var m moments
	for n, pn := range p {
		m.mean += float64(n) * pn
	}
	var m2, m3, m4 float64
	for n, pn := range p {
		d := float64(n) - m.mean
		m2 += d * d * pn
		m3 += d * d * d * pn
		m4 += d * d * d * d * pn
	}
	m.std = math.Sqrt(m2)
	if m.std > 1e-9 {
		m.skew = m3 / (m.std * m.std * m.std)
		m.kurt = m4 / (m2 * m2)
	}
	return m
}

// winProbRaw computes P(N_us ≤ N_them) by convolution and returns the two
// distributions' moments for the correction features.
func winProbRaw(us, them [6]int) (p float64, mu, mt moments, err error) {
	du, err := engine.RollDistribution(us)
	if err != nil {
		return 0, mu, mt, err
	}
	dt, err := engine.RollDistribution(them)
	if err != nil {
		return 0, mu, mt, err
	}
	// tail[n] = P(N_them ≥ n).
	tail := make([]float64, len(dt)+1)
	for n := len(dt) - 1; n >= 0; n-- {
		tail[n] = tail[n+1] + dt[n]
	}
	for n, pn := range du {
		if pn > 0 {
			p += pn * tail[n]
		}
	}
	return p, distMoments(du), distMoments(dt), nil
}

// correctionFeatures must match cmd/calibrace exactly: 8 powers of p, then
// {f, f·p, f·p²} for each of the 8 moment features.
func correctionFeatures(p float64, mu, mt moments) []float64 {
	x := make([]float64, 0, nCorrectionCoeffs)
	pk := 1.0
	for i := 0; i < 8; i++ {
		x = append(x, pk)
		pk *= p
	}
	for _, f := range []float64{mu.std, mt.std, mu.mean, mt.mean, mu.skew, mt.skew, mu.kurt, mt.kurt} {
		x = append(x, f, f*p, f*p*p)
	}
	return x
}

// RawWinProbFeatures exposes the uncorrected convolution and its feature
// vector. It exists for cmd/calibrace (offline calibration against the
// TS-06-11 oracle) and for the env-gated oracle tests; the panel path is
// EstimatedWinProb.
func RawWinProbFeatures(us, them [6]int) (p float64, features []float64, err error) {
	p, mu, mt, err := winProbRaw(us, them)
	if err != nil {
		return 0, nil, err
	}
	return p, correctionFeatures(p, mu, mt), nil
}

// EstimatedWinProb estimates the on-roll player's win probability for a
// pure-bearoff position outside the exact domain: raw convolution plus the
// calibrated correction, clamped to [0, 1].
func EstimatedWinProb(us, them [6]int) (float64, error) {
	p, mu, mt, err := winProbRaw(us, them)
	if err != nil {
		return 0, err
	}
	corr := 0.0
	for i, x := range correctionFeatures(p, mu, mt) {
		corr += correctionCoeffs[i] * x
	}
	p += corr
	return math.Max(0, math.Min(1, p)), nil
}
