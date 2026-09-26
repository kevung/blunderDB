// SPDX-License-Identifier: MIT

package gammonnet

import (
	_ "embed"
	"fmt"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
)

// Expectiminimax over dice, 0 to 4 ply, ported from gammonNet's gn_search.
//
// # The perspective rule, which is where this goes wrong silently
//
// Evaluate answers from the position's own Turn, and Play.Result already has
// the turn switched. So the value of a play TO THE PLAYER WHO MADE IT is the
// NEGATION of the network's answer on the result. Backwards, the engine
// confidently plays its opponent's best move. Every negation below is this.
//
// # The recursion
//
//	V(pos, 0) = cubeless money equity of pos, from pos.Turn's point of view
//	V(pos, k) = SUM over the 21 distinct rolls of
//	                w(roll) * max over plays of ( -V(play.Result, k-1) )
//
// A decision with known dice at depth k scores each play at -V(play.Result, k)
// — depth k, NOT k-1: the play itself is not one of the opponent rolls.
//
// # What the leaves are worth
//
// With UseMatch every node is valued through the match equity table
// (2×MWC−1, state swapped at every ply — ADR-0016); with UseCube every leaf
// goes through the cube model at the cube its mover sees, mirrored at every
// ply alongside the state (ADR-0023). Both keep the value negating between
// sides.
const (
	// MaxPly is the deepest search this engine will build — not a claim that
	// four plies are useful (upstream: +0.00022 equity per extra ply).
	MaxPly = 4
)

// DefaultPly, DefaultPruneK, DefaultPruneEquityLoss and its CI are the
// canonical "normal" level's fields (search_levels.go), read from the
// embedded export so they cannot drift from gammonNet's `gn_search_level`.
var (
	DefaultPly                   = mustLevel("normal").Ply
	DefaultPruneK                = mustLevel("normal").PruneK
	DefaultPruneEquityLoss       = mustLevel("normal").PruneEquityLoss
	DefaultPruneEquityLossCILow  = mustLevel("normal").PruneEquityLossCILow
	DefaultPruneEquityLossCIHigh = mustLevel("normal").PruneEquityLossCIHigh
)

//go:embed strehl-prune-32_v1.0.1_2026-08-27.bin
var embeddedPruneWeights []byte

var (
	pruneOnce sync.Once
	pruneNet  *Network
	pruneErr  error
)

// embeddedPruneNetwork returns the pruning network (196→32→5, distilled from
// the big one), which sorts candidates so the big network scores only the
// survivors.
func embeddedPruneNetwork() (*Network, error) {
	pruneOnce.Do(func() { pruneNet, pruneErr = Load(embeddedPruneWeights) })
	return pruneNet, pruneErr
}

// SearchConfig is what a decision is searched with.
type SearchConfig struct {
	// Ply is the search depth of the decision. 0 evaluates each resulting
	// position directly; each further ply enumerates one more opponent roll.
	Ply int

	// Filter[d] is how many candidates survive to be searched deeper at depth
	// d, 0 meaning no filtering. Filter[0] is never read: a decision at depth k
	// reads Filter[k].
	Filter [MaxPly + 1]int

	// PruneK is how many candidates the small network lets through, 0 turning
	// it off. It is raised to Filter[depth] where that is larger: pruning
	// below the filter would silently search fewer candidates than asked.
	PruneK int

	// UseMatch and Match select the referential every node is valued in
	// (ADR-0016, gn_search.c's use_match/match): cubeless money equity, or
	// 2×MWC−1 through Match. Match is the state as the caller sees it; the
	// search swaps it at every ply itself (swap_sides). One Searcher per
	// score, as per Ply (or Reconfigure).
	UseMatch bool
	Match    MatchState

	// UseCube, CubeOwner and CubeX value every LEAF through the cube model
	// (gn_search.c's use_cube, t34-videau-spec §8 step 2, ADR-0023): Value at
	// efficiency CubeX, money or match recursion. CubeOwner is the cube as the
	// caller sees it; the search mirrors it at every ply exactly where it
	// swaps the match state — state and owner always travel together. No
	// double/take/pass branches in the tree; terminalValue ignores the cube,
	// as in the C.
	//
	// CubeX is fixed at the root while CubeOwner is mirrored: a known
	// divergence (ADR-0029). A mirrored leaf is priced with the other
	// branch's coefficient, exactly as gn_search.c:299/:740 do; correcting it
	// here would turn the cube gold red. The fix is gammonNet's to write
	// (spec §4, §8 step 2).
	UseCube   bool
	CubeOwner CubeOwner
	CubeX     float64
}

// defaultFilterPrefix is the "normal" level's own filter — (0,1,3), read
// from the same embedded export as DefaultPruneK, never retyped.
var defaultFilterPrefix = mustLevel("normal").Filter

