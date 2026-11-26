package jpx

import "testing"

// raceEnabled is set by race_on_test.go.
var raceEnabled bool

// slack scales a stop-latency bound. The tests prove that a cancelled decode
// stops in bounded time rather than running on for minutes; the race
// detector, coverage instrumentation and a loaded CI machine slow it by an
// order of magnitude, so the bound stretches while an uncancellable loop
// still overruns it.
func slack() int {
	if raceEnabled || testing.CoverMode() != "" {
		return 10
	}
	return 1
}
