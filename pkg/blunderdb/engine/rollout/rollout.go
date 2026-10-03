package rollout

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
)

// batchGames is how many games run between two looks at the stopping rule:
// one stratum of first rolls, so a stop never lands mid-stratum.
const batchGames = 36

// preselectPly is the least depth the candidates of a checker rollout are
// chosen at when none is named.
const preselectPly = 2

// z95 is the two-sided 95 % normal quantile.
const z95 = 1.959963984540054

// jsdFloor guards the JSD's denominator, as gnubg's check_jsds does.
const jsdFloor = 1e-8

// Kind says what was rolled out.
type Kind string

const (
	KindMoves Kind = "moves"
	KindCube  Kind = "cube"
)

// Stop says why a rollout ended.
type Stop string

const (
	StopMaxGames  Stop = "max_games"
	StopJSD       Stop = "jsd"
	StopCancelled Stop = "cancelled"
)

// The cube rollout's three candidates, by name.
const (
	NoDouble   = "No double"
	DoubleTake = "Double/Take"
	DoublePass = "Double/Pass"
)

// Estimate is what the rollout concluded about one candidate, in the
// position's output equity (ADR-0019): money points per unit of the
// position's cube, or normalised equity at a match score.
type Estimate struct {
	Equity float64 `json:"equity"`
	// StdErr is the standard error of Equity — the σ gnubg and XG print.
	StdErr float64 `json:"std_err"`
	// CI95 is the half-width of the 95 % confidence interval.
	CI95  float64 `json:"ci95"`
	Games int     `json:"games"`
	// JSD is the gap to the reference in standard deviations of the
	// difference: to the best play for a checker rollout; for the cube, see
	// CubeResult.
	JSD float64 `json:"jsd"`
	// Chances are the mean outcome, root's view, in [0,1], nested as
	// gammonnet's outputs are; not luck-corrected, and cubeful games end on
	// a pass, so they are not true cubeless chances.
	Chances [gammonnet.NumOutputs]float64 `json:"chances"`
}

// Candidate is one rolled-out play, or one cube action.
type Candidate struct {
	Move string `json:"move"`
	Estimate
}

// CubeResult reads a cube rollout. Double/Pass is exact (+1, the cube's own
// value) and is never rolled.
type CubeResult struct {
	NoDouble   Estimate `json:"no_double"`
	DoubleTake Estimate `json:"double_take"`
	DoublePass Estimate `json:"double_pass"`
	Action     string   `json:"action"`
	// JSDDouble separates No double from Double (the lesser of take and
	// pass); JSDTake separates Double/Take from Double/Pass.
	JSDDouble float64 `json:"jsd_double"`
	JSDTake   float64 `json:"jsd_take"`
}

// Result is a finished, stopped or cancelled rollout.
type Result struct {
	Kind          Kind        `json:"kind"`
	EngineVersion string      `json:"engine_version"`
	Settings      Settings    `json:"settings"`
	Signature     string      `json:"signature"`
	Candidates    []Candidate `json:"candidates"`
	Cube          *CubeResult `json:"cube,omitempty"`
	Games         int         `json:"games"`
	Stop          Stop        `json:"stop"`
	// CubefulBias is set whenever the number leans on the cube model — a
	// cube policy inside the games or a cubeful leaf: the ranking is
	// trustworthy, the absolute equity less so (docs/recherche/P8).
	CubefulBias bool `json:"cubeful_bias"`
	// ExactBearoff says the two-sided table was available to end games.
	ExactBearoff bool `json:"exact_bearoff"`
}

// Progress is reported after each batch of games.
type Progress struct {
	Games      int         `json:"games"`
	MaxGames   int         `json:"max_games"`
	Candidates []Candidate `json:"candidates"`
}

// Options are what a caller adds to the position and the settings.
type Options struct {
	// Moves names the plays to roll out, in blunderDB notation; empty rolls
	// the Settings.Candidates best at Settings.Ply. Ignored for a cube.
	Moves []string
	// Progress, when set, is called after every batch from the calling
	// goroutine.
	Progress func(Progress)
	// NoBearoffTable turns the exact bearoff off, for a rollout that must
	// not depend on what this machine has generated.
	NoBearoffTable bool
	// withoutLuck turns the variance reduction off, for the test that
	// shows it leaves the mean where it was.
	withoutLuck bool
}