// DefaultConfig returns the canonical configuration for a given depth: pruning
// at DefaultPruneK and the published move filter (defaultFilterPrefix).
//
// The filter is what makes the depth reachable at all: a 2-ply opening
// decision costs ~13 400 evaluations with it, over 760 000 without.
//
// Depths beyond defaultFilterPrefix (3 and 4 ply) get 5 — this repository's
// own unmeasured extension; upstream only measured the 2-ply shape (0,1,3).
func DefaultConfig(ply int) SearchConfig {
	if ply < 0 {
		ply = 0
	}
	if ply > MaxPly {
		ply = MaxPly
	}
	var filter [MaxPly + 1]int
	copy(filter[:], defaultFilterPrefix)
	for d := len(defaultFilterPrefix); d < len(filter); d++ {
		filter[d] = 5
	}
	return SearchConfig{
		Ply:    ply,
		PruneK: DefaultPruneK,
		Filter: filter,
	}
}

// DepthLabel is the exact AnalysisDepth string a search at ply produces,
// after DefaultConfig's clamp to [0, MaxPly]. Staleness checks
// (database/db_gammonnet_batch.go) must compare against this, not the raw
// ply, or a clamped request reads as stale forever.
func DepthLabel(ply int) string {
	if ply < 0 {
		ply = 0
	}
	if ply > MaxPly {
		ply = MaxPly
	}
	return fmt.Sprintf("%d-ply", ply)
}

// Candidate is one legal play with what the search concluded about it.
type Candidate struct {
	Play Play
	// Probs is the distribution of the RESULTING position, from the opponent's
	// point of view — the network's raw answer, before the negation.
	Probs [NumOutputs]float32
	// Equity is the play's value to the player who made it.
	Equity float64
}

// Searcher owns everything one search needs. A Network is read-only and shared;
// a Searcher is not, and each goroutine takes its own.
type Searcher struct {
	cfg   SearchConfig
	net   *Network
	prune *Network

	ev      *Evaluator
	pruneEv *Evaluator
	cache   *evalCache
	rolls   [NumRolls]diceRoll

	evals      uint64 // big-network evaluations that actually ran
	pruneEvals uint64 // small-network evaluations
	cacheHits  uint64

	// batchFilled and batchSlotted: positions a fill pass evaluated, against
	// the lanes they occupy at EvalBatchWidth. Their ratio is the batch fill.
	batchFilled  uint64
	batchSlotted uint64

	// cubeValuations compte les distributions réellement valuées par le
	// modèle de videau (là où nodeValue appelle Value : jamais un nœud
	// terminal, jamais sous UseCube éteint) — le pendant de
	// gn_search_cube_valuations. Un nœud évalué ne porte pas toujours une
	// valuation : le dénominateur se compte, il ne se suppose pas.
	cubeValuations uint64

	// workers are independent searchers the root farms its roll loop out to.
	// Each owns its scratch, its generator and its cache; nothing is shared but
	// the read-only networks.
	workers []*Searcher

	// La file aplatie de deepenGroups et ses résultats, gardées avec le
	// brouillon pour ne pas allouer par décision.
	// frontierBest[(groupOffset+i)*NumRolls+r] est la valeur du lancer r pour
	// le candidat i d'un groupe : un emplacement par tâche, fixé avant que la
	// file ne démarre, jamais partagé.
	frontier     []rollTask
	frontierBest []float64

	// Brouillon de la racine de probsAt (search_probs.go) : un groupe de
	// candidats par lancer racine, pour que les 21 lancers passent ensemble
	// par deepenGroups. probeGroups est la sous-tranche soumise ;
	// probePassed/probeDanced tiennent la position d'un lancer qui danse,
	// résolu hors groupe.
	probeCands  [NumRolls][]Candidate
	probeGroups [][]Candidate
	probePassed [NumRolls]Position
	probeDanced [NumRolls]bool

	// matchStates holds the only two match states a decision's recursion
	// needs: even levels see [0], odd levels [1] (Swap only exchanges the
	// away scores). seedMatchState fixes both at level 0; deeper levels index
	// them (childMatchState) instead of allocating a swapped copy per step.
	// hasMatchState is false for money; each worker owns its own pair.
	matchStates   [2]MatchState
	hasMatchState bool

	gen   [MaxPly + 2]*Generator
	plays [MaxPly + 2][]Play
	cands [MaxPly + 2][]Candidate
	feat  [NumFeatures]float32

	// best is the candidate buffer (164 Ko) for the entry points that do not
	// take one: BestPlay and evaluateMoves. Allocated on first use, since a
	// worker never touches it.
	best []Candidate

	// Le brouillon du lot de shallowFill, le chemin le plus chaud du moteur,
	// alloué une fois. batchOf[l] est le candidat de la voie l : le lot ne
	// contient que les survivants (ni terminaux ni hits de cache), donc les
	// voies ne suivent pas les indices des candidats.
	batchFeat  [EvalBatchWidth][NumFeatures]float32
	batchProbs [EvalBatchWidth][NumOutputs]float32
	batchOf    [EvalBatchWidth]int
}

