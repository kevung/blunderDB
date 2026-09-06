package gammonnet

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The cube matrix (issue #267, fiche I.11).
//
// A cube decision is not a property of the board. The same checkers, the same
// pips, the same everything, are a clear double at 2-away/4-away and a clear
// no-double at 4-away/2-away — and a player who has learnt the money answer
// has learnt one cell of a grid. What blunderDB could always show was that one
// cell, the one the position happened to carry.
//
// The matrix shows the whole grid: the verdict at every away × away score of a
// match, on the position in front of the user.
//
// # Why this is a real sweep and not one search read many ways
//
// The search is MATCH-AWARE since ADR-0016: the distribution it returns
// already depends on the score, because the score changes which continuations
// are worth playing for. Reading one money search through a different match
// equity table per cell would produce a grid that looks right and is wrong
// exactly where the score matters most. So every cell is its own search, and
// the cost is the honest one: matchLength² searches.

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

	// Refused says the engine declined this cell — a score beyond its match
	// equity table, a cube state it will not judge. It is NOT an error and
	// NOT a "no double": a cell nobody can evaluate must look different from
	// a cell where the answer is "don't". Reason says why, for the tooltip.
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

// CubeMatrixLengths are the match lengths the interface offers. Three, not a
// free number: the grid is read by eye, and 9×9 is already at the edge of what
// fits beside a board.
var CubeMatrixLengths = []int{5, 7, 9}

// ComputeCubeMatrix evaluates pos's cube decision at every away × away score
// of a matchLength-point match.
//
// The position's own score is ignored — the grid replaces it — but everything
// else about the position is kept, cube included: the matrix answers "at what
// score would I double THIS", not "what would a different position do".
//
// workers searches run at once, each with its own Searcher. Cells are
// independent, so the grid is identical whatever workers says; only the time
// changes. ctx cancels between cells, and a cancelled call returns what it had.
func ComputeCubeMatrix(ctx context.Context, pos domain.Position, matchLength, ply, pruneK, workers int) (CubeMatrix, error) {
	if matchLength < 1 {
		return CubeMatrix{}, fmt.Errorf("gammonnet: cube matrix needs a match length of at least 1")
	}
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

	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if workers > len(jobs) {
		workers = len(jobs)
	}

	var next atomic.Int64
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			searcher, err := NewBatchSearcher(ply, pruneK)
			if err != nil {
				searcher = nil
			}
			for {
				if ctx.Err() != nil {
					return
				}
				n := next.Add(1) - 1
				if n >= int64(len(jobs)) {
					return
				}
				j := jobs[n]
				cells[n] = cubeMatrixCell(pos, matchLength, j.onRoll, j.opponent, searcher, ply, pruneK)
			}
		}()
	}
	wg.Wait()

	return CubeMatrix{MatchLength: matchLength, Ply: ply, Cells: cells}, ctx.Err()
}

// cubeMatrixCell evaluates one cell. A refusal is recorded in the cell, never
// returned as an error: one unevaluable score must not cost the whole grid.
func cubeMatrixCell(pos domain.Position, matchLength, awayOnRoll, awayOpponent int, searcher *Searcher, ply, pruneK int) CubeMatrixCell {
	cell := CubeMatrixCell{AwayOnRoll: awayOnRoll, AwayOpponent: awayOpponent}

	at := pos
	at.Score = scoreForCell(pos.PlayerOnRoll, awayOnRoll, awayOpponent)

	cfg, state, err := ConfigForPosition(&at, ply, pruneK)
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
	} else if err := s.Reconfigure(cfg); err != nil {
		cell.Refused, cell.Reason = true, err.Error()
		return cell
	}

	probs, ok := s.Probs(&gnPos)
	if !ok {
		cell.Refused, cell.Reason = true, "the position could not be evaluated"
		return cell
	}
	dec, ok := Decide(&probs, cfg.CubeOwner, state, DefaultEfficiency(cfg.CubeOwner), at.HasJacoby == 1)
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
// So a 1-away cell of the grid is stored as 0: the grid is the POST-Crawford
// one throughout. That is the only reading that carries information — during
// the Crawford game the cube is not in play at all, and a column of "you may
// not double" would say nothing about the position.
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
