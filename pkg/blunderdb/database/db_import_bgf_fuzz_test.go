package database

import "testing"

// FuzzBGFApplyCheckerMove exercises the legacy (GUI/CLI) BGF checker-move board
// update with arbitrary from/to values. Like its ingest twin, these come from
// parsed `.bgf` files (untrusted) and are mapped to board indices via
// `24-from` / `from-1` then used to index a fixed [28]int — an out-of-range
// from/to in a malformed file must be skipped, never panic.
func FuzzBGFApplyCheckerMove(f *testing.F) {
	f.Add(13, 7, 1)
	f.Add(24, 18, -1)
	f.Add(25, 0, 1)
	f.Add(25, 0, -1)
	f.Add(99, -3, 1) // malformed: historical index-out-of-range panic

	f.Fuzz(func(t *testing.T, from, to, player int) {
		if player >= 0 {
			player = 1
		} else {
			player = -1
		}
		board := [28]int{2, 0, 0, 0, 0, -5, 0, -3, 0, 0, 0, 5, -5, 0, 0, 0, 3, 0, 5, 0, 0, 0, 0, -2, 0, 0, 0, 0}
		moveData := map[string]interface{}{
			"from": []interface{}{float64(from)},
			"to":   []interface{}{float64(to)},
		}
		bgfApplyCheckerMove(&board, moveData, player)
	})
}
