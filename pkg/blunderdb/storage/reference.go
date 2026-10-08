package storage

import (
	"container/heap"
	"math"
	"sort"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// The reference positions of ADR-0079: the positions of a filter whose lesson
// would recover the most of the winning chances its neighbouring errors lost.
// The constants are fixed there, before any proposal was read, and change only
// with a new ADR.
const (
	// ReferenceRadius is ρ: two candidates of a group within this many
	// checker-pips (engine.SimilarityDistance) are neighbours, and a proposal
	// keeps no two positions this close.
	ReferenceRadius = 12
	// ReferenceDefaultSize is how many positions a proposal holds unless told
	// otherwise; ReferenceMaxSize the most it holds, the study queue's bound.
	ReferenceDefaultSize = 20
	ReferenceMaxSize     = domain.MaxStudyQueue
	// ReferenceDiscount multiplies a candidate's gain once per flag it raises
	// (a close lesson, an unstable verdict).
	ReferenceDiscount = 0.5
)

// ReferenceRow is one priced error with what a proposal reads beyond the
// study plan: the board, the score of a cube decision and how clean the
// position's lesson is. A backend hands these to SuggestReferences.
type ReferenceRow struct {
	StudyPlanRow
	Vector engine.SimilarityVector
	// AwayOnRoll and AwayOpponent are the points each side still needs, seen
	// from the side on roll; only cube candidates are grouped by them.
	AwayOnRoll   int
	AwayOpponent int
	// GapMP is what the second-best option costs, in millipoints of the
	// stored equity; -1 when the analysis does not say.
	GapMP int64
	// Unstable: another depth or a rollout rules the decision otherwise.
	// RolledOut: a rollout stands behind the verdict.
	Unstable  bool
	RolledOut bool
}

// ReferenceRequest is what a proposal is asked for: the statistics filter,
// optionally narrowed to some matches, and how many positions it holds (0 is
// ReferenceDefaultSize).
type ReferenceRequest struct {
	Filter   StatsFilter `json:"filter"`
	MatchIDs []int64     `json:"matchIds,omitempty"`
	Size     int         `json:"size,omitempty"`
}

// ReferenceLesson reads how clean a position's lesson is (ADR-0079 §4): the
// cost of the second-best option in millipoints (-1 when fewer than two are
// priced), whether another depth or a rollout rules the decision the player
// faced otherwise, and whether a rollout stands behind it. costs are the
// options' costs as the difficulty reads them, in equity.
func ReferenceLesson(ana *domain.PositionAnalysis, kind, cubeAction string, costs []float64) (gapMP int64, unstable, rolledOut bool) {
	gapMP = -1
	if len(costs) >= 2 {
		sorted := append([]float64(nil), costs...)
		sort.Float64s(sorted)
		gapMP = int64(math.Round((sorted[1] - sorted[0]) * 1000))
	}
	if ana == nil {
		return gapMP, false, false
	}
	isRollout := func(depth string) bool { return strings.Contains(strings.ToLower(depth), "rollout") }
	if kind == "cube" {
		dca := ana.DoublingCubeAnalysis
		if dca == nil {
			return gapMP, false, false
		}
		response := engine.IsResponseCubeAction(cubeAction)
		ruling := func(best string) (bool, bool) {
			v, ok := engine.BestCubeVerdict(best)
			if response {
				return v.ShouldPass, ok
			}
			return v.ShouldDouble, ok
		}
		want, ok := ruling(dca.BestCubeAction)
		if !ok {
			return gapMP, false, false
		}
		rolledOut = isRollout(dca.AnalysisDepth)
		for _, other := range ana.AllCubeAnalyses {
			if got, ok := ruling(other.BestCubeAction); ok && got != want {
				unstable = true
			}
			rolledOut = rolledOut || isRollout(other.AnalysisDepth)
		}
		for _, r := range ana.Rollouts {
			if r.Kind != domain.RolloutKindCube {
				continue
			}
			rolledOut = true
			if got, ok := ruling(r.BestCubeAction); ok && got != want {
				unstable = true
			}
		}
		return gapMP, unstable, rolledOut
	}
	ca := ana.CheckerAnalysis
	if ca == nil || len(ca.Moves) == 0 {
		return gapMP, false, false
	}
	best := normaliseMove(ca.Moves[0].Move)
	rolledOut = isRollout(ca.Moves[0].AnalysisDepth)
	for _, r := range ana.Rollouts {
		if r.Kind != domain.RolloutKindMoves || len(r.Candidates) == 0 {
			continue
		}
		rolledOut = true
		if normaliseMove(r.Candidates[0].Move) != best {
			unstable = true
		}
	}
	return gapMP, unstable, rolledOut
}

// normaliseMove lets two writers' spellings of one play compare equal.
func normaliseMove(m string) string {
	return strings.ToLower(strings.Join(strings.Fields(m), ""))
}

// ReferenceSuggestion is one proposed position with the components of its
// reason (ADR-0079 §7); each client words them in its own language.
type ReferenceSuggestion struct {
	PositionID int64 `json:"PositionID"`
	// MatchID and Label name a match the position's worst error was played in.
	MatchID  int64  `json:"MatchID"`
	Label    string `json:"Label"`
	GameType string `json:"GameType"`
	Kind     string `json:"Kind"`
	Theme    string `json:"Theme"`
	// AwayOnRoll and AwayOpponent are the score of a cube reference; zero
	// for a checker one.
	AwayOnRoll   int `json:"AwayOnRoll"`
	AwayOpponent int `json:"AwayOpponent"`
	// Gain is the discounted recoverable MWC it covers when picked; Covered
	// the same sum before the discount, MWC fraction.
	Gain    float64 `json:"Gain"`
	Covered float64 `json:"Covered"`
	// Errors and Matches count the errors it stands for (its own included)
	// and the matches they were played in.
	Errors  int `json:"Errors"`
	Matches int `json:"Matches"`
	// Excess is the position's own recoverable MWC.
	Excess float64 `json:"Excess"`
	// FamilyErrors is the size of its family, against the filter's decisions.
	FamilyErrors int   `json:"FamilyErrors"`
	GapMP        int64 `json:"GapMP"`
	Close        bool  `json:"Close"`
	Unstable     bool  `json:"Unstable"`
	RolledOut    bool  `json:"RolledOut"`
}

// ReferenceSuggestions is a proposal for one filter.
type ReferenceSuggestions struct {
	NumDecisions int `json:"NumDecisions"`
	ThresholdMP  int `json:"ThresholdMP"`
	Radius       int `json:"Radius"`
	Size         int `json:"Size"`
	// Candidates counts the positions weighed; Handled those left out
	// because something already deals with them.
	Candidates int                   `json:"Candidates"`
	Handled    int                   `json:"Handled"`
	References []ReferenceSuggestion `json:"References"`
}

// IDs are the proposed positions, in the proposal's order.
func (r *ReferenceSuggestions) IDs() []int64 {
	ids := []int64{}
	if r == nil {
		return ids
	}
	for _, s := range r.References {
		ids = append(ids, s.PositionID)
	}
	return ids
}

// ClampReferenceSize reads a requested size: 0 (or less) is the default, and
// nothing exceeds ReferenceMaxSize.
func ClampReferenceSize(size int) int {
	switch {
	case size <= 0:
		return ReferenceDefaultSize
	case size > ReferenceMaxSize:
		return ReferenceMaxSize
	}
	return size
}

// referenceKey is a candidate's group: its family and, for the cube, its score.
type referenceKey struct {
	gameType, kind, theme string
	away0, away1          int
}

// similarityKey is a board laid out for repeated distances: the prefix sums
// engine.SimilarityDistance takes the differences of, computed once, and their
// totals over four segments — each side's home half (points 0-12) and outer
// half (13 to the bar). The distance is the sum of the prefix sums' absolute
// differences, so it is at least the sum of the segments' absolute
// differences: a lower bound read in four subtractions.
type similarityKey struct {
	cum          [52]int16
	seg          [4]int
	kind         string
	away0, away1 int
}

func newSimilarityKey(v *engine.SimilarityVector, kind string, away0, away1 int) similarityKey {
	k := similarityKey{kind: kind, away0: away0, away1: away1}
	cm, co := 0, 0
	for i := range 26 {
		cm += v.Mover[i]
		co += v.Opponent[i]
		k.cum[i], k.cum[26+i] = int16(cm), int16(co)
		half := i / 13
		k.seg[half] += cm
		k.seg[2+half] += co
	}
	return k
}

// within says whether two boards lie within radius checker-pips, most pairs
// stopping at the segments' bound.
func (a *similarityKey) within(b *similarityKey, radius int) bool {
	bound := 0
	for i := range a.seg {
		bound += absInt(a.seg[i] - b.seg[i])
	}
	if bound > radius {
		return false
	}
	d := 0
	for i := range a.cum {
		d += absInt(int(a.cum[i]) - int(b.cum[i]))
		if d > radius {
			return false
		}
	}
	return true
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

type referenceCandidate struct {
	key        similarityKey
	group      referenceKey
	s          ReferenceSuggestion
	excess     float64
	matches    []int64
	discount   float64
	neighbours []int32
	handled    bool
	removed    bool
	covered    bool
	worst      float64
}

// SuggestReferences proposes up to size reference positions among priced,
// themed errors (ADR-0079): a lazy greedy cover of the recoverable MWC of each
// group's neighbourhoods, the positions in handled counting as already picked.
// It is pure and shared by every backend.
func SuggestReferences(rows []ReferenceRow, handled map[int64]bool, numDecisions, thresholdMP, size int) *ReferenceSuggestions {
	size = ClampReferenceSize(size)
	out := &ReferenceSuggestions{NumDecisions: numDecisions, ThresholdMP: thresholdMP, Radius: ReferenceRadius,
		Size: size, References: []ReferenceSuggestion{}}
	cands, groups := gatherReferenceCandidates(rows, handled, thresholdMP)
	for _, c := range cands {
		if c.handled {
			out.Handled++
		} else {
			out.Candidates++
		}
	}
	for _, members := range groups {
		linkNeighbours(cands, members)
	}
	// What is already dealt with is already picked: it covers its neighbours
	// and keeps their near-duplicates out.
	var picked []*referenceCandidate
	for _, c := range cands {
		if c.handled {
			pick(cands, c)
			picked = append(picked, c)
		}
	}
	h := &gainHeap{cands: cands}
	for i, c := range cands {
		if !c.handled && !c.removed {
			h.items = append(h.items, gainItem{idx: int32(i), gain: marginalGain(cands, c)})
		}
	}
	heap.Init(h)
	for h.Len() > 0 && len(out.References) < size {
		top := heap.Pop(h).(gainItem)
		c := cands[top.idx]
		if c.removed || c.covered {
			continue
		}
		g := marginalGain(cands, c)
		if h.Len() > 0 && h.less(gainItem{idx: h.items[0].idx, gain: h.items[0].gain}, gainItem{idx: top.idx, gain: g}) {
			heap.Push(h, gainItem{idx: top.idx, gain: g})
			continue
		}
		if g <= 0 {
			break
		}
		if nearPicked(picked, c) {
			c.removed = true
			continue
		}
		s := c.s
		s.Gain = g
		s.Covered, s.Errors, s.Matches = coverage(cands, c)
		pick(cands, c)
		picked = append(picked, c)
		out.References = append(out.References, s)
	}
	return out
}

// gatherReferenceCandidates folds errors into one candidate per position and
// group, and returns the groups' members.
func gatherReferenceCandidates(rows []ReferenceRow, handled map[int64]bool, thresholdMP int) ([]*referenceCandidate, map[referenceKey][]int32) {
	type ck struct {
		group referenceKey
		id    int64
	}
	family := map[[3]string]int{}
	byKey := map[ck]int32{}
	var cands []*referenceCandidate
	groups := map[referenceKey][]int32{}
	for i := range rows {
		r := &rows[i]
		if r.Loss == nil || r.Difficulty == nil || r.Theme == RecurringThemeNone {
			continue
		}
		family[[3]string{r.GameType, r.Kind, r.Theme}]++
		g := referenceKey{gameType: r.GameType, kind: r.Kind, theme: r.Theme}
		if r.Kind == "cube" {
			g.away0, g.away1 = r.AwayOnRoll, r.AwayOpponent
		}
		excess := *r.Loss - *r.Difficulty
		k := ck{g, r.PositionID}
		idx, ok := byKey[k]
		if !ok {
			idx = int32(len(cands))
			byKey[k] = idx
			c := &referenceCandidate{group: g, handled: handled[r.PositionID], discount: 1,
				key: newSimilarityKey(&r.Vector, r.Kind, g.away0, g.away1), worst: excess}
			c.s = ReferenceSuggestion{PositionID: r.PositionID, MatchID: r.MatchID, Label: r.Label,
				GameType: r.GameType, Kind: r.Kind, Theme: r.Theme, AwayOnRoll: g.away0, AwayOpponent: g.away1,
				GapMP: r.GapMP, Unstable: r.Unstable, RolledOut: r.RolledOut}
			c.s.Close = r.GapMP >= 0 && 2*r.GapMP < int64(thresholdMP)
			if c.s.Close {
				c.discount *= ReferenceDiscount
			}
			if c.s.Unstable {
				c.discount *= ReferenceDiscount
			}
			cands = append(cands, c)
			groups[g] = append(groups[g], idx)
		}
		c := cands[idx]
		if excess > 0 {
			c.excess += excess
		}
		c.s.Errors++
		if excess > c.worst {
			c.worst, c.s.MatchID, c.s.Label = excess, r.MatchID, r.Label
		}
		if !containsID(c.matches, r.MatchID) {
			c.matches = append(c.matches, r.MatchID)
		}
	}
	for _, c := range cands {
		c.s.Excess = c.excess
		c.s.FamilyErrors = family[[3]string{c.s.GameType, c.s.Kind, c.s.Theme}]
	}
	return cands, groups
}

func containsID(ids []int64, id int64) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// linkNeighbours fills the neighbour lists of one group. Two neighbours'
// segment totals differ by at most the radius in each segment, so a member
// only meets the members of its own and the adjacent cells of a grid on the
// four totals, cells one radius wide: that grid is the neighbourhood index of
// ADR-0079 §8.
func linkNeighbours(cands []*referenceCandidate, members []int32) {
	const w = ReferenceRadius + 1
	type cell [4]int
	grid := map[cell][]int32{}
	for _, mi := range members {
		k := &cands[mi].key
		grid[cell{k.seg[0] / w, k.seg[1] / w, k.seg[2] / w, k.seg[3] / w}] = append(grid[cell{k.seg[0] / w, k.seg[1] / w, k.seg[2] / w, k.seg[3] / w}], mi)
	}
	var offsets []cell
	for a := -1; a <= 1; a++ {
		for b := -1; b <= 1; b++ {
			for c := -1; c <= 1; c++ {
				for d := -1; d <= 1; d++ {
					offsets = append(offsets, cell{a, b, c, d})
				}
			}
		}
	}
	for c, own := range grid {
		for _, off := range offsets {
			other, ok := grid[cell{c[0] + off[0], c[1] + off[1], c[2] + off[2], c[3] + off[3]}]
			if !ok {
				continue
			}
			for _, mi := range own {
				a := cands[mi]
				for _, mj := range other {
					// Each unordered pair once: the smaller index links both.
					if mj <= mi {
						continue
					}
					b := cands[mj]
					if a.key.within(&b.key, ReferenceRadius) {
						a.neighbours = append(a.neighbours, mj)
						b.neighbours = append(b.neighbours, mi)
					}
				}
			}
		}
	}
	// Map order must not reach the proposal: neighbour lists in index order.
	for _, mi := range members {
		n := cands[mi].neighbours
		sort.Slice(n, func(i, j int) bool { return n[i] < n[j] })
	}
}

// marginalGain is what picking c would still cover, discounted.
func marginalGain(cands []*referenceCandidate, c *referenceCandidate) float64 {
	sum := 0.0
	if !c.covered {
		sum = c.excess
	}
	for _, n := range c.neighbours {
		if x := cands[n]; !x.covered {
			sum += x.excess
		}
	}
	return c.discount * sum
}

// coverage sums the uncovered excess of c and its neighbours, with the errors
// and the distinct matches they hold.
func coverage(cands []*referenceCandidate, c *referenceCandidate) (float64, int, int) {
	var sum float64
	errs := 0
	matches := map[int64]bool{}
	add := func(x *referenceCandidate) {
		if x.covered {
			return
		}
		sum += x.excess
		errs += x.s.Errors
		for _, m := range x.matches {
			matches[m] = true
		}
	}
	add(c)
	for _, n := range c.neighbours {
		add(cands[n])
	}
	return sum, errs, len(matches)
}

// pick covers c and its neighbours and keeps them out of the proposal.
func pick(cands []*referenceCandidate, c *referenceCandidate) {
	c.covered, c.removed = true, true
	for _, n := range c.neighbours {
		cands[n].covered, cands[n].removed = true, true
	}
}

// nearPicked catches a near-duplicate of a picked or handled position from another
// group: the same kind of decision and, for the cube, the same score.
func nearPicked(picked []*referenceCandidate, c *referenceCandidate) bool {
	for _, p := range picked {
		if p.key.kind == c.key.kind && p.key.away0 == c.key.away0 && p.key.away1 == c.key.away1 &&
			p.key.within(&c.key, ReferenceRadius) {
			return true
		}
	}
	return false
}

type gainItem struct {
	idx  int32
	gain float64
}

// gainHeap orders candidates by gain, then a rollout behind the verdict, the
// larger own excess and the smaller position id: a total order, so both
// backends propose the same list.
type gainHeap struct {
	cands []*referenceCandidate
	items []gainItem
}

func (h *gainHeap) Len() int           { return len(h.items) }
func (h *gainHeap) Less(i, j int) bool { return h.less(h.items[i], h.items[j]) }
func (h *gainHeap) Swap(i, j int)      { h.items[i], h.items[j] = h.items[j], h.items[i] }
func (h *gainHeap) Push(x any)         { h.items = append(h.items, x.(gainItem)) }
func (h *gainHeap) Pop() any {
	n := len(h.items)
	x := h.items[n-1]
	h.items = h.items[:n-1]
	return x
}

func (h *gainHeap) less(a, b gainItem) bool {
	if a.gain != b.gain {
		return a.gain > b.gain
	}
	ca, cb := h.cands[a.idx], h.cands[b.idx]
	if ca.s.RolledOut != cb.s.RolledOut {
		return ca.s.RolledOut
	}
	if ca.excess != cb.excess {
		return ca.excess > cb.excess
	}
	return ca.s.PositionID < cb.s.PositionID
}
