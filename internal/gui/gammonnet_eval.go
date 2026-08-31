package gui

import (
	"context"
	"fmt"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
)

// The Eval panel's live evaluation (#125, ADR-0013): two tiers, never a
// stored Analysis (that is the batch job, #129 — this panel only ever
// computes, it never writes). At the gesture, a cheap 0-ply call, synchronous
// — measured at ~376µs/evaluation (ADR-0011), a round trip is not worth a
// goroutine for. At rest (debounced 500ms, frontend-side), the configured
// display ply (canonically 2, k=12) runs in the background on the same
// goroutine+context+mutex+EventsEmit shape as DownloadBearoffDB
// (bearoff.go): one search at a time, a new gesture cancels the one in
// flight.
//
// KNOWN LIMIT: gammonnet.Searcher has no cancellation checkpoint inside its
// own search loop (WithWorkers parallelises across cores, not across time) —
// "cancelling" here discards a superseded result once the search actually
// finishes, it does not stop the CPU work already in flight. That work is
// bounded (a 2-ply k=12 decision, ADR-0011: ~0.63s on eight cores) and never
// literally orphaned — it always terminates and its goroutine always exits —
// so the acceptance criterion ("no orphaned search survives") holds, but a
// user editing faster than that bound briefly runs more than one search at
// once. Preemptive cancellation would need a context threaded through
// Searcher's recursion, which is out of this ticket's scope.

// gammonNetEngineVersion is kept as an alias so existing call sites and tests
// in this package don't need to change — the canonical constant now lives
// alongside the shared conversion logic in package gammonnet (#129), so the
// batch job (database) and this panel never drift on the label.
const gammonNetEngineVersion = gammonnet.EngineVersion

// GammonNetEvalResult is what the Eval panel receives: candidate moves when
// dice are set, a cube decision otherwise — never both, mirroring the
// position itself (CONTEXT.md: dice set → checker decision, no dice → cube
// decision). Race, independently, carries the race panel's "evaluated"
// regime (#126, ADR-0012) whenever the position is a pure bearoff outside
// the exact table's domain — set alongside either Moves or Cube, since the
// race question ("what should the on-roll player do about the cube, pre-
// roll") is asked regardless of whether dice happen to be sitting on the
// board.
type GammonNetEvalResult struct {
	Moves []domain.CheckerMove         `json:"moves,omitempty"`
	Cube  *domain.DoublingCubeAnalysis `json:"cube,omitempty"`
	Race  *race.Eval                   `json:"race,omitempty"`
}

// EvaluatePositionImmediate is the 0-ply, synchronous tier: called on every
// position edit. pruneK/candidates come from Config (GetGammonNetPruneK /
// GetGammonNetCandidates) — internal/gui cannot import package main's Config
// (it would be a circular import), so the frontend reads its own settings
// and passes them here, exactly as any other parameterised RPC call.
func (a *App) EvaluatePositionImmediate(pos domain.Position, pruneK, candidates int) (GammonNetEvalResult, error) {
	return evaluateGammonNet(pos, 0, pruneK, candidates)
}

var (
	gammonNetEvalMu     sync.Mutex
	gammonNetEvalCancel context.CancelFunc
)

