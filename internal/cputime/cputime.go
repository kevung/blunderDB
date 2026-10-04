// Package cputime reads the processor time the current process has consumed,
// for the tests that guard a cost.
//
// A wall clock measures what a loaded machine makes of a computation: when
// `go test ./...` runs a heavy package beside the one measuring, every
// scheduler stall lands inside the measure and a fixed ceiling fails without
// any change in the code. The processor time the process actually received
// does not grow while it waits for a core, so a guard stated on it fails on a
// costlier computation, not on a busier machine. It still counts every thread
// of the process: a computation that fans out to more cores costs more, not
// less.
package cputime

import "time"

// Process returns the user and system time consumed so far by every thread of
// the current process. ok is false where the platform does not expose it; a
// caller then falls back to the wall clock.
func Process() (spent time.Duration, ok bool) {
	return process()
}

// Clock measures a stretch of work: processor time where the platform has it,
// wall time otherwise.
type Clock struct {
	// CPU is true when the readings are processor time.
	CPU   bool
	start time.Duration
	wall  time.Time
}

// Start begins a measure.
func Start() Clock {
	if spent, ok := process(); ok {
		return Clock{CPU: true, start: spent}
	}
	return Clock{wall: time.Now()}
}

// Elapsed returns what the work has cost since Start.
func (c Clock) Elapsed() time.Duration {
	if c.CPU {
		if spent, ok := process(); ok {
			return spent - c.start
		}
	}
	return time.Since(c.wall)
}

// Unit names what Elapsed measures, for a test's log.
func (c Clock) Unit() string {
	if c.CPU {
		return "CPU time"
	}
	return "wall time"
}
