// Package training makes the Training questions that need the neural
// evaluator (ADR-0040, ADR-0041).
//
// The Evaluation exercise's domain is ANY position (ADR-0041 rule 4), so it
// needs gammonNet; race cannot import gammonnet (the reverse import exists),
// hence a package above both.
//
// As in race/generate.go, a question is a SEED plus k PLIES thrown away with
// it — not a play mode (ADR-0037). The plies are chosen by gammonNet at 0-ply
// (« gammonNet at one ply », ADR-0041 rule 1: the 0-ply search values the
// position after each play).
//
// # Where the truth comes from
//
// Money play on the one scale that leaves the engine (ADR-0019), so every
// number is the one the Eval panel shows for the same position:
//
//   - in the EXACT regime (a pure bear-off the two-sided table covers,
//     ADR-0012), the table's own win probability and money verdict;
//   - everywhere else, gammonNet at its canonical depth (the « normal » level,
//     DefaultPly/DefaultPruneK) — the regime ADR-0012 calls EVALUATED. The
//     convolution estimate is never used: a cube verdict is never estimated
//     (ADR-0009), and a win chance graded against an estimate would grade the
//     estimator.
//
// The cube answer is decided here, not in the interface: the three buttons are
// graded against CubeAnswer and the interface never compares equities itself.
//
// Parity: bound on *gui.App (GenerateEvaluationQuestion), with no CLI or
// daemon face, for the same reason as GenerateBearoffQuestion.
package training

import (
	"math/rand/v2"
	"sync"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
)

// Seed sources (ADR-0041 rule 2) — the same three words race uses, so the
// interface speaks one vocabulary for every exercise.
const (
	SourcePool    = race.SourcePool
	SourceBoard   = race.SourceBoard
	SourceLibrary = race.SourceLibrary
)

// Refusal codes. The interface owns the sentence in nine languages.
const (
	// RefusalEmptyBoard — nothing on the board. Rule 3 sends it back to the
	// pool, and the question comes back WITH this code so the panel says why.
	RefusalEmptyBoard = race.RefusalEmptyBoard
	// RefusalUnknownSource — a source this exercise does not serve.
	RefusalUnknownSource = race.RefusalUnknownSource
	// RefusalNotAGame — the board is not a position of a game in progress:
	// a side does not have fifteen chequers, or has already borne them all
	// off. The domain is « any position », and this is the one thing a
	// position has to be.
	RefusalNotAGame = "notAGame"
	// RefusalNotMoneyCubeDecision — a library position the exercise cannot
	// ask AS IT IS: played at a match score, carrying dice (a chequer
	// decision, not a pre-roll one), or with the cube in the opponent's
	// hands. The library source hands the position back unchanged, so it
	// cannot quietly move it to money or clear its dice.

	RefusalNotMoneyCubeDecision = "notMoneyCubeDecision"
	// RefusalNotEvaluable — the engine could not evaluate the question. Not
	// expected on a valid money position; named rather than turned into an
	// estimate.
	RefusalNotEvaluable = "notEvaluable"
)

// Cube answers — the three buttons, and the only values CubeAnswer takes.
// The same words the Decision exercise already sends its judge
// (engine.CubeActionNoDouble and friends).
const (
	AnswerNoDouble   = "nd"
	AnswerDoubleTake = "dt"
	AnswerDoublePass = "dp"
)

// The ply budgets of each source (ADR-0041 rule 2).
const (
	maxPoolPlies  = 10
	maxBoardPlies = 4
)

// questionDeadline bounds the WALK, as race's does: past it the generator
// drops the walk and falls back to a pool seed at k = 0 (rule 5). A walk ply
// costs about 0.2 ms, so ten plies never come near it on a working machine.
// The truth that follows is not preempted — gammonNet has no checkpoint
// inside a search — and its cost is what TestTheCostOfAQuestionStaysUnder
// ItsBudget holds.
const questionDeadline = 50 * time.Millisecond

// EvaluationRequest asks for one question. Seed is required for the board and
// library sources and ignored for the pool.
type EvaluationRequest struct {
	Source string           `json:"source"`
	Seed   *domain.Position `json:"seed,omitempty"`
}

