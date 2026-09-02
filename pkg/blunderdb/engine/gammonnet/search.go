// SPDX-License-Identifier: MIT

package gammonnet

import (
	_ "embed"
	"fmt"
	"slices"
	"sync"
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

	// workers are independent searchers the root farms its roll loop out to.
	// Each owns its scratch, its generator and its cache; nothing is shared but
	// the read-only networks.
	workers []*Searcher

	gen   [MaxPly + 2]Generator
	plays [MaxPly + 2][]Play
	cands [MaxPly + 2][]Candidate
	feat  [NumFeatures]float32
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
	for i := range s.plays {
		s.plays[i] = make([]Play, MaxPlays)
		s.cands[i] = make([]Candidate, MaxPlays)
	}
	return s
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
	out := make([]Candidate, MaxPlays)
	n, err := s.Plays(pos, d1, d2, out)
	if err != nil || n == 0 {
		return Candidate{}, false, err
	}
	return out[0], true, nil
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
	plays := s.plays[level]
	count := s.gen[level].LegalPlays(pos, d1, d2, plays)
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
	for i := 0; i < searched; i++ {
		if out[i].Play.Result.isOver() {
			continue // keeps the exact terminal value from the sweep
		}
		v, ok := s.positionEquity(&out[i].Play.Result, depth, level+1, level == 0, theirs, theirOwner)
		if !ok {
			return -1
		}
		out[i].Equity = -v
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
func (s *Searcher) shallowFill(ev *Evaluator, cands []Candidate, useCache bool) {
	filled := 0
	defer func() {
		s.batchFilled += uint64(filled)
		s.batchSlotted += uint64(batchSlots(filled))
	}()
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
		if !Encode(res, &s.feat) {
			cands[i].Probs = [NumOutputs]float32{}
			continue
		}
		_ = ev.Evaluate(s.feat[:], &cands[i].Probs)
		filled++
		if useCache {
			s.evals++
			s.cache.store(res, &cands[i].Probs)
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
	cands := s.cands[level]

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

// rollsInParallel spreads the twenty-one rolls over the workers. Each worker
// runs on its own scratch and its own cache, so nothing needs a lock. state
// is read-only here (MatchState is a plain value; Swap never mutates it in
// place), so sharing one pointer across goroutines is safe.
func (s *Searcher) rollsInParallel(pos *Position, depth int, state *MatchState, owner CubeOwner, best *[NumRolls]float64) bool {
	nw := len(s.workers)
	var wg sync.WaitGroup
	ok := make([]bool, nw)
	for w := 0; w < nw; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			worker := s.workers[w]
			ok[w] = true
			for r := w; r < NumRolls; r += nw {
				v, good := worker.oneRoll(pos, depth, 0, r, state, owner, worker.cands[0])
				if !good {
					ok[w] = false
					return
				}
				best[r] = v
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
		if !Encode(pos, &s.feat) {
			return 0
		}
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

// ResetCounters zeroes them, workers included.
func (s *Searcher) ResetCounters() {
	s.evals, s.pruneEvals, s.cacheHits = 0, 0, 0
	s.batchFilled, s.batchSlotted = 0, 0
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
	if n > NumRolls {
		n = NumRolls
	}
	s.workers = make([]*Searcher, n)
	for i := range s.workers {
		s.workers[i] = NewSearcherWith(s.cfg, s.net, s.prune)
	}
	return s
}
