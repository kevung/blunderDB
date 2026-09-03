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
// the turn switched to the opponent. So the value of a play, TO THE PLAYER WHO
// MADE IT, is the NEGATION of the network's answer on the resulting position.
// Get that backwards and the engine plays its opponent's best move with total
// confidence — no crash, no warning, a perfectly plausible output. Every
// negation below is there for that reason.
//
// # The recursion
//
//	V(pos, 0) = cubeless money equity of pos, from pos.Turn's point of view
//	V(pos, k) = SUM over the 21 distinct rolls of
//	                w(roll) * max over plays of ( -V(play.Result, k-1) )
//
// A decision with known dice at depth k scores each play at -V(play.Result, k)
// — depth k, NOT k-1: the play itself is not one of the opponent rolls to
// enumerate. The reference carries a comment saying that mistake was made once.
//
// # What the leaves are worth
//
// V(pos, 0) above is the cubeless money equity only in the simplest
// configuration. With UseMatch every node is valued through the match equity
// table (2×MWC−1, the state swapped at every ply — ADR-0016), and with
// UseCube every LEAF goes through the cube model at the cube state that
// node's mover sees, mirrored at every ply alongside the state (ADR-0023).
// Both keep the one property the negations rely on: the value negates
// between sides. Valuing a gammonish move at 4-away/2-away correctly needs
// both — the table for the score, the cube for the double that makes the
// gammon worth the match — and no money test will ever say otherwise.
const (
	// MaxPly is the deepest search this engine will build. It is not a claim
	// that four plies are useful: upstream measured a whole extra ply at
	// +0.00022 equity per decision, inside the noise.
	MaxPly = 4

	// DefaultPruneK is the pruning width the reference documents as its
	// default: ×3.9 cheaper for an equity loss in the noise.
	DefaultPruneK = 12
)

//go:embed strehl-prune-32_v1.0.1_2026-08-27.bin
var embeddedPruneWeights []byte

var (
	pruneOnce sync.Once
	pruneNet  *Network
	pruneErr  error
)

// EmbeddedPruneNetwork returns the pruning network: 196→32→5, distilled from
// the big one and measured 92.5× cheaper per evaluation. It sorts the
// candidates so the big network only ever scores the survivors.
func EmbeddedPruneNetwork() (*Network, error) {
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
	// the mechanism off entirely. It is raised to Filter[depth] where that is
	// larger — pruning below the filter would silently search fewer candidates
	// than asked, and no test would see it.
	PruneK int

	// UseMatch and Match select the referential every node is valued in
	// (ADR-0016, gn_search.c's use_match/match): cubeless money equity when
	// UseMatch is false, 2×MWC−1 through Match otherwise. Match is the state
	// AS THE SEARCHER'S OWN CALLER SEES IT — Plays/BestPlay/Probs swap it at
	// every ply themselves, exactly as gn_search.c's swap_sides does; a
	// caller building SearchConfig never has to.
	//
	// A Searcher is already bound to one Ply for its whole life (NewSearcher
	// takes SearchConfig once); binding it to one Match is the same contract,
	// not a new one — a caller evaluating many different scores builds one
	// Searcher per score, exactly as it already builds one per Ply.
	UseMatch bool
	Match    MatchState

	// UseCube, CubeOwner and CubeX value every LEAF through the cube model
	// instead of cubeless (gn_search.c's use_cube, t34-videau-spec §8 step 2,
	// ADR-0023): Value at efficiency CubeX — the money model, or the match
	// redouble recursion when UseMatch is also set. CubeOwner is the cube AS
	// THE SEARCHER'S OWN CALLER SEES IT (the player on roll at the root); the
	// search mirrors it (Owned <-> Opponent) at every ply exactly where it
	// swaps the match state, and nowhere else. Two sign conventions in one
	// recursion is the failure mode this file's header warns about, which is
	// why state and owner travel together, in the same calls, always.
	//
	// No double/take/pass branches in the tree: the cube-aware value is
	// applied at the leaves and rides the same expectiminimax, which is what
	// the reference engines do (§8 records why). A finished game is worth its
	// stake whatever the cube — terminalValue is untouched, as in the C.
	//
	// What this buys is not a bolder or more sober search in the abstract; at
	// a match score it is the WHOLE gammon-go / gammon-save effect. Cubeless,
	// the trailer at 4-away/2-away prices their own gammons below the
	// leader's and plays 24/18 13/9 with the opening 6-4; with the cube they
	// double early, at 2 their gammon wins the match, and 8/2 6/2 comes first
	// — exactly gnubg's cubeful choice (docs/adr/0023).
	//
	// CUBEX IS FIXED AT THE ROOT WHILE CUBEOWNER IS MIRRORED, AND THAT IS A
	// KNOWN DIVERGENCE, NOT AN OVERSIGHT (#192/C.5, ADR-0029).
	// DefaultEfficiency returns a coefficient PER CUBE STATE, each fitted
	// against a different column of gammonNet's exact two-sided table, so a
	// leaf whose owner has been mirrored is priced with the coefficient
	// fitted for the OTHER branch — one leaf in two, whenever the root cube
	// is not centred. gn_search.c does exactly the same (`config->cube_x` at
	// :299 and :740, passed beside the mirrored owner), so correcting it HERE
	// would manufacture the port divergence cube.go's header forbids and turn
	// the cube gold red. The correction is gammonNet's to write (cube_x
	// indexed by the local owner; spec §4 and §8 step 2). Measured meanwhile
	// on 669 real analysed decisions: 0.005 normalised equity per leaf, and
	// 0 of 60 best moves changed. Read ADR-0029 before touching this.
	UseCube   bool
	CubeOwner CubeOwner
	CubeX     float64
}

