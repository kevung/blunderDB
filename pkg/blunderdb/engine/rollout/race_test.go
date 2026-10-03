//go:build race

package rollout

// raceEnabled skips the rollouts that play hundreds of full games: under the
// race detector they cost minutes and check nothing about concurrency that
// TestReproducibleAcrossWorkers does not.
const raceEnabled = true
