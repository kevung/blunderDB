//go:build !unix && !windows

package cputime

import "time"

func process() (time.Duration, bool) { return 0, false }