// accumulator is a running mean and variance (Welford), fed in game order
// so the sum is the same whatever order the games finished in.
type accumulator struct {
	n       int
	mean    float64
	m2      float64
	chances [gammonnet.NumOutputs]float64
}

func (a *accumulator) add(o outcome) {
	a.n++
	d := o.value - a.mean
	a.mean += d / float64(a.n)
	a.m2 += d * (o.value - a.mean)
	for i := range a.chances {
		a.chances[i] += o.chances[i]
	}
}

// stdErr is the standard error of the mean, on the native scale.
func (a *accumulator) stdErr() float64 {
	if a.n < 2 {
		return 0
	}
	return math.Sqrt(a.m2 / float64(a.n-1) / float64(a.n))
}

// Run rolls out pos: its plays when dice are set, its cube decision
// otherwise. A cancelled ctx returns what the completed batches concluded,
// with ctx's error.
func Run(ctx context.Context, pos domain.Position, s Settings, opt Options) (*Result, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	gnPos, err := gammonnet.FromDomain(&pos)
	if err != nil {
		return nil, err
	}
	_, state, err := gammonnet.ConfigForPosition(&pos, s.Ply, 0)
	if err != nil {
		return nil, err
	}
	scale, ok := gammonnet.NewEquityScale(state)
	if !ok {
		return nil, fmt.Errorf("rollout: no equity referential at score %v", pos.Score)
	}

	t := &table{
		root:     gnPos.Turn,
		rootCube: 1 << uint(pos.Cube.Value),
		settings: s,
		jacoby:   state == nil && pos.HasJacoby != 0,
	}
	if state != nil {
		t.match = true
		t.away[t.root] = state.AwayOnRoll
		t.away[1-t.root] = state.AwayOpponent
		t.crawford = state.Crawford
	}
	t.withoutLuck = opt.withoutLuck
	if !opt.NoBearoffTable {
		t.bearoff = race.Resolve()
	}
	rootCube := cubeState{value: t.rootCube, owner: noOwner}
	switch gammonnet.CubeOwnerOf(&pos) {
	case gammonnet.CubeOwned:
		rootCube.owner = int(t.root)
	case gammonnet.CubeOpponent:
		rootCube.owner = int(1 - t.root)
	}

	res := &Result{
		EngineVersion: EngineVersion,
		Settings:      s,
		Signature:     s.Signature(),
		ExactBearoff:  t.bearoff != nil,
		CubefulBias:   s.Truncation > 0 || !t.crawford,
	}

	var branches []branch
	var names []string
	if hasDice(&pos) {
		res.Kind = KindMoves
		names, branches, err = moveBranches(&pos, s, opt.Moves, rootCube)
		if err != nil {
			return nil, err
		}
		if len(opt.Moves) > 0 {
			// The plays as resolved, not as typed.
			res.Signature = s.SignatureFor(names)
		}
	} else {
		res.Kind = KindCube
		res.Signature = s.CubeSignature()
		if !t.canDouble(rootCube, t.root) {
			return nil, fmt.Errorf("rollout: the player on roll cannot double here")
		}
		names = []string{NoDouble, DoubleTake}
		branches = []branch{
			{start: gnPos, cube: rootCube, skipFirstCube: true},
			{start: gnPos, cube: cubeState{value: rootCube.value * 2, owner: int(1 - t.root)}},
		}
	}

	accs, stop, runErr := t.rollAll(ctx, branches, res.Kind, opt.Progress, names, scale)
	res.Stop = stop
	res.Candidates = candidates(names, accs, scale)
	for _, a := range accs {
		res.Games = max(res.Games, a.n)
	}
	if res.Kind == KindCube {
		res.Cube = cubeResult(res.Candidates)
	} else {
		markMoveJSD(res.Candidates)
	}
	return res, runErr
}

// moveBranches is one branch per candidate play: the position after it, the
// opponent on roll.
// hasDice reports a position whose player on roll has rolled: a rollout of
// its plays; without dice, of its cube decision.
func hasDice(pos *domain.Position) bool {
	return pos.Dice[0] >= 1 && pos.Dice[0] <= 6 && pos.Dice[1] >= 1 && pos.Dice[1] <= 6
}