// DefaultConfig returns the canonical configuration for a given depth: pruning
// at k=12 and the published move filter (0,1,3).
//
// The filter is not an optimisation, it is what makes the depth reachable at
// all. Measured on this build, a 2-ply decision from the opening costs 13 400
// big-network evaluations WITH the filter; without one it costs upwards of
// 760 000, because every one of the twelve survivors at every node is searched
// deeper. A config with a zero filter at ply 2 is not a slower search, it is an
// unusable one.
func DefaultConfig(ply int) SearchConfig {
	if ply < 0 {
		ply = 0
	}
	if ply > MaxPly {
		ply = MaxPly
	}
	return SearchConfig{
		Ply:    ply,
		PruneK: DefaultPruneK,
		Filter: [MaxPly + 1]int{0, 1, 3, 5, 5},
	}
}

// DepthLabel is the exact AnalysisDepth string a search at ply produces,
// after the same clamp to [0, MaxPly] DefaultConfig applies. A caller that
// decides whether a stored analysis is stale at some target depth (the
// gammonNet batch's staleness predicate, database/db_gammonnet_batch.go)
// must compare against this, not against the raw ply it was asked for —
// asking for ply 9 and getting a MaxPly search back must not read as
// "stale forever" against a target that was silently clamped the same way.
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

	// batchFilled and batchSlotted measure what a batched kernel would carry:
	// the positions a fill pass actually had to evaluate, against the lanes
	// those positions would occupy at EvalBatchWidth. Their ratio is the fill,
	// and the fill is what decides whether the twenty-one rolls need grouping
	// (#145, #146). Counted whether or not a batched kernel exists yet — the
	// figure has to be comparable across that change.
	batchFilled  uint64
	batchSlotted uint64

	// cubeValuations est le dénominateur COMPTÉ du poste videau : le nombre
	// de distributions réellement valuées par le modèle de videau, incrémenté
	// là où nodeValue appelle Value — donc jamais sur un nœud terminal, et
	// jamais sous UseCube éteint. gammonNet a ajouté le même compteur
	// (gn_search_cube_valuations) en découvrant que sa mesure d'entrée
	// supposait « un nœud évalué porte une valuation », ce qui est faux dans
	// les deux sens. Une mesure en ns par valuation qui divise par un nombre
	// supposé mesure la supposition.
	cubeValuations uint64

	// workers are independent searchers the root farms its roll loop out to.
	// Each owns its scratch, its generator and its cache; nothing is shared but
	// the read-only networks.
	workers []*Searcher

	// La file d'un niveau aplati et ses résultats (deepenLevel). Elles vivent
	// avec le chercheur, comme le reste du brouillon : la racine les remplit
	// une fois par décision, et une allocation par décision suffirait à faire
	// travailler le ramasse-miettes pendant que les ouvriers calculent.
	// frontierBest[i*NumRolls+r] est la valeur du lancer r pour le candidat i
	// — un emplacement par tâche, fixé avant que la file ne démarre, jamais
	// partagé entre deux ouvriers.
	frontier     []rollTask
	frontierBest []float64

	gen   [MaxPly + 2]*Generator
	plays [MaxPly + 2][]Play
	cands [MaxPly + 2][]Candidate
	feat  [NumFeatures]float32

	// best is the candidate buffer the two entry points that do not take
	// one rank into: BestPlay, and evaluateMoves at the domain edge. A
	// Candidate is 80 octets and MaxPlays is 2048, so allocating one per
	// call meant 164 Ko allocated and zeroed for every decision — for a
	// caller that then reads the first entry and, in the batch job, a
	// handful more. It belongs with the rest of the searcher's scratch,
	// which already holds six buffers of exactly this size — but allocated
	// on first use, not at construction: a worker built by WithWorkers only
	// ever ranks into its caller's buffer and would never touch this one.
	best []Candidate

	// Le brouillon du lot que shallowFill remplit. Il vit ici, dimensionné
	// une fois avec le Searcher, parce que shallowFill est le chemin le plus
	// chaud du moteur : un lot alloué par nœud coûterait 6 Ko à chacun des
	// milliers de nœuds d'une décision. batchOf[l] dit à quel candidat la
	// voie l appartient — le lot ne contient que les survivants, pas les
	// positions terminales ni les hits de cache, donc les voies ne suivent
	// pas les indices des candidats.
	batchFeat  [EvalBatchWidth][NumFeatures]float32
	batchProbs [EvalBatchWidth][NumOutputs]float32
	batchOf    [EvalBatchWidth]int
}

