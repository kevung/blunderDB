package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/rollout"
)

func TestRolloutCommandRollsNamedPlays(t *testing.T) {
	cli := &CLI{}
	var runErr error
	out := captureStdout(t, func() {
		runErr = cli.runRollout([]string{"--games", "36", "--truncation", "1", "--jsd", "0", "--format", "json",
			"--move", "8/5 6/5", "--move", "13/10 10/9",
			"XGID=-b----E-C---eE---c-e----B-:0:0:1:31:0:0:0:0:10"})
	})
	if runErr != nil {
		t.Fatal(runErr)
	}
	var res rollout.Result
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if res.Kind != rollout.KindMoves || len(res.Candidates) != 2 || res.Games != 36 {
		t.Fatalf("kind %s, %d candidates, %d games", res.Kind, len(res.Candidates), res.Games)
	}
	if res.Candidates[0].Move != "8/5 6/5" {
		t.Fatalf("best %q, want 8/5 6/5", res.Candidates[0].Move)
	}
}

func TestRolloutCommandRefusesAnIllegalPlay(t *testing.T) {
	err := (&CLI{}).runRollout([]string{"--games", "36", "--move", "24/1",
		"XGID=-b----E-C---eE---c-e----B-:0:0:1:31:0:0:0:0:10"})
	if err == nil || !strings.Contains(err.Error(), "not a legal play") {
		t.Fatalf("err %v, want a refusal naming the play", err)
	}
}