// NewSearcher builds a searcher over the embedded networks. A UseMatch config
// whose Match is not IsValid() is an error, never a silent fall-back to money
// (ADR-0016).
func NewSearcher(cfg SearchConfig) (*Searcher, error) {
	if cfg.UseMatch && !cfg.Match.IsValid() {
		return nil, fmt.Errorf("%w: match state %+v", ErrNotEvaluable, cfg.Match)
	}
	net, err := embeddedNetwork()
	if err != nil {
		return nil, err
	}
	var pn *Network
	if cfg.PruneK > 0 {
		if pn, err = embeddedPruneNetwork(); err != nil {
			return nil, err
		}
	}
	return newSearcherWith(cfg, net, pn), nil
}

// newSearcherWith builds a searcher over explicit networks. prune may be nil,
// which turns pruning off whatever PruneK says.
func newSearcherWith(cfg SearchConfig, net, prune *Network) *Searcher {
	return newSearcherWithCache(cfg, net, prune, defaultCacheLog2)
}

// newSearcherWithCache is newSearcherWith with the cache size broken out, so
// WithWorkers can give a worker a smaller cache: it only catches repeats
// within the queue it drains, a much smaller working set than the root's.
func newSearcherWithCache(cfg SearchConfig, net, prune *Network, cacheLog2 uint) *Searcher {
	if cfg.Ply < 0 {
		cfg.Ply = 0
	}
	if cfg.Ply > MaxPly {
		cfg.Ply = MaxPly
	}
	s := &Searcher{
		cfg:   cfg,
		net:   net,
		prune: prune,
		ev:    NewEvaluator(net),
		cache: newEvalCache(cacheLog2),
		rolls: buildRolls(),
	}
	if prune != nil {
		s.pruneEv = NewEvaluator(prune)
	}
	return s
}

// playsAt et candsAt sont les brouillons du niveau level, alloués au premier
// usage : un ouvrier par cœur les multiplie, et une décision à 2 ply ne
// descend que trois niveaux sur six. Aucun verrou : un Searcher appartient à
// une goroutine.
func (s *Searcher) playsAt(level int) []Play {
	if s.plays[level] == nil {
		s.plays[level] = make([]Play, MaxPlays)
	}
	return s.plays[level]
}

func (s *Searcher) candsAt(level int) []Candidate {
	if s.cands[level] == nil {
		s.cands[level] = make([]Candidate, MaxPlays)
	}
	return s.cands[level]
}

// genAt est le générateur de coups du niveau level, alloué au premier usage
// (166 Ko chacun, la moitié du coût de construction d'un chercheur).
func (s *Searcher) genAt(level int) *Generator {
	if s.gen[level] == nil {
		s.gen[level] = &Generator{}
	}
	return s.gen[level]
}

// pruneKeep is how many candidates survive the small network at a given depth.
func (s *Searcher) pruneKeep(depth int) int {
	if s.prune == nil || s.cfg.PruneK <= 0 {
		return 0
	}
	keep := s.cfg.PruneK
	if depth >= 0 && depth <= MaxPly && s.cfg.Filter[depth] > keep {
		keep = s.cfg.Filter[depth]
	}
	return keep
}

// matchState is s.cfg.Match as a pointer, or nil under money valuation.
func (s *Searcher) matchState() *MatchState {
	if !s.cfg.UseMatch {
		return nil
	}
	m := s.cfg.Match
	return &m
}

// Plays scores every legal play for the given dice and writes them into out,
// best first. It returns how many there are; 0 means a dance.
func (s *Searcher) Plays(pos *Position, d1, d2 int, out []Candidate) (int, error) {
	if !pos.Valid() {
		return 0, fmt.Errorf("gammonnet: position is not structurally valid")
	}
	if d1 < 1 || d1 > 6 || d2 < 1 || d2 > 6 {
		return 0, fmt.Errorf("gammonnet: dice %d-%d out of range", d1, d2)
	}
	n := s.rankPlays(pos, d1, d2, s.cfg.Ply, 0, s.matchState(), s.cfg.CubeOwner, out)
	if n < 0 {
		return 0, fmt.Errorf("gammonnet: play generation refused or overflowed")
	}
	return n, nil
}

