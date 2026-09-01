package gammonnet

import (
	"errors"
	"fmt"
	"math"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// EngineVersion is the AnalysisEngine string every gammonNet-produced
// analysis carries — the same one gammonGo already writes (ADR-0011): a
// weights bump changes it, a depth change never does (depth belongs in
// AnalysisDepth).
//
// v1.1.0 is ADR-0016: the search became match-aware (use_match), so a
// checker-move analysis stored under v1.0.1 at a match score is money-only
// and, on the same scale, sits next to an imported XG/gnubg analysis that is
// not — see the batch job's staleness check, db_gammonnet_batch.go.
//
// v1.2.0 is ADR-0019: every equity leaving this package at a match score is
// now normalised equity (the scale XG and gnubg print), where v1.1.0 wrote
// the search's own 2×MWC−1 for moves and Decide's raw MWC for the cube —
// three scales in one panel, all but money's wrong against the imported
// analyses in the same column. Stored v1.1.0 analyses at a match score are
// therefore stale and re-run; money ones are unchanged but re-run too, the
// version being per-analysis and not per-referential.
//
// v1.3.0 is ADR-0022: the live cube curve got its tails back. Every cube
// equity this package reports for a position past the cash point moves up —
// on the reference position of that ADR, no-double goes from +0.995 to
// +1.14 — and with it the verdict, which can now be TooGood where v1.2.0
// could only ever say Double/Pass. Checker-move equities are untouched (the
// search values plays, not cube states), but the version is per-analysis,
// so a stored v1.2.0 position is stale as a whole and re-run as a whole.
const EngineVersion = "gammonNet v1.3.0"

// ErrNotEvaluable marks a position this build declines to answer for — a
// match score beyond the MET's horizon, or a cube decision the model refuses
// — as opposed to a malformed position or an engine failure. ADR-0019 rule 4
// makes a refusal a state the panel NAMES, so a caller has to be able to tell
// one from a breakage; before this, both arrived as an opaque error and the
// Eval panel swallowed them alike (ADR-0017 left exactly this as a
// follow-up).
var ErrNotEvaluable = errors.New("gammonnet: not evaluable")

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
	// PreRoll is the position's fact vector before any dice are rolled —
	// win/gammon/backgammon chances and the cubeless equity, in the
	// referential the search currently uses (money at every score, see
	// ADR-0016). It is the "position fact" ADR-0017 always shows, whatever
	// question the board is asking.
	//
	// Populated only on the no-dice (Cube) branch, where it costs nothing:
	// Searcher.Probs has already run to produce Cube, and this just reads
	// the same probs. On the with-dice (Moves) branch it stays nil — the
	// equivalent search would be a second one (measured +36% at display
	// depth, ADR-0017), which this shared function must not spend on every
	// batch-analysed checker decision (#129). The live Eval panel is the
	// only caller that ever needs it there, and pays for it itself
	// (internal/gui/gammonnet_eval.go's preRollFacts).
	PreRoll *PreRollFacts
	// CubeAction is the cube verdict as a VALUE, alongside the string
	// Cube.BestCubeAction (ADR-0019 rule 3). The string is a storage field:
	// importers write their analysing engine's own words into it, sometimes
	// already localised, and cubeActionLabel below has to flatten TooGood
	// into "No Double" to stay inside the three labels the rest of the app
	// parses. This field loses neither the fourth value nor the reader's
	// language, and it never reaches a domain type — so no schema, no
	// migration, and the batch analysis job keeps storing exactly what it
	// stores today. Meaningful only when Cube is non-nil.
	CubeAction CubeAction
}

// PreRollFacts is EvalResult's pre-roll fact vector — see its doc comment.
type PreRollFacts struct {
	PlayerWinChance          float64
	PlayerGammonChance       float64
	PlayerBackgammonChance   float64
	OpponentWinChance        float64
	OpponentGammonChance     float64
	OpponentBackgammonChance float64
	CubelessEquity           float64
}

