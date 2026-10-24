package domain

import (
	"strconv"
	"strings"
)

// AwayFilterBounds reads an `aw` token (`aw:4`, `aw:2,5`, `aw<5`, `aw>3`)
// into inclusive bounds on the distance to victory, the PointsAway of each
// player. ok is false for a token of another shape: the caller matches
// nothing rather than dropping the filter.
func AwayFilterBounds(tok string) (lo, hi int, hasLo, hasHi, ok bool) {
	rest, found := strings.CutPrefix(tok, "aw")
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

// AwayStoredRange turns an `aw` token into the range of STORED away scores
// it admits, the same for both players, so a backend can compare the columns
// as they are. A distance of one is stored as 0 (after the Crawford game) or
// 1 (the Crawford game), so a lower bound of one or less opens at 0, which
// also shuts money play (-1) out. none reports a token that admits nothing:
// malformed, or an upper bound below one point.
func AwayStoredRange(tok string) (lo, hi int, hasHi, none bool) {
	dlo, dhi, hasLo, hasHi, ok := AwayFilterBounds(tok)
	if !ok || (hasHi && dhi < Crawford) || (hasLo && hasHi && dlo > dhi) {
		return 0, 0, false, true
	}
	lo = PostCrawford
	if hasLo && dlo > Crawford {
		lo = dlo
	}
	return lo, dhi, hasHi, false
}
