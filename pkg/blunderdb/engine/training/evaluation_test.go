package training

import (
	"math/rand/v2"
	"os"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/bearoffgen/bearofftest"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
)

// Both tables, as the application has them once generated: the one-sided one
// for the EPC shown after the answer, the two-sided one for the exact regime.
func TestMain(m *testing.M) {
	dir, err := bearofftest.EnsureDataDir()
	if err != nil {
		panic("training tests: " + err.Error())
	}
	race.SetDataDir(dir)
	race.Invalidate()
	oneSided, err := bearofftest.EnsureOneSided()
	if err != nil {
		panic("training tests: " + err.Error())
	}
	if err := engine.LoadOneSided(oneSided); err != nil {
		panic("training tests: loading the one-sided table: " + err.Error())
	}
	os.Exit(m.Run())
}

func seededRNG() *rand.Rand { return rand.New(rand.NewPCG(0x5eed, 0xe7a1)) }

// generator is one serial Generator for the whole package: a Searcher costs
// megabytes, and a serial one keeps every walk replayable.
var generator = func() *Generator {
	g, err := NewGenerator(1)
	if err != nil {
		panic("training tests: " + err.Error())
	}
	return g
}()

func ask(req EvaluationRequest) EvaluationQuestion {
	return generator.evaluation(req, seededRNG(), time.Now)
}

// boardOf lays out a board from own-frame sides — the pool's own convention.
func boardOf(black, white side) domain.Board {
	gp := layOut(black, white, domain.Black)
	return toDomain(&gp)
}

// opening is the starting position, Black on roll: contact everywhere.
func opening() domain.Position {
	start := side{6: 5, 8: 3, 13: 5, 24: 2}
	return domain.Position{
		Board:        boardOf(start, start),
		Cube:         domain.Cube{Owner: domain.None},
		Score:        [2]int{domain.Unlimited, domain.Unlimited},
		PlayerOnRoll: domain.Black,
		DecisionType: domain.CubeAction,
	}
}

// smallBearoff is a pure bear-off the two-sided table covers: three chequers a
// side, the rest off.
func smallBearoff() domain.Position {
	pos := opening()
	pos.Board = boardOf(side{1: 2, 3: 1}, side{2: 1, 4: 2})
	return pos
}

// hasContact reports whether some Black chequer still has a White one to pass.
func hasContact(b *domain.Board) bool {
	if b.Points[domain.WhiteBar].Checkers > 0 || b.Points[domain.BlackBar].Checkers > 0 {
		return true
	}
	blackRear, whiteRear := 0, 25
	for i := 1; i <= domain.NumPoints; i++ {
		pt := b.Points[i]
		if pt.Checkers == 0 {
			continue
		}
		if pt.Color == domain.Black && i > blackRear {
			blackRear = i
		}
		if pt.Color == domain.White && i < whiteRear {
			whiteRear = i
		}
	}
	return blackRear > whiteRear
}

// assertAskable is what every generated question must be: a valid game in
// progress, money, cube centred or the roller's, no dice, with a truth.
func assertAskable(t *testing.T, q EvaluationQuestion) {
	t.Helper()
	if !q.Generated {
		t.Fatalf("no question: refusal %q", q.Refusal)
	}
	pos := q.Position
	if _, err := gammonnet.FromDomain(&pos); err != nil {
		t.Fatalf("the question is not a valid position: %v", err)
	}
	if gameOver(&pos.Board) {
		t.Fatal("the question is a finished game")
	}
	if !pos.IsMoney() {
		t.Fatalf("the question is played at a score %v: this tranche is money only", pos.Score)
	}
	if hasDice(&pos) {
		t.Fatalf("the question carries dice %v: it asks a pre-roll cube action", pos.Dice)
	}
	if q.WinChance < 0 || q.WinChance > 100 {
		t.Fatalf("win chance %v is not a percentage", q.WinChance)
	}
	switch q.CubeAnswer {
	case AnswerNoDouble, AnswerDoubleTake, AnswerDoublePass:
	default:
		t.Fatalf("cube answer %q is not one of the three buttons", q.CubeAnswer)
	}
	if q.Regime != race.RegimeExact && q.Regime != race.RegimeEvaluated {
		t.Fatalf("regime %q: the truth is read or evaluated, never estimated", q.Regime)
	}
}