// rankPlaysShallow fait les phases une et deux de rankPlays — générer,
// élaguer, valoriser, trier — sans approfondir, pour que la racine de probsAt
// assemble plusieurs groupes avant d'approfondir.
func (s *Searcher) rankPlaysShallow(pos *Position, d1, d2, depth, level int, state *MatchState, owner CubeOwner, out []Candidate) int {
	if level >= len(s.plays) {
		return -1
	}
	if level == 0 {
		s.seedMatchState(state) // #197/C.10: fixes matchStates[0]/[1] for this whole chain
	}
	plays := s.playsAt(level)
	count := s.genAt(level).LegalPlays(pos, d1, d2, plays)
	if count <= 0 {
		return count // 0 is a dance, -1 a refusal
	}
	written := count
	if written > len(out) {
		return -1
	}
	for i := 0; i < written; i++ {
		out[i].Play = plays[i]
	}

	theirs := s.childMatchState(level)
	theirOwner := owner.Mirror()

	if keep := s.pruneKeep(depth); keep > 0 && written > keep {
		s.shallowFill(s.pruneEv, out[:written], false)
		s.valueSweep(out[:written], theirs, theirOwner)
		sortByEquity(out[:written])
		written = keep
	}

	s.shallowFill(s.ev, out[:written], true)
	s.valueSweep(out[:written], theirs, theirOwner)
	sortByEquity(out[:written])
	return written
}

// BestPlay returns the highest-valued play. It reports false on a dance.
func (s *Searcher) BestPlay(pos *Position, d1, d2 int) (Candidate, bool, error) {
	n, err := s.Plays(pos, d1, d2, s.scratch())
	if err != nil || n == 0 {
		return Candidate{}, false, err
	}
	return s.best[0], true, nil
}

// scratch is the ranking buffer, allocated on first use. Not goroutine-safe,
// like every other piece of a Searcher's scratch.
func (s *Searcher) scratch() []Candidate {
	if s.best == nil {
		s.best = make([]Candidate, MaxPlays)
	}
	return s.best
}

// rankPlays generates, scores and orders the plays at one node.
//
// level indexes the scratch buffers (the nesting, not the search depth).
// state and owner are as pos's own mover sees them (nil state for money).
// theirs and theirOwner (swapped/mirrored once) value the results and go to
// the deep pass, both derived from the same unswapped pair, as in
// gn_search.c's rank_plays_finish/rank_plays_deepen.
func (s *Searcher) rankPlays(pos *Position, d1, d2, depth, level int, state *MatchState, owner CubeOwner, out []Candidate) int {
	// Phases one et deux : générer, élaguer, valoriser, trier.
	written := s.rankPlaysShallow(pos, d1, d2, depth, level, state, owner, out)
	if written <= 0 {
		return written // 0 est une danse, -1 un refus
	}

	// Phase trois : approfondir les meilleurs.
	if depth <= 0 {
		return written
	}
	theirs := s.childMatchState(level)
	theirOwner := owner.Mirror()
	searched := written
	if f := s.cfg.Filter[depth]; f > 0 && f < searched {
		searched = f
	}
	// À la racine, tous les candidats partent dans une seule file
	// (deepenGroups) : une barrière par décision. Ailleurs, la boucle
	// sérielle, terme pour terme identique.
	if level == 0 && len(s.workers) > 0 {
		if !s.deepenGroups([][]Candidate{out[:searched]}, depth, theirs, theirOwner) {
			return -1
		}
	} else {
		for i := 0; i < searched; i++ {
			if out[i].Play.Result.isOver() {
				continue // keeps the exact terminal value from the sweep
			}
			v, ok := s.positionEquity(&out[i].Play.Result, depth, level+1, theirs, theirOwner)
			if !ok {
				return -1
			}
			out[i].Equity = -v
		}
	}
	sortByEquity(out[:searched])
	return written
}

// seedMatchState fixes matchStates[0]/[1] from state as pos's mover sees it
// at level 0. Called wherever a chain begins at level 0; idempotent.
func (s *Searcher) seedMatchState(state *MatchState) {
	if state == nil {
		s.hasMatchState = false
		return
	}
	s.hasMatchState = true
	s.matchStates[0] = *state
	s.matchStates[1] = state.Swap()
}

// childMatchState is the state seen from the other side at level+1 —
// gn_search.c's swap_sides, as an index into matchStates. nil for money.
func (s *Searcher) childMatchState(level int) *MatchState {
	if !s.hasMatchState {
		return nil
	}
	return &s.matchStates[(level+1)&1]
}

