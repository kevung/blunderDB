package engine

import (
	"math/rand/v2"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The binary state reads back the board the compact array did, for any
// count a point can hold.
func TestBoardStateRoundTrip(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for range 2000 {
		var b domain.Board
		for i := 0; i < domain.NumPoints+2; i++ {
			if n := r.IntN(31) - 15; n > 0 {
				b.Points[i] = domain.Point{Checkers: n, Color: domain.White}
			} else if n < 0 {
				b.Points[i] = domain.Point{Checkers: -n, Color: domain.Black}
			}
		}
		b.Bearoff = [2]int{r.IntN(16), r.IntN(16)}
		bin := string(EncodeBoardState(b))
		if !IsCompactState(bin) {
			t.Fatalf("binary state %x not recognised", bin)
		}
		if got, want := DecodeBoardCompact(bin), DecodeBoardCompact(EncodeBoardCompact(b)); got != want {
			t.Fatalf("binary %v, compact %v", got, want)
		}
	}
}
