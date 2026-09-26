package race

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/bearoffgen/bearofftest"
)

// seededRNG gives every test the same dice, so a walk that broke once breaks
// again with the same numbers.
func seededRNG() *rand.Rand { return rand.New(rand.NewPCG(0x5eed, 0xb0a4d)) }

// boardWith lays chequers out the way seedFromBoard reads them: `black` on
// points 1..6, `white` on points 24..19.
func boardWith(black, white sideBoard) domain.Board {
	return boardOf(bearoffSeed{black: black, white: white})
}

// ── The rules of a bear-off play ────────────────────────────────────────────

func TestEveryDieIsPlayableWhileCheckersRemain(t *testing.T) {
	// The invariant legalPlays leans on, and the reason the rule that forbids
	// wasting a die is not implemented there: in a bear-off the highest
	// occupied point always has a play — it runs down under a smaller die,
	// bears off on an equal one, and bears off on a bigger one BECAUSE it is
	// the highest. Widen the domain (a chequer on the bar, one outside the
	// home board) and this is the guard that falls first.
	rng := seededRNG()
	for trial := 0; trial < 2000; trial++ {
		var side sideBoard
		for i := 0; i < 1+rng.IntN(maxDomainCheckers); i++ {
			side[rng.IntN(6)]++
		}
		for die := 1; die <= 6; die++ {
			if len(withDie(side, die)) == 0 {
				t.Fatalf("board %v has no play for a %d", side, die)
			}
		}
	}
	if len(withDie(sideBoard{}, 4)) != 0 {
		t.Error("a side that is out still has a play")
	}
}

func TestEveryDieIsPlayedWhenTheSideSurvivesTheTurn(t *testing.T) {
	// The other half: legalPlays consumes the whole roll. Two chequers on the
	// six point and a 5-3 must end on the three and the one point, never on a
	// board that spent a single die.
	plays := legalPlays(sideBoard{0, 0, 0, 0, 0, 2}, []int{5, 3})
	for _, play := range plays {
		if play.checkers() != 2 {
			t.Fatalf("play %v changed the chequer count", play)
		}
	}
	if want := (sideBoard{1, 0, 1, 0, 0, 0}); len(plays) != 1 || plays[0] != want {
		t.Fatalf("plays = %v, want exactly %v", plays, want)
	}
}

func TestADieLargerThanThePointBearsOffOnlyFromTheHighest(t *testing.T) {
	// Chequers on the ace and the three point, a five. The three point is the
	// highest occupied, so it comes off; the ace point may not, and a
	// generator that let it would invent chequers out of the position.
	plays := withDie(sideBoard{1, 0, 1, 0, 0, 0}, 5)
	if len(plays) != 1 {
		t.Fatalf("got %d plays, want one: %v", len(plays), plays)
	}
	if want := (sideBoard{1, 0, 0, 0, 0, 0}); plays[0] != want {
		t.Errorf("play = %v, want %v", plays[0], want)
	}
}

func TestDoublesPlayFourDice(t *testing.T) {
	// Four chequers on the six point, double 6: all four come off.
	plays := legalPlays(sideBoard{0, 0, 0, 0, 0, 4}, []int{6, 6, 6, 6})
	if len(plays) != 1 || plays[0] != (sideBoard{}) {
		t.Fatalf("plays = %v, want the side borne off with four sixes", plays)
	}
}

func TestBestPlayReadsTheExactTableRatherThanTakingTheFirstMove(t *testing.T) {
	// Two chequers on the ace point and one on the six, a 2-1. Both dice must
	// be played, and the two ways of doing it leave either three chequers
	// (2 on the ace, 1 on the three) or two (1 on the ace, 1 on the four).
	// The pip counts are equal — five each — so nothing short of the table
	// separates them, and the table is unambiguous: fewer chequers bear off in
	// fewer rolls.
	plays := legalPlays(sideBoard{2, 0, 0, 0, 0, 1}, []int{2, 1})
	if len(plays) != 2 {
		t.Fatalf("plays = %v, want two", plays)
	}
	best := bestPlay(plays)
	if want := (sideBoard{1, 0, 0, 1, 0, 0}); best != want {
		t.Errorf("bestPlay = %v, want %v", best, want)
	}
	if meanRolls(sideBoard{1, 0, 0, 1, 0, 0}) >= meanRolls(sideBoard{2, 0, 1, 0, 0, 0}) {
		t.Error("the premise of this test no longer holds: the table no longer prefers the two-chequer board")
	}
}

