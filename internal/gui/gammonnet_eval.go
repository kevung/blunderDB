package gui

import (
	"context"
	"fmt"
	"math"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
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

const gammonNetEngineVersion = "gammonNet v1.0.1"

// GammonNetEvalResult is what the Eval panel receives: candidate moves when
// dice are set, a cube decision otherwise — never both, mirroring the
// position itself (CONTEXT.md: dice set → checker decision, no dice → cube
// decision).
type GammonNetEvalResult struct {
	Moves []domain.CheckerMove         `json:"moves,omitempty"`
	Cube  *domain.DoublingCubeAnalysis `json:"cube,omitempty"`
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

// evaluateGammonNet does the actual work, shared by both tiers. Dice set on
// the position → candidate moves; no dice → a cube decision. Never both.
func evaluateGammonNet(pos domain.Position, ply, pruneK, candidates int) (GammonNetEvalResult, error) {
	gnPos, err := gammonnet.FromDomain(&pos)
	if err != nil {
		return GammonNetEvalResult{}, err
	}

	cfg := gammonnet.DefaultConfig(ply)
	if pruneK > 0 {
		cfg.PruneK = pruneK
	}
	searcher, err := gammonnet.NewSearcher(cfg)
	if err != nil {
		return GammonNetEvalResult{}, err
	}

	depthLabel := fmt.Sprintf("%d-ply", cfg.Ply) // the depth that RAN, never the one requested (ticket #125's non-negotiable rule)

	hasDice := pos.Dice[0] >= 1 && pos.Dice[0] <= 6 && pos.Dice[1] >= 1 && pos.Dice[1] <= 6
	if hasDice {
		moves, err := evaluateMoves(&gnPos, &pos, searcher, cfg, depthLabel, candidates)
		if err != nil {
			return GammonNetEvalResult{}, err
		}
		return GammonNetEvalResult{Moves: moves}, nil
	}

	cube, err := evaluateCube(&gnPos, &pos, searcher, depthLabel)
	if err != nil {
		return GammonNetEvalResult{}, err
	}
	return GammonNetEvalResult{Cube: cube}, nil
}

// evaluateMoves ranks every legal play for the position's own dice and
// attaches each candidate's blunderDB notation by matching its resulting
// board against domain.LegalMoves — the canonical generator, used here at
// the panel's edge (a cold path), never inside the search's own loop
// (ADR-0011's boundary rule).
func evaluateMoves(gnPos *gammonnet.Position, pos *domain.Position, searcher *gammonnet.Searcher, cfg gammonnet.SearchConfig, depthLabel string, maxCandidates int) ([]domain.CheckerMove, error) {
	out := make([]gammonnet.Candidate, gammonnet.MaxPlays)
	n, err := searcher.Plays(gnPos, pos.Dice[0], pos.Dice[1], out)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil // a dance: no legal play
	}
	out = out[:n]
	if maxCandidates > 0 && len(out) > maxCandidates {
		out = out[:maxCandidates]
	}

	legal := domain.LegalMoves(pos)
	opponent := domain.White
	if pos.PlayerOnRoll == domain.White {
		opponent = domain.Black
	}

	moves := make([]domain.CheckerMove, 0, len(out))
	for i, c := range out {
		notation := notationForCandidate(&c, legal, opponent)

		mine := gammonnet.InvertProbs(&c.Probs) // Candidate.Probs is the RESULTING position's distribution, opponent's POV
		moves = append(moves, domain.CheckerMove{
			Index:                    i,
			AnalysisDepth:            depthLabel,
			AnalysisEngine:           gammonNetEngineVersion,
			Move:                     notation,
			Equity:                   c.Equity,
			PlayerWinChance:          float64(mine[gammonnet.PWin]),
			PlayerGammonChance:       float64(mine[gammonnet.PWinGammon]),
			PlayerBackgammonChance:   float64(mine[gammonnet.PWinBackgammon]),
			OpponentWinChance:        1 - float64(mine[gammonnet.PWin]),
			OpponentGammonChance:     float64(mine[gammonnet.PLoseGammon]),
			OpponentBackgammonChance: float64(mine[gammonnet.PLoseBackgammon]),
		})
	}
	return moves, nil
}

// notationForCandidate finds the domain.LegalPlay whose resulting board
// matches c.Play.Result and returns its Notation — gammonnet.Position is
// directly comparable (fixed-size arrays and scalars only), the same
// technique integration_gate_test.go uses to judge the search against
// domain.LegalMoves. An unmatched candidate (should not happen: moves_diff_test.go
// holds the two generators to the same set) renders as "" rather than
// panicking — a cold-path display gap, not a crash.
func notationForCandidate(c *gammonnet.Candidate, legal []domain.LegalPlay, opponent int) string {
	for _, play := range legal {
		res := play.Result
		res.PlayerOnRoll = opponent
		gresult, err := gammonnet.FromDomain(&res)
		if err != nil {
			continue
		}
		if gresult == c.Play.Result {
			return play.Notation
		}
	}
	return ""
}

// evaluateCube runs the pre-roll distribution (Searcher.Probs) through the
// Janowski cube decision. Score/cube/Jacoby come straight off the position;
// Crawford is inferred the same way positionService.js already does for the
// XGID it copies to the clipboard (either away score == 1) — there is no
// dedicated Crawford field on domain.Position to read instead.
func evaluateCube(gnPos *gammonnet.Position, pos *domain.Position, searcher *gammonnet.Searcher, depthLabel string) (*domain.DoublingCubeAnalysis, error) {
	probs, ok := searcher.Probs(gnPos)
	if !ok {
		return nil, fmt.Errorf("gammonnet: could not evaluate the position for a cube decision")
	}

	mover := pos.PlayerOnRoll
	opponent := domain.White
	if mover == domain.White {
		opponent = domain.Black
	}

	owner := gammonnet.CubeCentred
	switch pos.Cube.Owner {
	case mover:
		owner = gammonnet.CubeOwned
	case opponent:
		owner = gammonnet.CubeOpponent
	}
	efficiency := gammonnet.DefaultEfficiency(owner)
	jacoby := pos.HasJacoby == 1

	var state *gammonnet.MatchState
	if pos.Score[0] != -1 || pos.Score[1] != -1 {
		crawford := pos.Score[0] == 1 || pos.Score[1] == 1
		state = &gammonnet.MatchState{
			AwayOnRoll:   pos.Score[mover],
			AwayOpponent: pos.Score[opponent],
			Cube:         1 << uint(pos.Cube.Value),
			Crawford:     crawford,
		}
	}

	dec, ok := gammonnet.Decide(&probs, owner, state, efficiency, jacoby)
	if !ok {
		return nil, fmt.Errorf("gammonnet: cube decision not evaluable at this score")
	}

	// Same convention as ingest/xgmap.go's computeBestCubeAction: the double
	// branch is worth the CHEAPER of take/pass (the opponent picks), and the
	// reported best is whichever of that and no-double is higher — errors
	// below are each option's equity minus this.
	best := dec.EquityNoDouble
	if effectiveDouble := math.Min(dec.EquityDoubleTake, dec.EquityDoublePass); effectiveDouble > best {
		best = effectiveDouble
	}

	return &domain.DoublingCubeAnalysis{
		AnalysisDepth:             depthLabel,
		AnalysisEngine:            gammonNetEngineVersion,
		PlayerWinChances:          float64(probs[gammonnet.PWin]),
		PlayerGammonChances:       float64(probs[gammonnet.PWinGammon]),
		PlayerBackgammonChances:   float64(probs[gammonnet.PWinBackgammon]),
		OpponentWinChances:        1 - float64(probs[gammonnet.PWin]),
		OpponentGammonChances:     float64(probs[gammonnet.PLoseGammon]),
		OpponentBackgammonChances: float64(probs[gammonnet.PLoseBackgammon]),
		CubefulNoDoubleEquity:     dec.EquityNoDouble,
		CubefulNoDoubleError:      dec.EquityNoDouble - best,
		CubefulDoubleTakeEquity:   dec.EquityDoubleTake,
		CubefulDoubleTakeError:    dec.EquityDoubleTake - best,
		CubefulDoublePassEquity:   dec.EquityDoublePass,
		CubefulDoublePassError:    dec.EquityDoublePass - best,
		BestCubeAction:            cubeActionLabel(dec.Action),
	}, nil
}

// cubeActionLabel renders gammonnet.CubeAction in the same three-way
// vocabulary XG/gnubg imports already write into BestCubeAction (see
// ingest/xgmap.go's computeBestCubeAction) — the only one the frontend
// understands (CubeVerdictTable, utils/cubeAction.js's normalizeCubeAction).
// TooGood has no such precedent to render as: it still means "don't double",
// so it folds into "No Double" rather than inventing a fourth label nothing
// downstream parses. The distinction is not lost — dec.Action is available
// to a future caller that wants it; only this string label collapses it.
func cubeActionLabel(a gammonnet.CubeAction) string {
	switch a {
	case gammonnet.DoubleTake:
		return "Double, Take"
	case gammonnet.DoublePass:
		return "Double, Pass"
	default: // NoDouble, TooGood
		return "No Double"
	}
}
