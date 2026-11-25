package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

func TestRunScoreMap(t *testing.T) {
	var out bytes.Buffer
	cli := &CLI{out: &out}
	withDice := "XGID=-b----E-C---eE---c-e----B-:0:0:1:31:6:4:0:9:10"
	if err := cli.runScoreMap([]string{"--match-length", "3", "--ply", "0", withDice}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "3/3") {
		t.Fatalf("text output:\n%s", out.String())
	}

	out.Reset()
	if err := cli.runScoreMap([]string{"--match-length", "3", "--ply", "0", "--format", "json", rollsTestXGID}); err != nil {
		t.Fatal(err)
	}
	var m gammonnet.ScoreMap
	if err := json.Unmarshal(out.Bytes(), &m); err != nil || m.Decision != "cube" || len(m.Cells) != 9 {
		t.Fatalf("json output (%v):\n%s", err, out.String())
	}

	for _, args := range [][]string{
		{"--match-length", "8", withDice},
		{"--ply", "3", withDice},
	} {
		if err := cli.runScoreMap(args); err == nil {
			t.Errorf("scoremap %v: accepted", args)
		}
	}
}
