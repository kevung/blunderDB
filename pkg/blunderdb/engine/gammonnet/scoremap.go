package gammonnet

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// The score map: the cube matrix generalised to checker play. For a position
// with dice, every away × away cell holds the best play at that score, its
// equity, the runner-up and the gap between them; for a position without
// dice, the cube verdict, as the cube matrix gives it. The grid keeps the
// matrix's convention: post-Crawford throughout, the position's cube kept.

// ScoreMapCell is one score's answer. The embedded cube fields carry the
// coordinates, the refusal and, for a cube decision, the verdict.
type ScoreMapCell struct {
	CubeMatrixCell
	// Ply is the depth this cell was searched at: the grid is filled at
	// 0-ply first, then refined cell by cell.
	Ply int `json:"ply"`
	// Move and Equity are the best play and its equity (ADR-0019 scale at
	// this score). Checker decisions only.
	Move   string  `json:"move,omitempty"`
	Equity float64 `json:"equity,omitempty"`
	// SecondMove, SecondEquity and Gap (best minus second, ≥ 0) are absent
	// when the roll has a single legal play.
	SecondMove   string   `json:"secondMove,omitempty"`
	SecondEquity *float64 `json:"secondEquity,omitempty"`
	Gap          *float64 `json:"gap,omitempty"`
}

// ScoreMap is the grid of one position at one match length.
type ScoreMap struct {
	MatchLength int `json:"matchLength"`
	// Ply is the depth the cells were refined to.
	Ply int `json:"ply"`
	// Decision is "checker" for a position with dice, "cube" otherwise.
	Decision string `json:"decision"`
	// Cells run row by row: away on roll 1..MatchLength, then away opponent.
	Cells []ScoreMapCell `json:"cells"`
}

// DefaultMapMatchLength is the match length a map opens on.
const DefaultMapMatchLength = 9

// ScoreMapCellFunc receives a cell as soon as it is computed, at 0-ply then
// at the display depth; index is its place in ScoreMap.Cells. Calls are
// serialised.
type ScoreMapCellFunc func(index int, cell ScoreMapCell)

// ComputeScoreMap evaluates pos at every away × away score of a
// matchLength-point match: the best play when pos has dice, the cube decision
// otherwise. The grid is filled at 0-ply, then refined at ply starting from
// the position's own score and moving outwards; onCell (may be nil) sees each
// cell at each depth. A cell where the position's cube cannot have been
// reached is refused (ErrNotEvaluable), never re-centred. ctx cancels between
// cells and inside a search; a cancelled call returns what it had.
func ComputeScoreMap(ctx context.Context, pos domain.Position, met *engine.MET, matchLength, ply, pruneK int, exec CellExec, onCell ScoreMapCellFunc) (ScoreMap, error) {
	if matchLength < 1 || matchLength > engine.MaxScore {
		return ScoreMap{}, fmt.Errorf("gammonnet: match length %d is outside 1-%d", matchLength, engine.MaxScore)
	}
	if ply < 0 || ply > MaxPly {
		return ScoreMap{}, fmt.Errorf("gammonnet: ply %d is outside 0-%d", ply, MaxPly)
	}
	checker := pos.Dice[0] >= 1 && pos.Dice[0] <= 6 && pos.Dice[1] >= 1 && pos.Dice[1] <= 6
	if !checker {
		pos.Dice = [2]int{0, 0}
	}
	gnPos, err := FromDomain(&pos)
	if err != nil {
		return ScoreMap{}, err
	}
	var notations map[Position]string
	if checker {
		legal := domain.LegalMoves(&pos)
		if len(legal) == 0 {
			return ScoreMap{}, fmt.Errorf("%w: %d-%d has no legal play", ErrNotEvaluable, pos.Dice[0], pos.Dice[1])
		}
		notations = notationIndex(legal, pos.PlayerOnRoll)
	}

	m := ScoreMap{MatchLength: matchLength, Ply: ply, Decision: "cube", Cells: make([]ScoreMapCell, matchLength*matchLength)}
	if checker {
		m.Decision = "checker"
	}
	order := scoreMapOrder(&pos, matchLength)

	var mu sync.Mutex
	phase := func(depth int) error {
		err := exec(len(order), func(k int, s *Searcher) {
			if ctx.Err() != nil {
				return
			}
			idx := order[k]
			onRoll, opponent := idx/matchLength+1, idx%matchLength+1
			var cell ScoreMapCell
			if checker {
				cell = scoreMapCheckerCell(ctx, pos, gnPos, notations, met, onRoll, opponent, s, depth, pruneK)
			} else {
				cell = scoreMapCubeCell(ctx, pos, met, onRoll, opponent, s, depth, pruneK)
			}
			if ctx.Err() != nil {
				return // a cancelled search is not an answer
			}
			mu.Lock()
			defer mu.Unlock()
			m.Cells[idx] = cell
			if onCell != nil {
				onCell(idx, cell)
			}
		})
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}

	if err := phase(0); err != nil {
		return m, err
	}
	if ply > 0 {
		if err := phase(ply); err != nil {
			return m, err
		}
	}
	return m, nil
}