// shallowFill writes each candidate's resulting distribution. useCache is false
// for the pruning pass: the small network's ordering must not depend on
// evaluation history.
//
// Elle alimente le noyau groupé (ADR-0024) : les candidats d'un appel sont
// des frères, dont l'union des entrées actives est petite, le lot où le noyau
// est au mieux. Terminaux, hits de cache et encodages refusés sortent du lot ;
// les survivants partent par tranches de EvalBatchWidth (la queue partielle
// est l'affaire du noyau). Le lot cherche en cache avant de ranger, ce qui
// peut rendre quelques hits de plus : seuls les compteurs bougent, un hit
// rend les bits d'un calcul (cache.go).
func (s *Searcher) shallowFill(ev *Evaluator, cands []Candidate, useCache bool) {
	filled, lanes := 0, 0
	for i := range cands {
		res := &cands[i].Play.Result
		if res.isOver() {
			cands[i].Probs = [NumOutputs]float32{}
			continue
		}
		if useCache && s.cache.lookup(res, &cands[i].Probs) {
			s.cacheHits++
			continue
		}
		// encodeLegal : une position générée est légale par construction, et
		// les points d'entrée publics valident une fois, à l'entrée.
		encodeLegal(res, &s.batchFeat[lanes])
		s.batchOf[lanes] = i
		lanes++
		if lanes == EvalBatchWidth {
			s.flushBatch(ev, cands, lanes, useCache)
			filled += lanes
			lanes = 0
		}
	}
	if lanes > 0 {
		s.flushBatch(ev, cands, lanes, useCache)
		filled += lanes
	}
	// Les tranches sont pleines sauf la dernière : les voies occupées sont
	// exactement batchSlots(filled).
	s.batchFilled += uint64(filled)
	s.batchSlotted += uint64(batchSlots(filled))
}

// flushBatch évalue les n premières voies du brouillon et redistribue les
// résultats aux candidats de batchOf, avec les mêmes compteurs qu'une
// évaluation unitaire. Le petit réseau ne touche jamais le cache.
//
// Le repli scalaire est inatteignable en pratique (EvaluateBatch ne refuse
// que ce qu'Evaluate refuse) ; il reproduit l'évaluation unitaire plutôt que
// d'inventer une valeur.
func (s *Searcher) flushBatch(ev *Evaluator, cands []Candidate, n int, useCache bool) {
	err := ev.EvaluateBatch(&s.batchFeat, n, &s.batchProbs)
	for l := 0; l < n; l++ {
		i := s.batchOf[l]
		if err == nil {
			cands[i].Probs = s.batchProbs[l]
		} else {
			_ = ev.Evaluate(s.batchFeat[l][:], &cands[i].Probs)
		}
		if useCache {
			s.evals++
			s.cache.store(&cands[i].Play.Result, &cands[i].Probs)
		} else {
			s.pruneEvals++
		}
	}
}

// valueSweep turns each candidate's distribution into its value to the player
// who made the play — hence the negation. state and owner are as the
// RESULTING position's mover sees them.
func (s *Searcher) valueSweep(cands []Candidate, state *MatchState, owner CubeOwner) {
	for i := range cands {
		res := &cands[i].Play.Result
		if res.isOver() {
			cands[i].Equity = -terminalValue(res, state)
			continue
		}
		cands[i].Equity = -s.nodeValue(&cands[i].Probs, state, owner)
	}
}

// nodeValue is the value of one evaluated node from its own mover's view:
// valueFromProbs, or under UseCube Value at owner, on the same scale. A
// failure values as 0.
//
// gn_search.c's node_value also reads the exact two-sided table for money
// leaves; this port never does, and the search gold is produced with no
// table loaded — a documented divergence.
func (s *Searcher) nodeValue(probs *[NumOutputs]float32, state *MatchState, owner CubeOwner) float64 {
	if !s.cfg.UseCube {
		return valueFromProbs(probs, state)
	}
	s.cubeValuations++
	v, ok := Value(probs, owner, state, s.cfg.CubeX)
	if !ok {
		return 0
	}
	return v
}

// positionEquity is the value of a position to the player on turn, at depth.
// state is the match state as pos's OWN mover sees it, or nil under money
// valuation; owner the cube as that mover sees it.
//
// Always serial: its callers are below the root, where the workers are never
// consulted (the roots farm out through deepenGroups).
func (s *Searcher) positionEquity(pos *Position, depth, level int, state *MatchState, owner CubeOwner) (float64, bool) {
	if level == 0 {
		s.seedMatchState(state) // #197/C.10: only a direct test entry point takes this in production
	}
	if pos.isOver() {
		return terminalValue(pos, state), true
	}
	if depth <= 0 {
		return s.leafValue(pos, state, owner), true
	}
	if level >= len(s.cands) {
		return 0, false
	}
	cands := s.candsAt(level)

	var best [NumRolls]float64
	for r := 0; r < NumRolls; r++ {
		v, ok := s.oneRoll(pos, depth, level, r, state, owner, cands)
		if !ok {
			return 0, false
		}
		best[r] = v
	}

	var sum float64
	for r := 0; r < NumRolls; r++ {
		sum += s.rolls[r].weight * best[r]
	}
	return sum, true
}

