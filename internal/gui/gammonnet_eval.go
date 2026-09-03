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
	// PreRoll is the position's fact vector (ADR-0017): win/gammon/
	// backgammon chances and the cubeless equity, before any roll, in
	// gammonNet's own referential (money at every score, ADR-0016). Always
	// present when a search could produce one — on Moves it comes from an
	// extra search paid only here (ADR-0017's measured +36% at display
	// depth), on Cube it is free (gammonnet.EvalResult.PreRoll).
	PreRoll *PositionFacts `json:"preRoll,omitempty"`
	// CubeVerdict is the cube verdict as a KEY the panel translates —
	// "no_double", "double_take", "double_pass" or "too_good", the same four
	// race.CubeVerdict.Verdict already carries (ADR-0019 rule 3). Cube's own
	// BestCubeAction string stays what it has always been: a stored field
	// carrying an analysing engine's own words, which is right for an
	// imported record and wrong for our own live evaluation (it arrived in
	// English, and had lost too_good on the way).
	CubeVerdict race.Verdict `json:"cubeVerdict,omitempty"`
	// Refused is true when this build declines the position outright — a
	// match score beyond the MET's horizon, typically. It is DATA, not an
	// error: an error becomes a rejected promise the panel logs and swallows,
	// leaving the previous position's numbers on screen forever under a
	// placeholder that promises an evaluation is coming (ADR-0017 noted this
	// and left it; ADR-0019 rule 4 closes it). A refusal is a state the panel
	// names, so it has to arrive as a value.
	Refused bool `json:"refused,omitempty"`
}

// PositionFacts is GammonNetEvalResult.PreRoll's JSON shape — the wire
// mirror of gammonnet.PreRollFacts (an internal Go struct with no json
// tags of its own).
type PositionFacts struct {
	PlayerWinChance          float64 `json:"playerWinChance"`
	PlayerGammonChance       float64 `json:"playerGammonChance"`
	PlayerBackgammonChance   float64 `json:"playerBackgammonChance"`
	OpponentWinChance        float64 `json:"opponentWinChance"`
	OpponentGammonChance     float64 `json:"opponentGammonChance"`
	OpponentBackgammonChance float64 `json:"opponentBackgammonChance"`
	CubelessEquity           float64 `json:"cubelessEquity"`
}

// EvaluatePositionImmediate is the 0-ply, synchronous tier: called on every
// position edit. pruneK/candidates come from Config (GetGammonNetPruneK /
// GetGammonNetCandidates) — internal/gui cannot import package main's Config
// (it would be a circular import), so the frontend reads its own settings
// and passes them here, exactly as any other parameterised RPC call.
func (a *App) EvaluatePositionImmediate(pos domain.Position, pruneK, candidates int) (GammonNetEvalResult, error) {
	return a.evaluateGammonNet(pos, 0, pruneK, candidates)
}

// gammonNetLivePool is the Eval panel's reused search apparatus (#196/C.9):
// the up to three searches one evaluateGammonNet call makes — the
// moves-or-cube decision itself, preRollFacts' own Probs, and
// evaluateRaceRegime's own Probs — share ONE Searcher (and its worker pool)
// across a call, and across GESTURES, instead of each allocating and
// zeroing its own every time. Before this, the display-depth tier (2-ply,
// LiveWorkers(2) = NumCPU) built up to three WithWorkers(NumCPU) pools per
// gesture — about 190 MB on a 16-core machine, cold caches every time.
//
// Reconfigure carries the risk this pool exists to manage: it keeps the
// searcher's own scratch AND its cache, but it can never turn the prune
// network on or off (Reconfigure's own doc comment) — that is fixed at
// construction. So acquire rebuilds from scratch whenever pruneK changes,
// and only Reconfigures when it has not.
//
// acquire is a no-op pool below LiveWorkers' own ply-2 floor: the
// SYNCHRONOUS, per-keystroke 0-ply tier (EvaluatePositionImmediate) never
// takes gnEvalMu at all — LiveWorkers(0) is 1, a searcher with no worker
// pool is cheap to build and discard, and sharing this pool across tiers
// would make every keystroke wait behind whatever the background "at rest"
// search (StartEvaluationAtRest) is doing, exactly the stall the panel's
// two-tier split exists to avoid.
//
// mu also serialises the rare case this file's own KNOWN LIMIT note already
// accepts: a superseded "at rest" search still runs to completion
// (cancellation is cooperative, not preemptive — Searcher has no checkpoint
// inside its own recursion). Before, that superseded search and the new one
// ran on two independent pools, wasting cores but never blocking each
// other; now they serialise on one pool instead — still bounded by the same
// one-search's-worth of latency the KNOWN LIMIT already names, never an
// orphan, just no longer doubling the memory to get there.
type gammonNetLivePool struct {
	mu       sync.Mutex
	searcher *gammonnet.Searcher
	pruneK   int // the value the current searcher was BUILT with
}