// ── The domain: any position, money play ────────────────────────────────────

func TestAContactPositionIsAsked(t *testing.T) {
	q := ask(EvaluationRequest{Source: SourceBoard, Seed: ptr(opening())})
	assertAskable(t, q)
	if q.Plies < 1 || q.Plies > maxBoardPlies {
		t.Errorf("plies = %d, want 1..%d from the board", q.Plies, maxBoardPlies)
	}
	if !hasContact(&q.Position.Board) {
		t.Error("four plies from the opening left no contact: the walk is not playing the seed")
	}
	if q.Regime != race.RegimeEvaluated || q.Depth != gammonnet.DepthLabel(gammonnet.DefaultPly) {
		t.Errorf("regime %q depth %q, want evaluated at %s", q.Regime, q.Depth, gammonnet.DepthLabel(gammonnet.DefaultPly))
	}
	if q.EPC != nil {
		t.Error("a contact position has no exact EPC, and none may be shown")
	}
}

func TestARacePositionIsAsked(t *testing.T) {
	rng := seededRNG()
	for i := 0; i < 20; i++ {
		q := generator.evaluation(EvaluationRequest{Source: SourcePool}, rng, time.Now)
		assertAskable(t, q)
		if hasContact(&q.Position.Board) {
			t.Fatalf("draw %d: a pool question has contact — « contact just broken » is a race", i)
		}
		if q.Plies > maxPoolPlies {
			t.Fatalf("draw %d: %d plies, over the pool's %d", i, q.Plies, maxPoolPlies)
		}
	}
}

func TestPoolShapesAreRacesJustBroken(t *testing.T) {
	for i, shape := range pool {
		n, h := shape.checkers(), shape.highest()
		switch {
		case n == gammonnet.NumCheckers && h >= 10 && h <= 18:
			// a running break, or the side that has just left its anchor
		case n < gammonnet.NumCheckers && n >= 8 && h <= 6:
			// the side that was bearing off while the other held
		default:
			t.Errorf("shape %d (%d chequers, rearmost on the %d-point) is neither side of a contact that has JUST broken", i, n, h)
		}
	}
	used := make(map[int]bool)
	for _, pair := range poolPairs {
		used[pair[0]], used[pair[1]] = true, true
		b := boardOf(pool[pair[0]], pool[pair[1]])
		if hasContact(&b) {
			t.Fatalf("pair %v has contact", pair)
		}
	}
	if len(used) != len(pool) {
		t.Errorf("only %d of %d shapes can be drawn: a shape no pair admits is dead data", len(used), len(pool))
	}
}

func TestTheBoardSourceNeverAsksTheSeedItself(t *testing.T) {
	seed := opening()
	rng := seededRNG()
	for i := 0; i < 10; i++ {
		q := generator.evaluation(EvaluationRequest{Source: SourceBoard, Seed: &seed}, rng, time.Now)
		assertAskable(t, q)
		if q.Position.Board == seed.Board && q.Position.PlayerOnRoll == seed.PlayerOnRoll {
			t.Fatalf("draw %d asked the seed back: the board source is its neighbourhood, never itself", i)
		}
	}
}

func TestTheBoardSourceMakesAMoneyQuestionOfAScoredSeed(t *testing.T) {
	// « la source n'en produit pas »: the board gives its geometry,
	// the question is money at a centred cube.
	seed := opening()
	seed.Score = [2]int{3, 5}
	seed.Cube = domain.Cube{Owner: domain.White, Value: 2}
	seed.Dice = [2]int{6, 5}
	q := ask(EvaluationRequest{Source: SourceBoard, Seed: &seed})
	assertAskable(t, q)
	if q.Position.Cube.Owner != domain.None || q.Position.Cube.Value != 0 {
		t.Errorf("cube %+v, want centred", q.Position.Cube)
	}
}

