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
// IT NAMES A REAL UPSTREAM TAG, and that is the whole rule. ADR-0011 put it
// that way — "the label gammonNet v1.0.1 is honest" — because the string's
// job is to say which engine produced a stored row, and a number blunderDB
// minted for itself says nothing a reader can go and check.
//
// v1.1.0 is ADR-0016: the search became match-aware (use_match), so a
// checker-move analysis stored under v1.0.1 at a match score is money-only
// and, on the same scale, sits next to an imported XG/gnubg analysis that is
// not — see the batch job's staleness check, db_gammonnet_batch.go.
//
// v1.2.0 is gammonNet's cube fix (ADR-0022): the live cube curve got its
// tails back, so every cube equity past the cash point moves up — on that
// ADR's reference position, no-double goes from +0.995 to +1.14 — and with
// it the verdict, which can now be TooGood where v1.1.0 could only ever say
// Double/Pass. It also carries ADR-0019, which is blunderDB-side (every
// equity leaving this package at a match score is now normalised equity, the
// scale XG and gnubg print, where v1.1.0 wrote the search's own 2×MWC−1 for
// moves and Decide's raw MWC for the cube). ADR-0019 briefly stamped rows
// "gammonNet v1.2.0" on main before that tag existed; no released build ever
// did (0.33.1 predates it), so the label means the upstream release and
// nothing else.
//
// Checker-move equities are untouched by the cube fix — the search values
// plays, not cube states, and leafValue is cubeless — but the version is
// per-analysis, so a stored position is stale as a whole and re-run as a
// whole.
//
// v1.2.1 is gammonNet's Crawford fix (the cube VALUE in the Crawford game is
// the dead one, not a live blend) — and the release under which this port
// switched its search to use_cube (ADR-0023): every leaf is now valued
// through the cube model at the position's own cube state, so checker-move
// equities at a match score are cubeful, the scale XG and gnubg print, and
// gammon-go / gammon-save come out of the search where v1.2.0's cubeless
// leaves could not see them. Money equities move too, slightly. Every row
// stored under an earlier label is therefore stale as a whole.
// v1.2.1 ET PAS v1.3.0, alors que le fichier d'or est désormais produit par
// v1.3.0 : les deux épingles ne disent pas la même chose. EngineVersion nomme
// la version amont que ce portage IMPLÉMENTE, et c'est la clé de péremption
// d'une analyse stockée (AnalyzeStaleGammonNet) ; l'épingle du gold nomme la
// build de référence contre laquelle la porte compare. Mesuré avant de les
// laisser diverger (2026-09-03, verticale ADR-0003 poste 3) : de v1.2.1 à
// v1.3.0, le gold argent est bit à bit identique, le gold match+videau bouge
// 41 équités de candidat d'au plus 2,2e-16 — une contraction en FMA que ce
// portage s'interdit de toute façon (ADR-0024) — et AUCUN coup choisi ne
// change. Les deux versions calculent donc la même chose pour ce que ce
// portage lit, et monter l'étiquette rendrait périmée chaque ligne stockée
// pour rien.
const EngineVersion = "gammonNet v1.2.1"

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
	// win/gammon/backgammon chances and the CUBELESS equity, in the
	// position's own referential (ADR-0016/ADR-0019). It is the "position
	// fact" ADR-0017 always shows, whatever question the board is asking;
	// cubeless by definition, even though the search that produced the
	// distribution values its leaves with the cube (ADR-0023).
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

// CubeOwnerOf is pos's cube as the player on roll sees it — the one
// translation from domain.Cube.Owner (a player index, or None for a centred
// cube) to the engine's three-valued owner, shared by the search
// configuration, the cube decision and internal/gui's race regime.
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
// -1 sentinel (CONTEXT.md) — as opposed to played at a match score. This is
// THE predicate every caller asking "money or match" reads: before #190/C.3
// it had two independent, silently divergent forms (this package's own
// `Score[0] < 0 && Score[1] < 0` here, internal/gui/gammonnet_eval.go's
// `Score[0] != -1 || Score[1] != -1` there), which agreed on a clean score
// and could disagree on a malformed one — exactly the mixed money/match case
// ConfigForPosition below now names instead of silently refusing under the
// same message as a genuine horizon refusal.
func IsMoneyPosition(pos *domain.Position) bool {
	return pos.Score[0] < 0 && pos.Score[1] < 0
}