// NewSearcher builds a searcher over the embedded networks. Refused, never
// degraded (ADR-0016): an UseMatch config with a Match that IsValid() rejects
// is an error here, not a silent fall-back to money — gn_search_config_match's
// own comment names exactly this trap.
func NewSearcher(cfg SearchConfig) (*Searcher, error) {
	if cfg.UseMatch && !cfg.Match.IsValid() {
		return nil, fmt.Errorf("%w: match state %+v", ErrNotEvaluable, cfg.Match)
	}
	net, err := Embedded()
	if err != nil {
		return nil, err
	}
	var pn *Network
	if cfg.PruneK > 0 {
		if pn, err = EmbeddedPruneNetwork(); err != nil {
			return nil, err
		}
	}
	return NewSearcherWith(cfg, net, pn), nil
}

// NewSearcherWith builds a searcher over explicit networks. prune may be nil,
// which turns pruning off whatever PruneK says.
func NewSearcherWith(cfg SearchConfig, net, prune *Network) *Searcher {
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
		cache: newEvalCache(defaultCacheLog2),
		rolls: buildRolls(),
	}
	if prune != nil {
		s.pruneEv = NewEvaluator(prune)
	}
	return s
}

// playsAt et candsAt sont les brouillons du niveau level, alloués au premier
// usage. Les six niveaux étaient alloués d'avance, ce qui coûtait 1,6 Mo par
// chercheur ; depuis que la recherche de fond se donne un ouvrier par cœur,
// ce sont dix-sept chercheurs à construire par décision, et une décision à
// 2 ply n'en descend que trois. Le reste était payé et jamais lu.
//
// Aucun verrou : un Searcher appartient à une goroutine, comme tout le reste
// de son brouillon.
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

// genAt est le générateur de coups du niveau level, lui aussi alloué au
// premier usage. Un Generator pèse 166 Ko à lui seul (ses deux demi-niveaux
// de 2 048 entrées), soit près d'un mégaoctet par chercheur pour les six
// niveaux — la moitié du coût de construction d'un chercheur, et le seul
// poste qui comptait vraiment une fois le pool d'ouvriers branché.
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

