package engine

import (
	"fmt"
	"slices"
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
// CanonicalMove folds all three and gives the play's shape; HitPoints gives
// where it hits. SamePlay compares on both, and reads a missing "*" as "no
// hit" unless the source never writes one (MarksHits): only then is an
// unmarked play a wildcard over the hits.

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

// HitPoints lists the points a play hits on, as its "*" markers name them,
// sorted and without repeats: "8/5*/4" hits on 5, "6/5* 5/1*" on 1 and 5.
func HitPoints(move string) []string {
	var hits []string
	for _, tok := range strings.Fields(move) {
		if i := strings.IndexByte(tok, '('); i >= 0 && strings.HasSuffix(tok, ")") {
			tok = tok[:i]
		}
		points := strings.Split(tok, "/")
		for _, p := range points[1:] {
			if strings.HasSuffix(p, "*") {
				hits = append(hits, strings.TrimRight(p, "*"))
			}
		}
	}
	sort.Strings(hits)
	return slices.Compact(hits)
}

// PlayKey is CanonicalMove with the play's hit points: equal keys name the
// same play in any dialect that writes its hits.
func PlayKey(move string) string {
	canon := CanonicalMove(move)
	if hits := HitPoints(move); len(hits) > 0 {
		return canon + " *" + strings.Join(hits, ",")
	}
	return canon
}

// MarksHits reports whether a source writes its hits: whether any play of
// its list carries a "*". A source that never does (gnubg's SGF candidates)
// says nothing about hits, so its plays match on shape alone.
func MarksHits(moves ...string) bool {
	for _, m := range moves {
		if strings.Contains(m, "*") {
			return true
		}
	}
	return false
}

// SamePlay reports whether two notations name the same play. aMarks and
// bMarks say whether each one's source writes its hits (MarksHits): when
// both do, a missing "*" means no hit and the hit points must agree; when
// one does not, the shape decides and the caller must make sure it names a
// single play.
func SamePlay(a string, aMarks bool, b string, bMarks bool) bool {
	if CanonicalMove(a) != CanonicalMove(b) {
		return false
	}
	if !aMarks || !bMarks {
		return true
	}
	return slices.Equal(HitPoints(a), HitPoints(b))
}

// ResolvePlay finds the legal play a notation stands for: as written first,
// then as the play it describes (SamePlay, the generator always writing its
// hits). It refuses rather than guesses: a notation that names no legal
// play, or plays leaving different boards, is an error.
func ResolvePlay(legal []domain.LegalPlay, notation string, marks bool) (domain.LegalPlay, error) {
	notation = strings.TrimSpace(notation)
	norm := NormalizeMove(notation)
	for _, p := range legal {
		if p.Notation == notation || NormalizeMove(p.Notation) == norm {
			return p, nil
		}
	}
	var same []domain.LegalPlay
	for _, p := range legal {
		if SamePlay(notation, marks, p.Notation, true) {
			same = append(same, p)
		}
	}
	if len(same) == 0 {
		return domain.LegalPlay{}, fmt.Errorf("%q is not a legal play", notation)
	}
	for _, p := range same[1:] {
		if p.Result.Board != same[0].Result.Board {
			names := make([]string, len(same))
			for i, q := range same {
				names[i] = q.Notation
			}
			return domain.LegalPlay{}, fmt.Errorf("%q is ambiguous: %s", notation, strings.Join(names, ", "))
		}
	}
	return same[0], nil
}