func TestTheLibrarySourceKeepsThePositionAsItIs(t *testing.T) {
	seed := opening()
	seed.HasJacoby = 1
	seed.Cube = domain.Cube{Owner: domain.Black, Value: 1}
	q := ask(EvaluationRequest{Source: SourceLibrary, Seed: &seed})
	assertAskable(t, q)
	if q.Plies != 0 || q.Position.Board != seed.Board || q.Position.Cube != seed.Cube || q.Position.HasJacoby != 1 {
		t.Errorf("the library position came back changed: plies %d, cube %+v, jacoby %d", q.Plies, q.Position.Cube, q.Position.HasJacoby)
	}
}

func TestALibraryPositionThatIsNotAMoneyCubeDecisionIsRefusedByName(t *testing.T) {
	scored := opening()
	scored.Score = [2]int{3, 5}
	withDice := opening()
	withDice.Dice = [2]int{3, 1}
	withDice.DecisionType = domain.CheckerAction
	cubeAgainst := opening()
	cubeAgainst.Cube = domain.Cube{Owner: domain.White, Value: 1}
	for name, seed := range map[string]domain.Position{"at a score": scored, "with dice": withDice, "cube against": cubeAgainst} {
		q := ask(EvaluationRequest{Source: SourceLibrary, Seed: &seed})
		if q.Generated || q.Refusal != RefusalNotMoneyCubeDecision {
			t.Errorf("%s: generated %v, refusal %q — want refused as %q", name, q.Generated, q.Refusal, RefusalNotMoneyCubeDecision)
		}
	}
}

func TestABoardThatIsNotAGameIsRefusedByName(t *testing.T) {
	missing := opening()
	missing.Board.Points[13].Checkers = 4 // Black is one chequer short
	finished := opening()
	finished.Board = boardOf(side{1: 1}, side{})
	for name, seed := range map[string]domain.Position{"fourteen chequers": missing, "a finished game": finished} {
		for _, source := range []string{SourceBoard, SourceLibrary} {
			q := ask(EvaluationRequest{Source: source, Seed: &seed})
			if q.Generated || q.Refusal != RefusalNotAGame {
				t.Errorf("%s from %s: generated %v, refusal %q — want refused as %q", name, source, q.Generated, q.Refusal, RefusalNotAGame)
			}
		}
	}
}

func TestAnEmptyBoardFallsBackToThePoolAndStillSaysWhy(t *testing.T) {
	var empty domain.Position
	for i := range empty.Board.Points {
		empty.Board.Points[i] = domain.Point{Color: domain.None}
	}
	q := ask(EvaluationRequest{Source: SourceBoard, Seed: &empty})
	assertAskable(t, q)
	if q.Source != SourcePool || q.Refusal != RefusalEmptyBoard {
		t.Errorf("source %q refusal %q, want a pool question that says %q", q.Source, q.Refusal, RefusalEmptyBoard)
	}
}

func TestAnUnknownSourceIsRefused(t *testing.T) {
	if q := ask(EvaluationRequest{Source: "elsewhere"}); q.Generated || q.Refusal != RefusalUnknownSource {
		t.Errorf("generated %v refusal %q", q.Generated, q.Refusal)
	}
}

// ── The truth: the engine's, in the referential that leaves it ──────────────