// matchState is s.cfg.Match as a pointer, or nil under money valuation — the
// form every recursive entry point below threads instead of re-reading
// s.cfg.UseMatch at each node.
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
// level indexes the scratch buffers; it is the recursion's nesting, not the
// search depth. state is the match state AS pos's OWN mover sees it, or nil
// under money valuation — the same on every call here, since every play
// generated shares pos's mover; owner is the cube as that same mover sees it
// (read only under UseCube). theirs (state, swapped once) and theirOwner
// (owner, mirrored once) are what value the results and what the deep pass
// hands to the position on the other side of each play — gn_search.c's
// rank_plays_finish/rank_plays_deepen both derive them exactly this way,
// from the SAME unswapped pair, never from one another's swap.
func (s *Searcher) rankPlays(pos *Position, d1, d2, depth, level int, state *MatchState, owner CubeOwner, out []Candidate) int {
	if level >= len(s.plays) {
		return -1
	}
	plays := s.playsAt(level)
	count := s.genAt(level).LegalPlays(pos, d1, d2, plays)
	if count <= 0 {
		return count // 0 is a dance, -1 a refusal
	}
	written := count
	if written > len(out) {
		// The reference truncates here, keeping the first max_out plays in
		// generation order. Refusing is the same choice it makes everywhere
		// else, and a silently short candidate list is exactly what it warns
		// about: indistinguishable from a position with fewer options.
		return -1
	}
	for i := 0; i < written; i++ {
		out[i].Play = plays[i]
	}

	theirs := swapMatchState(state)
	theirOwner := owner.Mirror()

	// Phase one: the small network sorts, and only the survivors go on.
	if keep := s.pruneKeep(depth); keep > 0 && written > keep {
		s.shallowFill(s.pruneEv, out[:written], false)
		s.valueSweep(out[:written], theirs, theirOwner)
		sortByEquity(out[:written])
		written = keep
	}

	// Phase two: the big network scores what survived. It overwrites the small
	// network's probabilities — five plausible numbers from the wrong network
	// must never reach a caller.
	s.shallowFill(s.ev, out[:written], true)
	s.valueSweep(out[:written], theirs, theirOwner)
	sortByEquity(out[:written])

	// Phase three: search the best few deeper.
	if depth <= 0 {
		return written
	}
	searched := written
	if f := s.cfg.Filter[depth]; f > 0 && f < searched {
		searched = f
	}
	// À la racine, tous les candidats à approfondir partent dans une seule
	// file (deepenLevel) : une barrière par décision au lieu d'une par
	// candidat, et trois fois plus de tâches à répartir. Ailleurs — et sans
	// ouvriers — la boucle sérielle, terme pour terme identique.
	if level == 0 && len(s.workers) > 0 {
		if !s.deepenLevel(out[:searched], depth, theirs, theirOwner) {
			return -1
		}
	} else {
		for i := 0; i < searched; i++ {
			if out[i].Play.Result.isOver() {
				continue // keeps the exact terminal value from the sweep
			}
			v, ok := s.positionEquity(&out[i].Play.Result, depth, level+1, false, theirs, theirOwner)
			if !ok {
				return -1
			}
			out[i].Equity = -v
		}
	}
	sortByEquity(out[:searched])
	return written
}

// swapMatchState is state seen from the other side of the table, or nil when
// state is nil — the one place rankPlays computes gn_search.c's swap_sides,
// reused for both the value sweep and the deep pass so the two can never
// drift apart on which state a result is valued in.
func swapMatchState(state *MatchState) *MatchState {
	if state == nil {
		return nil
	}
	swapped := state.Swap()
	return &swapped
}

