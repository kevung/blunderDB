//go:build race

package transcript

// raceEnabled lets a timing guard stand aside under the race detector, which
// slows the move generator several times over.
const raceEnabled = true