// oneRoll is the value of the best reply to one roll. state and owner are
// pos's own — unswapped, since pos and its mover are unchanged by which dice
// came up.
func (s *Searcher) oneRoll(pos *Position, depth, level, r int, state *MatchState, owner CubeOwner, cands []Candidate) (float64, bool) {
	roll := s.rolls[r]
	n := s.rankPlays(pos, int(roll.d1), int(roll.d2), depth-1, level, state, owner, cands)
	if n < 0 {
		return 0, false
	}
	if n > 0 {
		return cands[0].Equity, true
	}
	// No legal play: the turn passes, so state is swapped and owner mirrored.
	passed := *pos
	passed.swapTurn()
	v, ok := s.positionEquity(&passed, depth-1, level+1, s.childMatchState(level), owner.Mirror())
	if !ok {
		return 0, false
	}
	return -v, true
}

// rollTask est un lancer à évaluer et l'emplacement où ranger sa valeur,
// fixé avant que la file ne démarre : l'ordonnancement choisit qui calcule,
// jamais l'ordre d'addition.
type rollTask struct {
	pos  *Position
	roll int // index du lancer dans s.rolls, 0..20
	slot int // index dans le tableau de résultats du niveau
}

// rollsByCost est l'ordre LPT (Longest Processing Time first) des 21 lancers :
// coût décroissant, ce qui borne le makespan à 4/3 − 1/(3m) de l'optimum
// (Graham 1969) au lieu de laisser la tâche la plus longue tomber en dernier.
//
// Le coût est mesuré (TestProbeRollCost, évaluations par sous-arbre, 24
// positions à 2 ply) et contredit le proxy « les doubles d'abord » :
//
//	2-6 17 256   3-6 17 256   2-3 17 184   3-4 17 160   1-4 16 920
//	2-5 16 896   4-5 16 704   5-6 16 584   1-5 16 536   1-2 14 928
//	2-2 14 400   2-4 13 056   4-4 12 960   4-6 12 888   1-3 12 672
//	1-6 12 456   3-5 12 120   5-5 11 976   3-3 11 928   1-1 11 736
//	6-6 11 232
//
// Les doubles sont parmi les moins chers : l'élagage ne garde que douze de
// leurs nombreux coups, et leur sous-arbre est plus étroit. Les évaluations
// sont retenues plutôt que le temps parce qu'elles sont déterministes.
var rollsByCost = [NumRolls]int{10, 14, 7, 12, 3, 9, 16, 19, 4, 1, 6, 8, 15, 17, 2, 5, 13, 18, 11, 0, 20}

// deepenGroups approfondit tous les candidats d'un ou plusieurs groupes (un
// niveau de rankPlays, ou un lancer racine de probsAt) dans une seule file et
// derrière une seule barrière : une file de 63 tâches se répartit bien mieux
// sur les ouvriers que trois de 21.
//
// Le résultat ne bouge pas d'un bit : la somme pondérée reste sérielle, par
// candidat, en index de lancer croissant, en float64 (l'équivalent aplati de
// positionEquity) ; un groupe n'écrit jamais l'emplacement d'un autre.
//
// state et owner sont ceux de la position résultante, déjà échangés et
// miroités, les mêmes pour tous les groupes.
func (s *Searcher) deepenGroups(groups [][]Candidate, depth int, state *MatchState, owner CubeOwner) bool {
	need := 0
	for _, g := range groups {
		need += len(g) * NumRolls
	}
	if need == 0 {
		return true
	}
	if cap(s.frontierBest) < need {
		s.frontierBest = make([]float64, need)
		s.frontier = make([]rollTask, 0, need)
	}
	best := s.frontierBest[:need]
	tasks := s.frontier[:0]

	// groupOffset[g] est le nombre de candidats des groupes avant g : le
	// candidat i du groupe g occupe (groupOffset[g]+i)*NumRolls+r.
	// len(groups) <= NumRolls.
	var groupOffset [NumRolls]int
	off := 0
	for g, cands := range groups {
		groupOffset[g] = off
		off += len(cands)
	}

	// Ordre LPT sur la file entière, tous groupes confondus.
	for _, r := range rollsByCost {
		for g, cands := range groups {
			base := groupOffset[g] * NumRolls
			for i := range cands {
				if cands[i].Play.Result.isOver() {
					continue // la valeur terminale du sweep est déjà la bonne
				}
				tasks = append(tasks, rollTask{pos: &cands[i].Play.Result, roll: r, slot: base + i*NumRolls + r})
			}
		}
	}
	s.frontier = tasks
	if !s.runRollTasks(tasks, depth, state, owner, best) {
		return false
	}

	for g, cands := range groups {
		base := groupOffset[g] * NumRolls
		for i := range cands {
			if cands[i].Play.Result.isOver() {
				continue
			}
			var sum float64
			b := base + i*NumRolls
			for r := 0; r < NumRolls; r++ {
				sum += s.rolls[r].weight * best[b+r]
			}
			cands[i].Equity = -sum
		}
	}
	return true
}

