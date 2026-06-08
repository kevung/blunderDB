package domain

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// xgidContractCase mirrors one entry of testdata/xgid_corpus.json. The same
// corpus is asserted by the GUI parser in
// frontend/src/__tests__/xgidContract.test.js, so any drift between the Go
// decoder and the GUI clipboard parser fails a test on at least one side.
// See the corpus file's _comment for conventions and intentionally excluded
// fields (decision_type) / edge cases (turn=0, match away score of 1).
type xgidContractCase struct {
	Name         string `json:"name"`
	XGID         string `json:"xgid"`
	CubeOwner    int    `json:"cubeOwner"`
	CubeValueExp int    `json:"cubeValueExp"`
	Dice         [2]int `json:"dice"`
	PlayerOnRoll int    `json:"playerOnRoll"`
	Score        [2]int `json:"score"`
	HasJacoby    int    `json:"hasJacoby"`
	HasBeaver    int    `json:"hasBeaver"`
}

func TestDecodeXGIDContract(t *testing.T) {
	// domain test cwd is the package dir; the shared corpus lives at repo root.
	path := filepath.Join("..", "..", "..", "testdata", "xgid_corpus.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read corpus %s: %v", path, err)
	}
	var corpus struct {
		Cases []xgidContractCase `json:"cases"`
	}
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatalf("parse corpus: %v", err)
	}
	if len(corpus.Cases) == 0 {
		t.Fatal("corpus has no cases")
	}

	for _, c := range corpus.Cases {
		t.Run(c.Name, func(t *testing.T) {
			pos, err := DecodeXGID(c.XGID)
			if err != nil {
				t.Fatalf("DecodeXGID(%q): %v", c.XGID, err)
			}
			if pos.Cube.Owner != c.CubeOwner {
				t.Errorf("cubeOwner: got %d, want %d", pos.Cube.Owner, c.CubeOwner)
			}
			if pos.Cube.Value != c.CubeValueExp {
				t.Errorf("cubeValueExp: got %d, want %d", pos.Cube.Value, c.CubeValueExp)
			}
			if pos.Dice != c.Dice {
				t.Errorf("dice: got %v, want %v", pos.Dice, c.Dice)
			}
			if pos.PlayerOnRoll != c.PlayerOnRoll {
				t.Errorf("playerOnRoll: got %d, want %d", pos.PlayerOnRoll, c.PlayerOnRoll)
			}
			if pos.Score != c.Score {
				t.Errorf("score: got %v, want %v", pos.Score, c.Score)
			}
			if pos.HasJacoby != c.HasJacoby {
				t.Errorf("hasJacoby: got %d, want %d", pos.HasJacoby, c.HasJacoby)
			}
			if pos.HasBeaver != c.HasBeaver {
				t.Errorf("hasBeaver: got %d, want %d", pos.HasBeaver, c.HasBeaver)
			}
		})
	}
}