// ── The domain, and refusing by name ────────────────────────────────────────

func TestAContactSeedIsRefusedByName(t *testing.T) {
	var start domain.Board
	put := func(i, n, color int) { start.Points[i] = domain.Point{Checkers: n, Color: color} }
	put(24, 2, domain.Black)
	put(13, 5, domain.Black)
	put(8, 3, domain.Black)
	put(6, 5, domain.Black)
	put(1, 2, domain.White)
	put(12, 5, domain.White)
	put(17, 3, domain.White)
	put(19, 5, domain.White)

	q := generateBearoff(BearoffRequest{Source: SourceBoard, Seed: &domain.Position{Board: start}}, seededRNG())
	if q.Generated {
		t.Fatal("the opening position started a Bearoff session")
	}
	if q.Refusal != RefusalNotBearoff {
		t.Errorf("refusal = %q, want %q", q.Refusal, RefusalNotBearoff)
	}
}

func TestACheckerOnTheBarIsRefused(t *testing.T) {
	b := boardWith(sideBoard{2, 2, 2, 2, 2, 2}, sideBoard{2, 2, 2, 2, 2, 2})
	b.Points[domain.BlackBar] = domain.Point{Checkers: 1, Color: domain.Black}
	q := generateBearoff(BearoffRequest{Source: SourceBoard, Seed: &domain.Position{Board: b}}, seededRNG())
	if q.Generated || q.Refusal != RefusalNotBearoff {
		t.Errorf("generated=%v refusal=%q, want a notBearoff refusal", q.Generated, q.Refusal)
	}
}

func TestASideBelowFourCheckersIsRefusedByItsOwnName(t *testing.T) {
	// A perfectly legal bear-off, but outside what this exercise trains. The
	// refusal must not say "this is not a bear-off": it is one.
	b := boardWith(sideBoard{1, 1, 1, 0, 0, 0}, sideBoard{2, 2, 2, 2, 2, 2})
	q := generateBearoff(BearoffRequest{Source: SourceLibrary, Seed: &domain.Position{Board: b}}, seededRNG())
	if q.Generated {
		t.Fatal("a three-chequer side started a session")
	}
	if q.Refusal != RefusalTooFewCheckers {
		t.Errorf("refusal = %q, want %q", q.Refusal, RefusalTooFewCheckers)
	}
}

func TestAnEmptyBoardFallsBackToThePoolAndStillSaysWhy(t *testing.T) {
	q := generateBearoff(BearoffRequest{Source: SourceBoard, Seed: &domain.Position{}}, seededRNG())
	if !q.Generated {
		t.Fatal("an empty board produced no question; rule 3 falls back to the pool")
	}
	if q.Source != SourcePool {
		t.Errorf("source = %q, want %q", q.Source, SourcePool)
	}
	if q.Refusal != RefusalEmptyBoard {
		t.Errorf("refusal = %q, want %q — the fallback still names why the board was not used", q.Refusal, RefusalEmptyBoard)
	}
}

func TestAnUnknownSourceIsRefused(t *testing.T) {
	q := generateBearoff(BearoffRequest{Source: "elsewhere"}, seededRNG())
	if q.Generated || q.Refusal != RefusalUnknownSource {
		t.Errorf("generated=%v refusal=%q, want an unknownSource refusal", q.Generated, q.Refusal)
	}
}