// shallowFill writes each candidate's resulting distribution. useCache is false
// for the pruning pass: letting the small network read or write the cache would
// make its ordering depend on evaluation history, which is the one way a cache
// could start changing results.
//
// C'est ici que la recherche alimente le noyau groupé (#146, ADR-0024). Les
// candidats d'un même appel sont les coups d'un même lancer depuis une même
// position — des frères, dont l'union des entrées actives est petite (~32 sur
// 196), soit exactement le lot sur lequel le noyau est au mieux : 16,9 µs par
// position contre 39 µs sur des plateaux sans rapport, et 505 µs en scalaire.
//
// La passe se fait en deux temps. D'abord le tri : les positions terminales,
// les hits de cache et les encodages refusés n'ont pas besoin du réseau et
// sortent du lot — leur traitement est inchangé. Ensuite les survivants
// partent par tranches de EvalBatchWidth ; la convention du lot partiel
// appartient au noyau (les voies au-delà de n dupliquent la position n-1), et
// l'appelant ne raisonne donc jamais sur la queue.
//
// Une nuance sur le cache, qui ne change aucun résultat : la version position
// par position rangeait le candidat i avant de chercher le candidat i+1, si
// bien qu'un rangement pouvait évincer, dans cette table à adressage direct,
// une entrée que le candidat suivant aurait trouvée. Le lot cherche les huit
// avant de ranger les huit, donc il peut rendre quelques hits de plus. Un hit
// rend les bits qu'un calcul aurait rendus (cache.go) : seuls les compteurs
// bougent, jamais l'évaluation. Deux candidats d'un même appel ne peuvent pas
// être la même position — moves_gen déduplique par plateau résultant — donc
// aucun doublon ne s'évalue deux fois dans un même lot.
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
		// encodeLegal plutôt qu'Encode : une position produite par la
		// génération est légale par construction, et la validation était la
		// moitié du coût de l'encodage (#150). Elle n'a donc plus de branche
		// d'échec — les points d'entrée publics valident une fois, à l'entrée.
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
	// Les mêmes deux compteurs qu'avant, et la même mesure : les positions
	// qu'il a fallu évaluer, contre les voies qu'elles occupent. La différence
	// est qu'ils ne simulent plus le lot, ils le décrivent — les tranches
	// envoyées ci-dessus sont pleines sauf la dernière, donc leur total de
	// voies est exactement batchSlots(filled).
	s.batchFilled += uint64(filled)
	s.batchSlotted += uint64(batchSlots(filled))
}

// flushBatch évalue les n premières voies du brouillon et redistribue les
// résultats dans les candidats que batchOf désigne, en tenant à jour les
// mêmes compteurs qu'une évaluation unitaire : evals et le rangement en cache
// pour la passe grand réseau, pruneEvals pour l'élagage. Le petit réseau passe
// par ce chemin comme le grand, et n'a toujours ni lecture ni écriture du
// cache (cache.go) : son ordre ne doit dépendre d'aucun historique.
//
// Le repli scalaire est là par prudence, et inatteignable en pratique :
// EvaluateBatch ne refuse que ce qu'Evaluate refuse aussi (une largeur d'entrée
// qui n'est pas celle du réseau), et un sélecteur de noyau invalide a déjà fait
// échouer Load. S'il se déclenche, il reproduit exactement l'ancien
// comportement plutôt que d'inventer une valeur.
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
// who made the play — hence the negation. state and owner are theirs: the
// match state and the cube as the RESULTING position's own mover sees them
// (rankPlays already swapped and mirrored them), or nil/unused under money
// cubeless valuation.
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

