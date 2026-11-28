package progress

import (
	"context"
	"testing"
	"time"
)

func TestReportWithoutReporterIsANoop(t *testing.T) {
	Report(context.Background(), 1, 2)
}

func TestReportReachesTheReporter(t *testing.T) {
	var got [2]int
	ctx := With(context.Background(), func(d, n int) { got = [2]int{d, n} })
	Report(ctx, 3, 7)
	if got != [2]int{3, 7} {
		t.Fatalf("got %v", got)
	}
}

func TestThrottleKeepsFirstLastAndTotalChanges(t *testing.T) {
	var calls [][2]int
	f := Throttle(time.Hour, func(d, n int) { calls = append(calls, [2]int{d, n}) })
	for i := 0; i <= 100; i++ {
		f(i, 100)
	}
	f(0, 5)
	want := [][2]int{{0, 100}, {100, 100}, {0, 5}}
	if len(calls) != len(want) {
		t.Fatalf("calls %v, want %v", calls, want)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("calls %v, want %v", calls, want)
		}
	}
}
