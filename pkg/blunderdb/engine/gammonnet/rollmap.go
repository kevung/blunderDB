package gammonnet

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// The roll map: for a position where a player is about to roll, what each of
// the 21 rolls does — its best play, the equity after it, and whether it
// loses the market. The same map answers "what do my rolls do" and "what can
// the opponent reply" (the position after a candidate play, opponent on
// roll); PositionAfterPlay builds the second from the first.
//
// A row's equity is the search's own value of its best play, so the
// roll-weighted average of the rows is, term for term, the position's
// equity one ply deeper: the map decomposes the number the panel shows.

// RollMapRow is one roll's answer.
type RollMapRow struct {
	// Die1 ≤ Die2, in the search's reference order (1-1, 1-2, … 6-6).
	Die1 int `json:"die1"`
	Die2 int `json:"die2"`
	// Weight is the roll's chance out of 36: 1 for a double, 2 otherwise.
	Weight int `json:"weight"`
	// Move is the best play's notation; empty when the roll dances.
	Move  string `json:"move,omitempty"`
	Dance bool   `json:"dance,omitempty"`
	// Equity is the roller's equity after the roll and its best play, in the
	// scale ADR-0019 defines (money points, or normalised at a score).
	Equity float64 `json:"equity"`
	// MarketLost says that, after this roll and its best play, a double from
	// the roller would be one the opponent passes (double/pass or too good).
	// Nil when the roller has no access to the cube.
	MarketLost *bool `json:"marketLost,omitempty"`
	// Ply is the depth this row was searched at: the map is filled at 0-ply
	// first, then refined row by row.
	Ply int `json:"ply"`
}

// RollMap is the 21 rows of one position.
type RollMap struct {
	// Ply is the depth the rows were refined to.
	Ply  int          `json:"ply"`
	Rows []RollMapRow `json:"rows"`
	// Average is the roll-weighted mean of the rows' equities: the position's
	// equity at Ply+1, cube not turned.
	Average float64 `json:"average"`
	// CubeAccess says the roller may double here, so MarketLost is set.
	CubeAccess bool `json:"cubeAccess"`
}

// RollMapRowFunc receives a row as soon as it is computed, at 0-ply then at
// the display depth. Calls are serialised.
type RollMapRowFunc func(index int, row RollMapRow)

// LocalExec is the CellExec of a computation that owns its cores: workers
// goroutines (0: one per core), each reusing one searcher built for
// ply/pruneK, each holding a core of the process budget while it searches.
func LocalExec(workers, ply, pruneK int) CellExec {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	return cubeMatrixGoroutines(workers, ply, pruneK)
}

// ComputeRollMap evaluates the 21 rolls of pos, a position whose player on
// roll has not rolled yet (its dice are ignored). Rows go to onRow (may be
// nil) at 0-ply first, then at ply. exec runs the rows; ctx cancels between
// rolls and inside a row's search, and a cancelled call returns what it had
// with ctx's error. Nothing is stored.
func ComputeRollMap(ctx context.Context, pos domain.Position, met *engine.MET, ply, pruneK int, exec CellExec, onRow RollMapRowFunc) (RollMap, error) {
	if IsResponsePosition(&pos) {
		return RollMap{}, fmt.Errorf("%w: a take/pass decision has no rolls", ErrNotEvaluable)
	}
	if ply < 0 || ply > MaxPly {
		return RollMap{}, fmt.Errorf("gammonnet: ply %d is outside 0-%d", ply, MaxPly)
	}
	pos.Dice = [2]int{0, 0}
	gnPos, err := FromDomain(&pos)
	if err != nil {
		return RollMap{}, err
	}
	if gnPos.isOver() {
		return RollMap{}, fmt.Errorf("%w: the game is over", ErrNotEvaluable)
	}
	cfg, state, err := ConfigForPositionMET(&pos, met, ply, pruneK)
	if err != nil {
		return RollMap{}, err
	}
	scale, ok := NewEquityScale(state)
	if !ok {
		return RollMap{}, fmt.Errorf("%w: no equity referential at score %v", ErrNotEvaluable, pos.Score)
	}

	m := RollMap{Ply: ply, Rows: make([]RollMapRow, NumRolls), CubeAccess: hasCubeAccess(cfg.CubeOwner, state)}
	job := rollMapJob{pos: pos, gnPos: gnPos, cfg: cfg, state: state, scale: scale, access: m.CubeAccess, rolls: buildRolls()}

	var mu sync.Mutex
	var firstErr error
	phase := func(depth int) error {
		job.cfg = cfg
		job.cfg.Ply = depth
		err := exec(NumRolls, func(i int, s *Searcher) {
			if ctx.Err() != nil {
				return
			}
			row, err := job.row(ctx, i, s, pruneK)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil && !IsCanceled(err) {
					firstErr = err
				}
				return
			}
			m.Rows[i] = row
			if onRow != nil {
				onRow(i, row)
			}
		})
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			return err
		}
		return firstErr
	}

	// The whole map at 0-ply first, so a reader sees every roll at once; then
	// the display depth replaces each row as it lands.
	if err := phase(0); err != nil {
		return m, err
	}
	if ply > 0 {
		if err := phase(ply); err != nil {
			return m, err
		}
	}
	// Ascending roll index, in float64: the order the search sums in.
	for i, r := range job.rolls {
		m.Average += r.weight * m.Rows[i].Equity
	}
	return m, nil
}

