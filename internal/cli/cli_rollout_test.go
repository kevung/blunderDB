package cli

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
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

// --store writes the rollout on the position of --db; analyze --rollout then
// finds nothing left to do with the same settings.
func TestRolloutCommandStoresAndAnalyzeResumes(t *testing.T) {
	seed, dbPath := setupCLIWithDB(t)
	p := domain.InitializePosition()
	id, err := seed.db.SavePosition(&p)
	if err != nil {
		t.Fatal(err)
	}
	seed.db.Close() // one writer at a time: the commands below open it
	tiny := []string{"--games", "36", "--min-games", "36", "--truncation", "2", "--candidates", "2"}

	if err := (&CLI{}).runRollout([]string{"--store"}); err == nil {
		t.Error("--store without --db/--id accepted")
	}
	c := &CLI{db: NewDatabase()}
	var runErr error
	out := captureStdout(t, func() {
		runErr = c.runRollout(append(tiny, "--db", dbPath, "--id", strconv.FormatInt(id, 10), "--store"))
	})
	if runErr != nil || !strings.Contains(out, "Stored on position") {
		t.Fatalf("rollout --store: %v\n%s", runErr, out)
	}
	c.db.Close()
	l := &CLI{db: NewDatabase()}
	out = captureStdout(t, func() {
		runErr = l.runRollout([]string{"--db", dbPath, "--id", strconv.FormatInt(id, 10), "--list"})
	})
	l.db.Close()
	if runErr != nil || !strings.Contains(out, "Rollout 36 games") {
		t.Fatalf("rollout --list: %v\n%s", runErr, out)
	}

	a := &CLI{db: NewDatabase()}
	defer a.db.Close()
	out = captureStdout(t, func() {
		runErr = a.runAnalyze([]string{"--db", dbPath, "--rollout", "fast,games=36,min-games=36,truncation=2,candidates=2", "--query", "s"})
	})
	if runErr != nil || !strings.Contains(out, "Nothing to do") {
		t.Fatalf("analyze --rollout over a rolled-out position: %v\n%s", runErr, out)
	}
	if err := (&CLI{db: NewDatabase()}).runAnalyze([]string{"--db", dbPath, "--rollout", "fast", "--stale"}); err == nil {
		t.Error("--rollout with --stale accepted")
	}
}