// acquire returns a searcher ready for a search at ply/pruneK and a release
// function the caller must invoke once every phase of this evaluation has
// run (a single defer around the whole evaluateGammonNet call is right: the
// three searches inside it are already sequential, never concurrent with
// each other).
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

// StartEvaluationAtRest starts the display-depth (canonically 2-ply k=12)
// search in the background — bearoff.go's DownloadBearoffDB pattern. Any
// evaluation already in flight is cancelled first: one at a time. Emits
// "gammonnet-eval:done" (GammonNetEvalResult) on success,
// "gammonnet-eval:cancelled" if a newer call superseded this one before it
// finished (not an error), or "gammonnet-eval:error".
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

// CancelEvaluationAtRest aborts an in-flight background evaluation, if any —
// called by the frontend before starting a new one for a fresh gesture, and
// on its own when the panel no longer wants an answer at all (e.g. closed).
func (a *App) CancelEvaluationAtRest() {
	a.gnEvalMu.Lock()
	if a.gnEvalCancel != nil {
		a.gnEvalCancel()
		a.gnEvalCancel = nil
	}
	a.gnEvalMu.Unlock()
}

// evaluateGammonNet does the actual work, shared by both tiers. The
// moves-or-cube conversion itself (#125) now lives in package gammonnet
// (gammonnet.EvaluatePosition, #129) so the batch analysis job — which never
// touches internal/gui — can call the exact same logic; this function's own
// job is what stays specific to the live panel: the race-regime bonus.
//
// All three searches below share the ONE searcher a.gnLivePool.acquire
// hands back (#196/C.9) — released once, when every phase has run.
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

// preRollFacts is the position's fact vector (ADR-0017), whatever question
// the board is asking. When gammonnet.EvaluatePosition already produced one
// (the no-dice/Cube branch — free, see EvalResult.PreRoll's doc comment)
// this just relabels it for the wire. With dice set, EvaluatePosition never
// computes one (evaluateMoves has no reason to), so this pays for a second,
// dice-independent search — the only case that does, measured at +36% over
// the moves search itself at display depth (ADR-0017's cost table).
//
// searcher is the pool evaluateGammonNet already acquired for this call
// (#196/C.9) — reconfigured here rather than built fresh, so this second
// search reuses the same warm cache and worker pool the first one just used.
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

	// The very configuration EvaluatePosition ran for this position — same
	// referential (ADR-0016), same cube (ADR-0023) — so the fact vector and
	// the decision next to it come from one and the same search. A match
	// state this build cannot evaluate is refused there, and the fact vector
	// simply is not built — never a silent fall to money.
	cfg, state, err := gammonnet.ConfigForPosition(pos, ply, pruneK)
	if err != nil {
		return nil
	}
	scale, ok := gammonnet.NewEquityScale(state)
	if !ok {
		return nil // no referential to state the equity in (ADR-0019)
	}

	// The same searcher the decision next to it just used (#196/C.9) —
	// Reconfigure aims it back at cfg, keeping its cache and worker pool
	// (previously: a second, freshly-built WithWorkers(NumCPU) pool per
	// call, on top of the one evaluateGammonNet's main search already
	// built and the one evaluateRaceRegime is about to build).
	if err := searcher.Reconfigure(cfg); err != nil {
		return nil
	}
	// pos's own dice-free representation — Searcher.Plays takes the dice
	// separately (see evaluateMoves), so gnPos here is already the pre-roll
	// position, no clone/clear needed.
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
		// Follows pos's own referential (ADR-0016/ADR-0019), same as
		// evaluateCube's own PreRollFacts on the no-dice branch: money
		// points at money play, normalised equity at a match score.
		CubelessEquity: scale.FromSearch(gammonnet.CubelessValue(&probs, state)),
	}
}

