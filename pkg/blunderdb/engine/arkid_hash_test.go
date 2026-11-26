package engine

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// A position written as an ArkID and read back falls on the same row: the
// same Zobrist hash, whichever player was on roll in the source.
func TestArkIDKeepsTheZobristHash(t *testing.T) {
	for _, xgid := range []string{
		"XGID=-b----E-C---eE---c-e----B-:0:0:1:65:0:0:0:0:10",
		"XGID=-b----E-C---eE---c-e----B-:1:1:-1:31:0:0:3:7:10",
		"XGID=a--aB-BBA--acDa-Ab-db---BA:0:0:1:64:2:0:0:13:10",
		"XGID=a--aB-BBA--acDa-Ab-db---BA:2:-1:-1:00:1:0:0:13:10",
		"XGID=bA----D-C---dE---d-e----B-:0:0:1:00:0:0:3:0:10",
		"XGID=-b----E-C---eE---c-e----B-:0:0:1:42:0:0:1:5:10",
		"XGID=-b----E-C---eE---c-e----B-:1:0:1:00:0:0:0:7:10",
		"XGID=-b----E-C---eE---c-e----B-:2:0:-1:00:0:0:0:7:10",
	} {
		pos, err := domain.DecodeXGID(xgid)
		if err != nil {
			t.Fatalf("%s: %v", xgid, err)
		}
		id, err := domain.EncodeArkID(&pos)
		if err != nil {
			t.Fatalf("%s: %v", xgid, err)
		}
		back, err := domain.DecodeArkID(id)
		if err != nil {
			t.Fatalf("%s -> %q: %v", xgid, id, err)
		}
		if ZobristHash(&back) != ZobristHash(&pos) {
			t.Errorf("%s -> %q: hash changed", xgid, id)
		}
	}
}