// EvaluatePosition runs a gammonNet search at the given ply (canonical
// parameters when pruneK/candidates are 0) and returns the moves-or-cube
// verdict for pos, in pos's own referential (ADR-0016): cubeless money at
// money play, 2×MWC−1 at a match score. The depth label on the result always
// reports the depth that actually ran, never the one requested (DefaultConfig
// clamps).
//
// A match score this build cannot evaluate (MatchStateFromPosition refuses)
// is an error here, never a silent fall to money — the same refusal
// NewSearcher itself would make, raised earlier so the message names the
// score rather than an opaque internal state.
func EvaluatePosition(pos domain.Position, ply, pruneK, candidates int) (EvalResult, error) {
	gnPos, err := FromDomain(&pos)
	if err != nil {
		return EvalResult{}, err
	}

	cfg := DefaultConfig(ply)
	if pruneK > 0 {
		cfg.PruneK = pruneK
	}

	isMoney := pos.Score[0] < 0 && pos.Score[1] < 0
	var state *MatchState
	if !isMoney {
		m, ok := MatchStateFromPosition(&pos)
		if !ok {
			return EvalResult{}, fmt.Errorf("%w: match state at score %v", ErrNotEvaluable, pos.Score)
		}
		cfg.UseMatch = true
		cfg.Match = m
		state = &m
	}

	searcher, err := NewSearcher(cfg)
	if err != nil {
		return EvalResult{}, err
	}

	depthLabel := fmt.Sprintf("%d-ply", cfg.Ply)

	// The one referential conversion (ADR-0019): the search and the cube
	// model keep gammonNet's own scales inside, everything that leaves this
	// function is normalised equity. Refused rather than emitted unconverted
	// — an unscaled match number is plausible-looking and wrong by a factor
	// of five.
	scale, ok := NewEquityScale(state)
	if !ok {
		return EvalResult{}, fmt.Errorf("gammonnet: no equity referential at score %v", pos.Score)
	}

	hasDice := pos.Dice[0] >= 1 && pos.Dice[0] <= 6 && pos.Dice[1] >= 1 && pos.Dice[1] <= 6
	if hasDice {
		moves, err := evaluateMoves(&gnPos, &pos, searcher, depthLabel, candidates, scale)
		if err != nil {
			return EvalResult{}, err
		}
		return EvalResult{Moves: moves}, nil
	}

	cube, preRoll, action, err := evaluateCube(&gnPos, &pos, searcher, depthLabel, state, scale)
	if err != nil {
		return EvalResult{}, err
	}
	return EvalResult{Cube: cube, PreRoll: preRoll, CubeAction: action}, nil
}

// MatchStateFromScores builds a MatchState from raw away scores as the
// on-roll player sees them, the cube (blunderDB's log2 exponent convention,
// not a literal value — see the XGID contract), and an explicit Crawford
// flag — decoding the away=0 sentinel (CONTEXT.md's Away score entry:
// "1-away, post-Crawford") into the away=1 the MET expects. ok is false when
// the resulting state cannot be evaluated at all (MatchState.IsValid):
// refused, never silently degraded to money (ADR-0016).
//
// Split from MatchStateFromPosition because not every caller has the same
// source for Crawford: a domain.Position on its own only has the score
// sentinel (MatchStateFromPosition reads it), but a decision extracted from
// a parsed match graph knows better — gnubgmap.go/xgmap.go never write the
// sentinel (every 1-away position reads raw away=1, Crawford or not), so
// integration_gate_test.go derives Crawford from the match's own game
// history (isCrawfordGame) and calls this directly.
func MatchStateFromScores(awayOnRoll, awayOpponent, cubeExponent int, crawford bool) (MatchState, bool) {
	decode := func(raw int) int {
		if raw == 0 {
			return 1
		}
		return raw
	}
	state := MatchState{
		AwayOnRoll:   decode(awayOnRoll),
		AwayOpponent: decode(awayOpponent),
		Cube:         1 << uint(cubeExponent),
		Crawford:     crawford,
	}
	if !state.IsValid() {
		return MatchState{}, false
	}
	return state, true
}