func TestWithoutAOneSidedTableTheExerciseSaysSo(t *testing.T) {
	// The exercise asks for the EPC: with no table there is no truth to grade
	// against, and a question nobody can answer is worse than a refusal.
	// ADR-0027 generates the tables in the background, so this is the state of
	// a first launch and not a hypothesis.
	if err := engine.LoadOneSided(""); err != nil {
		t.Fatalf("unloading the table: %v", err)
	}
	t.Cleanup(func() {
		if err := engine.LoadOneSided(bearofftest.OneSidedPath(t)); err != nil {
			t.Fatalf("reloading the table: %v", err)
		}
	})

	q := generateBearoff(BearoffRequest{Source: SourcePool}, seededRNG())
	if q.Generated || q.Refusal != RefusalNoTable {
		t.Errorf("generated=%v refusal=%q, want a noTable refusal", q.Generated, q.Refusal)
	}
}

// ── What comes out ──────────────────────────────────────────────────────────

func TestPoolShapesAreCompleteBearIns(t *testing.T) {
	for i, shape := range pool {
		if n := shape.checkers(); n != maxDomainCheckers {
			t.Errorf("pool[%d] = %v holds %d chequers, want %d", i, shape, n, maxDomainCheckers)
		}
	}
}

func TestEveryGeneratedQuestionIsInTheDomain(t *testing.T) {
	// The three sources, not just the pool: they differ in where the seed
	// comes from and in how many plies are played, so each can leave the
	// domain in its own way.
	seed := &domain.Position{
		Board:        boardWith(sideBoard{3, 2, 3, 2, 3, 2}, sideBoard{2, 3, 2, 3, 2, 3}),
		PlayerOnRoll: domain.White,
	}
	for _, source := range []string{SourcePool, SourceBoard, SourceLibrary} {
		t.Run(source, func(t *testing.T) {
			request := BearoffRequest{Source: source}
			if source != SourcePool {
				request.Seed = seed
			}
			assertQuestionsAreInDomain(t, request)
		})
	}
}

func assertQuestionsAreInDomain(t *testing.T, request BearoffRequest) {
	t.Helper()
	rng := seededRNG()
	for i := 0; i < 500; i++ {
		q := generateBearoff(request, rng)
		if !q.Generated {
			t.Fatalf("draw %d refused: %q", i, q.Refusal)
		}
		if q.Plies > maxPoolPlies {
			t.Fatalf("draw %d played %d plies, over the budget of %d", i, q.Plies, maxPoolPlies)
		}
		seed, refusal := seedFromBoard(&q.Position.Board)
		if refusal != "" {
			t.Fatalf("draw %d produced a position its own exercise refuses (%q): %+v", i, refusal, q.Position.Board)
		}
		for _, side := range []sideBoard{seed.black, seed.white} {
			if !inDomain(side) {
				t.Fatalf("draw %d: side %v is outside %d..%d chequers", i, side, minDomainCheckers, maxDomainCheckers)
			}
		}
		if q.Position.Board.Bearoff[domain.Black]+seed.black.checkers() != maxDomainCheckers ||
			q.Position.Board.Bearoff[domain.White]+seed.white.checkers() != maxDomainCheckers {
			t.Fatalf("draw %d: the rest is not borne off: %+v", i, q.Position.Board.Bearoff)
		}
		if q.Position.Cube != (domain.Cube{Owner: domain.None, Value: 0}) {
			t.Fatalf("draw %d: cube %+v, want centred", i, q.Position.Cube)
		}
		if !q.Position.IsMoney() {
			t.Fatalf("draw %d: score %v is not money play", i, q.Position.Score)
		}
		if q.Position.PlayerOnRoll != domain.Black && q.Position.PlayerOnRoll != domain.White {
			t.Fatalf("draw %d: nobody on roll", i)
		}
		if q.EPC.Bottom.EPC == nil || q.EPC.Top.EPC == nil {
			t.Fatalf("draw %d: a side has no EPC, so the question has no truth", i)
		}
	}
}

