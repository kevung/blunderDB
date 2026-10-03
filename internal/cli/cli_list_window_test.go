package cli

import (
	"strings"
	"testing"
)

// --offset windows the text listing of positions; given with another type it
// is refused rather than silently ignored. The check runs before the database
// is opened, so no file is needed.
func TestListRefusesOffsetOutsidePositions(t *testing.T) {
	for _, args := range [][]string{
		{"--type", "matches", "--offset", "5"},
		{"--type", "positions", "--format", "csv", "--offset", "5"},
	} {
		err := NewCLI().Run(append([]string{"list", "--db", "unused.db"}, args...))
		if err == nil || !strings.Contains(err.Error(), "--offset") {
			t.Errorf("list %v: err = %v, want a refusal of --offset", args, err)
		}
	}
}