// EvaluationQuestion is one question, or the named reason there is none.
// Generated and Refusal are independent, as in race.BearoffQuestion: both set
// is the empty-board fallback.
type EvaluationQuestion struct {
	Generated bool   `json:"generated"`
	Refusal   string `json:"refusal,omitempty"`
	// Source is where the seed actually came from.
	Source string `json:"source"`
	// Plies is how many rolls were actually played from the seed.
	Plies    int             `json:"plies"`
	Position domain.Position `json:"position"`

	// WinChance is the player on roll's chance of winning, in PERCENT,
	// before the roll — the number the user types.
	WinChance float64 `json:"winChance"`
	// CubeVerdict is the engine's four-way verdict, for the correction
	// ("too_good" included, which the three buttons cannot say).
	CubeVerdict race.Verdict `json:"cubeVerdict"`
	// CubeAnswer is the button that is right: "nd", "dt" or "dp". Too good
	// is "nd" — the player keeps the cube and plays on — and the correction
	// names why through CubeVerdict.
	CubeAnswer string `json:"cubeAnswer"`
	// Regime is where the truth came from: exact or evaluated, never
	// estimated.
	Regime race.Regime `json:"regime"`
	// Depth is the search depth that produced the truth in the evaluated
	// regime ("2-ply"), empty in the exact one.
	Depth string `json:"depth,omitempty"`
	// EPC is present only when BOTH sides have an exact one (ADR-0027): it
	// is shown after the answer, never asked.
	EPC *race.EPC `json:"epc,omitempty"`
}

// Generator owns the two searchers a question needs: a serial 0-ply one that
// chooses the plays of the walk, and one at the canonical depth that computes
// the truth. A Searcher costs megabytes to build, so they are built once and
// reused; the mutex makes one question at a time, which is all a session
// asks for (the current one is answered while the next one is prepared).
type Generator struct {
	mu     sync.Mutex
	walker *gammonnet.Searcher
	judge  *gammonnet.Searcher
}

// NewGenerator builds the two searchers. workers ≤ 1 keeps the judge serial.
func NewGenerator(workers int) (*Generator, error) {
	walker, err := gammonnet.NewBatchSearcher(0, 0)
	if err != nil {
		return nil, err
	}
	judge, err := gammonnet.NewBatchSearcher(gammonnet.DefaultPly, gammonnet.DefaultPruneK)
	if err != nil {
		return nil, err
	}
	if workers > 1 {
		judge = judge.WithWorkers(workers)
	}
	return &Generator{walker: walker, judge: judge}, nil
}

// Evaluation makes one Evaluation question.
func (g *Generator) Evaluation(req EvaluationRequest) EvaluationQuestion {
	return g.evaluation(req, rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())), time.Now)
}

// evaluation is the seeded core, with the clock the deadline is read on.
func (g *Generator) evaluation(req EvaluationRequest, rng *rand.Rand, now func() time.Time) EvaluationQuestion {
	g.mu.Lock()
	defer g.mu.Unlock()
	w := walk{walker: g.walker, rng: rng, now: now, deadline: now().Add(questionDeadline)}

	switch req.Source {
	case SourcePool:
		return g.ask(w.playOut(poolSeed(rng), rng.IntN(maxPoolPlies+1), SourcePool, ""))

	case SourceBoard:
		if req.Seed == nil {
			return EvaluationQuestion{Source: req.Source, Refusal: RefusalNotAGame}
		}
		if emptyBoard(&req.Seed.Board) {
			// Rule 3: an empty board is no position. It falls back to the
			// pool and keeps the sentence.
			return g.ask(w.playOut(poolSeed(rng), rng.IntN(maxPoolPlies+1), SourcePool, RefusalEmptyBoard))
		}
		// Only the GEOMETRY and the roller are the seed's: the question is
		// a money position at a centred cube, whatever the seed's score and
		// cube — the source makes no other kind.
		seed, ok := seedOf(req.Seed, rng)
		if !ok {
			return EvaluationQuestion{Source: req.Source, Refusal: RefusalNotAGame}
		}
		return g.ask(w.playOut(seed, 1+rng.IntN(maxBoardPlies), SourceBoard, ""))

	case SourceLibrary:
		if req.Seed == nil {
			return EvaluationQuestion{Source: req.Source, Refusal: RefusalNotAGame}
		}
		// k = 0: the position AS IT IS, so it is refused rather than
		// reframed when it is not a money cube decision.
		if !req.Seed.IsMoney() || hasDice(req.Seed) || !rollerMayDouble(req.Seed) {
			return EvaluationQuestion{Source: req.Source, Refusal: RefusalNotMoneyCubeDecision}
		}
		pos := *req.Seed
		pos.Dice = [2]int{0, 0}
		pos.DecisionType = domain.CubeAction
		if _, err := gammonnet.FromDomain(&pos); err != nil || gameOver(&pos.Board) {
			return EvaluationQuestion{Source: req.Source, Refusal: RefusalNotAGame}
		}
		return g.ask(EvaluationQuestion{Generated: true, Source: SourceLibrary, Position: pos})

	default:
		return EvaluationQuestion{Source: req.Source, Refusal: RefusalUnknownSource}
	}
}