func TestTheBoardSourceNeverAsksTheSeedItself(t *testing.T) {
	// Rule 2: from the board, k is drawn in 1..4 and never 0 — the user has
	// just been looking at the seed, so asking for it would ask for nothing.
	seed := &domain.Position{Board: boardWith(sideBoard{3, 2, 3, 2, 3, 2}, sideBoard{2, 3, 2, 3, 2, 3})}
	rng := seededRNG()
	for i := 0; i < 200; i++ {
		q := generateBearoff(BearoffRequest{Source: SourceBoard, Seed: seed}, rng)
		if !q.Generated {
			t.Fatalf("draw %d refused: %q", i, q.Refusal)
		}
		if q.Plies < 1 || q.Plies > maxBoardPlies {
			t.Fatalf("draw %d played %d plies, want 1..%d", i, q.Plies, maxBoardPlies)
		}
		if q.Position.Board == seed.Board {
			t.Fatalf("draw %d handed back the seed the user is already looking at", i)
		}
	}
}

func TestTheLibrarySourceKeepsThePositionAsItIs(t *testing.T) {
	// Rule 2: a library position is already real; simulating from it would add
	// nothing, so k = 0 and the chequers must not move.
	board := boardWith(sideBoard{3, 2, 3, 2, 3, 2}, sideBoard{2, 3, 2, 3, 2, 3})
	q := generateBearoff(BearoffRequest{Source: SourceLibrary, Seed: &domain.Position{Board: board}}, seededRNG())
	if !q.Generated {
		t.Fatalf("refused: %q", q.Refusal)
	}
	if q.Plies != 0 {
		t.Errorf("plies = %d, want 0", q.Plies)
	}
	if q.Position.Board != board {
		t.Errorf("the library position was played on:\n got %+v\nwant %+v", q.Position.Board, board)
	}
}

func TestPastItsDeadlineAQuestionFallsBackToAPoolSeedAtZeroPlies(t *testing.T) {
	// Rule 5: « past a deadline the generator falls back to k = 0 on a pool
	// seed rather than wait ». The clock below jumps an hour at every reading,
	// so the deadline is behind the walk before its first ply — on the two
	// sources that walk at all. The library source plays no ply, so it has
	// nothing to fall back from.
	late := time.Unix(0, 0)
	clock := func() time.Time {
		late = late.Add(time.Hour)
		return late
	}
	seed := &domain.Position{Board: boardWith(sideBoard{3, 2, 3, 2, 3, 2}, sideBoard{2, 3, 2, 3, 2, 3})}
	for _, request := range []BearoffRequest{{Source: SourcePool}, {Source: SourceBoard, Seed: seed}} {
		t.Run(request.Source, func(t *testing.T) {
			rng := seededRNG()
			for i := 0; i < 50; i++ {
				q := generateBearoffWithClock(request, rng, clock)
				if !q.Generated {
					t.Fatalf("draw %d refused (%q): a late walk still owes a question", i, q.Refusal)
				}
				if q.Plies != 0 {
					t.Fatalf("draw %d played %d plies past its deadline, want 0", i, q.Plies)
				}
				if q.Source != SourcePool {
					t.Fatalf("draw %d: source = %q, want %q — the fallback says where the seed came from", i, q.Source, SourcePool)
				}
				laid, _ := seedFromBoard(&q.Position.Board)
				if laid.black.checkers() != maxDomainCheckers || laid.white.checkers() != maxDomainCheckers {
					t.Fatalf("draw %d is not a pool shape at zero plies: %+v", i, q.Position.Board)
				}
			}
		})
	}
}

// ── The cost, measured ──────────────────────────────────────────────────────

// budgetPerQuestion is the stated ceiling of ADR-0041 rule 5. It is generous on
// purpose: it guards against a change of approach (searching instead of
// looking up, rebuilding the table per question), which lands two or three
// decades above it, while a question costs about 50-200 µs even under load.
const budgetPerQuestion = 100 * time.Millisecond

