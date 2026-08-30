package gammonnet

import (
	"fmt"
	"math"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// EngineVersion is the AnalysisEngine string every gammonNet-produced
// analysis carries — the same one gammonGo already writes (ADR-0011): a
// weights bump changes it, a depth change never does (depth belongs in
// AnalysisDepth).
const EngineVersion = "gammonNet v1.0.1"

// EvalResult is what evaluating a domain.Position through gammonNet
// produces: candidate moves when dice are set, a cube decision otherwise —
// never both, mirroring the position itself (CONTEXT.md: dice set → checker
// decision, no dice → cube decision). Shared by the Eval panel's live
// evaluation (internal/gui, #125) and the batch analysis job (database,
// #129) — one conversion, two callers, so the two never drift apart on what
// "gammonNet says about this position" means.
type EvalResult struct {
	Moves []domain.CheckerMove
	Cube  *domain.DoublingCubeAnalysis
}

// EvaluatePosition runs a gammonNet search at the given ply (canonical
// parameters when pruneK/candidates are 0) and returns the moves-or-cube
// verdict for pos. The depth label on the result always reports the depth
// that actually ran, never the one requested (DefaultConfig clamps).
func EvaluatePosition(pos domain.Position, ply, pruneK, candidates int) (EvalResult, error) {
	gnPos, err := FromDomain(&pos)
	if err != nil {
		return EvalResult{}, err
	}

	cfg := DefaultConfig(ply)
	if pruneK > 0 {
		cfg.PruneK = pruneK
	}
	searcher, err := NewSearcher(cfg)
	if err != nil {
		return EvalResult{}, err
	}

	depthLabel := fmt.Sprintf("%d-ply", cfg.Ply)

	hasDice := pos.Dice[0] >= 1 && pos.Dice[0] <= 6 && pos.Dice[1] >= 1 && pos.Dice[1] <= 6
	if hasDice {
		moves, err := evaluateMoves(&gnPos, &pos, searcher, depthLabel, candidates)
		if err != nil {
			return EvalResult{}, err
		}
		return EvalResult{Moves: moves}, nil
	}

	cube, err := evaluateCube(&gnPos, &pos, searcher, depthLabel)
	if err != nil {
		return EvalResult{}, err
	}
	return EvalResult{Cube: cube}, nil
}

// evaluateMoves ranks every legal play for the position's own dice and
// attaches each candidate's blunderDB notation by matching its resulting
// board against domain.LegalMoves — the canonical generator, used here at
// the engine's edge (a cold path), never inside the search's own loop
// (ADR-0011's boundary rule).
func evaluateMoves(gnPos *Position, pos *domain.Position, searcher *Searcher, depthLabel string, maxCandidates int) ([]domain.CheckerMove, error) {
	out := make([]Candidate, MaxPlays)
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

	// out comes back best-first (Searcher.Plays), so out[0].Equity is the
	// reference every other candidate's loss is measured against — same
	// convention as ingest/merge.go's sortCheckerMovesByEquity and
	// database/db_analysis.go's two save-time copies: nil for the best move,
	// a non-negative bestEquity-equity for the rest.
	bestEquity := out[0].Equity

	moves := make([]domain.CheckerMove, 0, len(out))
	for i, c := range out {
		notation := notationForCandidate(&c, legal, opponent)

		var equityError *float64
		if i > 0 {
			diff := bestEquity - c.Equity
			equityError = &diff
		}

		mine := InvertProbs(&c.Probs) // Candidate.Probs is the RESULTING position's distribution, opponent's POV
		moves = append(moves, domain.CheckerMove{
			Index:                    i,
			AnalysisDepth:            depthLabel,
			AnalysisEngine:           EngineVersion,
			Move:                     notation,
			Equity:                   c.Equity,
			EquityError:              equityError,
			PlayerWinChance:          float64(mine[PWin]),
			PlayerGammonChance:       float64(mine[PWinGammon]),
			PlayerBackgammonChance:   float64(mine[PWinBackgammon]),
			OpponentWinChance:        1 - float64(mine[PWin]),
			OpponentGammonChance:     float64(mine[PLoseGammon]),
			OpponentBackgammonChance: float64(mine[PLoseBackgammon]),
		})
	}
	return moves, nil
}

// notationForCandidate finds the domain.LegalPlay whose resulting board
// matches c.Play.Result and returns its Notation — Position is directly
// comparable (fixed-size arrays and scalars only), the same technique
// integration_gate_test.go uses to judge the search against domain.LegalMoves.
// An unmatched candidate (should not happen: moves_diff_test.go holds the two
// generators to the same set) renders as "" rather than panicking — a
// cold-path display gap, not a crash.
func notationForCandidate(c *Candidate, legal []domain.LegalPlay, opponent int) string {
	for _, play := range legal {
		res := play.Result
		res.PlayerOnRoll = opponent
		gresult, err := FromDomain(&res)
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
// Crawford is inferred the same way positionService.js infers it for the
// XGID it copies to the clipboard (either away score == 1) — there is no
// dedicated Crawford field on domain.Position to read instead.
func evaluateCube(gnPos *Position, pos *domain.Position, searcher *Searcher, depthLabel string) (*domain.DoublingCubeAnalysis, error) {
	probs, ok := searcher.Probs(gnPos)
	if !ok {
		return nil, fmt.Errorf("gammonnet: could not evaluate the position for a cube decision")
	}

	mover := pos.PlayerOnRoll
	opponent := domain.White
	if mover == domain.White {
		opponent = domain.Black
	}

	owner := CubeCentred
	switch pos.Cube.Owner {
	case mover:
		owner = CubeOwned
	case opponent:
		owner = CubeOpponent
	}
	efficiency := DefaultEfficiency(owner)
	jacoby := pos.HasJacoby == 1

	var state *MatchState
	if pos.Score[0] != -1 || pos.Score[1] != -1 {
		crawford := pos.Score[0] == 1 || pos.Score[1] == 1
		state = &MatchState{
			AwayOnRoll:   pos.Score[mover],
			AwayOpponent: pos.Score[opponent],
			Cube:         1 << uint(pos.Cube.Value),
			Crawford:     crawford,
		}
	}

	dec, ok := Decide(&probs, owner, state, efficiency, jacoby)
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
		AnalysisEngine:            EngineVersion,
		PlayerWinChances:          float64(probs[PWin]),
		PlayerGammonChances:       float64(probs[PWinGammon]),
		PlayerBackgammonChances:   float64(probs[PWinBackgammon]),
		OpponentWinChances:        1 - float64(probs[PWin]),
		OpponentGammonChances:     float64(probs[PLoseGammon]),
		OpponentBackgammonChances: float64(probs[PLoseBackgammon]),
		CubefulNoDoubleEquity:     dec.EquityNoDouble,
		CubefulNoDoubleError:      dec.EquityNoDouble - best,
		CubefulDoubleTakeEquity:   dec.EquityDoubleTake,
		CubefulDoubleTakeError:    dec.EquityDoubleTake - best,
		CubefulDoublePassEquity:   dec.EquityDoublePass,
		CubefulDoublePassError:    dec.EquityDoublePass - best,
		BestCubeAction:            cubeActionLabel(dec.Action),
	}, nil
}

// cubeActionLabel renders CubeAction in the same three-way vocabulary
// XG/gnubg imports already write into BestCubeAction (see
// ingest/xgmap.go's computeBestCubeAction) — the only one the frontend
// understands (CubeVerdictTable, utils/cubeAction.js's normalizeCubeAction).
// TooGood has no such precedent to render as: it still means "don't double",
// so it folds into "No Double" rather than inventing a fourth label nothing
// downstream parses. The distinction is not lost — dec.Action is available
// to a caller that wants it (race.VerdictTooGood, #126); only this string
// label collapses it.
func cubeActionLabel(a CubeAction) string {
	switch a {
	case DoubleTake:
		return "Double, Take"
	case DoublePass:
		return "Double, Pass"
	default: // NoDouble, TooGood
		return "No Double"
	}
}
