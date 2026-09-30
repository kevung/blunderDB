package direction

import (
	"regexp"
	"slices"
	"strconv"
	"testing"
	"time"
)

// Reloading restarts the CSS animation, so the delays must come from the wall clock: two renders
// at different instants put the same view at the same point of the cycle as real time says.
func TestRotation_DelaysFollowTheWallClock(t *testing.T) {
	delays := func(now time.Time) []int {
		_, markup := rotation([]string{"a", "b", "c"}, now)
		var out []int
		for _, m := range regexp.MustCompile(`animation-delay:(-?\d+)s`).FindAllStringSubmatch(markup, -1) {
			n, _ := strconv.Atoi(m[1])
			out = append(out, n)
		}
		return out
	}
	// Cycle = 3 views x 12 s = 36 s. At 0 s the first view has just begun.
	if got := delays(time.Unix(36*1000, 0)); !slices.Equal(got, []int{0, -24, -12}) {
		t.Errorf("at cycle start: %v", got)
	}
	// 14 s later the first view is 14 s in (its turn is over), the second 2 s in.
	if got := delays(time.Unix(36*1000+14, 0)); !slices.Equal(got, []int{-14, -2, -26}) {
		t.Errorf("14 s into the cycle: %v", got)
	}
	// A render a whole number of cycles later is identical.
	if !slices.Equal(delays(time.Unix(36*7+5, 0)), delays(time.Unix(36*9+5, 0))) {
		t.Error("delays are not periodic in the cycle")
	}
}
