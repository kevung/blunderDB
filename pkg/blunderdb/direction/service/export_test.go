package service

import "testing"

// ObserveSimulation calls observe on every event of the runs simulated until the test ends.
func ObserveSimulation(t testing.TB, observe func()) {
	simulateObserve = observe
	t.Cleanup(func() { simulateObserve = nil })
}
