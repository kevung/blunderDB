package domain

import (
	"strconv"
	"strings"
)

// The values of SearchFilters.CrawfordFilter, written `cr` and `pcr`.
const (
	CrawfordGame  = "game"
	CrawfordAfter = "post"
	// CrawfordNone is both tokens at once: no game is both, so nothing matches.
	CrawfordNone = "none"
)

// AwayFilterBounds reads an away token — prefix `aw` for the player, `AW` for
// the opponent — (`aw:4`, `aw:2,5`, `aw<5`, `aw>3`) into inclusive bounds on
// a distance to victory, the PointsAway of that player. ok is false for a
// token of another shape: the caller matches nothing rather than dropping the
// filter.
func AwayFilterBounds(tok, prefix string) (lo, hi int, hasLo, hasHi, ok bool) {
	rest, found := strings.CutPrefix(tok, prefix)
	if !found || rest == "" {
		return 0, 0, false, false, false
	}
	switch rest[0] {
	case '<':
		v, err := strconv.Atoi(rest[1:])
		return 0, v, false, true, err == nil
	case '>':
		v, err := strconv.Atoi(rest[1:])
		return v, 0, true, false, err == nil
	case ':':
		a, b, pair := strings.Cut(rest[1:], ",")
		v1, err1 := strconv.Atoi(a)
		if err1 != nil {
			return 0, 0, false, false, false
		}
		if !pair {
			return v1, v1, true, true, true
		}
		v2, err2 := strconv.Atoi(b)
		if err2 != nil {
			return 0, 0, false, false, false
		}
		if v1 > v2 {
			v1, v2 = v2, v1
		}
		return v1, v2, true, true, true
	}
	return 0, 0, false, false, false
}

// AwayStoredRange turns an away token into the range of STORED away scores it
// admits, so a backend can compare the column as it is. A distance of one is
// stored as 0 (after the Crawford game) or 1 (the Crawford game), so a lower
// bound of one or less opens at 0, which also shuts money play (-1) out. none
// reports a token that admits nothing: malformed, or an upper bound below one
// point.
func AwayStoredRange(tok, prefix string) (lo, hi int, hasHi, none bool) {
	dlo, dhi, hasLo, hasHi, ok := AwayFilterBounds(tok, prefix)
	if !ok || (hasHi && dhi < Crawford) || (hasLo && hasHi && dlo > dhi) {
		return 0, 0, false, true
	}
	lo = PostCrawford
	if hasLo && dlo > Crawford {
		lo = dlo
	}
	return lo, dhi, hasHi, false
}

// AwayAdmits reports whether one player's stored away score meets the away
// token: AwayStoredRange read in Go, for the mirror search, which decides its
// filters on the mirrored board.
func AwayAdmits(tok, prefix string, stored int) bool {
	lo, hi, hasHi, none := AwayStoredRange(tok, prefix)
	return !none && stored >= lo && (!hasHi || stored <= hi)
}

// MatchesCrawfordFilter reports whether a pair of stored away scores is the
// game state the filter names. The state belongs to the game, not to one
// player: in the Crawford game one score is the Crawford sentinel (a 1-point
// match, [1, 1], is its own Crawford game), after it one score is the
// post-Crawford sentinel. Money play is neither.
func MatchesCrawfordFilter(filter string, score [2]int) bool {
	switch filter {
	case "":
		return true
	case CrawfordGame:
		return score[0] == Crawford || score[1] == Crawford
	case CrawfordAfter:
		return score[0] == PostCrawford || score[1] == PostCrawford
	}
	return false
}
