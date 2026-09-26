package sqlshared

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The regime is tested twice — once in Go by domain.Position.IsMoney, once in
// SQL by Similar's WHERE clause — and this holds the two forms together.
// Two spellings of "money or match" can agree on a clean score and part on a
// malformed one, silently splitting a class in two.
func TestSimilarRegimePredicateMatchesIsMoney(t *testing.T) {
	// The SQL is `score_1 < 0 AND score_2 < 0` for money, and its negation
	// `score_1 >= 0 OR score_2 >= 0` for a match score. Evaluated here on the
	// same scores domain.Position.IsMoney sees.
	sqlIsMoney := func(s1, s2 int) bool { return s1 < 0 && s2 < 0 }
	sqlIsMatch := func(s1, s2 int) bool { return s1 >= 0 || s2 >= 0 }

	scores := [][2]int{
		{-1, -1}, // money, as every writer spells it
		{0, 0},   // a match at 0-0, which is NOT money
		{3, 5},
		{-1, 3}, // malformed: half money, half score
		{3, -1},
		{-2, -7}, // a sentinel some importer wrote wider
	}
	for _, sc := range scores {
		p := domain.Position{Score: sc}
		if got, want := sqlIsMoney(sc[0], sc[1]), p.IsMoney(); got != want {
			t.Errorf("score %v: SQL says money=%v, domain.Position.IsMoney says %v", sc, got, want)
		}
		if sqlIsMatch(sc[0], sc[1]) == sqlIsMoney(sc[0], sc[1]) {
			t.Errorf("score %v: the two SQL branches must partition the library, not overlap or leave a gap", sc)
		}
	}
}
