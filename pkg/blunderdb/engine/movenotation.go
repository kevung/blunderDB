package engine

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Comparing a move between two engines.
//
// The same checker play is written differently by different engines, and the
// differences are notation, not disagreement:
//
//   - hit markers: gnubg's SGF candidates render "6/3 3/1" for a move that
//     hits on the way, XG's and gammonNet's carry the "*". Where a hit lands
//     is part of which play was chosen: "6/5* 5/1*" and "6/2 2/1*" (4-1) move
//     the same checker to the same point and leave different boards.
//   - repetition: "4/2 4/2" and "4/2(2)" are the same move; which form appears
//     depends on which side collapsed it.
//   - chained hops: XG writes one checker's two dice as one step, "13/7", or
//     as one token naming the hop, "8/5*/4"; gammonNet writes both, "13/8
//     8/7". They leave the SAME board.
//
// CanonicalMove folds all three and gives the play's shape, for display and
// counting. Deciding whether two notations are the same play is a question
// about boards, not text: ResolveBoard replays each on the position, the hits
// read in its dialect (EngineMarksHits).

// CanonicalMove renders a checker play in a form two engines can be compared
// on: hit markers dropped, "(n)" expanded, chained hops merged, steps sorted.
//
// It is a NOTATION-level canonicalisation, not a board simulation: it never
// looks at a position. Two moves that canonicalise the same leave the same
// board; the converse is not guaranteed for pathological input, which is why
// a caller that needs certainty (the .mat exporter, the move generator) works
// from the board instead.
func CanonicalMove(move string) string {
	type hop struct{ from, to string }
	var hops []hop
	for _, tok := range strings.Fields(move) {
		tok = strings.ReplaceAll(tok, "*", "")
		count := 1
		if i := strings.IndexByte(tok, '('); i >= 0 && strings.HasSuffix(tok, ")") {
			if n, err := strconv.Atoi(tok[i+1 : len(tok)-1]); err == nil && n > 0 {
				count = n
			}
			tok = tok[:i]
		}
		points := strings.Split(tok, "/")
		if len(points) < 2 {
			// Not a step at all ("cannot move", a bare word): keep it as is,
			// so two spellings of the same non-move still compare equal.
			points = []string{tok, ""}
		}
		// "8/5*/4" is one checker hopping twice, written as one token.
		for i := 0; i < count; i++ {
			for j := 0; j+1 < len(points); j++ {
				hops = append(hops, hop{points[j], points[j+1]})
			}
		}
	}

	// Merge x→y with y→z into x→z, repeatedly. A hop into "off" or out of
	// "bar" is a fine link in a chain; only "off" cannot be departed from,
	// and there is no such hop to find.
	for merged := true; merged; {
		merged = false
		for i := range hops {
			if hops[i].to == "" {
				continue
			}
			for j := range hops {
				if i == j || hops[j].from != hops[i].to {
					continue
				}
				hops[i] = hop{hops[i].from, hops[j].to}
				hops = append(hops[:j], hops[j+1:]...)
				merged = true
				break
			}
			if merged {
				break
			}
		}
	}

	steps := make([]string, 0, len(hops))
	for _, h := range hops {
		if h.to == "" {
			steps = append(steps, h.from)
			continue
		}
		steps = append(steps, h.from+"/"+h.to)
	}
	sort.Strings(steps)
	return strings.Join(steps, " ")
}

// EngineMarksHits says whether an analysis engine's dialect, as blunderDB's
// importers write it, carries hit markers; known=false for an engine it does
// not name. XG, gammonNet and blunderDB's own move generator write "*"; the
// GNU Backgammon, BGBlitz and HedgeHog importers never do. A stored rollout
// ("blunderDB rollout") names its plays as they were picked, possibly from an
// unmarked analysis, so it is read as one that does not.
func EngineMarksHits(engine string) (marks, known bool) {
	switch {
	case strings.HasPrefix(engine, "blunderDB rollout"):
		return false, true
	case engine == "XG", strings.HasPrefix(engine, "gammonNet"), strings.HasPrefix(engine, "blunderDB"):
		return true, true
	case engine == "GNUbg", engine == "BGBlitz", strings.HasPrefix(engine, "HedgeHog"):
		return false, true
	}
	return false, false
}

// MarksHits reports whether any of the notations carries a "*": what can be
// told of a source whose dialect is unknown, from the plays it wrote.
func MarksHits(moves ...string) bool {
	for _, m := range moves {
		if strings.Contains(m, "*") {
			return true
		}
	}
	return false
}

// ResolveBoard returns the one position a notation leaves from pos
// (domain.PlayBoards); marks says whether its dialect writes its hits. It
// refuses rather than guesses: a notation that names no play, or plays
// leaving different boards, is an error.
func ResolveBoard(pos *domain.Position, notation string, marks bool) (domain.Position, error) {
	boards := domain.PlayBoards(pos, notation, marks)
	switch len(boards) {
	case 0:
		return domain.Position{}, fmt.Errorf("%q is not a legal play of %d-%d", notation, pos.Dice[0], pos.Dice[1])
	case 1:
		return boards[0], nil
	}
	return domain.Position{}, fmt.Errorf("%q is ambiguous: it leaves %d different boards", notation, len(boards))
}

// ResolvePlay finds, among legal (domain.LegalMoves of pos), the play a
// notation stands for: the one leaving the board ResolveBoard finds.
func ResolvePlay(pos *domain.Position, legal []domain.LegalPlay, notation string, marks bool) (domain.LegalPlay, error) {
	board, err := ResolveBoard(pos, notation, marks)
	if err != nil {
		return domain.LegalPlay{}, err
	}
	for _, p := range legal {
		if domain.SameBoard(p.Result.Board, board.Board) {
			return p, nil
		}
	}
	return domain.LegalPlay{}, fmt.Errorf("%q is not a legal play", notation)
}

// SamePlay reports whether two notations leave the same board from pos, each
// read in its dialect (aMarks, bMarks: EngineMarksHits). ok is false when
// either cannot be resolved to one board: then nothing can be said.
func SamePlay(pos *domain.Position, a string, aMarks bool, b string, bMarks bool) (same, ok bool) {
	ab, err := ResolveBoard(pos, a, aMarks)
	if err != nil {
		return false, false
	}
	bb, err := ResolveBoard(pos, b, bMarks)
	if err != nil {
		return false, false
	}
	return domain.SameBoard(ab.Board, bb.Board), true
}