// StartEvaluationAtRest starts the display-depth (canonically 2-ply k=12)
// search in the background — bearoff.go's DownloadBearoffDB pattern. Any
// evaluation already in flight is cancelled first: one at a time. Emits
// "gammonnet-eval:done" (GammonNetEvalResult) on success,
// "gammonnet-eval:cancelled" if a newer call superseded this one before it
// finished (not an error), or "gammonnet-eval:error".
func (a *App) StartEvaluationAtRest(pos domain.Position, ply, pruneK, candidates int) {
	gammonNetEvalMu.Lock()
	if gammonNetEvalCancel != nil {
		gammonNetEvalCancel()
	}
	ctx, cancel := context.WithCancel(a.ctx)
	gammonNetEvalCancel = cancel
	gammonNetEvalMu.Unlock()

	go func() {
		result, err := evaluateGammonNet(pos, ply, pruneK, candidates)

		gammonNetEvalMu.Lock()
		gammonNetEvalCancel = nil
		gammonNetEvalMu.Unlock()

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

// CancelEvaluationAtRest aborts an in-flight background evaluation, if any —
// called by the frontend before starting a new one for a fresh gesture, and
// on its own when the panel no longer wants an answer at all (e.g. closed).
func (a *App) CancelEvaluationAtRest() {
	gammonNetEvalMu.Lock()
	if gammonNetEvalCancel != nil {
		gammonNetEvalCancel()
		gammonNetEvalCancel = nil
	}
	gammonNetEvalMu.Unlock()
}

// evaluateGammonNet does the actual work, shared by both tiers. The
// moves-or-cube conversion itself (#125) now lives in package gammonnet
// (gammonnet.EvaluatePosition, #129) so the batch analysis job — which never
// touches internal/gui — can call the exact same logic; this function's own
// job is what stays specific to the live panel: the race-regime bonus.
func evaluateGammonNet(pos domain.Position, ply, pruneK, candidates int) (GammonNetEvalResult, error) {
	result, err := gammonnet.EvaluatePosition(pos, ply, pruneK, candidates)
	if err != nil {
		return GammonNetEvalResult{}, err
	}

	raceEval := evaluateRaceRegime(&pos, ply, pruneK)

	return GammonNetEvalResult{Moves: result.Moves, Cube: result.Cube, Race: raceEval}, nil
}

// evaluateRaceRegime fills the race panel's "evaluated" regime (#126, ADR-0012)
// when the position is a pure bearoff outside the exact table's domain —
// cheap to check first: race.Evaluate is the fast convolution path already
// driving the panel's synchronous refresh (positionService.js's updateEPC),
// so it doubles as the domain predicate here for free, and exact-regime
// positions short-circuit before the engine is ever asked. A nil return (not
// a race, already exact, or the engine declined the position) just means the
// panel keeps showing whatever "estimated"/"exact" it already had — never an
// error, since this is a bonus on top of Moves/Cube, not the request itself.
//
// This logic lives here rather than in package race because race must not
// import gammonnet: pkg/blunderdb/engine/gammonnet/eval_measure_test.go
// (#127, internal test file, package gammonnet) already imports race for its
// exact-table comparison, and Go refuses the resulting cycle for an internal
// test augmentation (gammonnet-with-tests -> race -> gammonnet). gui already
// imports both with no such constraint, so the Decision -> race.Eval mapping
// is done here. cubeStateFor below is a 3-line duplicate of race/eval.go's
// unexported helper of the same name, kept in sync by inspection — small
// enough that a shared symbol is not worth reopening the import question.
//
// Builds its own Searcher (cheap: gammonnet.Embedded() is sync.Once-cached,
// see gammonnet.EvaluatePosition) rather than sharing the one Moves/Cube
// used — the two conversions live in different packages since #129, and this
// one is a bonus computed on a dice-cleared clone anyway (race.Evaluate
// ignores dice), so there is no result to share even when they do coincide.
func evaluateRaceRegime(pos *domain.Position, ply, pruneK int) *race.Eval {
	fast := race.Evaluate(pos)
	if fast.Race == nil || fast.Race.Regime == race.RegimeExact {
		return nil
	}

	cfg := gammonnet.DefaultConfig(ply)
	if pruneK > 0 {
		cfg.PruneK = pruneK
	}
	searcher, err := gammonnet.NewSearcher(cfg)
	if err != nil {
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

	var owner gammonnet.CubeOwner
	switch pos.Cube.Owner {
	case mover:
		owner = gammonnet.CubeOwned
	case domain.None:
		owner = gammonnet.CubeCentred
	default:
		owner = gammonnet.CubeOpponent
	}
	efficiency := gammonnet.DefaultEfficiency(owner)
	jacoby := pos.HasJacoby == 1

	// One translation (ADR-0016): the same MatchStateFromPosition
	// EvaluatePosition itself uses, not a second inline copy of it.
	var state *gammonnet.MatchState
	if m, ok := gammonnet.MatchStateFromPosition(pos); ok {
		state = &m
	}

	dec, ok := gammonnet.Decide(&probs, owner, state, efficiency, jacoby)
	if !ok {
		return nil
	}

	money := race.Money{
		// Uniform with evaluateCube above: ND/DT/DP are reported regardless
		// of who owns the cube, exactly as the Eval panel's own cube table
		// already does — no separate "cube against" special case invented
		// here that does not exist there.
		//
		// Cubeless follows pos's own referential (ADR-0016), same as the
		// other three fields: money at money play, 2×MWC−1 at a match
		// score — it would otherwise sit in a different scale from its own
		// row's NoDouble/DoubleTake/DoublePass whenever a score is present.
		CubeState:  raceCubeStateFor(pos, mover),
		Cubeless:   gammonnet.CubelessValue(&probs, state),
		NoDouble:   dec.EquityNoDouble,
		DoubleTake: dec.EquityDoubleTake,
		DoublePass: dec.EquityDoublePass,
		Verdict:    raceVerdictFromCubeAction(dec.Action),
	}

	return &race.Eval{
		Regime:  race.RegimeEvaluated,
		OnRoll:  mover,
		WinProb: float64(probs[gammonnet.PWin]),
		Money:   &money,
		Depth:   depthLabel,
	}
}

// raceCubeStateFor mirrors race/eval.go's unexported cubeStateFor.
func raceCubeStateFor(pos *domain.Position, onRoll int) race.CubeState {
	switch pos.Cube.Owner {
	case onRoll:
		return race.CubeOwned
	case domain.None:
		return race.CubeCentered
	default:
		return race.CubeAgainst
	}
}

// raceVerdictFromCubeAction maps gammonnet's four-way cube action onto
// race.Verdict — a straight rename: all four gammonnet.CubeAction values
// have a Verdict counterpart (race.VerdictTooGood, #126).
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

// evaluateMoves/notationForCandidate/evaluateCube/cubeActionLabel moved to
// package gammonnet (gammonnet.EvaluatePosition, #129) — the batch analysis
// job needs the exact same conversion and internal/gui cannot be imported
// from pkg/blunderdb/database, so the logic now lives where both callers can
// reach it without a backwards import.
