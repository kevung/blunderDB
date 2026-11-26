package parallel

import (
	"sync/atomic"
	"testing"
)

func TestForCoversEachIndexOnce(t *testing.T) {
	for _, c := range []struct{ n, grain int }{{0, 1}, {1, 1}, {7, 1}, {1000, 3}, {1000, 2000}} {
		seen := make([]int32, c.n)
		For(c.n, c.grain, func(lo, hi int) {
			for i := lo; i < hi; i++ {
				atomic.AddInt32(&seen[i], 1)
			}
		})
		for i, v := range seen {
			if v != 1 {
				t.Fatalf("n %d grain %d: index %d run %d times", c.n, c.grain, i, v)
			}
		}
	}
}

func TestForRaisesAPanicOnTheCaller(t *testing.T) {
	defer func() {
		if e := recover(); e != "boom" {
			t.Errorf("recovered %v", e)
		}
	}()
	For(1000, 1, func(lo, hi int) {
		if lo == 0 {
			panic("boom")
		}
	})
	t.Error("no panic")
}
