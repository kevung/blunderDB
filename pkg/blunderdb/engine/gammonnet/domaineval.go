package gammonnet

import (
	"errors"
	"fmt"
	"math"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// EngineVersion is the AnalysisEngine string every gammonNet-produced
// analysis carries, as gammonGo writes it (ADR-0011). It names a real
// upstream tag — the version this port implements — and is the staleness key
// of a stored analysis (AnalyzeStaleGammonNet): any change re-runs every
// stored row as a whole. Depth belongs in AnalysisDepth, never here.
//
// v1.2.1 carries the match-aware search (ADR-0016), the cube tails
// (ADR-0022), the dead cube in the Crawford game and cubeful leaves
// (ADR-0023). The gold file's pin (v1.3.0) is a different thing: the
// reference build the gate compares against. v1.3.0 differs only by FMA
// contractions this port forbids (ADR-0024), so bumping this label would
// stale every stored row for nothing.
const EngineVersion = "gammonNet v1.2.1"

// ErrNotEvaluable marks a position this build declines to answer for (a
// score beyond the MET's horizon, a cube decision the model refuses), as
// opposed to a malformed position or an engine failure: ADR-0019 rule 4 has
// the panel name a refusal.
var ErrNotEvaluable = errors.New("gammonnet: not evaluable")

// EvalResult is what evaluating a domain.Position through gammonNet
// produces: candidate moves when dice are set, a cube decision otherwise,
// never both. Shared by the live Eval panel and the batch analysis job so
// the two never drift apart.
type EvalResult struct {
	Moves []domain.CheckerMove
	Cube  *domain.DoublingCubeAnalysis
	// PreRoll is the position's fact vector before any dice are rolled —
	// chances and the CUBELESS equity, in the position's own referential
	// (ADR-0017, ADR-0019). Populated only on the no-dice branch, where it
	// reads the probs already computed; with dice it would cost a second
	// search, which the live panel pays for itself
	// (internal/gui/gammonnet_eval.go's preRollFacts).
	PreRoll *PreRollFacts
	// CubeAction is the cube verdict as a VALUE, alongside the storage string
	// Cube.BestCubeAction (ADR-0019 rule 6), so a reader gets it in their own
	// language. It never reaches a domain type. Meaningful only when Cube is
	// non-nil.
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

// CubeOwnerOf is pos's cube as the player on roll sees it — the one
// translation from domain.Cube.Owner to CubeOwner.
func CubeOwnerOf(pos *domain.Position) CubeOwner {
	switch pos.Cube.Owner {
	case pos.PlayerOnRoll:
		return CubeOwned
	case domain.None:
		return CubeCentred
	default:
		return CubeOpponent
	}
}

// IsMoneyPosition reports whether pos is unscored — both away scores at the
// -1 sentinel (CONTEXT.md). It is the one "money or match" predicate; a
// single -1 is malformed, which ConfigForPosition names.
func IsMoneyPosition(pos *domain.Position) bool {
	return pos.IsMoney()
}

// ConfigForPosition is THE search configuration gammonNet runs for pos: the
// canonical depth and pruning (pruneK overriding when > 0), the position's
// referential (ADR-0016: refused, never degraded to money, beyond the MET)
// and its cube (ADR-0023). The match state comes back alongside (nil at
// money) for the caller's EquityScale. Every caller goes through it, so a
// panel never mixes differently configured searches.
func ConfigForPosition(pos *domain.Position, ply, pruneK int) (SearchConfig, *MatchState, error) {
	cfg := DefaultConfig(ply)
	if pruneK > 0 {
		cfg.PruneK = pruneK
	}

	var state *MatchState
	if !IsMoneyPosition(pos) {
		// One -1 sentinel alone is malformed data, not a score past the
		// MET's horizon: named apart.
		if pos.Score[0] < 0 || pos.Score[1] < 0 {
			return SearchConfig{}, nil, fmt.Errorf("%w: mixed money/match score %v (one side carries the -1 money sentinel, the other a real away score)", ErrNotEvaluable, pos.Score)
		}
		m, ok := MatchStateFromPosition(pos)
		if !ok {
			return SearchConfig{}, nil, fmt.Errorf("%w: match score %v is beyond this build's MET horizon", ErrNotEvaluable, pos.Score)
		}
		cfg.UseMatch = true
		cfg.Match = m
		state = &m
	}

	owner := CubeOwnerOf(pos)
	cfg.UseCube = true
	cfg.CubeOwner = owner
	cfg.CubeX = DefaultEfficiency(owner)
	return cfg, state, nil
}

// EvaluatePosition runs a gammonNet search at the given ply (canonical
// parameters when pruneK/candidates are 0) and returns the moves-or-cube
// verdict for pos, cubeful, in pos's own referential (ADR-0016, ADR-0019):
// money points, or normalised equity at a score. The depth label reports the
// depth that actually ran. An unevaluable score is an error naming the
// score, never a fall to money.
func EvaluatePosition(pos domain.Position, ply, pruneK, candidates int) (EvalResult, error) {
	gnPos, err := FromDomain(&pos)
	if err != nil {
		return EvalResult{}, err
	}

	cfg, state, err := ConfigForPosition(&pos, ply, pruneK)
	if err != nil {
		return EvalResult{}, err
	}

	searcher, err := NewSearcher(cfg)
	if err != nil {
		return EvalResult{}, err
	}
	// Recherche au premier plan : tous les cœurs (ADR-0011). Un lot n'entre
	// jamais par ici (NewBatchSearcher/EvaluatePositionWith).
	searcher = searcher.WithWorkers(LiveWorkers(cfg.Ply))

	return evaluateConfigured(&gnPos, &pos, searcher, cfg, state, candidates)
}

// NewBatchSearcher builds the searcher a batch job keeps for the whole run:
// the canonical configuration at the given depth and pruning width, with no
// position aimed at it yet. EvaluatePositionWith re-aims it position by
// position (Searcher.Reconfigure).
//
// A Searcher costs about 5.5 MB; one per goroutine, reused, pays it once and
// keeps the cache warm, which is licit (cache.go).
func NewBatchSearcher(ply, pruneK int) (*Searcher, error) {
	cfg := DefaultConfig(ply)
	if pruneK > 0 {
		cfg.PruneK = pruneK
	}
	return NewSearcher(cfg)
}

// EvaluatePositionWith is EvaluatePosition on a searcher the caller owns and
// reuses — same result, bit for bit: the searcher is re-pointed at pos's
// configuration first, and only the cache survives. searcher must not be
// shared between goroutines.
func EvaluatePositionWith(searcher *Searcher, pos domain.Position, ply, pruneK, candidates int) (EvalResult, error) {
	if searcher == nil {
		return EvaluatePosition(pos, ply, pruneK, candidates)
	}

	gnPos, err := FromDomain(&pos)
	if err != nil {
		return EvalResult{}, err
	}

	cfg, state, err := ConfigForPosition(&pos, ply, pruneK)
	if err != nil {
		return EvalResult{}, err
	}

	if err := searcher.Reconfigure(cfg); err != nil {
		return EvalResult{}, err
	}

	return evaluateConfigured(&gnPos, &pos, searcher, cfg, state, candidates)
}

// evaluateConfigured is the body the two entry points share, with the
// searcher already aimed at cfg.
func evaluateConfigured(gnPos *Position, pos *domain.Position, searcher *Searcher, cfg SearchConfig, state *MatchState, candidates int) (EvalResult, error) {
	depthLabel := DepthLabel(cfg.Ply)

	// The one referential conversion (ADR-0019): everything that leaves is
	// normalised equity. Refused rather than emitted unconverted — an
	// unscaled match number looks plausible and is wrong by a factor of five.
	scale, ok := NewEquityScale(state)
	if !ok {
		return EvalResult{}, fmt.Errorf("gammonnet: no equity referential at score %v", pos.Score)
	}

	hasDice := pos.Dice[0] >= 1 && pos.Dice[0] <= 6 && pos.Dice[1] >= 1 && pos.Dice[1] <= 6
	if hasDice {
		moves, err := evaluateMoves(gnPos, pos, searcher, depthLabel, candidates, scale)
		if err != nil {
			return EvalResult{}, err
		}
		return EvalResult{Moves: moves}, nil
	}

	cube, preRoll, action, err := evaluateCube(gnPos, pos, searcher, depthLabel, state, cfg.CubeOwner, scale)
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
// Split from MatchStateFromPosition for callers with a better Crawford
// source than the score sentinel, such as a parsed match's game history
// (integration_gate_test.go): the importers never write the sentinel.
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

// MatchStateFromPosition is MatchStateFromScores for a lone domain.Position:
// Crawford comes from the score sentinel (either raw away score equal to 1),
// as positionService.js reads it. ok is false for money play or a state
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

// evaluateMoves ranks every legal play for the position's dice and attaches
// each candidate's notation by matching its resulting board against
// domain.LegalMoves — at the edge, never inside the search (ADR-0011).
func evaluateMoves(gnPos *Position, pos *domain.Position, searcher *Searcher, depthLabel string, maxCandidates int, scale EquityScale) ([]domain.CheckerMove, error) {
	// The searcher's own buffer: every candidate is copied out below.
	out := searcher.scratch()
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

	notations := notationIndex(domain.LegalMoves(pos), pos.PlayerOnRoll)

	// out is best-first, so each loss is bestEquity-equity (nil for the
	// best), as ingest/merge.go does. Both sides are converted first
	// (ADR-0019): the map is increasing and affine, so the ranking holds and
	// the error is in the unit imported analyses use.
	bestEquity := scale.FromSearch(out[0].Equity)

	moves := make([]domain.CheckerMove, 0, len(out))
	for i, c := range out {
		notation := notationForCandidate(&c, notations)
		equity := scale.FromSearch(c.Equity)

		var equityError *float64
		if i > 0 {
			diff := bestEquity - equity
			equityError = &diff
		}

		mine := invertProbs(&c.Probs) // Candidate.Probs is the RESULTING position's distribution, opponent's POV
		// Chance fields are percentages [0,100], like every importer's; not
		// the fraction PreRollFacts uses.
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

// notationIndex maps each legal play's resulting board (a comparable
// Position) to its blunderDB notation, built once per position. mover is the
// player on roll: Play.Result already has the turn switched, so the domain
// boards get that same turn before conversion, or nothing matches. First
// writer wins on a duplicate board.
func notationIndex(legal []domain.LegalPlay, mover int) map[Position]string {
	opponent := domain.White
	if mover == domain.White {
		opponent = domain.Black
	}
	index := make(map[Position]string, len(legal))
	for _, play := range legal {
		res := play.Result
		res.PlayerOnRoll = opponent
		gresult, err := FromDomain(&res)
		if err != nil {
			continue
		}
		if _, seen := index[gresult]; !seen {
			index[gresult] = play.Notation
		}
	}
	return index
}

// notationForCandidate reads c's notation out of that index; an unmatched
// candidate (moves_diff_test.go says never) renders as "".
func notationForCandidate(c *Candidate, index map[Position]string) string {
	return index[c.Play.Result]
}

// evaluateCube runs the pre-roll distribution (Searcher.Probs) through the
// Janowski cube decision, with the search's own state (nil at money). It also
// returns the same probs as PreRollFacts and the verdict as a value.
func evaluateCube(gnPos *Position, pos *domain.Position, searcher *Searcher, depthLabel string, state *MatchState, owner CubeOwner, scale EquityScale) (*domain.DoublingCubeAnalysis, *PreRollFacts, CubeAction, error) {
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
		// pos's own referential, like Cubeful*Equity (ADR-0016, ADR-0019).
		CubelessEquity: scale.FromSearch(CubelessValue(&probs, state)),
	}

	// owner is the searcher's root cube state: decision and leaves share it.
	efficiency := DefaultEfficiency(owner)
	jacoby := pos.HasJacoby == 1

	dec, ok := Decide(&probs, owner, state, efficiency, jacoby)
	if !ok {
		return nil, nil, NoDouble, fmt.Errorf("%w: cube decision at this score", ErrNotEvaluable)
	}

	// Into blunderDB's scale (ADR-0019) before anything is compared.
	noDouble := scale.FromDecision(dec.EquityNoDouble)
	doubleTake := scale.FromDecision(dec.EquityDoubleTake)
	doublePass := scale.FromDecision(dec.EquityDoublePass)

	// As ingest/xgmap.go's computeBestCubeAction: double is worth the cheaper
	// of take/pass, best the higher of that and no-double.
	best := noDouble
	if effectiveDouble := math.Min(doubleTake, doublePass); effectiveDouble > best {
		best = effectiveDouble
	}

	// Percentages [0,100] here; preRoll stays a fraction.
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

// cubeActionLabel renders the decision in BestCubeAction's vocabulary: the
// labels the XG/gnubg imports write plus the "too good" spelling
// engine.BestCubeVerdict decodes. TooGood's take/pass suffix names what the
// opponent would do if the cube came anyway, as XG writes it.
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
