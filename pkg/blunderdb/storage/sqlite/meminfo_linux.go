//go:build linux

package sqlite

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// availableMemoryBytes reports MemAvailable from /proc/meminfo; ok is false
// when it cannot be read, in which case the caller skips the check.
func availableMemoryBytes() (bytes uint64, ok bool) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		rest, found := strings.CutPrefix(sc.Text(), "MemAvailable:")
		if !found {
			continue
		}
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			return 0, false
		}
		kb, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			return 0, false
		}
		return kb * 1024, true
	}
	return 0, false
}