// PositionAfterPlay is pos played with d1-d2 and move (any notation of a
// legal play, not necessarily the best), handed to the opponent before their
// roll: the position whose roll map is the opponent's replies. An empty move
// is accepted only for a roll that dances.
func PositionAfterPlay(pos domain.Position, d1, d2 int, move string) (domain.Position, error) {
	if d1 < 1 || d1 > 6 || d2 < 1 || d2 > 6 {
		return domain.Position{}, fmt.Errorf("gammonnet: dice %d-%d out of range", d1, d2)
	}
	pos.Dice = [2]int{d1, d2}
	legal := domain.LegalMoves(&pos)
	opponent := domain.White
	if pos.PlayerOnRoll == domain.White {
		opponent = domain.Black
	}
	next := pos
	switch {
	case len(legal) == 0:
		if move != "" {
			return domain.Position{}, fmt.Errorf("gammonnet: %d-%d dances, there is no play %q", d1, d2, move)
		}
	case move == "":
		return domain.Position{}, errors.New("gammonnet: a play is required for this roll")
	default:
		want := engine.CanonicalMove(move)
		found := false
		for _, p := range legal {
			if engine.CanonicalMove(p.Notation) == want {
				next = p.Result
				found = true
				break
			}
		}
		if !found {
			return domain.Position{}, fmt.Errorf("gammonnet: %q is not a legal play of %d-%d", move, d1, d2)
		}
	}
	next.PlayerOnRoll = opponent
	next.Dice = [2]int{0, 0}
	next.ID = 0
	return next, nil
}

// hasCubeAccess: the roller may double — the cube is theirs or centred, and
// this is not the Crawford game.
func hasCubeAccess(owner CubeOwner, state *MatchState) bool {
	if owner == CubeOpponent {
		return false
	}
	return state == nil || !state.Crawford
}

// rollMapJob is what every row of one phase shares.
type rollMapJob struct {
	pos    domain.Position
	gnPos  Position
	cfg    SearchConfig
	state  *MatchState
	scale  EquityScale
	access bool
	rolls  [NumRolls]diceRoll
}

// row evaluates roll i at j.cfg's depth on s (nil: a fresh searcher).
func (j *rollMapJob) row(ctx context.Context, i int, s *Searcher, pruneK int) (RollMapRow, error) {
	roll := j.rolls[i]
	d1, d2 := int(roll.d1), int(roll.d2)
	depth := j.cfg.Ply
	row := RollMapRow{Die1: d1, Die2: d2, Weight: 2, Ply: depth}
	if d1 == d2 {
		row.Weight = 1
	}

	if s == nil {
		var err error
		if s, err = NewBatchSearcher(depth, pruneK); err != nil {
			return row, err
		}
	}
	if err := s.Reconfigure(j.cfg); err != nil {
		return row, err
	}
	defer s.BindContext(ctx)()

	var child *MatchState
	if j.state != nil {
		c := j.state.Swap()
		child = &c
	}
	theirOwner := j.cfg.CubeOwner.Mirror()

	// after is the position the opponent rolls from; afterProbs, once known,
	// its 0-ply distribution from the opponent's side.
	var after Position
	var afterProbs *[NumOutputs]float32

	n, err := s.Plays(&j.gnPos, d1, d2, s.scratch())
	switch {
	case err != nil:
		return row, err
	case n == 0:
		// A dance hands the turn over with the board unchanged: the value
		// the search gives it, as its own roll loop does.
		row.Dance = true
		after = j.gnPos
		after.swapTurn()
		v, ok := s.positionEquity(&after, depth, 0, child, theirOwner)
		if !ok {
			return row, j.failure(s)
		}
		row.Equity = j.scale.FromSearch(-v)
	default:
		best := s.best[0]
		after = best.Play.Result
		afterProbs = &best.Probs
		row.Equity = j.scale.FromSearch(best.Equity)
		withDice := j.pos
		withDice.Dice = [2]int{d1, d2}
		row.Move = notationForCandidate(&best, notationIndex(domain.LegalMoves(&withDice), j.pos.PlayerOnRoll))
	}

	if j.access {
		lost, ok, err := j.marketLost(s, &after, afterProbs, depth, child, theirOwner)
		if err != nil {
			return row, err
		}
		if ok {
			row.MarketLost = &lost
		}
	}
	return row, nil
}

// marketLost asks the cube decision of the roller, read on the position the
// opponent is about to roll from, as if the roller could double there: a
// double/pass or a too-good means the market went with this roll. The
// distribution is searched one ply short of the row (the network's answer at
// 0- and 1-ply, already at hand for a play), which keeps the map at the cost
// of its plays. ok is false where the model declines the decision.
func (j *rollMapJob) marketLost(s *Searcher, after *Position, known *[NumOutputs]float32, depth int, child *MatchState, theirOwner CubeOwner) (lost, ok bool, err error) {
	probsDepth := max(depth-1, 0)
	var theirs [NumOutputs]float32
	if known != nil && probsDepth == 0 {
		theirs = *known
	} else {
		p, good := s.probsAt(after, probsDepth, 0, child, theirOwner)
		if !good {
			return false, false, j.failure(s)
		}
		theirs = p
	}
	mine := invertProbs(&theirs)
	dec, good := DecideForSession(&mine, j.cfg.CubeOwner, j.state, DefaultEfficiency(j.cfg.CubeOwner), &j.pos)
	if !good {
		return false, false, nil
	}
	return dec.Action == DoublePass || dec.Action == TooGood, true, nil
}

// failure names why a search returned nothing: cancelled, or broken.
func (j *rollMapJob) failure(s *Searcher) error {
	if s.stopped() {
		return ErrCanceled
	}
	return errors.New("gammonnet: the position could not be evaluated")
}
