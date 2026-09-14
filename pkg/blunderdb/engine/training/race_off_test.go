//go:build !race

package training

// raceEnabled lets a timing assertion stand aside under the race detector,
// which slows a network evaluation by an order of magnitude.
const raceEnabled = false
