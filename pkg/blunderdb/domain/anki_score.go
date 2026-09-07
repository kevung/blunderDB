package domain

import (
	"fmt"
	"strconv"
	"strings"
)

// The away range the score cards cover (ADR-0040 rule 4): 2 to 9 away, the
// scores whose take points and gammon values are worth knowing by heart. Below
// 2 away there is no cube decision left to take, and beyond 9 the numbers are
// close enough to money play that a table adds nothing.
//
// The Training exercise draws from the same range, in
// frontend/src/services/scoreCard.js. The two lists are written twice because
// they are read on two sides of the Wails boundary — but only ONE of them
// creates cards: a deck is filled here, and the review view renders whatever
// key it is handed. A drift would therefore change what is drilled under the
// clock, never what a deck holds.
const (
	ScoreAwayMin = 2
	ScoreAwayMax = 9
)

// ScoreKey names an unordered score: "3:5" and "5:3" are the same question
// asked from two sides, so the smaller away is always written first. That is
// what makes the 36 keys of UnorderedScoreKeys a set rather than a grid of 64.
func ScoreKey(a, b int) string {
	if b < a {
		a, b = b, a
	}
	return strconv.Itoa(a) + ":" + strconv.Itoa(b)
}

// ParseScoreKey reads a key written by ScoreKey back into its two aways.
func ParseScoreKey(key string) (int, int, error) {
	lo, hi, ok := strings.Cut(key, ":")
	if !ok {
		return 0, 0, fmt.Errorf("score key %q: want \"a:b\"", key)
	}
	a, err := strconv.Atoi(lo)
	if err != nil {
		return 0, 0, fmt.Errorf("score key %q: %w", key, err)
	}
	b, err := strconv.Atoi(hi)
	if err != nil {
		return 0, 0, fmt.Errorf("score key %q: %w", key, err)
	}
	if a < ScoreAwayMin || b < ScoreAwayMin || a > ScoreAwayMax || b > ScoreAwayMax {
		return 0, 0, fmt.Errorf("score key %q: aways outside %d-%d", key, ScoreAwayMin, ScoreAwayMax)
	}
	return a, b, nil
}

// UnorderedScoreKeys is the content of a score deck (ADR-0042 rule 2): the 36
// unordered scores of ScoreAwayMin to ScoreAwayMax away, in reading order.
// The user picks none of them — the one choice at "New deck > Score sheets"
// is to want the deck or not.
func UnorderedScoreKeys() []string {
	keys := make([]string, 0, 36)
	for a := ScoreAwayMin; a <= ScoreAwayMax; a++ {
		for b := a; b <= ScoreAwayMax; b++ {
			keys = append(keys, ScoreKey(a, b))
		}
	}
	return keys
}