// ask fills a laid-out question with its truth — or turns it into a refusal
// when the engine cannot give one. A question without a truth is never sent:
// it would be graded against nothing.
func (g *Generator) ask(q EvaluationQuestion) EvaluationQuestion {
	if !q.Generated {
		return q
	}
	pos := q.Position

	if epc := race.ComputeEPC(&pos.Board); epc.Bottom.EPC != nil && epc.Top.EPC != nil {
		q.EPC = &epc
	}

	// The exact regime first: a two-sided lookup is the truth where it
	// answers, and it is cheaper than any search.
	if r := race.Evaluate(&pos); r.Race != nil && r.Race.Regime == race.RegimeExact && r.Race.Money != nil && r.Race.Money.Verdict != "" {
		q.WinChance = 100 * r.Race.WinProb
		q.CubeVerdict = r.Race.Money.Verdict
		q.CubeAnswer = answerFor(q.CubeVerdict)
		q.Regime = race.RegimeExact
		return q
	}

	// Everywhere else, gammonNet — a refusal (ErrNotEvaluable) and a breakage
	// alike end in a NAMED refusal: neither is turned into an estimate, and
	// a question without a truth is not sent.
	result, err := gammonnet.EvaluatePositionWith(g.judge, pos, gammonnet.DefaultPly, gammonnet.DefaultPruneK, 0)
	if err != nil || result.Cube == nil || result.PreRoll == nil {
		return EvaluationQuestion{Source: q.Source, Refusal: RefusalNotEvaluable}
	}
	q.WinChance = 100 * result.PreRoll.PlayerWinChance
	q.CubeVerdict = verdictOf(result.CubeAction)
	q.CubeAnswer = answerFor(q.CubeVerdict)
	q.Regime = race.RegimeEvaluated
	q.Depth = result.Cube.AnalysisDepth
	return q
}

// answerFor is the button a verdict makes right. Too good is no double: the
// player does not turn the cube, and the three buttons name what one DOES.
func answerFor(v race.Verdict) string {
	switch v {
	case race.VerdictDoubleTake:
		return AnswerDoubleTake
	case race.VerdictDoublePass:
		return AnswerDoublePass
	default:
		return AnswerNoDouble
	}
}

// verdictOf renames gammonNet's cube action into the verdict the rest of the
// application reads — the same four values, one to one.
func verdictOf(a gammonnet.CubeAction) race.Verdict {
	switch a {
	case gammonnet.DoubleTake:
		return race.VerdictDoubleTake
	case gammonnet.DoublePass:
		return race.VerdictDoublePass
	case gammonnet.TooGood:
		return race.VerdictTooGood
	default:
		return race.VerdictNoDouble
	}
}

func hasDice(pos *domain.Position) bool {
	return pos.Dice[0] >= 1 && pos.Dice[0] <= 6 && pos.Dice[1] >= 1 && pos.Dice[1] <= 6
}

// rollerMayDouble is false only when the opponent holds the cube.
func rollerMayDouble(pos *domain.Position) bool {
	return pos.Cube.Owner == domain.None || pos.Cube.Owner == pos.PlayerOnRoll
}

func emptyBoard(b *domain.Board) bool {
	for _, pt := range b.Points {
		if pt.Checkers > 0 {
			return false
		}
	}
	return true
}

func gameOver(b *domain.Board) bool {
	return b.Bearoff[domain.Black] >= 15 || b.Bearoff[domain.White] >= 15
}

// seedOf reads a seed off a board the user brought: its geometry and its
// roller, in gammonNet's representation, or false when it is not a game in
// progress. A seed that names no roller gets one drawn, as race.rollerOf does.
func seedOf(p *domain.Position, rng *rand.Rand) (gammonnet.Position, bool) {
	pos := *p
	if pos.PlayerOnRoll != domain.Black && pos.PlayerOnRoll != domain.White {
		pos.PlayerOnRoll = rng.IntN(2)
	}
	if gameOver(&pos.Board) {
		return gammonnet.Position{}, false
	}
	gp, err := gammonnet.FromDomain(&pos)
	if err != nil {
		return gammonnet.Position{}, false
	}
	return gp, true
}
