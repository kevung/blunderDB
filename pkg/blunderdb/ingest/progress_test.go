package ingest

import (
	"testing"
	"time"
)

// The meter emits at most once per interval however many files go by, and
// always once at the end.
func TestProgressMeter_BoundedCadence(t *testing.T) {
	clock := time.Unix(0, 0)
	var got []BatchProgress
	m := NewProgressMeter(10000, 10000, ProgressInterval, func(p BatchProgress) { got = append(got, p) })
	m.now = func() time.Time { return clock }
	m.start, m.last = clock, clock
	for i := 0; i < 10000; i++ {
		clock = clock.Add(time.Millisecond) // 10 s in all
		m.Read(1)
		m.File(FileOutcome{Status: FileImported, Positions: 2})
	}
	final := m.Finish()
	if len(got) > 41+1 {
		t.Errorf("%d emissions over 10 s, want at most 4 per second", len(got))
	}
	if !final.Done || final.FilesDone != 10000 || final.Positions != 20000 || final.BytesRead != 10000 {
		t.Errorf("final = %+v", final)
	}
	if got[len(got)-1] != final {
		t.Error("the final state was not emitted")
	}
	if final.ETASeconds != 0 {
		t.Errorf("ETA at the end = %v, want 0", final.ETASeconds)
	}
}
