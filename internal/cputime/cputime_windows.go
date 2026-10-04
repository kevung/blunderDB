//go:build windows

package cputime

import (
	"syscall"
	"time"
)

func process() (time.Duration, bool) {
	h, err := syscall.GetCurrentProcess()
	if err != nil {
		return 0, false
	}
	var creation, exit, kernel, user syscall.Filetime
	if err := syscall.GetProcessTimes(h, &creation, &exit, &kernel, &user); err != nil {
		return 0, false
	}
	return ticks(kernel) + ticks(user), true
}

// ticks reads a FILETIME holding a duration, counted in 100 ns intervals.
// Filetime.Nanoseconds would subtract the 1601 epoch, which only a date has.
func ticks(f syscall.Filetime) time.Duration {
	return time.Duration(int64(f.HighDateTime)<<32|int64(f.LowDateTime)) * 100
}