// ConfigForPosition is THE search configuration gammonNet runs for pos: the
// canonical depth and pruning (DefaultConfig, pruneK overriding when > 0),
// the position's own referential (ADR-0016: UseMatch at a score, refused
// rather than degraded to money when the score is beyond the MET), and,
// since ADR-0023, its cube: UseCube at the cube state pos carries, at that
// state's measured efficiency. The match state comes back alongside (nil at
// money play) so a caller builds its EquityScale from the same translation.
//
// One function, every caller — EvaluatePosition, internal/gui's pre-roll
// facts and race-regime bonus — so "what gammonNet was asked" has exactly
// one definition, and a panel can never show a fact vector or a race verdict
// from a differently configured search than the decision next to it.
func ConfigForPosition(pos *domain.Position, ply, pruneK int) (SearchConfig, *MatchState, error) {
	cfg := DefaultConfig(ply)
	if pruneK > 0 {
		cfg.PruneK = pruneK
	}

	var state *MatchState
	if !IsMoneyPosition(pos) {
		// Exactly one side carrying the -1 sentinel is neither money nor a
		// valid match state — malformed data, not merely a score past the
		// MET's horizon. Naming the two apart is #190/C.3's point 3: they
		// used to collapse into the same "not evaluable at this score".
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
// verdict for pos, in pos's own referential (ADR-0016, ADR-0019): money
// points at money play, normalised equity at a match score — cubeful either
// way since ADR-0023, the leaves being valued with pos's own cube. The depth
// label on the result always reports the depth that actually ran, never the
// one requested (DefaultConfig clamps).
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

	cfg, state, err := ConfigForPosition(&pos, ply, pruneK)
	if err != nil {
		return EvalResult{}, err
	}

	searcher, err := NewSearcher(cfg)
	if err != nil {
		return EvalResult{}, err
	}
	// Un appelant qui entre par ici attend son résultat, et c'est la seule
	// recherche en cours : elle prend tous les cœurs (#148, ADR-0011). Le
	// palier 0-ply synchrone n'en prend aucun — LiveWorkers rend 1 en
	// dessous de 2 ply —, et un LOT n'entre pas par ici : il garde son propre
	// chercheur série (NewBatchSearcher/EvaluatePositionWith), son
	// parallélisme étant entre positions.
	searcher = searcher.WithWorkers(LiveWorkers(cfg.Ply))

	return evaluateConfigured(&gnPos, &pos, searcher, cfg, state, candidates)
}

// NewBatchSearcher builds the searcher a batch job keeps for the whole run:
// the canonical configuration at the given depth and pruning width, with no
// position aimed at it yet. EvaluatePositionWith re-aims it position by
// position (Searcher.Reconfigure).
//
// It exists because a Searcher costs about 5.5 MB to allocate and zero, and
// the batch used to pay that per position (#147). One per goroutine, reused,
// pays it once — and the evaluation cache stays warm from one position to
// the next, which is free and licit (cache.go: a hit returns exactly what a
// miss would have computed).
func NewBatchSearcher(ply, pruneK int) (*Searcher, error) {
	cfg := DefaultConfig(ply)
	if pruneK > 0 {
		cfg.PruneK = pruneK
	}
	return NewSearcher(cfg)
}

// EvaluatePositionWith is EvaluatePosition on a searcher the caller owns and
// reuses — same result, bit for bit, for the same position and parameters:
// the searcher is re-pointed at pos's own configuration before the search
// runs, and nothing that survives from the previous position can change an
// answer (only the cache does, and a cache hit is by construction what a
// miss would have computed).
//
// The plain EvaluatePosition remains the form for one-shot callers — the
// Eval panel, the server's per-position endpoint — which have no reason to
// manage a searcher's lifetime. searcher must not be shared between
// goroutines: it owns mutable scratch buffers, so a batch gives each
// goroutine its own.
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

// evaluateConfigured is the body the two entry points share: the searcher is
// already aimed at cfg, all that is left is to ask the position's own
// question and convert the answer once.
func evaluateConfigured(gnPos *Position, pos *domain.Position, searcher *Searcher, cfg SearchConfig, state *MatchState, candidates int) (EvalResult, error) {
	depthLabel := DepthLabel(cfg.Ply)

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
	// The searcher's own ranking buffer, not a fresh 164 Ko one per
	// position: nothing below outlives this function — every candidate read
	// out of it is copied into a domain.CheckerMove.
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
		notation := notationForCandidate(&c, notations)
		equity := scale.FromSearch(c.Equity)

		var equityError *float64
		if i > 0 {
			diff := bestEquity - equity
			equityError = &diff
		}

		mine := invertProbs(&c.Probs) // Candidate.Probs is the RESULTING position's distribution, opponent's POV
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

// notationIndex maps each legal play's resulting board, in the engine's own
// representation, to that play's blunderDB notation. Position is directly
// comparable (fixed-size arrays and scalars only) and therefore a map key —
// the same property integration_gate_test.go relies on to judge the search
// against domain.LegalMoves.
//
// Built once per position. It used to be built implicitly, and quadratically:
// notationForCandidate rescanned the whole legal list for every candidate,
// converting each of its plays through FromDomain every time, so a position
// with thirty plays paid nine hundred conversions where thirty are enough.
//
// mover is the player on roll in pos: a Candidate's Play.Result already has
// the turn switched to the opponent (Play's own doc comment), so the domain
// boards are given that same turn before conversion — otherwise nothing
// matches and every notation comes back empty.
//
// First writer wins, so a duplicate board keeps the notation the linear scan
// would have returned.
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

// notationForCandidate reads c's notation out of that index. An unmatched
// candidate (should not happen: moves_diff_test.go holds the two generators
// to the same set) renders as "" rather than panicking — a cold-path display
// gap, not a crash.
func notationForCandidate(c *Candidate, index map[Position]string) string {
	return index[c.Play.Result]
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
		// Follows pos's own referential (ADR-0016), same as Cubeful*Equity
		// below: money points at money play, normalised equity at a match
		// score (ADR-0019).
		CubelessEquity: scale.FromSearch(CubelessValue(&probs, state)),
	}

	// owner is the searcher's own root cube state (ConfigForPosition), so
	// the decision below and the leaves that produced probs were priced at
	// the same cube, by construction.
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
