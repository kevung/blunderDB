package gui

import (
	"sort"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Les seuils de latence de la saisie d'une transcription (T3.5, ux.md §7).
//
// # Pourquoi ce fichier est ici et pas dans pkg/blunderdb/transcript
//
// Le rejeu du document a déjà ses mesures, dans le paquet qui le calcule
// (replay_test.go : rejeu complet, rejeu incrémental sous 5 ms, égalité des
// deux). Ce qui manquait est l'AUTRE moitié du tour de saisie, celle que le
// paquet transcript ne voit pas : la liste des coups du jet qui vient d'être
// tapé. Elle est faite de deux appels, et ce sont les deux que le panneau fait
// lui-même (TranscriptionPanel.svelte, `computeCandidates`) —
// domain.LegalMoves puis EvaluatePositionImmediate. Ils vivent ici, dans
// internal/gui ; les mesurer ailleurs mesurerait autre chose.
//
// # Ce qui est mesuré, et ce qui ne l'est pas
//
// CLAUDE.md, « le parallélisme est un comportement de production » : le chemin
// mesuré ici est celui de la saisie interactive — un `Searcher` série par
// appel, ce que `gammonNetLivePool.acquire` rend pour un 0-ply — et non le lot
// d'analyse, qui répartit les positions sur NumCPU. Ce n'est pas un banc
// d'essai du lot ; un chiffre pris ici ne dit rien de lui.
//
// # Mesures du 2026-09-07 (dev, AVX2, 16 cœurs logiques, best-of-N)
//
//	un jet : LegalMoves + classement 0-ply de ses coups     0,75 – 1,23 ms
//	les 21 jets, LegalMoves seul (le coup joué au plateau)   4,7 – 6,6 ms
//	les 21 jets, chacun classé au 0-ply                     34,6 – 46,6 ms
//
// Et, sur une machine chargée (12 de charge moyenne, une autre suite en
// cours), les mêmes trois mesures ont donné 2,6 ms, 18,8 ms et 177 ms — un
// facteur trois à quatre, malgré le meilleur de cinq passes. Les seuils
// ci-dessous en tiennent compte : ils cherchent un ordre de grandeur perdu,
// pas un pour cent, et une suite qui rougit une fois sur dix ne garde rien.
//
// Le seuil que la fiche T3.5 proposait — 30 ms pour « les 21 jets triés au
// 0-ply » — est donc DÉMENTI par la mesure, et ux.md §7 porte le chiffre
// mesuré depuis. Il ne l'est que pour un geste qui n'existe pas : rien dans
// l'application ne classe vingt et un jets. Les deux gestes réels tiennent
// tous deux sous leur seuil, avec un facteur trente pour l’un et douze pour
// l’autre, et ce sont eux que ce fichier garde.

// contactBoard is a mid-game CONTACT position — the expensive case for
// domain.LegalMoves and for a 0-ply ranking alike, and the one a transcription
// spends a match in. Both sides have checkers past each other, so no roll is
// forced and most rolls open a dozen distinct plays (892 over the 21 rolls).
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

// allRolls is the 21 distinct rolls — the triangle of ux.md §4.1, and the sweep
// the board-play entry makes to deduce which roll a play was made with.
func allRolls() [][2]int {
	out := make([][2]int, 0, 21)
	for high := 1; high <= 6; high++ {
		for low := 1; low <= high; low++ {
			out = append(out, [2]int{high, low})
		}
	}
	return out
}

// rankRoll is `computeCandidates`, in Go: the legal plays of a roll, the 0-ply
// ranking of them, and the order the panel shows. It returns how many plays
// were ranked.
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

// TestTranscriptionRollLatencyPerKeystroke is the guard on the gesture of every
// turn: the second die is typed, and that roll's plays have to be on screen
// before the next keystroke. ux.md §1 prices the whole turn at 0,56 s of
// typing; whatever the software adds comes out of that budget.
//
// The ceiling is the 30 ms of ux.md §7 — some thirty times the measured cost,
// which is the margin a CI runner sharing its cores needs, and still a tenth of
// one keystroke.
func TestTranscriptionRollLatencyPerKeystroke(t *testing.T) {
	const ceiling = 30 * time.Millisecond

	pos := contactPosition()
	app := &App{}
	// A running application has long since paid for the network's scratch: the
	// first call is not what a keystroke costs.
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

// TestTranscriptionBoardPlayLatency is the guard on the OTHER gesture that
// waits on the engine: a play made at the board with no dice typed (T2.4). The
// panel deduces the roll by asking domain.LegalMoves for all 21 of them
// (`loadBoardPlay`), once per Action and not once per click.
//
// No 0-ply ranking here, on purpose: the sweep only has to find WHICH rolls
// produce the play, and it is the third measurement below that says the ranking
// is what costs — 5 ms for the 21 sweeps, 39 ms once each is also ranked.
//
// The ceiling is TWICE the 30 ms of ux.md §7, and the doubling is measured
// rather than cautious: the same 5 ms of work took 19 ms on a machine under
// load, best-of-five included, and a guard that goes red on a busy CI runner
// guards nothing.
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

// TestTranscriptionRankedSweepCost prices the design question ux.md §4.1 leaves
// open — what a triangle that showed each roll's best play WOULD cost — and it
// is the measurement that denied the fiche's 30 ms. It is a report, not a
// budget: nothing in the application ranks 21 rolls, so its guard sits an
// order of magnitude over the measurement — 500 ms against 39 — because the
// same sweep was seen at 177 ms on a loaded machine, and a report that goes
// red on the neighbours' build is worse than no report.
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
