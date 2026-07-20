package storage

import "testing"

func acc(sumErr int64, cnt int, matches ...int64) *TournamentPlayerAcc {
	m := make(map[int64]struct{}, len(matches))
	for _, id := range matches {
		m[id] = struct{}{}
	}
	return &TournamentPlayerAcc{SumErr: sumErr, Cnt: cnt, Matches: m}
}

func TestPickReferencePlayer(t *testing.T) {
	t.Run("empty map yields zero badge", func(t *testing.T) {
		b := PickReferencePlayer(nil)
		if b.RefPlayer != "" || b.PR != 0 {
			t.Fatalf("got %+v, want zero badge", b)
		}
	})

	t.Run("most matches wins over lower per-decision error", func(t *testing.T) {
		// A is in 2 matches (cleaner), B in 1 (dirtier) — A is the reference.
		b := PickReferencePlayer(map[string]*TournamentPlayerAcc{
			"A": acc(100, 50, 1, 2),
			"B": acc(400, 20, 3),
		})
		if b.RefPlayer != "A" {
			t.Fatalf("RefPlayer = %q, want A", b.RefPlayer)
		}
		// PR = 500 * 100 / 1000 / 50 = 1.0
		if b.PR != 1.0 {
			t.Fatalf("PR = %v, want 1.0", b.PR)
		}
	})

	t.Run("equal match count breaks on more decisions", func(t *testing.T) {
		b := PickReferencePlayer(map[string]*TournamentPlayerAcc{
			"A": acc(120, 50, 1),
			"B": acc(80, 30, 2),
		})
		if b.RefPlayer != "A" {
			t.Fatalf("RefPlayer = %q, want A (50 > 30 decisions)", b.RefPlayer)
		}
	})

	t.Run("full tie breaks on name, empty name loses", func(t *testing.T) {
		b := PickReferencePlayer(map[string]*TournamentPlayerAcc{
			"":      acc(120, 50, 1),
			"Bravo": acc(120, 50, 2),
			"Alpha": acc(120, 50, 3),
		})
		if b.RefPlayer != "Alpha" {
			t.Fatalf("RefPlayer = %q, want Alpha (named, lexicographically first)", b.RefPlayer)
		}
	})
}
