package gui

import (
	"sort"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Les seuils de latence de la saisie d'une transcription (ux.md §7). Le rejeu
// est mesuré dans transcript ; ici, l'autre moitié du tour : les deux appels
// du panneau (`computeCandidates` : domain.LegalMoves puis
// EvaluatePositionImmediate), sur le chemin interactif série, pas celui du lot.
// Les seuils tiennent compte d'un facteur trois à quatre sous charge.

// contactBoard is a mid-game contact position, the expensive case: 892
// distinct plays over the 21 rolls.
func contactBoard() domain.Board {
	var b domain.Board
	for i := range b.Points {
		b.Points[i] = domain.Point{Color: domain.None}
	}
	put := func(idx, n, color int) { b.Points[idx] = domain.Point{Checkers: n, Color: color} }
	// Black (player 1) moves 24 → 1.
	put(24, 2, domain.Black)
	put(13, 4, domain.Black)
	put(11, 2, domain.Black)
	put(8, 3, domain.Black)
	put(6, 3, domain.Black)
	put(4, 1, domain.Black)
	// White (player 2) mirrors it.
	put(1, 2, domain.White)
	put(12, 4, domain.White)
	put(14, 2, domain.White)
	put(17, 3, domain.White)
	put(19, 3, domain.White)
	put(21, 1, domain.White)
	return b
}

// contactPosition is that board as a money position with nobody's cube.
func contactPosition() domain.Position {
	return domain.Position{
		Board:        contactBoard(),
		PlayerOnRoll: domain.Black,
		Score:        [2]int{-1, -1},
		Cube:         domain.Cube{Owner: domain.None, Value: 0},
	}
}

// allRolls is the 21 distinct rolls (ux.md §4.1).
func allRolls() [][2]int {
	out := make([][2]int, 0, 21)
	for high := 1; high <= 6; high++ {
		for low := 1; low <= high; low++ {
			out = append(out, [2]int{high, low})
		}
	}
	return out
}

// rankRoll is `computeCandidates` in Go; it returns how many plays were
// ranked.
func rankRoll(t *testing.T, app *App, pos domain.Position, roll [2]int) int {
	t.Helper()
	p := pos
	p.Dice = roll
	plays := domain.LegalMoves(&p)
	res, err := app.evaluateGammonNet(p, 0, 0, 0)
	if err != nil {
		t.Fatalf("roll %v: %v", roll, err)
	}
	sort.SliceStable(res.Moves, func(i, j int) bool { return res.Moves[i].Equity > res.Moves[j].Equity })
	if len(plays) == 0 {
		t.Fatalf("roll %v has no legal play on a contact board", roll)
	}
	return len(plays)
}

// TestTranscriptionRollLatencyPerKeystroke guards the roll's plays being on
// screen before the next keystroke. Ceiling: ux.md §7's 30 ms, about thirty
// times the cost, the margin a shared CI runner needs.
func TestTranscriptionRollLatencyPerKeystroke(t *testing.T) {
	const ceiling = 30 * time.Millisecond

	pos := contactPosition()
	app := &App{}
	// Warm-up: a running application has already paid for the scratch.
	rankRoll(t, app, pos, [2]int{6, 5})

	best := time.Duration(1<<63 - 1)
	plays := 0
	for i := 0; i < 5; i++ {
		start := time.Now()
		plays = rankRoll(t, app, pos, [2]int{3, 1})
		if e := time.Since(start); e < best {
			best = e
		}
	}
	t.Logf("one roll, %d plays ranked at 0-ply: %v", plays, best)
	if best > ceiling {
		t.Errorf("ranking one roll took %v, over the %v guard of ux.md §7", best, ceiling)
	}
}

// TestTranscriptionBoardPlayLatency guards a play made at the board with no
// dice typed: `loadBoardPlay` asks domain.LegalMoves for all 21 rolls, with no
// ranking. Ceiling: twice ux.md §7's 30 ms, as ~5 ms of work reached 19 ms
// under load.
func TestTranscriptionBoardPlayLatency(t *testing.T) {
	const ceiling = 60 * time.Millisecond

	pos := contactPosition()
	rolls := allRolls()

	best := time.Duration(1<<63 - 1)
	plays := 0
	for i := 0; i < 5; i++ {
		start := time.Now()
		plays = 0
		for _, roll := range rolls {
			p := pos
			p.Dice = roll
			plays += len(domain.LegalMoves(&p))
		}
		if e := time.Since(start); e < best {
			best = e
		}
	}
	t.Logf("the 21 rolls of a board play, %d plays generated: %v", plays, best)
	if best > ceiling {
		t.Errorf("deducing the roll of a board play took %v, over the %v guard of ux.md §7", best, ceiling)
	}
}

// TestTranscriptionRankedSweepCost prices ranking all 21 rolls (ux.md §4.1's
// open question). A report, not a budget: nothing ranks 21 rolls, so the guard
// sits at 500 ms against ~39 ms (177 ms seen under load).
func TestTranscriptionRankedSweepCost(t *testing.T) {
	if testing.Short() {
		t.Skip("a measurement of a gesture that does not exist; -short skips it")
	}
	const guard = 500 * time.Millisecond

	pos := contactPosition()
	app := &App{}
	rolls := allRolls()
	rankRoll(t, app, pos, rolls[0])

	best := time.Duration(1<<63 - 1)
	plays := 0
	for i := 0; i < 3; i++ {
		start := time.Now()
		plays = 0
		for _, roll := range rolls {
			plays += rankRoll(t, app, pos, roll)
		}
		if e := time.Since(start); e < best {
			best = e
		}
	}
	t.Logf("the 21 rolls, each ranked at 0-ply, %d plays: %v (%v per roll)", plays, best, best/21)
	if best > guard {
		t.Errorf("ranking the 21 rolls took %v, over the %v guard — an order of magnitude worse than the 39 ms measured on 2026-09-07", best, guard)
	}
}
