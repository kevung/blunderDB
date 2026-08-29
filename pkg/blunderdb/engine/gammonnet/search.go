// SPDX-License-Identifier: MIT

package gammonnet

import (
	_ "embed"
	"fmt"
	"sort"
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
// # What this tranche does not do
//
// The search is CUBELESS and MONEY-only. Valuing nodes through the match equity
// table is what makes a gammonish move worth what it is actually worth at
// 4-away/2-away, and no money test will ever say otherwise — it arrives with
// the MET work, together with the cube decision.
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

	// workers are independent searchers the root farms its roll loop out to.
	// Each owns its scratch, its generator and its cache; nothing is shared but
	// the read-only networks.
	workers []*Searcher

	gen   [MaxPly + 2]Generator
	plays [MaxPly + 2][]Play
	cands [MaxPly + 2][]Candidate
	feat  [NumFeatures]float32
}

// NewSearcher builds a searcher over the embedded networks.
func NewSearcher(cfg SearchConfig) (*Searcher, error) {
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

// Plays scores every legal play for the given dice and writes them into out,
// best first. It returns how many there are; 0 means a dance.
func (s *Searcher) Plays(pos *Position, d1, d2 int, out []Candidate) (int, error) {
	if !pos.Valid() {
		return 0, fmt.Errorf("gammonnet: position is not structurally valid")
	}
	if d1 < 1 || d1 > 6 || d2 < 1 || d2 > 6 {
		return 0, fmt.Errorf("gammonnet: dice %d-%d out of range", d1, d2)
	}
	n := s.rankPlays(pos, d1, d2, s.cfg.Ply, 0, out)
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
// search depth.
func (s *Searcher) rankPlays(pos *Position, d1, d2, depth, level int, out []Candidate) int {
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

	// Phase one: the small network sorts, and only the survivors go on.
	if keep := s.pruneKeep(depth); keep > 0 && written > keep {
		s.shallowFill(s.pruneEv, out[:written], false)
		s.valueSweep(out[:written])
		sortByEquity(out[:written])
		written = keep
	}

	// Phase two: the big network scores what survived. It overwrites the small
	// network's probabilities — five plausible numbers from the wrong network
	// must never reach a caller.
	s.shallowFill(s.ev, out[:written], true)
	s.valueSweep(out[:written])
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
		v, ok := s.positionEquity(&out[i].Play.Result, depth, level+1, level == 0)
		if !ok {
			return -1
		}
		out[i].Equity = -v
	}
	sortByEquity(out[:searched])
	return written
}

// shallowFill writes each candidate's resulting distribution. useCache is false
// for the pruning pass: letting the small network read or write the cache would
// make its ordering depend on evaluation history, which is the one way a cache
// could start changing results.
func (s *Searcher) shallowFill(ev *Evaluator, cands []Candidate, useCache bool) {
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
		if useCache {
			s.evals++
			s.cache.store(res, &cands[i].Probs)
		} else {
			s.pruneEvals++
		}
	}
}

// valueSweep turns each candidate's distribution into its value to the player
// who made the play — hence the negation.
func (s *Searcher) valueSweep(cands []Candidate) {
	for i := range cands {
		res := &cands[i].Play.Result
		if res.isOver() {
			cands[i].Equity = -terminalEquity(res)
			continue
		}
		cands[i].Equity = -float64(MoneyEquity(&cands[i].Probs))
	}
}

// positionEquity is the value of a position to the player on turn, at depth.
func (s *Searcher) positionEquity(pos *Position, depth, level int, parallel bool) (float64, bool) {
	if pos.isOver() {
		return terminalEquity(pos), true
	}
	if depth <= 0 {
		return s.leafValue(pos), true
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
		if !s.rollsInParallel(pos, depth, &best) {
			return 0, false
		}
	} else {
		for r := 0; r < NumRolls; r++ {
			v, ok := s.oneRoll(pos, depth, level, r, cands)
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

// oneRoll is the value of the best reply to one roll.
func (s *Searcher) oneRoll(pos *Position, depth, level, r int, cands []Candidate) (float64, bool) {
	roll := s.rolls[r]
	n := s.rankPlays(pos, int(roll.d1), int(roll.d2), depth-1, level, cands)
	if n < 0 {
		return 0, false
	}
	if n > 0 {
		return cands[0].Equity, true
	}
	// No legal play: the turn passes. Not an error.
	passed := *pos
	passed.swapTurn()
	v, ok := s.positionEquity(&passed, depth-1, level+1, false)
	if !ok {
		return 0, false
	}
	return -v, true
}

// rollsInParallel spreads the twenty-one rolls over the workers. Each worker
// runs on its own scratch and its own cache, so nothing needs a lock.
func (s *Searcher) rollsInParallel(pos *Position, depth int, best *[NumRolls]float64) bool {
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
				v, good := worker.oneRoll(pos, depth, 0, r, worker.cands[0])
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

// leafValue is the cubeless money equity of a position, from its own turn's
// point of view.
func (s *Searcher) leafValue(pos *Position) float64 {
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
	return float64(MoneyEquity(&probs))
}

// sortByEquity orders candidates best first.
//
// The reference uses qsort, which is NOT stable and whose implementation has
// changed across libc versions — on an exact tie the play it picks depends on
// the C library. A stable sort here is deterministic, which is what a gold file
// needs; where two equities tie exactly, the two engines may legitimately pick
// different plays, and the comparison has to allow for that rather than pretend
// the tie is a disagreement.
func sortByEquity(c []Candidate) {
	sort.SliceStable(c, func(i, j int) bool { return c[i].Equity > c[j].Equity })
}

// Counters reports what the last searches cost: big-network evaluations that
// actually ran, small-network (pruning) evaluations, and cache hits. A cache
// hit is an evaluation that did not happen.
func (s *Searcher) Counters() (evals, pruneEvals, cacheHits uint64) {
	return s.evals, s.pruneEvals, s.cacheHits
}

// ResetCounters zeroes them.
func (s *Searcher) ResetCounters() { s.evals, s.pruneEvals, s.cacheHits = 0, 0, 0 }

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