// nodeValue is the value of one evaluated node from its own mover's point of
// view: valueFromProbs (cubeless money, or 2×MWC−1) unless UseCube, in which
// case Value at owner — the cube as THAT node's mover sees it — on the very
// same scale, which is what lets the two valuations share one recursion. A
// failure (only reachable with a state NewSearcher already refuses) values
// the node as 0, the same choice valueFromProbs makes for its own.
//
// gn_search.c's node_value also reads the exact two-sided table for money
// leaves inside its domain; this port never does — blunderDB carries no such
// table in this package (engine/race has its own, for its own regime) — and
// the search gold is produced with no shared table loaded, so the two agree
// on the model path and the divergence is a documented one, not a drift.
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
func (s *Searcher) positionEquity(pos *Position, depth, level int, parallel bool, state *MatchState, owner CubeOwner) (float64, bool) {
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

	// The twenty-one rolls are independent, so the root farms them out. The
	// weighted sum is still accumulated afterwards in ascending roll index
	// order, in float64 — the parallelism changes who computes each term, never
	// the order they are added in, so the answer is bit-identical to the serial
	// one. A parallel reduction would not be.
	var best [NumRolls]float64
	if parallel && len(s.workers) > 0 {
		if !s.rollsInParallel(pos, depth, state, owner, &best) {
			return 0, false
		}
	} else {
		for r := 0; r < NumRolls; r++ {
			v, ok := s.oneRoll(pos, depth, level, r, state, owner, cands)
			if !ok {
				return 0, false
			}
			best[r] = v
		}
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
	// No legal play: the turn passes. Not an error. The passed position's own
	// mover is the opponent, so its state is state, swapped, and its cube
	// owner, mirrored.
	passed := *pos
	passed.swapTurn()
	v, ok := s.positionEquity(&passed, depth-1, level+1, false, swapMatchState(state), owner.Mirror())
	if !ok {
		return 0, false
	}
	return -v, true
}

// rollTask est un lancer à évaluer depuis une position, et l'emplacement où
// en ranger la valeur. L'emplacement est FIXE, calculé avant que la file ne
// démarre : c'est ce qui rend l'ordonnancement libre de choisir qui calcule
// quoi sans jamais toucher à l'ordre dans lequel les termes seront additionnés.
type rollTask struct {
	pos  *Position
	roll int // index du lancer dans s.rolls, 0..20
	slot int // index dans le tableau de résultats du niveau
}

// rollsByCost est l'ordre LPT (Longest Processing Time first) des 21 lancers :
// coût décroissant, ce qui borne le makespan à 4/3 − 1/(3m) de l'optimum
// (Graham 1969) au lieu de laisser la tâche la plus longue tomber en dernier.
//
// Le coût est mesuré, pas supposé, et la mesure contredit le proxy « les
// doubles d'abord » que la littérature suggère. TestProbeRollCost, sur 24
// positions du corpus à 2 ply, donne le nombre d'évaluations que déclenche le
// sous-arbre de chaque lancer :
//
//	2-6 17 256   3-6 17 256   2-3 17 184   3-4 17 160   1-4 16 920
//	2-5 16 896   4-5 16 704   5-6 16 584   1-5 16 536   1-2 14 928
//	2-2 14 400   2-4 13 056   4-4 12 960   4-6 12 888   1-3 12 672
//	1-6 12 456   3-5 12 120   5-5 11 976   3-3 11 928   1-1 11 736
//	6-6 11 232
//
// Les doubles sont parmi les MOINS chers, à l'inverse de ce qu'on attend de
// leurs 4 demi-coups : ils génèrent bien plus de coups légaux (1 800 pour 2-2
// contre 168 pour 5-6), mais l'élagage n'en garde que douze, et la position
// qu'ils laissent est plus contrainte, donc son sous-arbre est plus étroit.
// Seule la passe du petit réseau paie la largeur, et elle est bon marché.
// L'écart total n'est que de 1,54× en évaluations et 1,30× en temps — ce qui
// dit aussi que l'ordre LPT ne peut pas rendre grand-chose ici, et la mesure
// le confirme : l'essentiel du gain vient de l'aplatissement du niveau, pas
// du tri.
//
// Le nombre d'évaluations est retenu plutôt que le temps : il est déterministe
// et indépendant de la machine, là où le classement par temps sur un portable
// à budget thermique change d'un tour à l'autre. Les deux classements se
// recoupent d'ailleurs largement.
var rollsByCost = [NumRolls]int{10, 14, 7, 12, 3, 9, 16, 19, 4, 1, 6, 8, 15, 17, 2, 5, 13, 18, 11, 0, 20}

// deepenLevel approfondit d'un coup TOUS les candidats d'un niveau de la
// racine, au lieu d'en approfondir un, d'attendre, puis de passer au suivant.
//
// C'est le point trois de la fiche F4. Avant, chaque candidat approfondi
// ouvrait sa propre file de 21 lancers et sa propre barrière : à 2 ply, trois
// barrières de 21 tâches pour 8 ouvriers, soit trois fois ⌈21/8⌉ = 9 tours
// pour 7,875 tours de travail — 14 % perdus en quantification, avant même de
// compter le déséquilibre entre lancers. En une seule file de 3 × 21 = 63
// tâches, ⌈63/8⌉ = 8 tours : 1,6 % de quantification, et une seule barrière
// par décision.
//
// Le résultat ne bouge pas d'un bit. La somme pondérée reste sérielle, par
// candidat, en index de lancer croissant, en float64 (voir positionEquity,
// dont ceci est l'exact équivalent aplati) ; le parallélisme choisit qui
// calcule chaque terme, jamais l'ordre où ils s'ajoutent.
//
// state et owner sont ceux de la position RÉSULTANTE — rankPlays les a déjà
// échangés et miroités —, les mêmes pour tous les candidats du niveau puisque
// tous naissent du même coup de dés depuis la même position.
func (s *Searcher) deepenLevel(cands []Candidate, depth int, state *MatchState, owner CubeOwner) bool {
	need := len(cands) * NumRolls
	if cap(s.frontierBest) < need {
		s.frontierBest = make([]float64, need)
		s.frontier = make([]rollTask, 0, need)
	}
	best := s.frontierBest[:need]
	tasks := s.frontier[:0]

	// Ordre LPT : lancer par lancer, du plus cher au moins cher, tous
	// candidats confondus. Les copies du lancer le plus lourd partent donc en
	// premier, ce qui est exactement ce que la borne de Graham demande.
	for _, r := range rollsByCost {
		for i := range cands {
			if cands[i].Play.Result.isOver() {
				continue // la valeur terminale du sweep est déjà la bonne
			}
			tasks = append(tasks, rollTask{pos: &cands[i].Play.Result, roll: r, slot: i*NumRolls + r})
		}
	}
	s.frontier = tasks
	if !s.runRollTasks(tasks, depth, state, owner, best) {
		return false
	}

	for i := range cands {
		if cands[i].Play.Result.isOver() {
			continue
		}
		var sum float64
		base := i * NumRolls
		for r := 0; r < NumRolls; r++ {
			sum += s.rolls[r].weight * best[base+r]
		}
		cands[i].Equity = -sum
	}
	return true
}

// runRollTasks vide la file sur les ouvriers. Chaque ouvrier pioche la tâche
// suivante par un compteur atomique — l'équivalent du schedule(dynamic,1)
// d'OpenMP, deux ou trois ordres de grandeur moins cher qu'un canal — au lieu
// du tourniquet statique r += nw d'avant, qui figeait la répartition avant de
// savoir ce que chaque tâche coûterait.
//
// Chaque ouvrier travaille sur son propre brouillon, son propre générateur et
// son propre cache : rien n'est partagé que les réseaux, en lecture seule, et
// state (MatchState est une valeur, Swap ne mute jamais sur place). Les
// résultats vont dans des emplacements dédiés que la file a fixés d'avance,
// donc aucun accès concurrent à un même mot.
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

// rollsInParallel est le cas à une seule position de la même file : les 21
// lancers d'une position, ordonnés LPT, piochés par compteur atomique. Il
// reste le chemin des appelants qui entrent par positionEquity plutôt que par
// la racine de rankPlays.
func (s *Searcher) rollsInParallel(pos *Position, depth int, state *MatchState, owner CubeOwner, best *[NumRolls]float64) bool {
	var tasks [NumRolls]rollTask
	for i, r := range rollsByCost {
		tasks[i] = rollTask{pos: pos, roll: r, slot: r}
	}
	return s.runRollTasks(tasks[:], depth, state, owner, best[:])
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
// The reference uses qsort, which is NOT stable and whose implementation has
// changed across libc versions — on an exact tie the play it picks depends on
// the C library. A stable sort here is deterministic, which is what a gold file
// needs; where two equities tie exactly, the two engines may legitimately pick
// different plays, and the comparison has to allow for that rather than pretend
// the tie is a disagreement.
//
// Typed, because the reflective one was not free: sort.SliceStable builds a
// reflect swapper and two closures per call, and moves 80-octet candidates
// through them. Three sorts per node, ~1 400 nodes, and the allocation
// counter of a single 2-ply decision read 7 432 — nearly all of them here.
//
// Two shapes, one order. Under sortInsertionMax the insertion sort below is
// stable by construction (it only ever swaps a strictly better candidate
// past a worse one, never past an equal one) and is what a real node runs:
// a search node ranks a few dozen plays. Above it — the pruning pass on a
// double, which can face hundreds — the standard library's typed stable sort
// takes over, so the cost stays O(n log² n) rather than quadratic on
// 80-octet moves. A stable sort's output permutation is unique, so which of
// the two ran can never change the order.
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

// sortInsertionMax is where sortByEquity stops inserting and starts merging.
// Measured on the candidate lists a 2-ply search actually produces: the vast
// majority hold fewer than thirty plays, and an insertion sort beats a merge
// there by a wide margin on 80-octet elements.
const sortInsertionMax = 48

// Counters reports what the last searches cost: big-network evaluations that
// actually ran, small-network (pruning) evaluations, and cache hits. A cache
// hit is an evaluation that did not happen.
//
// The workers are counted in. They are where a parallel search does most of
// its work, so a root-only figure would report a fraction of the cost and
// shrink as cores are added — the opposite of what a cost probe is for. It is
// safe to read them here because Counters is called between searches, never
// during one.
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
// valued — the counted denominator of the cube post, workers counted in for
// the same reason Counters counts them.
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
// The answer is unchanged, bit for bit. Parallelism decides who computes each
// of the twenty-one terms, never the order they are summed in — a parallel
// reduction would change the last bit, and this deliberately is not one.
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
		s.workers[i] = NewSearcherWith(s.cfg, s.net, s.prune)
	}
	return s
}

