package domain

import (
	"strings"
	"testing"
)

// FuzzDecodeXGID exercises the XGID parser against arbitrary input. XGIDs come
// from untrusted sources (clipboard paste, file/HTTP import), so the contract is
// strict: DecodeXGID must NEVER panic, whatever bytes it is handed. When it does
// return a position without error, the decoded board must round-trip back into a
// well-formed 26-char board string via EncodeXGIDBoard.
func FuzzDecodeXGID(f *testing.F) {
	seeds := []string{
		"",
		"XGID=",
		"XGID=a--aB-BBA--acDa-Ab-db---BA:0:0:1:64:2:0:0:13:10",
		"XGID=bA----D-C---dE---d-e----B-:0:0:1:00:0:0:3:0:10",
		"XGID=---BBaB-BbA-bC-b--BdAca---:0:0:1:00:0:5:0:9:10",
		"a--aB-BBA--acDa-Ab-db---BA:0:0:1:64:2:0:0:13:10",
		"XGID=:::::::::",
		"XGID=" + strings.Repeat("Z", 26) + ":0:0:1:00:0:0:0:0:10",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, xgid string) {
		pos, err := DecodeXGID(xgid)
		if err != nil {
			return
		}
		board := EncodeXGIDBoard(&pos)
		if len(board) != 26 {
			t.Fatalf("DecodeXGID accepted %q but EncodeXGIDBoard produced a %d-char board (want 26): %q",
				xgid, len(board), board)
		}
	})
}
