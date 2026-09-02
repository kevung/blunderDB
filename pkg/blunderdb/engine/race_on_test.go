//go:build race

package engine

// raceEnabled lets a timing assertion stand aside under the race detector,
// which slows byte-level work by an order of magnitude.
const raceEnabled = true
