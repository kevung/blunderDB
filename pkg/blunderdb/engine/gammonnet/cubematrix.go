package gammonnet

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// The cube matrix: the cube verdict of one position at every away × away
// score of a match.
//
// Every cell is its own search (matchLength² of them): the search is
// match-aware (ADR-0016), so reading one money search through a different
// MET per cell would be wrong exactly where the score matters most.

// CubeMatrixCell is one score's verdict.
type CubeMatrixCell struct {
	// AwayOnRoll and AwayOpponent are "points still needed to win", from the
	// side of the player on roll — the one deciding.
	AwayOnRoll   int `json:"awayOnRoll"`
	AwayOpponent int `json:"awayOpponent"`

	// Verdict is the stable token of the decision: "no_double",
	// "double_take", "double_pass", "too_good". Empty when Refused.
	Verdict string `json:"verdict,omitempty"`

	// The three option equities, in the normalised scale ADR-0019 defines at
	// a match score. Zero when Refused.
	NoDouble   float64 `json:"noDouble"`
	DoubleTake float64 `json:"doubleTake"`
	DoublePass float64 `json:"doublePass"`

	// Refused says the engine declined this cell: neither an error nor a
	// "no double". Reason says why, for the tooltip.
	Refused bool   `json:"refused,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// CubeMatrix is the grid for one position at one match length.
type CubeMatrix struct {
	// MatchLength is the match the grid is for; cells run 1..MatchLength away
	// on each axis.
	MatchLength int `json:"matchLength"`
	// Ply is the depth every cell was searched at, so a caller showing a
	// 0-ply grid cannot present it as the 2-ply one.
	Ply   int              `json:"ply"`
	Cells []CubeMatrixCell `json:"cells"`
}

// CubeMatrixLengths are the match lengths the cube matrix and the score map
// offer: the odd ones, as matches are played, up to the 15 points of the
// longest common format. DefaultMapMatchLength is where they open.
var CubeMatrixLengths = []int{3, 5, 7, 9, 11, 13, 15}

// ComputeCubeMatrix evaluates pos's cube decision at every away × away score
// of a matchLength-point match.
//
// The position's own score is replaced by the grid; everything else, cube
// included, is kept. workers searches run at once, each with its own
// Searcher; the grid does not depend on workers. ctx cancels between cells,
// and a cancelled call returns what it had.
func ComputeCubeMatrix(ctx context.Context, pos domain.Position, matchLength, ply, pruneK, workers int) (CubeMatrix, error) {
	return ComputeCubeMatrixMET(ctx, pos, nil, matchLength, ply, pruneK, workers)
}

// ComputeCubeMatrixMET is ComputeCubeMatrix valued with the match equity
// table met (nil: the built-in Kazaross-XG2), the table of the library the
// position is studied in (ADR-0068).
func ComputeCubeMatrixMET(ctx context.Context, pos domain.Position, met *engine.MET, matchLength, ply, pruneK, workers int) (CubeMatrix, error) {
	if matchLength < 1 {
		return CubeMatrix{}, fmt.Errorf("gammonnet: cube matrix needs a match length of at least 1")
	}
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	return computeCubeMatrix(ctx, pos, met, matchLength, ply, pruneK, cubeMatrixGoroutines(workers, ply, pruneK))
}

// CellExec runs the n cells of a cube matrix: it calls cell(i, s) once for
// every i in [0, n), on any goroutines and in any order, s a Searcher built
// by NewBatchSearcher(ply, pruneK) that the goroutine reuses (nil: one is
// made per cell), and returns once every call it made has returned. It may
// leave cells uncalled only when ctx has ended or by returning an error.
type CellExec func(n int, cell func(i int, s *Searcher)) error

// mapDepths are the depths a map is filled at, shallow to deep, each phase
// replacing the previous one's answers: 0-ply for an immediate whole map,
// 1-ply because a whole map costs only a few times the 0-ply one, then ply.
func mapDepths(ply int) []int {
	depths := []int{0}
	for d := 1; d <= ply; d++ {
		if d == 1 || d == ply {
			depths = append(depths, d)
		}
	}
	return depths
}

// mapDeepWidth is how many cells a deep phase (depth >= 2) searches at once.
// A depth-2 search forks onto the idle cores of the budget (parallel.go), so
// a few cells at a time finish one after another, in the phase's order, about
// as fast in total as every cell at once finishing together at the end.
var mapDeepWidth = 4

// runMapPhase runs the n cells of one map phase through exec, in index order,
// ctx checked before each cell. A shallow phase hands exec every cell; a deep
// one hands it mapDeepWidth units, each taking the next cell in order as it
// frees, so cell i starts no later than cell i+1 and the cores a unit does
// not use go to its searches' forks.
func runMapPhase(ctx context.Context, exec CellExec, n, depth int, cell func(i int, s *Searcher)) error {
	if depth < 2 {
		return exec(n, func(i int, s *Searcher) {
			if ctx.Err() != nil {
				return
			}
			// exec's searchers are built for the map's own depth and fork
			// from 2 ply; a shallow cell searches serially, as a searcher
			// built for its depth would (CoresFor): its root's parallel
			// passes cost a 2-ply search for a 1-ply answer.
			if s != nil && s.team != nil {
				t := s.team
				s.team = nil
				defer func() { s.team = t }()
			}
			cell(i, s)
		})
	}
	var next atomic.Int64
	return exec(min(max(mapDeepWidth, 1), n), func(_ int, s *Searcher) {
		for ctx.Err() == nil {
			i := int(next.Add(1) - 1)
			if i >= n {
				return
			}
			cell(i, s)
		}
	})
}

// ComputeCubeMatrixExec is ComputeCubeMatrixMET with the cells run by exec,
// for a caller that shares its cores between several computations.
func ComputeCubeMatrixExec(ctx context.Context, pos domain.Position, met *engine.MET, matchLength, ply, pruneK int, exec CellExec) (CubeMatrix, error) {
	if matchLength < 1 {
		return CubeMatrix{}, fmt.Errorf("gammonnet: cube matrix needs a match length of at least 1")
	}
	return computeCubeMatrix(ctx, pos, met, matchLength, ply, pruneK, exec)
}

func computeCubeMatrix(ctx context.Context, pos domain.Position, met *engine.MET, matchLength, ply, pruneK int, exec CellExec) (CubeMatrix, error) {
	// Pre-roll, like every cube decision: the dice on the position are not
	// part of the question.
	pos.Dice = [2]int{0, 0}
	if _, err := FromDomain(&pos); err != nil {
		return CubeMatrix{}, err
	}

	type job struct{ onRoll, opponent int }
	jobs := make([]job, 0, matchLength*matchLength)
	for i := 1; i <= matchLength; i++ {
		for j := 1; j <= matchLength; j++ {
			jobs = append(jobs, job{i, j})
		}
	}
	cells := make([]CubeMatrixCell, len(jobs))
	err := exec(len(jobs), func(n int, searcher *Searcher) {
		if ctx.Err() != nil {
			return
		}
		j := jobs[n]
		cells[n] = cubeMatrixCell(pos, met, j.onRoll, j.opponent, searcher, ply, pruneK)
	})
	if ctx.Err() != nil {
		return CubeMatrix{MatchLength: matchLength, Ply: ply, Cells: cells}, ctx.Err()
	}
	return CubeMatrix{MatchLength: matchLength, Ply: ply, Cells: cells}, err
}

// cubeMatrixGoroutines is the CellExec of a computation that owns its
// cores: workers goroutines, each reusing one searcher.
func cubeMatrixGoroutines(workers, ply, pruneK int) CellExec {
	return func(n int, cell func(int, *Searcher)) error {
		var next atomic.Int64
		var wg sync.WaitGroup
		for range min(max(workers, 1), n) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				searcher, err := NewBatchSearcher(ply, pruneK)
				if err != nil {
					searcher = nil
				}
				for {
					i := next.Add(1) - 1
					if i >= int64(n) {
						return
					}
					release := HoldCore()
					cell(int(i), searcher)
					release()
				}
			}()
		}
		wg.Wait()
		return nil
	}
}

// cubeMatrixCell evaluates one cell. A refusal is recorded in the cell, never
// returned as an error: one unevaluable score must not cost the whole grid.
func cubeMatrixCell(pos domain.Position, met *engine.MET, awayOnRoll, awayOpponent int, searcher *Searcher, ply, pruneK int) CubeMatrixCell {
	cell := CubeMatrixCell{AwayOnRoll: awayOnRoll, AwayOpponent: awayOpponent}

	at := pos
	at.Score = scoreForCell(pos.PlayerOnRoll, awayOnRoll, awayOpponent)

	cfg, state, err := ConfigForPositionMET(&at, met, ply, pruneK)
	if err != nil {
		cell.Refused, cell.Reason = true, err.Error()
		return cell
	}
	scale, ok := NewEquityScale(state)
	if !ok {
		cell.Refused, cell.Reason = true, "no equity referential at this score"
		return cell
	}
	gnPos, err := FromDomain(&at)
	if err != nil {
		cell.Refused, cell.Reason = true, err.Error()
		return cell
	}

	s := searcher
	if s == nil {
		if s, err = NewBatchSearcher(ply, pruneK); err != nil {
			cell.Refused, cell.Reason = true, err.Error()
			return cell
		}
	}
	// A fresh searcher too: it is built for money, and the cell's score is
	// in cfg.
	if err := s.Reconfigure(cfg); err != nil {
		cell.Refused, cell.Reason = true, err.Error()
		return cell
	}

	probs, ok := s.Probs(&gnPos)
	if !ok {
		cell.Refused, cell.Reason = true, "the position could not be evaluated"
		return cell
	}
	dec, ok := DecideForSession(&probs, cfg.CubeOwner, state, DefaultEfficiency(cfg.CubeOwner), &at)
	if !ok {
		cell.Refused, cell.Reason = true, "cube decision at this score"
		return cell
	}

	cell.NoDouble = scale.FromDecision(dec.EquityNoDouble)
	cell.DoubleTake = scale.FromDecision(dec.EquityDoubleTake)
	cell.DoublePass = scale.FromDecision(dec.EquityDoublePass)
	cell.Verdict = cubeVerdictToken(dec.Action)
	return cell
}

// scoreForCell writes an away × away pair into a Position's score, in
// blunderDB's own storage convention: Score is indexed BY PLAYER, not by
// on-roll, and a raw 1 is the Crawford sentinel while a raw 0 decodes back to
// 1-away (parser.go's remap, MatchStateFromScores's decode).
//
// So a 1-away cell is stored as 0: the grid is post-Crawford throughout,
// since in the Crawford game the cube is not in play.
func scoreForCell(playerOnRoll, awayOnRoll, awayOpponent int) [2]int {
	crawfordSentinel := func(away int) int {
		if away == 1 {
			return 0
		}
		return away
	}
	opponent := domain.White
	if playerOnRoll == domain.White {
		opponent = domain.Black
	}
	var score [2]int
	score[playerOnRoll] = crawfordSentinel(awayOnRoll)
	score[opponent] = crawfordSentinel(awayOpponent)
	return score
}

// cubeVerdictToken is the stable name of a CubeAction, the one the wire and
// the interface both use. It is not the engine's own String(): a token that
// travels has to stay put across versions and languages.
func cubeVerdictToken(a CubeAction) string {
	switch a {
	case DoubleTake:
		return "double_take"
	case DoublePass:
		return "double_pass"
	case TooGood:
		return "too_good"
	default:
		return "no_double"
	}
}
