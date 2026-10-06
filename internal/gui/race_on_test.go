//go:build race

package gui

// raceEnabled lets a timing guard stand aside under the race detector, which
// slows the move generator and the evaluator several times over.
const raceEnabled = true