func moveBranches(pos *domain.Position, s Settings, named []string, cube cubeState) ([]string, []branch, error) {
	legal := domain.LegalMoves(pos)
	if len(legal) == 0 {
		return nil, nil, fmt.Errorf("rollout: no legal play for %d-%d", pos.Dice[0], pos.Dice[1])
	}
	byNotation := make(map[string]domain.Position, len(legal))
	for _, p := range legal {
		byNotation[p.Notation] = p.Result
	}

	if len(named) == 0 {
		k := s.Candidates
		if k <= 0 {
			k = 5
		}
		// Chosen at 2 ply at least: a 0-ply preset must not leave out the
		// play a deeper search ranks first.
		eval, err := gammonnet.EvaluatePosition(*pos, max(s.Ply, preselectPly), 0, k)
		if err != nil {
			return nil, nil, err
		}
		for _, m := range eval.Moves {
			named = append(named, m.Move)
		}
	}

	opponent := domain.White
	if pos.PlayerOnRoll == domain.White {
		opponent = domain.Black
	}
	var names []string
	var branches []branch
	for _, n := range named {
		n = strings.TrimSpace(n)
		res, ok := byNotation[n]
		if !ok {
			options := make([]string, 0, len(byNotation))
			for k := range byNotation {
				options = append(options, k)
			}
			slices.Sort(options)
			return nil, nil, fmt.Errorf("rollout: %q is not a legal play; legal: %s", n, strings.Join(options, ", "))
		}
		if slices.Contains(names, n) {
			continue
		}
		res.PlayerOnRoll = opponent
		res.Dice = [2]int{0, 0}
		gp, err := gammonnet.FromDomain(&res)
		if err != nil {
			return nil, nil, err
		}
		names = append(names, n)
		branches = append(branches, branch{start: gp, cube: cube})
	}
	return names, branches, nil
}

// rollAll plays the games batch by batch. Each game of each branch is a
// pure function of (seed, game number, branch) — the unit a worker takes —
// and results are summed in
// game order after each batch, so the numbers never depend on Workers or
// on which goroutine finished first. A batch interrupted by ctx is dropped
// whole.
func (t *table) rollAll(ctx context.Context, branches []branch, kind Kind, progress func(Progress), names []string, scale gammonnet.EquityScale) ([]accumulator, Stop, error) {
	s := t.settings
	nWorkers := s.Workers
	if nWorkers == 0 {
		nWorkers = runtime.NumCPU()
	}
	nWorkers = min(nWorkers, batchGames*len(branches))
	workers := make([]*worker, nWorkers)
	for i := range workers {
		w, err := newWorker(t)
		if err != nil {
			return nil, StopMaxGames, err
		}
		workers[i] = w
	}

	accs := make([]accumulator, len(branches))
	active := make([]bool, len(branches))
	for i := range active {
		active[i] = true
	}
	results := make([][]outcome, batchGames)
	for i := range results {
		results[i] = make([]outcome, len(branches))
	}

	for done := 0; done < s.MaxGames; {
		size := min(batchGames, s.MaxGames-done)
		var next atomic.Int64
		var firstErr error
		var errOnce sync.Once
		var wg sync.WaitGroup
		for _, w := range workers {
			wg.Add(1)
			go func(w *worker) {
				defer wg.Done()
				for {
					if ctx.Err() != nil {
						return
					}
					task := int(next.Add(1)) - 1
					if task >= size*len(branches) {
						return
					}
					g, b := task/len(branches), task%len(branches)
					if !active[b] {
						continue
					}
					o, err := w.play(&branches[b], newGameDice(s.Seed, done+g))
					if err != nil {
						errOnce.Do(func() { firstErr = err })
						return
					}
					results[g][b] = o
				}
			}(w)
		}
		wg.Wait()
		if firstErr != nil {
			return accs, StopMaxGames, firstErr
		}
		if err := ctx.Err(); err != nil {
			return accs, StopCancelled, err
		}
		for g := 0; g < size; g++ {
			for b := range branches {
				if active[b] {
					accs[b].add(results[g][b])
				}
			}
		}
		done += size

		if progress != nil {
			progress(Progress{Games: done, MaxGames: s.MaxGames, Candidates: candidates(names, accs, scale)})
		}
		if s.JSDLimit > 0 && done >= s.MinGames && done < s.MaxGames {
			if t.stopByJSD(kind, accs, active) {
				return accs, StopJSD, nil
			}
		}
	}
	return accs, StopMaxGames, nil
}

