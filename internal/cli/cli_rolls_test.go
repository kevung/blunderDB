package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

const rollsTestXGID = "XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:0:10"

func TestRunRolls_TextAndJSON(t *testing.T) {
	var out bytes.Buffer
	cli := &CLI{out: &out}
	if err := cli.runRolls([]string{"--ply", "0", rollsTestXGID}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "6-6") || !strings.Contains(out.String(), "Average") {
		t.Fatalf("text output:\n%s", out.String())
	}

	out.Reset()
	if err := cli.runRolls([]string{"--ply", "0", "--format", "json", "--roll", "31", "--move", "8/5 6/5", rollsTestXGID}); err != nil {
		t.Fatal(err)
	}
	var m gammonnet.RollMap
	if err := json.Unmarshal(out.Bytes(), &m); err != nil || len(m.Rows) != gammonnet.NumRolls {
		t.Fatalf("json output (%v):\n%s", err, out.String())
	}
}

func TestRunRolls_Refusals(t *testing.T) {
	cli := &CLI{out: &bytes.Buffer{}}
	for _, args := range [][]string{
		{"--ply", "3", rollsTestXGID},
		{"--move", "8/5 6/5", rollsTestXGID},
		{"--roll", "37", rollsTestXGID},
		{"--roll", "31", "--move", "24/18", rollsTestXGID},
	} {
		if err := cli.runRolls(args); err == nil {
			t.Errorf("rolls %v: accepted", args)
		}
	}
}
