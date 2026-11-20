package sqlshared

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The least pips of a band count the checkers the masks hold on each point of
// the template — two at most — and the farthest checker of the band where the
// template puts none.
func TestBlackFloorBands(t *testing.T) {
	var b domain.Board
	b.Points[5] = domain.Point{Checkers: 2, Color: domain.Black}
	b.Points[8] = domain.Point{Checkers: 3, Color: domain.Black}
	b.Points[20] = domain.Point{Checkers: 2, Color: domain.White}
	f := newBlackFloor(b)
	if f == nil || f.top != 8 || f.pips != 26 {
		t.Fatalf("floor %+v, want top 8 and 26 pips", f)
	}
	for band, want := range map[int]int{9: 26, 10: 35, 26: 51} {
		if got := f.band(band); got != want {
			t.Errorf("band %d: least pips %d, want %d", band, got, want)
		}
	}
	var white domain.Board
	white.Points[20] = domain.Point{Checkers: 2, Color: domain.White}
	if newBlackFloor(white) != nil {
		t.Error("a structure without a black checker bounds no band")
	}
}

// An away bound past the longest match admitted keeps nothing; one within it,
// or a match length without an upper bound, is left to the search.
func TestAwayBeyondTheMatchLength(t *testing.T) {
	for _, c := range []struct {
		aw, AW string
		length int
		want   bool
	}{
		{"aw:8", "", 7, true},
		{"aw:7", "", 7, false},
		{"", "AW:9", 7, true},
		{"aw>8", "", 7, true},
		{"aw<9", "", 7, false},
		{"", "", 7, false},
	} {
		f := domain.SearchFilters{AwayFilter: c.aw, OpponentAwayFilter: c.AW}
		if got := awayBeyond(f, c.length); got != c.want {
			t.Errorf("aw %q AW %q in %d points: %v, want %v", c.aw, c.AW, c.length, got, c.want)
		}
	}
}