// MatchStateFromPosition is MatchStateFromScores for a domain.Position taken
// on its own, with no broader match context available: Crawford comes from
// the score sentinel itself (either away score, raw, equal to 1) — the
// reading positionService.js already relies on when it copies an XGID to the
// clipboard, and every GUI-edited or XG-clipboard-imported Position supports
// it. ok is false for money play (Score == [-1,-1]) or a state
// MatchStateFromScores refuses.
func MatchStateFromPosition(pos *domain.Position) (MatchState, bool) {
	if pos.Score[0] < 0 || pos.Score[1] < 0 {
		return MatchState{}, false
	}
	mover := pos.PlayerOnRoll
	opponent := domain.White
	if mover == domain.White {
		opponent = domain.Black
	}
	crawford := pos.Score[0] == 1 || pos.Score[1] == 1
	return MatchStateFromScores(pos.Score[mover], pos.Score[opponent], pos.Cube.Value, crawford)
}

// evaluateMoves ranks every legal play for the position's own dice and
// attaches each candidate's blunderDB notation by matching its resulting
// board against domain.LegalMoves — the canonical generator, used here at
// the engine's edge (a cold path), never inside the search's own loop
// (ADR-0011's boundary rule).
func evaluateMoves(gnPos *Position, pos *domain.Position, searcher *Searcher, depthLabel string, maxCandidates int, scale EquityScale) ([]domain.CheckerMove, error) {
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
	//
	// Both sides of that subtraction are converted first (ADR-0019): the map
	// is increasing and affine, so the ranking is untouched and the error is
	// the difference of the converted equities — the unit an imported XG
	// analysis, and therefore every blunder threshold and PR computed over
	// this column, is already stated in.
	bestEquity := scale.FromSearch(out[0].Equity)

	moves := make([]domain.CheckerMove, 0, len(out))
	for i, c := range out {
		notation := notationForCandidate(&c, legal, opponent)
		equity := scale.FromSearch(c.Equity)

		var equityError *float64
		if i > 0 {
			diff := bestEquity - equity
			equityError = &diff
		}

		mine := InvertProbs(&c.Probs) // Candidate.Probs is the RESULTING position's distribution, opponent's POV
		// domain.CheckerMove's chance fields are percentages [0,100] — the
		// scale every importer (xgmap.go, gnubgmap.go, bgfmap.go) and
		// CandidateMovesTable.svelte's unscaled .toFixed(2) already use, not
		// the fraction PreRollFacts below is in.
		moves = append(moves, domain.CheckerMove{
			Index:                    i,
			AnalysisDepth:            depthLabel,
			AnalysisEngine:           EngineVersion,
			Move:                     notation,
			Equity:                   equity,
			EquityError:              equityError,
			PlayerWinChance:          100 * float64(mine[PWin]),
			PlayerGammonChance:       100 * float64(mine[PWinGammon]),
			PlayerBackgammonChance:   100 * float64(mine[PWinBackgammon]),
			OpponentWinChance:        100 * (1 - float64(mine[PWin])),
			OpponentGammonChance:     100 * float64(mine[PLoseGammon]),
			OpponentBackgammonChance: 100 * float64(mine[PLoseBackgammon]),
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
// Janowski cube decision. state is the same MatchState the search itself was
// built with (nil at money play) — EvaluatePosition's one MatchStateFromPosition
// call, not a second copy of it.
//
// Also returns the same probs as a PreRollFacts — this is the position's
// fact vector (EvalResult.PreRoll's doc comment), and building it here is
// free: probs already exists for the cube decision itself — and the verdict
// as a value (EvalResult.CubeAction), which the string label below cannot
// carry into a reader's own language.
func evaluateCube(gnPos *Position, pos *domain.Position, searcher *Searcher, depthLabel string, state *MatchState, scale EquityScale) (*domain.DoublingCubeAnalysis, *PreRollFacts, CubeAction, error) {
	probs, ok := searcher.Probs(gnPos)
	if !ok {
		return nil, nil, NoDouble, fmt.Errorf("gammonnet: could not evaluate the position for a cube decision")
	}
	preRoll := &PreRollFacts{
		PlayerWinChance:          float64(probs[PWin]),
		PlayerGammonChance:       float64(probs[PWinGammon]),
		PlayerBackgammonChance:   float64(probs[PWinBackgammon]),
		OpponentWinChance:        1 - float64(probs[PWin]),
		OpponentGammonChance:     float64(probs[PLoseGammon]),
		OpponentBackgammonChance: float64(probs[PLoseBackgammon]),
		// Follows pos's own referential (ADR-0016), same as Cubeful*Equity
		// below: money points at money play, normalised equity at a match
		// score (ADR-0019).
		CubelessEquity: scale.FromSearch(CubelessValue(&probs, state)),
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

	dec, ok := Decide(&probs, owner, state, efficiency, jacoby)
	if !ok {
		return nil, nil, NoDouble, fmt.Errorf("%w: cube decision at this score", ErrNotEvaluable)
	}

	// Out of Decide's own scale and into blunderDB's (ADR-0019) before
	// anything is compared or subtracted: money points here, normalised
	// equity at a score, the units the three columns of an imported XG
	// analysis are already in.
	noDouble := scale.FromDecision(dec.EquityNoDouble)
	doubleTake := scale.FromDecision(dec.EquityDoubleTake)
	doublePass := scale.FromDecision(dec.EquityDoublePass)

	// Same convention as ingest/xgmap.go's computeBestCubeAction: the double
	// branch is worth the CHEAPER of take/pass (the opponent picks), and the
	// reported best is whichever of that and no-double is higher — errors
	// below are each option's equity minus this.
	best := noDouble
	if effectiveDouble := math.Min(doubleTake, doublePass); effectiveDouble > best {
		best = effectiveDouble
	}

	// domain.DoublingCubeAnalysis's chance fields are percentages [0,100],
	// same convention as CheckerMove above and every importer — unlike
	// preRoll above, built from the same probs but staying a fraction for
	// EPCPanel's live "position fact" display (moverFactsToSides.js).
	return &domain.DoublingCubeAnalysis{
		AnalysisDepth:             depthLabel,
		AnalysisEngine:            EngineVersion,
		PlayerWinChances:          100 * float64(probs[PWin]),
		PlayerGammonChances:       100 * float64(probs[PWinGammon]),
		PlayerBackgammonChances:   100 * float64(probs[PWinBackgammon]),
		OpponentWinChances:        100 * (1 - float64(probs[PWin])),
		OpponentGammonChances:     100 * float64(probs[PLoseGammon]),
		OpponentBackgammonChances: 100 * float64(probs[PLoseBackgammon]),
		CubefulNoDoubleEquity:     noDouble,
		CubefulNoDoubleError:      noDouble - best,
		CubefulDoubleTakeEquity:   doubleTake,
		CubefulDoubleTakeError:    doubleTake - best,
		CubefulDoublePassEquity:   doublePass,
		CubefulDoublePassError:    doublePass - best,
		BestCubeAction:            cubeActionLabel(dec),
	}, preRoll, dec.Action, nil
}

// cubeActionLabel renders the decision in the vocabulary BestCubeAction
// already carries across the application — the three XG/gnubg imports write
// (see ingest/xgmap.go's computeBestCubeAction) plus the "too good" spelling
// engine.BestCubeVerdict already decodes and the frontend already
// translates (epc.race.verdicts.too_good, nine languages).
//
// Until ADR-0019 this collapsed TooGood onto "No Double" for want of a
// fourth label. It is not the same ruling — "no double" says the offer would
// be wrong, "too good" says the player is too strong to cash and plays on
// for the gammon — and the engine had been computing the distinction all
// along; only this string threw it away. The take/pass suffix names what the
// opponent WOULD do if the cube came anyway, which is what makes the label
// XG's own and what BestCubeVerdict reads out of it.
func cubeActionLabel(dec Decision) string {
	switch dec.Action {
	case DoubleTake:
		return "Double, Take"
	case DoublePass:
		return "Double, Pass"
	case TooGood:
		if dec.EquityDoubleTake <= dec.EquityDoublePass {
			return "Too good to double, take"
		}
		return "Too good to double, pass"
	default: // NoDouble
		return "No Double"
	}
}