// evaluateRaceRegime fills the race panel's "evaluated" regime (#126, ADR-0012)
// when the position is a pure bearoff outside the exact table's domain —
// cheap to check first: race.Evaluate is the fast convolution path already
// driving the panel's synchronous refresh (positionService.js's updateEPC),
// so it doubles as the domain predicate here for free, and exact-regime
// positions short-circuit before the engine is ever asked — UNLESS the
// position carries a score. The exact table is money-referential
// (MoneyFromEntry never reads pos.Score); at a match score its equities and
// verdict are in the wrong scale for what the board is asking, so the
// evaluated regime — already match-aware via Decide's MatchState — is
// computed anyway. The caller keeps exact's WinProb (a real lookup,
// referential-independent) and takes equities/verdict from this result
// (ADR-0017 decision 4); this function itself does not know how the two are
// combined, that merge lives in the frontend's displayRace. A nil return
// (not a race, exact-and-money, or the engine declined the position) just
// means the panel keeps showing whatever "estimated"/"exact" it already
// had — never an error, since this is a bonus on top of Moves/Cube, not the
// request itself.
//
// This logic lives here rather than in package race because race must not
// import gammonnet: pkg/blunderdb/engine/gammonnet/eval_measure_test.go
// (#127, internal test file, package gammonnet) already imports race for its
// exact-table comparison, and Go refuses the resulting cycle for an internal
// test augmentation (gammonnet-with-tests -> race -> gammonnet). gui already
// imports both with no such constraint, so the Decision -> race.Eval mapping
// is done here, calling race.CubeStateFor for the one piece that IS shared
// with race/eval.go's own Evaluate — exported (#197/C.10) rather than kept
// as a second 3-line copy in sync by inspection.
//
// searcher is the pool evaluateGammonNet already acquired for this call
// (#196/C.9) — reconfigured here, on a dice-cleared clone of pos, rather
// than built fresh: this used to be the THIRD WithWorkers(NumCPU) pool a
// single gesture allocated and threw away, on top of Moves/Cube's own and
// preRollFacts', all three cold-cache every time.
func evaluateRaceRegime(searcher *gammonnet.Searcher, pos *domain.Position, ply, pruneK int) *race.Eval {
	fast := race.Evaluate(pos)
	if fast.Race == nil {
		return nil
	}
	hasScore := !gammonnet.IsMoneyPosition(pos)
	if fast.Race.Regime == race.RegimeExact && !hasScore {
		return nil // exact and money: nothing this regime can add
	}

	// The exact configuration EvaluatePosition itself would build for pos
	// (#190/C.3 point 1). Before this fix, the bonus below built a plain
	// DefaultConfig — money, cubeless — and only fed its OWN, separately
	// built match state and cube owner to Decide at the very end: a
	// distribution read off a money-cubeless tree, tariffed by a
	// match-cubeful verdict. ADR-0023 already has a name for that
	// inconsistency ("Open"). ConfigForPosition also refuses a score it
	// cannot evaluate (beyond the MET horizon, or a mixed money/match
	// score) rather than silently degrading to money, which the manual
	// construction below used to do whenever its own inline
	// MatchStateFromPosition call failed.
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

	// cfg.CubeOwner/cfg.CubeX are the same CubeOwnerOf/DefaultEfficiency
	// ConfigForPosition itself derived from pos — read back off cfg rather
	// than recomputed, so this can never drift from what the search above
	// actually ran with.
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
		// Uniform with evaluateCube above: ND/DT/DP are reported regardless
		// of who owns the cube, exactly as the Eval panel's own cube table
		// already does — no separate "cube against" special case invented
		// here that does not exist there.
		//
		// All four fields go through the same EquityScale (ADR-0019): money
		// points at money play, normalised equity at a match score. They
		// come out of gammonNet on two DIFFERENT internal scales — the
		// cubeless one from the search (2×MWC−1), the three cube branches
		// from Decide (raw MWC) — so converting each from its own source is
		// what keeps this row internally consistent, and consistent with
		// the exact table's money equities the panel merges it with.
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
