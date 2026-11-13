package searchquery

import (
	"errors"
	"testing"
)

// TestLivingRefusesLike pins that a living collection never reads `like`: not
// the ranked form, which has no meaning for a set, nor the starred one, which
// skipped would widen the collection to the whole library.
func TestLivingRefusesLike(t *testing.T) {
	t.Parallel()
	for query, want := range map[string]error{
		"s like":             ErrLikeUnranked,
		"s like42<2 nc":      ErrLikeUnranked,
		"s like*":            ErrRefused,
		"s like42<12* w>50":  ErrRefused,
		"s like42* like<1":   ErrRefused,
		"s nc w>50 like<12*": ErrRefused,
	} {
		_, living, err := Living(7, query)
		if !errors.Is(err, want) || living {
			t.Errorf("Living(%q) = living %v, err %v; want %v", query, living, err, want)
		}
	}
	if _, living, err := Living(7, "s nc w>50"); err != nil || !living {
		t.Errorf("Living of a plain query: living %v, err %v", living, err)
	}
}
