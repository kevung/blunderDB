package bearoffgen

import (
	"runtime"
	"time"
)

// What a domain costs, so the interface can say it before the user commits to
// it rather than after.
//
// The two-sided sweep is O(n³): n² pairs, each looking at the successors of
// one of n positions. Measured on this repository's reference machine (Ryzen 7
// PRO 6850U, 16 threads, 2026-09-05), serially:
//
//	TS-06-06   n =   924     0.96 s    a = 1.22e-9
//	TS-06-07   n = 1 716     3.43 s    a = 6.8e-10
//	TS-06-08   n = 3 003    17.27 s    a = 6.4e-10
//	TS-06-09   n = 5 005    78.94 s    a = 6.3e-10
//
// a = t / n³ settles from TS-06-07 on; TS-06-06 is dominated by fixed costs
// and is not representative. So the model is t = a·n³ / (workers · efficiency),
// with `a` measured on the machine when a run has finished there and the
// reference constant until then.
const (
	// ReferenceRate is `a` on the reference machine: seconds per n³ per core.
	ReferenceRate = 6.5e-10

	// oneSidedRate is the one-sided sweep's cost per n², seconds. Fitted on
	// OS-08 (n = 490 314, 77 s), which the neighbouring domains bracket.
	oneSidedRate = 3.2e-10

	// ParallelEfficiency is what a core is worth beyond the first. The sweep
	// is memory-bound at these sizes: 16 threads measured 8.1× on TS-06-09
	// (78.9 s → 9.8 s) and 6.7× on TS-06-08.
	ParallelEfficiency = 0.6
)

// EstimateDuration is how long generating this domain should take on
// `workers` cores, given a measured rate in seconds per n³ per core. Pass 0
// for the rate to use the reference machine's.
//
// It is an estimate and reads like one in the interface: the measured ETA
// takes over as soon as the run reports its first progress.
func (d Domain) EstimateDuration(rate float64, workers int) time.Duration {
	if d.Kind == OneSidedKind {
		// The one-sided sweep is strictly sequential — position i reads only
		// positions j < i — so cores buy it nothing, and its cost is its own
		// curve. Measured on the reference machine: OS-06 5.5 s, OS-07 17 s,
		// OS-08 77 s, OS-09 ~6 min, OS-10 ~28 min, which is n² within a few
		// percent (n = C(p+15, p)).
		//
		// A measured rate still applies, as a ratio: what a two-sided run
		// learned is how fast THIS MACHINE is, and that is not a property of
		// which sweep measured it.
		speed := 1.0
		if rate > 0 {
			speed = rate / ReferenceRate
		}
		n := float64(NumPositions(d.Points, osCheckers))
		return time.Duration(speed * oneSidedRate * n * n * float64(time.Second))
	}
	if rate <= 0 {
		rate = ReferenceRate
	}
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	n := float64(NumPositions(d.Points, d.Checkers))
	speedup := 1.0
	if workers > 1 {
		speedup = 1 + float64(workers-1)*ParallelEfficiency
	}
	return time.Duration(rate * n * n * n / speedup * float64(time.Second))
}

// MeasuredRate turns one finished run into the rate to use for the next
// estimate. Returns 0 for a domain too small to be representative — TS-06-06
// runs in a second, most of which is not the sweep.
func (d Domain) MeasuredRate(elapsed time.Duration, workers int) float64 {
	if d.Kind != TwoSidedKind || d.Checkers < 7 || elapsed <= 0 {
		return 0
	}
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	speedup := 1.0
	if workers > 1 {
		speedup = 1 + float64(workers-1)*ParallelEfficiency
	}
	n := float64(NumPositions(d.Points, d.Checkers))
	return elapsed.Seconds() * speedup / (n * n * n)
}

// Candidates lists the two-sided domains the interface offers, widest last.
// Six points is the board the cube verdict is defined over (ADR-0009); the
// chequer count is what the user chooses, and what decides whether the table
// is 6 MB or 22 GB.
func Candidates() []Domain {
	out := make([]Domain, 0, 10)
	for c := 6; c <= 15; c++ {
		out = append(out, Domain{Kind: TwoSidedKind, Points: 6, Checkers: c})
	}
	return out
}

// OneSidedCandidates lists the one-sided domains the interface offers: the
// width of the board the EPC can answer for (ADR-0027 §9). It stops at ten
// points, where the file is already 117 MB and half an hour of one core, and
// where a "race" with a chequer on the 11-point has stopped being one.
func OneSidedCandidates() []Domain {
	out := make([]Domain, 0, 5)
	for p := 6; p <= 10; p++ {
		out = append(out, Domain{Kind: OneSidedKind, Points: p, Checkers: osCheckers})
	}
	return out
}