// runRollTasks vide la file sur les ouvriers, qui piochent la tâche suivante
// par un compteur atomique (le schedule(dynamic,1) d'OpenMP, bien moins cher
// qu'un canal). Rien n'est partagé que les réseaux en lecture seule et state
// (une valeur) ; chaque résultat va dans son emplacement fixé d'avance.
func (s *Searcher) runRollTasks(tasks []rollTask, depth int, state *MatchState, owner CubeOwner, out []float64) bool {
	nw := len(s.workers)
	if nw > len(tasks) {
		nw = len(tasks)
	}
	if nw <= 0 {
		return true
	}
	var next atomic.Int64
	var wg sync.WaitGroup
	ok := make([]bool, nw)
	for w := 0; w < nw; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			worker := s.workers[w]
			ok[w] = true
			for {
				i := int(next.Add(1)) - 1
				if i >= len(tasks) {
					return
				}
				t := tasks[i]
				v, good := worker.oneRoll(t.pos, depth, 0, t.roll, state, owner, worker.candsAt(0))
				if !good {
					ok[w] = false
					return
				}
				out[t.slot] = v
			}
		}(w)
	}
	wg.Wait()
	for _, good := range ok {
		if !good {
			return false
		}
	}
	return true
}

// leafValue is the value of a position from its own turn's point of view:
// evaluate once, then nodeValue — cubeless money equity with no match state,
// 2×MWC−1 otherwise, either one through the cube model under UseCube.
func (s *Searcher) leafValue(pos *Position, state *MatchState, owner CubeOwner) float64 {
	var probs [NumOutputs]float32
	if !s.cache.lookup(pos, &probs) {
		encodeLegal(pos, &s.feat)
		_ = s.ev.Evaluate(s.feat[:], &probs)
		s.evals++
		s.cache.store(pos, &probs)
	} else {
		s.cacheHits++
	}
	return s.nodeValue(&probs, state, owner)
}

// sortByEquity orders candidates best first.
//
// Stable: on an exact tie generation order is kept, the rule gammonNet also
// follows since v1.3.0.
//
// Typed (sort.SliceStable allocates per call). Under sortInsertionMax a
// stable insertion sort; above it (a pruning pass on a double) the typed
// stable sort. A stable permutation is unique, so which ran never changes
// the order.
func sortByEquity(c []Candidate) {
	if len(c) < 2 {
		return
	}
	if len(c) <= sortInsertionMax {
		for i := 1; i < len(c); i++ {
			for j := i; j > 0 && c[j].Equity > c[j-1].Equity; j-- {
				c[j], c[j-1] = c[j-1], c[j]
			}
		}
		return
	}
	slices.SortStableFunc(c, func(a, b Candidate) int {
		switch {
		case a.Equity > b.Equity:
			return -1
		case b.Equity > a.Equity:
			return 1
		default:
			return 0
		}
	})
}

// sortInsertionMax is where sortByEquity stops inserting and starts merging;
// most real candidate lists hold fewer than thirty plays.
const sortInsertionMax = 48

// Counters reports what the last searches cost: big-network evaluations that
// actually ran, small-network (pruning) evaluations, and cache hits. A cache
// hit is an evaluation that did not happen.
//
// Workers are counted in (a root-only figure would shrink as cores are
// added). Call between searches, never during one.
func (s *Searcher) Counters() (evals, pruneEvals, cacheHits uint64) {
	evals, pruneEvals, cacheHits = s.evals, s.pruneEvals, s.cacheHits
	for _, w := range s.workers {
		e, pe, ch := w.Counters()
		evals, pruneEvals, cacheHits = evals+e, pruneEvals+pe, cacheHits+ch
	}
	return evals, pruneEvals, cacheHits
}

// BatchFill reports how many positions the fill passes evaluated, and how many
// lanes those positions would occupy at EvalBatchWidth. filled/slotted is the
// batch fill ratio; the shortfall is the work a batched kernel computes and
// discards. Workers are counted in, for the same reason Counters counts them.
func (s *Searcher) BatchFill() (filled, slotted uint64) {
	filled, slotted = s.batchFilled, s.batchSlotted
	for _, w := range s.workers {
		f, sl := w.BatchFill()
		filled, slotted = filled+f, slotted+sl
	}
	return filled, slotted
}