// maxUsefulWorkers est le nombre de tâches que le plus gros niveau de cette
// configuration peut offrir : au-delà, un ouvrier de plus ne fait qu'ajouter
// sa table d'évaluation (3,7 Mo) et une goroutine qui ne pioche jamais rien.
//
// Le plafond était NumRolls, ce qui était juste tant qu'un candidat
// approfondi ouvrait sa propre file de 21 lancers. Depuis l'aplatissement
// (deepenLevel), un niveau porte Filter[depth] × 21 tâches — 63 à la
// configuration canonique 2 ply —, et brider à 21 laisserait les machines à
// plus de vingt cœurs sur la table.
func (s *Searcher) maxUsefulWorkers() int {
	widest := 1
	for depth := 1; depth <= s.cfg.Ply && depth < len(s.cfg.Filter); depth++ {
		if f := s.cfg.Filter[depth]; f > widest {
			widest = f
		}
	}
	return NumRolls * widest
}

// LiveWorkers is how many goroutines a FOREGROUND search — one the user is
// waiting on, and the only search running — should spread its roll queue
// over: every core the machine has, or one when the depth is too shallow for
// the pool to pay for itself.
//
// ADR-0011 calls intra-search parallelism a requirement rather than an
// optimisation ("without it the interactive promise does not hold"), and yet
// nothing in production called WithWorkers until #148: the panel ran a
// 2-ply decision on one core. This is the function that closes that gap, and
// the ply floor is the whole of its policy.
//
// The 2-ply floor is measured, not assumed. Building the pool costs about
// 6 ms (eight searchers to allocate), and that is the whole of the argument:
//
//	0 ply  286 µs serial, 352 µs with eight workers — there is no roll queue
//	       to spread at all, only barriers to pay for. This is the tier the
//	       board refreshes synchronously on every edit.
//	1 ply  3,5 ms serial, 1,8 ms with eight workers — a real speedup, and
//	       still a net LOSS once the 6 ms of pool cost the tier is on the hook
//	       for.
//	2 ply  250 ms serial, 55 ms with eight workers. Nothing to weigh.
//
// It is deliberately NOT what a batch job should use. There, the parallelism
// is across positions (#147): a pool per position on top of that would
// oversubscribe the cores and multiply the scratch by the number of workers
// for nothing. That path keeps its own serial searcher per goroutine
// (NewBatchSearcher, EvaluatePositionWith) and never comes through here.
func LiveWorkers(ply int) int {
	if ply < 2 {
		return 1
	}
	return runtime.NumCPU()
}

// Reconfigure points an existing searcher at a new configuration — the same
// networks, the same scratch buffers, the same cache, a different question.
// It is what makes a Searcher reusable across positions: a batch job builds
// one per goroutine (NewBatchSearcher) and re-aims it at each position's own
// referential and cube state instead of allocating a fresh 5.5 MB searcher
// per position.
//
// The cache is DELIBERATELY kept across the change. Its key is the whole
// position and its value is the network's own output for that position; a hit
// returns exactly what a miss would have computed, whatever the score, the
// cube or the depth the search arrived with (cache.go's own contract). So
// carrying it from one position to the next is free and licit — only the hit
// rate moves, never a bit of the answer.
//
// The networks are never swapped: pruning stays on exactly when the searcher
// was built with a prune network, whatever the new PruneK says — the same
// rule NewSearcherWith states. Refused, never degraded, on an invalid match
// state, exactly like NewSearcher.
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
