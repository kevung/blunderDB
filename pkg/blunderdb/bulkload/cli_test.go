package bulkload

import (
	"strconv"
	"strings"
	"testing"
)

// A limit that would overflow MiB << 20 or that is negative is refused before
// anything is read, not turned into a silent "no limit".
func TestRunCLIRefusesAnUnusableMemoryLimit(t *testing.T) {
	for _, v := range []int64{-1, 1 << 44, MaxMemoryLimitMiB + 1} {
		err := RunCLI([]string{"--dsn", "postgres://x@localhost/x", "--source", ".", "--tenant", "1", "--memory-limit-mb", strconv.FormatInt(v, 10)})
		if err == nil || !strings.Contains(err.Error(), "memory-limit-mb") {
			t.Errorf("--memory-limit-mb %d: %v", v, err)
		}
	}
}
