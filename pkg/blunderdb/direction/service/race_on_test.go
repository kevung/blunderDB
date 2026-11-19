//go:build race

package service_test

// raceEnabled lets a timing guard stand aside under the race detector, which slows the
// engine several times over: a draw the guard holds to a quarter of a second takes more.
const raceEnabled = true
