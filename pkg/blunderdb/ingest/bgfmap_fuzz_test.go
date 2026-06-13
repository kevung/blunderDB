package ingest

import "testing"

// FuzzBGFApplyCheckerMove exercises the BGF checker-move board update with
// arbitrary from/to values. These come from parsed `.bgf` files (untrusted
// input) and are mapped to 0-based board indices via `24-from` / `from-1`
// arithmetic, then used to index a fixed [28]int board — so an out-of-range
// from/to in a malformed file must be skipped, never panic.
func FuzzBGFApplyCheckerMove(f *testing.F) {
	// Seeds: a couple of legal moves plus the bar/off edge values.
	f.Add(13, 7, 1)
	f.Add(24, 18, -1)
	f.Add(25, 0, 1)  // from bar, bear off
	f.Add(25, 0, -1) // from bar, bear off
	f.Add(99, -3, 1) // malformed (the historical panic)

	f.Fuzz(func(t *testing.T, from, to, player int) {
		// player is only meaningfully -1 or 1; map any other value onto those
		// so the fuzzer spends its budget on the index arithmetic, not noise.
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
		// Contract: must not panic whatever from/to the parser produced.
		bgfApplyCheckerMove(&board, moveData, player)
	})
}