func TestTheTruthIsWhatTheEngineComputesForTheQuestion(t *testing.T) {
	// The win chance and the cube answer are gammonNet's own for the very
	// position asked, at the canonical depth — the same numbers
	// EvaluatePosition gives the Eval panel. Nothing recomputes them.
	q := ask(EvaluationRequest{Source: SourceBoard, Seed: ptr(opening())})
	assertAskable(t, q)
	want, err := gammonnet.EvaluatePosition(q.Position, gammonnet.DefaultPly, gammonnet.DefaultPruneK, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := 100 * want.PreRoll.PlayerWinChance; q.WinChance != got {
		t.Errorf("win chance %v, gammonNet says %v", q.WinChance, got)
	}
	if v := verdictOf(want.CubeAction); q.CubeVerdict != v || q.CubeAnswer != answerFor(v) {
		t.Errorf("verdict %q answer %q, gammonNet says %q", q.CubeVerdict, q.CubeAnswer, v)
	}
}

func TestAnExactBearoffReadsTheTwoSidedTableAndShowsItsEPC(t *testing.T) {
	seed := smallBearoff()
	q := ask(EvaluationRequest{Source: SourceLibrary, Seed: &seed})
	assertAskable(t, q)
	exact := race.Evaluate(&seed)
	if exact.Race == nil || exact.Race.Regime != race.RegimeExact {
		t.Fatal("the fixture is not in the exact regime: the two-sided table is not resolved")
	}
	if q.Regime != race.RegimeExact || q.Depth != "" {
		t.Errorf("regime %q depth %q, want exact", q.Regime, q.Depth)
	}
	if q.WinChance != 100*exact.Race.WinProb || q.CubeVerdict != exact.Race.Money.Verdict {
		t.Errorf("win %v verdict %q, the table says %v %q", q.WinChance, q.CubeVerdict, 100*exact.Race.WinProb, exact.Race.Money.Verdict)
	}
	if q.EPC == nil || q.EPC.Bottom.EPC == nil || q.EPC.Top.EPC == nil {
		t.Error("a pure bear-off has an exact EPC on both sides, and it is shown")
	}
}

func TestARaceOutsideTheOneSidedTableShowsNoEPC(t *testing.T) {
	seed := opening()
	seed.Board = boardOf(pool[0], pool[3])
	q := ask(EvaluationRequest{Source: SourceLibrary, Seed: &seed})
	assertAskable(t, q)
	if q.EPC != nil {
		t.Error("a race with chequers outside the home board has no exact EPC in this build: none may be shown")
	}
}

func TestTooGoodIsANoDouble(t *testing.T) {
	// The three buttons say what one DOES; too good keeps the cube.
	for verdict, want := range map[race.Verdict]string{
		race.VerdictNoDouble:   AnswerNoDouble,
		race.VerdictTooGood:    AnswerNoDouble,
		race.VerdictDoubleTake: AnswerDoubleTake,
		race.VerdictDoublePass: AnswerDoublePass,
	} {
		if got := answerFor(verdict); got != want {
			t.Errorf("answerFor(%q) = %q, want %q", verdict, got, want)
		}
	}
	for action, want := range map[gammonnet.CubeAction]race.Verdict{
		gammonnet.NoDouble: race.VerdictNoDouble, gammonnet.DoubleTake: race.VerdictDoubleTake,
		gammonnet.DoublePass: race.VerdictDoublePass, gammonnet.TooGood: race.VerdictTooGood,
	} {
		if got := verdictOf(action); got != want {
			t.Errorf("verdictOf(%v) = %q, want %q", action, got, want)
		}
	}
}

// ── The walk ────────────────────────────────────────────────────────────────

func TestToDomainIsTheInverseOfFromDomain(t *testing.T) {
	rng := seededRNG()
	w := walk{walker: generator.walker, rng: rng, now: time.Now, deadline: time.Now().Add(time.Hour)}
	gp, _ := seedOf(ptr(opening()), rng)
	// Forty plies from the opening meet hits, the bar and bear-offs.
	for i := 0; i < 40; i++ {
		back := domain.Position{Board: toDomain(&gp), PlayerOnRoll: playerOf(gp.Turn)}
		again, err := gammonnet.FromDomain(&back)
		if err != nil {
			t.Fatalf("ply %d: %v", i, err)
		}
		if again != gp {
			t.Fatalf("ply %d: the round trip changed the position\n got %+v\nwant %+v", i, again, gp)
		}
		next, ok := w.onePly(gp)
		if !ok {
			break
		}
		gp = next
	}
}

func TestPastItsDeadlineAQuestionFallsBackToAPoolSeedAtZeroPlies(t *testing.T) {
	// A clock that jumps an hour at every reading: the walk is late from its
	// first ply. Without the check, the board seed would be walked.
	var hours time.Duration
	late := func() time.Time { hours += time.Hour; return time.Unix(0, 0).Add(hours) }
	q := generator.evaluation(EvaluationRequest{Source: SourceBoard, Seed: ptr(opening())}, seededRNG(), late)
	assertAskable(t, q)
	if q.Source != SourcePool || q.Plies != 0 {
		t.Errorf("source %q plies %d, want the pool at zero plies", q.Source, q.Plies)
	}
}

func ptr(p domain.Position) *domain.Position { return &p }
