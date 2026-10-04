package cputime

import (
	"testing"
	"time"
)

// A sleep is not charged to the process, and a spin is:
// the reading tracks work done, not time passed.
func TestClockChargesWorkNotWaiting(t *testing.T) {
	c := Start()
	if !c.CPU {
		t.Skip("no processor time on this platform")
	}
	time.Sleep(200 * time.Millisecond)
	if slept := c.Elapsed(); slept > 100*time.Millisecond {
		t.Errorf("a 200ms sleep cost %v of processor time", slept)
	}
	// Spin until the reading moves rather than for a wall length: on a loaded
	// machine a spin of fixed wall length may receive little processor time.
	c = Start()
	for deadline := time.Now().Add(10 * time.Second); c.Elapsed() < 100*time.Millisecond; {
		if time.Now().After(deadline) {
			t.Fatalf("10s of spinning charged only %v of processor time", c.Elapsed())
		}
	}
}
