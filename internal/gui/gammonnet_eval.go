package gui

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
)

// The Eval panel's live evaluation (ADR-0013) computes, never writes. Two
// tiers: a synchronous 0-ply call at each gesture (~376µs, not worth a
// goroutine), and at rest (debounced frontend-side) the display ply in the
// background, one search at a time, a new gesture cancelling the old.
//
// KNOWN LIMIT: Searcher has no cancellation checkpoint, so cancelling only
// discards a superseded result; its bounded CPU work (~0.63s at 2-ply k=12)
// still runs to completion.

// gammonNetEngineVersion aliases gammonnet.EngineVersion, shared with the
// batch job so the label never drifts.
const gammonNetEngineVersion = gammonnet.EngineVersion

// GammonNetEvalResult is what the Eval panel receives: Moves when dice are
// set, Cube otherwise, never both. Race carries the "evaluated" regime
// (ADR-0012) for a pure bearoff outside the exact table, alongside either.
type GammonNetEvalResult struct {
	Moves []domain.CheckerMove         `json:"moves,omitempty"`
	Cube  *domain.DoublingCubeAnalysis `json:"cube,omitempty"`
	Race  *race.Eval                   `json:"race,omitempty"`
	// PreRoll is the pre-roll fact vector (ADR-0017). Free on Cube; on
	// Moves it costs an extra search (+36% at display depth).
	PreRoll *PositionFacts `json:"preRoll,omitempty"`
	// CubeVerdict is the verdict as a key the panel translates (ADR-0019
	// rule 6). Cube.BestCubeAction is English and has no too_good.
	CubeVerdict race.Verdict `json:"cubeVerdict,omitempty"`
	// Refused: this build declines the position (e.g. a score beyond the
	// MET). Data, not an error, so the panel can name it (ADR-0019 rule 4).
	Refused bool `json:"refused,omitempty"`
}

// PositionFacts is the JSON mirror of gammonnet.PreRollFacts.
type PositionFacts struct {
	PlayerWinChance          float64 `json:"playerWinChance"`
	PlayerGammonChance       float64 `json:"playerGammonChance"`
	PlayerBackgammonChance   float64 `json:"playerBackgammonChance"`
	OpponentWinChance        float64 `json:"opponentWinChance"`
	OpponentGammonChance     float64 `json:"opponentGammonChance"`
	OpponentBackgammonChance float64 `json:"opponentBackgammonChance"`
	CubelessEquity           float64 `json:"cubelessEquity"`
}

// EvaluatePositionImmediate is the synchronous 0-ply tier, called on every
// edit. The frontend passes pruneK/candidates: gui cannot import main's Config.
func (a *App) EvaluatePositionImmediate(pos domain.Position, pruneK, candidates int) (GammonNetEvalResult, error) {
	return a.evaluateGammonNet(pos, 0, pruneK, candidates)
}

// gammonNetLivePool shares ONE Searcher and worker pool across the up to
// three searches of an evaluation and across gestures (a fresh NumCPU pool
// per call costs ~190 MB on 16 cores, cold caches).
//
// Reconfigure cannot toggle the prune network, so acquire rebuilds when
// pruneK changes. Below LiveWorkers' ply-2 floor it bypasses the pool, so the
// per-keystroke 0-ply tier never waits behind the at-rest search. mu
// serialises a superseded search (see KNOWN LIMIT) rather than doubling memory.
type gammonNetLivePool struct {
	mu       sync.Mutex
	searcher *gammonnet.Searcher
	pruneK   int // the value the current searcher was BUILT with
}

// acquire returns a searcher for ply/pruneK and a release to call once the
// whole (sequential) evaluation has run.
func (p *gammonNetLivePool) acquire(ply, pruneK int) (*gammonnet.Searcher, func(), error) {
	if gammonnet.LiveWorkers(ply) <= 1 {
		s, err := gammonnet.NewBatchSearcher(ply, pruneK)
		if err != nil {
			return nil, nil, err
		}
		return s, func() {}, nil
	}

	p.mu.Lock()
	if p.searcher == nil || p.pruneK != pruneK {
		s, err := gammonnet.NewBatchSearcher(ply, pruneK)
		if err != nil {
			p.mu.Unlock()
			return nil, nil, err
		}
		p.searcher = s.WithWorkers(gammonnet.LiveWorkers(ply))
		p.pruneK = pruneK
	}
	return p.searcher, p.mu.Unlock, nil
}

