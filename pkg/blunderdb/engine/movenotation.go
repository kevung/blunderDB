package engine

import (
	"sort"
	"strconv"
	"strings"
)

// Comparing a move between two engines (#270).
//
// The same checker play is written differently by different engines, and the
// differences are notation, not disagreement:
//
//   - hit markers: gnubg's SGF candidates render "6/3 3/1" for a move that
//     hits on the way, XG's carry the "*". The hit is a derived fact about the
//     board, not part of which move was chosen.
//   - repetition: "4/2 4/2" and "4/2(2)" are the same move; which form appears
//     depends on which side collapsed it.
//   - chained hops: XG writes one checker's two dice as one step, "13/7";
//     gammonNet writes both, "13/8 8/7". They leave the SAME board — moving
//     one checker 13→8→7 and moving one checker 13→8 while another goes 8→7
//     are indistinguishable once the dust settles — so a comparison that told
//     them apart would report a disagreement nobody has.
//
// CanonicalMove folds all three. It is what lets a comparison between two
// engines count actual disagreements rather than dialects.

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
		from, to, ok := strings.Cut(tok, "/")
		if !ok {
			// Not a step at all ("cannot move", a bare word): keep it as is,
			// so two spellings of the same non-move still compare equal.
			from, to = tok, ""
		}
		for i := 0; i < count; i++ {
			hops = append(hops, hop{from, to})
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
