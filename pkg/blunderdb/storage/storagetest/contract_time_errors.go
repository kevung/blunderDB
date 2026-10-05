package storagetest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testStatsTimeErrors pins the duration bands of the time/error table for the
// players of a Match written the way a Duel writes it: each player under its
// own name, an unknown duration in no band, a cube decision added to the play
// after it, and an unscored decision counted but never averaged as zero.
func testStatsTimeErrors(t *testing.T, s storage.Storage) {
	ms := func(v int64) *int64 { return &v }
	timedMatch(t, s, [][2]*int64{
		{ms(3000), ms(3000)}, // player 1: 6 s, band 1
		{ms(40000), nil},     // player 2: band 3
		{nil, nil},           // player 1: unknown, no band
		{ms(1000), nil},      // player 2: band 0
	})
	got, err := s.Stats().TimeErrors(context.Background(), "")
	if err != nil {
		t.Fatalf("TimeErrors: %v", err)
	}
	want := []storage.TimeErrorRow{
		{Player: "me", Bucket: 1, Decisions: 1},
		{Player: "them", Bucket: 0, Decisions: 1},
		{Player: "them", Bucket: 3, Decisions: 1},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}
