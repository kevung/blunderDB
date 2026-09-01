package domain

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// xgidContractCase mirrors one entry of testdata/xgid_corpus.json. The corpus
// is shared with the GUI: this side decodes `xgid` and must produce exactly
// `position` and `pips` (and the board part of `xgidCanonical`); the GUI side
// (frontend/src/__tests__/xgidContract.test.js) starts from that same
// `position` and must re-encode it to `xgidCanonical` (generateXGID) and count
// `pips` (computePipCount). The GUI has no XGID parser any more (commit
// cd33de85), so `position` is the hand-off point between the two halves.
// The flat fields restate `position` for readability and are cross-checked
// against it. See the corpus _comment for conventions and the deliberately
// lossy re-encoding.
type xgidContractCase struct {
	Name          string   `json:"name"`
	XGID          string   `json:"xgid"`
	CubeOwner     int      `json:"cubeOwner"`
	CubeValueExp  int      `json:"cubeValueExp"`
	Dice          [2]int   `json:"dice"`
	PlayerOnRoll  int      `json:"playerOnRoll"`
	Score         [2]int   `json:"score"`
	HasJacoby     int      `json:"hasJacoby"`
	HasBeaver     int      `json:"hasBeaver"`
	Pips          [2]int   `json:"pips"`
	XGIDCanonical string   `json:"xgidCanonical"`
	Position      Position `json:"position"`
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

			// The flat fields are what the corpus reader sees first; they
			// must agree with the decoder and with the `position` document.
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

			// `position` is the exact document the GUI test starts from: the
			// decoder must reproduce it field for field (board, bearoff, cube,
			// dice, score, player on roll, decision type, jacoby, beaver).
			// ID and the provenance flags are not part of an XGID.
			want := c.Position
			want.ID, want.IndividuallyImported, want.Flagged = pos.ID, pos.IndividuallyImported, pos.Flagged
			if !reflect.DeepEqual(pos, want) {
				t.Errorf("position: decoded\n%+v\ndiffers from corpus\n%+v", pos, want)
			}

			p1, p2 := pos.ComputePipCounts()
			if got := [2]int{p1, p2}; got != c.Pips {
				t.Errorf("pips: got %v, want %v", got, c.Pips)
			}

			// The board is the only part of the XGID this package re-encodes;
			// the GUI test covers the full canonical string.
			board := strings.SplitN(c.XGIDCanonical, ":", 2)[0]
			if got := EncodeXGIDBoard(&pos); got != board {
				t.Errorf("EncodeXGIDBoard: got %q, want %q (board of xgidCanonical)", got, board)
			}
		})
	}
}
