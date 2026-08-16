package storage

import "testing"

// TestTallyCubeDirections is the whole point of keeping this classification in
// one pure function: both backends read the same raw (best, played) cells and
// must sort them identically. The retention predicate of this repo already
// lives in three places and has to be kept in sync by hand — this one does not.
func TestTallyCubeDirections(t *testing.T) {
	got := TallyCubeDirections([]CubeDirectionRow{
		// ── Offer axis ───────────────────────────────────────────────────────
		{Best: "No Double", Played: "No Double", Count: 10},
		{Best: "Double, Pass", Played: "Double/Pass", Count: 4},
		{Best: "Double, Take", Played: "Double", Count: 3},
		{Best: "Double, Take", Played: "No Double", Count: 5, ErrorMP: 300},
		{Best: "No Double", Played: "Double", Count: 2, ErrorMP: 200},
		// "Double No" is the importer's other spelling of a no-double: it must
		// land on the same side as "No Double", not on the doubling side.
		{Best: "Double, Take", Played: "Double No", Count: 1, ErrorMP: 60},

		// ── Answer axis ──────────────────────────────────────────────────────
		{Best: "Double, Pass", Played: "Pass", Count: 6},
		// best "No Double" still rules the answer: taking is right.
		{Best: "No Double", Played: "Take", Count: 7},
		{Best: "Double, Take", Played: "Pass", Count: 2, ErrorMP: 150},
		{Best: "Double, Pass", Played: "Take", Count: 1, ErrorMP: 90},

		// ── Ignored: nothing can be concluded ────────────────────────────────
		{Best: "", Played: "No Double", Count: 9, ErrorMP: 999},
		{Best: "No Double", Played: "garbage", Count: 8, ErrorMP: 888},
	})

	want := CubeDirections{
		Offer: CubeOfferCounts{
			Right:       17, // 10 + 4 + 3
			Missed:      6,  // 5 + 1 ("Double No")
			MissedMP:    360,
			Premature:   2,
			PrematureMP: 200,
		},
		Answer: CubeAnswerCounts{
			Right:       13, // 6 + 7
			WrongPass:   2,
			WrongPassMP: 150,
			WrongTake:   1,
			WrongTakeMP: 90,
		},
	}
	if got != want {
		t.Errorf("TallyCubeDirections()\n got %+v\nwant %+v", got, want)
	}
}

// A "Too good to double" verdict rules against doubling: not doubling is right,
// doubling is premature. It is the label that contains "double" while meaning
// the opposite, so it gets its own case rather than riding on the one above.
func TestTallyCubeDirections_TooGood(t *testing.T) {
	got := TallyCubeDirections([]CubeDirectionRow{
		{Best: "Too good to double, take", Played: "No Double", Count: 5},
		{Best: "Too good to double, take", Played: "Double", Count: 2, ErrorMP: 400},
	})
	if got.Offer.Right != 5 || got.Offer.Premature != 2 || got.Offer.PrematureMP != 400 {
		t.Errorf("too-good offer counts = %+v", got.Offer)
	}
	if got.Offer.Missed != 0 {
		t.Errorf("too-good must never count as a missed double, got %d", got.Offer.Missed)
	}
}