// CubeValuations reports how many distributions the cube model actually
// valued, workers counted in.
func (s *Searcher) CubeValuations() uint64 {
	n := s.cubeValuations
	for _, w := range s.workers {
		n += w.CubeValuations()
	}
	return n
}

// ResetCounters zeroes them, workers included.
func (s *Searcher) ResetCounters() {
	s.evals, s.pruneEvals, s.cacheHits = 0, 0, 0
	s.batchFilled, s.batchSlotted = 0, 0
	s.cubeValuations = 0
	for _, w := range s.workers {
		w.ResetCounters()
	}
}

// WithWorkers gives the searcher a pool to farm the root's roll loop out to.
// Each worker is an independent Searcher over the same read-only networks: its
// own scratch, its own generator, its own cache.
//
// The answer is unchanged, bit for bit: parallelism decides who computes each
// of the 21 terms, never the order they are summed in (no parallel reduction).
func (s *Searcher) WithWorkers(n int) *Searcher {
	if n <= 1 {
		s.workers = nil
		return s
	}
	if max := s.maxUsefulWorkers(); n > max {
		n = max
	}
	s.workers = make([]*Searcher, n)
	for i := range s.workers {
		s.workers[i] = newSearcherWithCache(s.cfg, s.net, s.prune, workerCacheLog2)
	}
	return s
}

// maxUsefulWorkers est le nombre de tâches de la plus grosse file de cette
// configuration : au-delà, un ouvrier n'ajoute que sa table (3,7 Mo). Deux
// files comptent : un niveau de rankPlays (Filter[depth] × 21) et la racine
// de probsAt (jusqu'à NumRolls² × Filter[Ply-1]).
func (s *Searcher) maxUsefulWorkers() int {
	widest := 1
	for depth := 1; depth <= s.cfg.Ply && depth < len(s.cfg.Filter); depth++ {
		if f := s.cfg.Filter[depth]; f > widest {
			widest = f
		}
	}
	tasks := NumRolls * widest
	if s.cfg.Ply >= 1 {
		rootDepth := s.cfg.Ply - 1
		rootWidth := 1
		if rootDepth >= 0 && rootDepth < len(s.cfg.Filter) && s.cfg.Filter[rootDepth] > rootWidth {
			rootWidth = s.cfg.Filter[rootDepth]
		}
		if probeTasks := NumRolls * NumRolls * rootWidth; probeTasks > tasks {
			tasks = probeTasks
		}
	}
	return tasks
}

// LiveWorkers is how many goroutines a FOREGROUND search (the only one
// running, with the user waiting) should spread its roll queue over: every
// core, or one when the depth is too shallow for the pool (~6 ms to build)
// to pay for itself (ADR-0011). The 2-ply floor is measured:
//
//	0 ply  286 µs serial, 352 µs with eight workers — there is no roll queue
//	       to spread at all, only barriers to pay for. This is the tier the
//	       board refreshes synchronously on every edit.
//	1 ply  3,5 ms serial, 1,8 ms with eight workers — a real speedup, and
//	       still a net LOSS once the 6 ms of pool cost the tier is on the hook
//	       for.
//	2 ply  250 ms serial, 55 ms with eight workers. Nothing to weigh.
//
// Never for a batch job: its parallelism is across positions, one serial
// searcher per goroutine (NewBatchSearcher); a pool on top would ask for
// NumCPU² goroutines.
func LiveWorkers(ply int) int {
	if ply < 2 {
		return 1
	}
	return runtime.NumCPU()
}

// Reconfigure points an existing searcher at a new configuration — same
// networks, scratch and cache — so a batch job reuses one searcher per
// goroutine instead of allocating 5.5 MB per position.
//
// The cache is deliberately kept: a hit returns exactly what a miss would
// compute, whatever the score, cube or depth (cache.go). The networks are
// never swapped: pruning stays on iff the searcher was built with a prune
// network. An invalid match state is refused, like NewSearcher.
func (s *Searcher) Reconfigure(cfg SearchConfig) error {
	if cfg.UseMatch && !cfg.Match.IsValid() {
		return fmt.Errorf("%w: match state %+v", ErrNotEvaluable, cfg.Match)
	}
	if cfg.Ply < 0 {
		cfg.Ply = 0
	}
	if cfg.Ply > MaxPly {
		cfg.Ply = MaxPly
	}
	s.cfg = cfg
	for _, w := range s.workers {
		w.cfg = cfg
	}
	return nil
}