// stopByJSD applies the stopping rule. A checker play whose gap to the best
// reaches the limit stops being rolled, and the rollout ends when one play
// is left. A cube rollout ends when both of its questions are settled.
// Equity scales are affine, so the JSD is the same on the native scale.
func (t *table) stopByJSD(kind Kind, accs []accumulator, active []bool) bool {
	limit := t.settings.JSDLimit
	if kind == KindCube {
		nd, dt := accs[0], accs[1]
		dp := t.nativeDoublePass()
		double, doubleErr := dt.mean, dt.stdErr()
		if dp < dt.mean {
			double, doubleErr = dp, 0
		}
		return jsd(nd.mean, nd.stdErr(), double, doubleErr) >= limit &&
			jsd(dt.mean, dt.stdErr(), dp, 0) >= limit
	}
	best := 0
	for i := range accs {
		if accs[i].mean > accs[best].mean {
			best = i
		}
	}
	left := 0
	for i := range accs {
		if i != best && active[i] && jsd(accs[best].mean, accs[best].stdErr(), accs[i].mean, accs[i].stdErr()) >= limit {
			active[i] = false
		}
		if active[i] {
			left++
		}
	}
	return left <= 1
}

// nativeDoublePass is Double/Pass on the native scale: the root's cube won.
func (t *table) nativeDoublePass() float64 {
	if t.match {
		return t.mwcAfter(t.rootCube, true)
	}
	return 1
}

// jsd is |a−b| over the standard deviation of the difference, the two
// estimates taken as independent, as gnubg's check_jsds does. Common dice
// correlate them positively, so this overstates the spread: conservative.
func jsd(a, sa, b, sb float64) float64 {
	return math.Abs(a-b) / math.Max(math.Sqrt(sa*sa+sb*sb), jsdFloor)
}

func candidates(names []string, accs []accumulator, scale gammonnet.EquityScale) []Candidate {
	slope := scale.FromDecision(1) - scale.FromDecision(0)
	out := make([]Candidate, len(names))
	for i, n := range names {
		a := accs[i]
		e := Estimate{Games: a.n}
		if a.n > 0 {
			e.Equity = scale.FromDecision(a.mean)
			e.StdErr = slope * a.stdErr()
			e.CI95 = z95 * e.StdErr
			for k := range e.Chances {
				e.Chances[k] = a.chances[k] / float64(a.n)
			}
		}
		out[i] = Candidate{Move: n, Estimate: e}
	}
	return out
}

// markMoveJSD sorts the plays best first and gives each its JSD to the best.
func markMoveJSD(cs []Candidate) {
	slices.SortStableFunc(cs, func(a, b Candidate) int {
		switch {
		case a.Equity > b.Equity:
			return -1
		case a.Equity < b.Equity:
			return 1
		}
		return 0
	})
	if len(cs) == 0 {
		return
	}
	for i := 1; i < len(cs); i++ {
		cs[i].JSD = jsd(cs[0].Equity, cs[0].StdErr, cs[i].Equity, cs[i].StdErr)
	}
}

// cubeResult reads the two rolled branches as a cube decision; Double/Pass
// is +1 on the output scale by construction (the cash anchor).
func cubeResult(cs []Candidate) *CubeResult {
	nd, dt := cs[0].Estimate, cs[1].Estimate
	dp := Estimate{Equity: 1}
	double := dt
	if dp.Equity < dt.Equity {
		double = dp
	}
	cr := &CubeResult{
		NoDouble:   nd,
		DoubleTake: dt,
		DoublePass: dp,
		JSDDouble:  jsd(nd.Equity, nd.StdErr, double.Equity, double.StdErr),
		JSDTake:    jsd(dt.Equity, dt.StdErr, dp.Equity, 0),
	}
	cr.NoDouble.JSD = cr.JSDDouble
	cr.DoubleTake.JSD = cr.JSDTake
	cr.Action = actionName(gammonnet.Verdict(nd.Equity, dt.Equity, dp.Equity))
	return cr
}

func actionName(a gammonnet.CubeAction) string {
	switch a {
	case gammonnet.DoubleTake:
		return "Double, take"
	case gammonnet.DoublePass:
		return "Double, pass"
	case gammonnet.TooGood:
		return "Too good"
	}
	return "No double"
}
