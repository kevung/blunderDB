package main

import (
	"os"
	"strconv"
	"syscall"
)

// lowerPriority renices every thread of the process: on Linux a nice value
// belongs to a thread, and the threads the runtime starts later inherit it
// from the one that clones them.
func lowerPriority() error {
	tasks, err := os.ReadDir("/proc/self/task")
	if err != nil {
		return syscall.Setpriority(syscall.PRIO_PROCESS, 0, 19)
	}
	var first error
	for _, t := range tasks {
		tid, err := strconv.Atoi(t.Name())
		if err != nil {
			continue
		}
		if err := syscall.Setpriority(syscall.PRIO_PROCESS, tid, 19); err != nil && first == nil {
			first = err
		}
	}
	return first
}
