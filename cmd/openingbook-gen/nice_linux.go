package main

import (
	"os"
	"strconv"
	"syscall"
	"unsafe"
)

// lowerPriority renices every thread of the process: on Linux a nice value
// belongs to a thread, and the threads the runtime starts later inherit it
// from the one that clones them.
func lowerPriority() error {
	return eachThread(func(tid int) error { return syscall.Setpriority(syscall.PRIO_PROCESS, tid, 19) })
}

// schedIdle is SCHED_IDLE (sched(7)): the thread runs only on a core nothing
// else wants, which a nice value does not promise.
const schedIdle = 5

// idlePriority moves every thread to SCHED_IDLE; threads started later
// inherit the policy from the one that clones them.
func idlePriority() error {
	return eachThread(func(tid int) error {
		var param struct{ priority int32 }
		_, _, errno := syscall.Syscall(syscall.SYS_SCHED_SETSCHEDULER, uintptr(tid), schedIdle, uintptr(unsafe.Pointer(&param)))
		if errno != 0 {
			return errno
		}
		return nil
	})
}

func eachThread(fn func(tid int) error) error {
	tasks, err := os.ReadDir("/proc/self/task")
	if err != nil {
		return fn(0)
	}
	var first error
	for _, t := range tasks {
		tid, err := strconv.Atoi(t.Name())
		if err != nil {
			continue
		}
		if err := fn(tid); err != nil && first == nil {
			first = err
		}
	}
	return first
}
