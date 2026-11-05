package changeset

import "testing"

// A last review written in either layout orders the same on both backends.
func TestCardAheadReadsEitherLayout(t *testing.T) {
	later := cardPayload{LastReview: "2026-11-05T10:00:00Z", Reps: 1}
	earlier := cardPayload{LastReview: "2026-11-05 09:00:00", Reps: 5}
	if later.ahead(earlier) <= 0 || earlier.ahead(later) >= 0 {
		t.Errorf("a review at 10:00 is not ahead of one at 09:00 across layouts")
	}
}