// scoreMapOrder is the order cells are searched in: the position's own score
// first, then by growing distance from it, so the cell the player is looking
// at sharpens first. Money or a score off the grid starts from the corner of
// the longest match.
func scoreMapOrder(pos *domain.Position, matchLength int) []int {
	homeRoll, homeOpp := matchLength, matchLength
	if state, ok := MatchStateFromPosition(pos); ok {
		homeRoll = min(max(state.AwayOnRoll, 1), matchLength)
		homeOpp = min(max(state.AwayOpponent, 1), matchLength)
	}
	dist := func(idx int) int {
		i, j := idx/matchLength+1, idx%matchLength+1
		return abs(i-homeRoll) + abs(j-homeOpp)
	}
	order := make([]int, matchLength*matchLength)
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return dist(order[a]) < dist(order[b]) })
	return order
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// cubeReachableAt reports whether pos's cube can stand at this score. The
// last double took the cube from half its value; a player already holding
// the match at that stake never had a reason to double, and a centred cube
// above 1 never happens in a match. The grid has no Crawford game, so that
// is the whole rule.
func cubeReachableAt(pos *domain.Position, awayOnRoll, awayOpponent int) error {
	if pos.Cube.Value <= 0 {
		return nil
	}
	value := 1 << uint(pos.Cube.Value)
	var doublerAway int
	switch pos.Cube.Owner {
	case pos.PlayerOnRoll:
		doublerAway = awayOpponent
	case domain.None:
		return fmt.Errorf("%w: a centred cube at %d cannot stand in a match", ErrNotEvaluable, value)
	default:
		doublerAway = awayOnRoll
	}
	if value/2 >= doublerAway {
		return fmt.Errorf("%w: a cube at %d means a double by a player %d-away, who already held the match at %d", ErrNotEvaluable, value, doublerAway, value/2)
	}
	return nil
}

func refusedCell(onRoll, opponent, depth int, err error) ScoreMapCell {
	c := ScoreMapCell{Ply: depth}
	c.AwayOnRoll, c.AwayOpponent = onRoll, opponent
	c.Refused, c.Reason = true, err.Error()
	return c
}

// scoreMapCubeCell is the cube matrix's cell, under the cube-reachability rule.
func scoreMapCubeCell(ctx context.Context, pos domain.Position, met *engine.MET, onRoll, opponent int, s *Searcher, depth, pruneK int) ScoreMapCell {
	if err := cubeReachableAt(&pos, onRoll, opponent); err != nil {
		return refusedCell(onRoll, opponent, depth, err)
	}
	if s != nil {
		defer s.BindContext(ctx)()
	}
	return ScoreMapCell{CubeMatrixCell: cubeMatrixCell(pos, met, onRoll, opponent, s, depth, pruneK), Ply: depth}
}

// scoreMapCheckerCell ranks pos's plays at one score.
func scoreMapCheckerCell(ctx context.Context, pos domain.Position, gnPos Position, notations map[Position]string, met *engine.MET, onRoll, opponent int, s *Searcher, depth, pruneK int) ScoreMapCell {
	if err := cubeReachableAt(&pos, onRoll, opponent); err != nil {
		return refusedCell(onRoll, opponent, depth, err)
	}
	at := pos
	at.Score = scoreForCell(pos.PlayerOnRoll, onRoll, opponent)
	cfg, state, err := ConfigForPositionMET(&at, met, depth, pruneK)
	if err != nil {
		return refusedCell(onRoll, opponent, depth, err)
	}
	scale, ok := NewEquityScale(state)
	if !ok {
		return refusedCell(onRoll, opponent, depth, fmt.Errorf("%w: no equity referential at this score", ErrNotEvaluable))
	}
	if s == nil {
		if s, err = NewBatchSearcher(depth, pruneK); err != nil {
			return refusedCell(onRoll, opponent, depth, err)
		}
	}
	if err := s.Reconfigure(cfg); err != nil {
		return refusedCell(onRoll, opponent, depth, err)
	}
	defer s.BindContext(ctx)()

	out := s.scratch()
	n, err := s.Plays(&gnPos, pos.Dice[0], pos.Dice[1], out)
	if err != nil {
		return refusedCell(onRoll, opponent, depth, err)
	}
	if n == 0 {
		return refusedCell(onRoll, opponent, depth, fmt.Errorf("%w: no legal play", ErrNotEvaluable))
	}
	cell := ScoreMapCell{Ply: depth}
	cell.AwayOnRoll, cell.AwayOpponent = onRoll, opponent
	cell.Move = notationForCandidate(&out[0], notations)
	cell.Equity = scale.FromSearch(out[0].Equity)
	if n > 1 {
		second := scale.FromSearch(out[1].Equity)
		gap := cell.Equity - second
		cell.SecondMove = notationForCandidate(&out[1], notations)
		cell.SecondEquity, cell.Gap = &second, &gap
	}
	return cell
}