func TestTheCostOfAQuestionStaysUnderItsBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("timing measurement")
	}
	const draws = 500
	rng := seededRNG()

	// The walk is measured on a clock that never moves, so its deadline never
	// falls. Otherwise the fallback of rule 5 would cap every question near
	// questionDeadline and a change of approach — the very thing this budget
	// is here to catch — would come back green, hidden behind pool shapes at
	// zero plies.
	frozen := time.Now()
	stopped := func() time.Time { return frozen }

	// Warm the table's first pages in, so the measurement is of the generator
	// and not of the first lookup of the process.
	generateBearoffWithClock(BearoffRequest{Source: SourcePool}, rng, stopped)

	start := time.Now()
	distinct := make(map[domain.Board]bool, draws)
	for i := 0; i < draws; i++ {
		q := generateBearoffWithClock(BearoffRequest{Source: SourcePool}, rng, stopped)
		if !q.Generated {
			t.Fatalf("draw %d refused: %q", i, q.Refusal)
		}
		distinct[q.Position.Board] = true
	}
	per := time.Since(start) / draws
	t.Logf("%v per question over %d draws (budget %v)", per, draws, budgetPerQuestion)

	// A generator that returned one constant position would be instant and
	// would pass a timing assertion on its own. It does not pass this one.
	if len(distinct) < draws/2 {
		t.Fatalf("only %d distinct positions in %d draws: the generator is not generating", len(distinct), draws)
	}
	if per > budgetPerQuestion {
		t.Errorf("a question costs %v, over the stated budget of %v", per, budgetPerQuestion)
	}
}

// rollers reports how many of `draws` questions put each side on roll.
func rollers(t *testing.T, req BearoffRequest, draws int) map[int]int {
	t.Helper()
	rng := seededRNG()
	seen := map[int]int{}
	for i := 0; i < draws; i++ {
		q := generateBearoff(req, rng)
		if !q.Generated {
			t.Fatalf("draw %d refused: %q", i, q.Refusal)
		}
		seen[q.Position.PlayerOnRoll]++
	}
	return seen
}

func TestTheRollerIsDrawnFromThePoolAndKeptFromASeed(t *testing.T) {
	// « Roller drawn » (rule 4). Checking only "0 or 1" would pass on the zero
	// value; the oracle has to tell a DRAW from a constant.

	pool := rollers(t, BearoffRequest{Source: SourcePool}, 400)
	if pool[domain.Black] == 0 || pool[domain.White] == 0 {
		t.Errorf("pool questions put %v on roll: the roller is not drawn", pool)
	}

	// A seed the user brought keeps its own roller: at k = 0 the library
	// source hands the position back as it is, and turning it over to the
	// other side would be a silent adaptation on a field nobody checks.
	for _, side := range []int{domain.Black, domain.White} {
		seed := &domain.Position{
			Board:        boardWith(sideBoard{3, 2, 3, 2, 3, 2}, sideBoard{2, 3, 2, 3, 2, 3}),
			PlayerOnRoll: side,
		}
		got := rollers(t, BearoffRequest{Source: SourceLibrary, Seed: seed}, 50)
		if len(got) != 1 || got[side] != 50 {
			t.Errorf("a library seed with %d on roll produced %v", side, got)
		}
	}

	// A board nobody assigned a side to falls back to the draw rather than to
	// the zero value.
	unassigned := &domain.Position{
		Board:        boardWith(sideBoard{3, 2, 3, 2, 3, 2}, sideBoard{2, 3, 2, 3, 2, 3}),
		PlayerOnRoll: domain.None,
	}
	got := rollers(t, BearoffRequest{Source: SourceLibrary, Seed: unassigned}, 400)
	if got[domain.Black] == 0 || got[domain.White] == 0 {
		t.Errorf("an unassigned seed produced %v: the roller is not drawn", got)
	}
}