// StartEvaluationAtRest runs the display-depth search in the background,
// cancelling any in flight. Emits "gammonnet-eval:done" (GammonNetEvalResult),
// "gammonnet-eval:cancelled" when superseded, or "gammonnet-eval:error".
func (a *App) StartEvaluationAtRest(pos domain.Position, ply, pruneK, candidates int) {
	a.gnEvalMu.Lock()
	if a.gnEvalCancel != nil {
		a.gnEvalCancel()
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.gnEvalCancel = cancel
	a.gnEvalMu.Unlock()

	go func() {
		defer recoverBackground(a.ctx, "gammonNet evaluation at rest")
		result, err := a.evaluateGammonNet(pos, ply, pruneK, candidates)

		a.gnEvalMu.Lock()
		a.gnEvalCancel = nil
		a.gnEvalMu.Unlock()

		if ctx.Err() != nil {
			runtime.EventsEmit(a.ctx, "gammonnet-eval:cancelled")
			return
		}
		if err != nil {
			runtime.EventsEmit(a.ctx, "gammonnet-eval:error", map[string]string{"message": err.Error()})
			return
		}
		runtime.EventsEmit(a.ctx, "gammonnet-eval:done", result)
	}()
}

// CancelEvaluationAtRest aborts an in-flight background evaluation, if any.
func (a *App) CancelEvaluationAtRest() {
	a.gnEvalMu.Lock()
	if a.gnEvalCancel != nil {
		a.gnEvalCancel()
		a.gnEvalCancel = nil
	}
	a.gnEvalMu.Unlock()
}

// evaluateGammonNet serves both tiers. The moves-or-cube conversion is
// gammonnet.EvaluatePosition, shared with the batch; only the pre-roll facts
// and race regime are specific to the panel.
func (a *App) evaluateGammonNet(pos domain.Position, ply, pruneK, candidates int) (GammonNetEvalResult, error) {
	searcher, release, err := a.gnLivePool.acquire(ply, pruneK)
	if err != nil {
		return GammonNetEvalResult{}, err
	}
	defer release()

	result, err := gammonnet.EvaluatePositionWith(searcher, pos, ply, pruneK, candidates)
	if err != nil {
		// A refusal is an answer ("this build cannot judge that score"), a
		// breakage is not. Only the first travels as data.
		if errors.Is(err, gammonnet.ErrNotEvaluable) {
			return GammonNetEvalResult{Refused: true}, nil
		}
		return GammonNetEvalResult{}, err
	}

	raceEval := evaluateRaceRegime(searcher, &pos, ply, pruneK)
	preRoll := preRollFacts(searcher, &pos, ply, pruneK, result.PreRoll)

	verdict := race.Verdict("")
	if result.Cube != nil {
		verdict = raceVerdictFromCubeAction(result.CubeAction)
	}

	return GammonNetEvalResult{Moves: result.Moves, Cube: result.Cube, Race: raceEval, PreRoll: preRoll, CubeVerdict: verdict}, nil
}

// preRollFacts is the position's fact vector (ADR-0017): relabelled when
// the Cube branch produced it for free, otherwise a second, dice-free search
// on the call's already-acquired searcher.
func preRollFacts(searcher *gammonnet.Searcher, pos *domain.Position, ply, pruneK int, free *gammonnet.PreRollFacts) *PositionFacts {
	if free != nil {
		return &PositionFacts{
			PlayerWinChance:          free.PlayerWinChance,
			PlayerGammonChance:       free.PlayerGammonChance,
			PlayerBackgammonChance:   free.PlayerBackgammonChance,
			OpponentWinChance:        free.OpponentWinChance,
			OpponentGammonChance:     free.OpponentGammonChance,
			OpponentBackgammonChance: free.OpponentBackgammonChance,
			CubelessEquity:           free.CubelessEquity,
		}
	}

	hasDice := pos.Dice[0] >= 1 && pos.Dice[0] <= 6 && pos.Dice[1] >= 1 && pos.Dice[1] <= 6
	if !hasDice {
		return nil // EvaluatePosition declined the cube decision too; nothing to build facts from
	}

	// Same configuration as the decision (ADR-0016, ADR-0023); an
	// unevaluable score yields no facts, never a silent fall to money.
	cfg, state, err := gammonnet.ConfigForPosition(pos, ply, pruneK)
	if err != nil {
		return nil
	}
	scale, ok := gammonnet.NewEquityScale(state)
	if !ok {
		return nil // no referential to state the equity in (ADR-0019)
	}

	// Reconfigure keeps the warm cache and worker pool.
	if err := searcher.Reconfigure(cfg); err != nil {
		return nil
	}
	// FromDomain ignores the dice: gnPos is already pre-roll.
	gnPos, err := gammonnet.FromDomain(pos)
	if err != nil {
		return nil
	}
	probs, ok := searcher.Probs(&gnPos)
	if !ok {
		return nil
	}
	return &PositionFacts{
		PlayerWinChance:          float64(probs[gammonnet.PWin]),
		PlayerGammonChance:       float64(probs[gammonnet.PWinGammon]),
		PlayerBackgammonChance:   float64(probs[gammonnet.PWinBackgammon]),
		OpponentWinChance:        1 - float64(probs[gammonnet.PWin]),
		OpponentGammonChance:     float64(probs[gammonnet.PLoseGammon]),
		OpponentBackgammonChance: float64(probs[gammonnet.PLoseBackgammon]),
		// ADR-0019: money points at money play, normalised at a score.
		CubelessEquity: scale.FromSearch(gammonnet.CubelessValue(&probs, state)),
	}
}

// evaluateRaceRegime fills the race panel's "evaluated" regime (ADR-0012)
// for a pure bearoff outside the exact table. race.Evaluate is the cheap
// domain predicate; exact positions short-circuit unless at a match score,
// where the money-only exact table is in the wrong scale (the frontend's
// displayRace merges the two, ADR-0017 decision 4). nil means the panel keeps
// what it had, never an error.
//
// It lives here, not in race, because gammonnet's internal tests import race
// and race importing gammonnet would be a cycle.
func evaluateRaceRegime(searcher *gammonnet.Searcher, pos *domain.Position, ply, pruneK int) *race.Eval {
	fast := race.Evaluate(pos)
	if fast.Race == nil {
		return nil
	}
	hasScore := !gammonnet.IsMoneyPosition(pos)
	if fast.Race.Regime == race.RegimeExact && !hasScore {
		return nil // exact and money: nothing this regime can add
	}

	// Same configuration as the decision next to it (ADR-0023); an
	// unevaluable score is refused, never degraded to money.
	cfg, state, err := gammonnet.ConfigForPosition(pos, ply, pruneK)
	if err != nil {
		return nil
	}
	if err := searcher.Reconfigure(cfg); err != nil {
		return nil
	}
	depthLabel := fmt.Sprintf("%d-ply", cfg.Ply)

	mover := pos.PlayerOnRoll

	// Pre-roll, like race.Evaluate: dice on the position are ignored.
	clone := *pos
	clone.Dice = [2]int{0, 0}
	gnPos, err := gammonnet.FromDomain(&clone)
	if err != nil {
		return nil
	}
	probs, ok := searcher.Probs(&gnPos)
	if !ok {
		return nil
	}

	// Read back off cfg so they cannot drift from what the search ran with.
	owner := cfg.CubeOwner
	efficiency := cfg.CubeX
	jacoby := pos.HasJacoby == 1

	scale, ok := gammonnet.NewEquityScale(state)
	if !ok {
		return nil // no referential to state the equity in (ADR-0019)
	}

	dec, ok := gammonnet.Decide(&probs, owner, state, efficiency, jacoby)
	if !ok {
		return nil
	}

	money := race.CubeVerdict{
		// ND/DT/DP whoever owns the cube, as evaluateCube. EquityScale
		// (ADR-0019) converts each from its own internal scale: the search
		// gives 2×MWC−1, Decide raw MWC.
		CubeState:  race.CubeStateFor(pos, mover),
		Cubeless:   scale.FromSearch(gammonnet.CubelessValue(&probs, state)),
		NoDouble:   scale.FromDecision(dec.EquityNoDouble),
		DoubleTake: scale.FromDecision(dec.EquityDoubleTake),
		DoublePass: scale.FromDecision(dec.EquityDoublePass),
		Verdict:    raceVerdictFromCubeAction(dec.Action),
	}

	return &race.Eval{
		Regime:         race.RegimeEvaluated,
		OnRoll:         mover,
		WinProb:        float64(probs[gammonnet.PWin]),
		WinGammon:      float64(probs[gammonnet.PWinGammon]),
		WinBackgammon:  float64(probs[gammonnet.PWinBackgammon]),
		LoseGammon:     float64(probs[gammonnet.PLoseGammon]),
		LoseBackgammon: float64(probs[gammonnet.PLoseBackgammon]),
		Money:          &money,
		Depth:          depthLabel,
	}
}

// raceVerdictFromCubeAction renames a gammonnet.CubeAction to race.Verdict.
func raceVerdictFromCubeAction(a gammonnet.CubeAction) race.Verdict {
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
