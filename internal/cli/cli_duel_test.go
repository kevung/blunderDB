package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/duel"
)

// Each call is its own process: a Duel is created by one, played by the next,
// and thrown away by a third, with nothing but the database between them.
func TestCLI_DuelOneActionPerCall(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	run := func(args ...string) string {
		t.Helper()
		return captureStdout(t, func() {
			if err := cli.Run(append([]string{"duel"}, args...)); err != nil {
				t.Fatalf("duel %v: %v", args, err)
			}
		})
	}

	var st duel.State
	if err := json.Unmarshal([]byte(run("create", "--db", dbPath, "--length", "3", "--name1", "Alice", "--name2", "Bob", "--format", "json")), &st); err != nil {
		t.Fatal(err)
	}
	if st.Awaiting == nil || st.Awaiting.Kind != duel.DecideMove {
		t.Fatalf("awaiting %+v, want the opening move", st.Awaiting)
	}

	shown := run("show", "--db", dbPath, "--id", "1")
	if !strings.Contains(shown, "Legal plays") || !strings.Contains(shown, "Alice") {
		t.Fatalf("show:\n%s", shown)
	}
	// The first legal play listed by show is accepted back as typed.
	lines := strings.Split(shown, "\n")
	var play string
	for i, l := range lines {
		if strings.HasPrefix(l, "Dice:") {
			play = strings.TrimSpace(lines[i+1])
			break
		}
	}
	if play == "" {
		t.Fatalf("no legal play in:\n%s", shown)
	}
	if out := run("move", "--db", dbPath, "--id", "1", "--play", play); !strings.Contains(out, "Actions:") {
		t.Fatalf("move:\n%s", out)
	}
	if out := run("list", "--db", dbPath); !strings.Contains(out, "Alice") {
		t.Errorf("list:\n%s", out)
	}
	if err := cli.Run([]string{"duel", "move", "--db", dbPath, "--id", "1", "--play", "24/1"}); err == nil {
		t.Error("an illegal play was accepted")
	}
	if out := run("discard", "--db", dbPath, "--id", "1"); !strings.Contains(out, "thrown away") {
		t.Errorf("discard:\n%s", out)
	}
	if out := run("list", "--db", dbPath); !strings.Contains(out, "No Duel") {
		t.Errorf("list after discard:\n%s", out)
	}
}

func TestCLI_DuelBotSideNotOfferedYet(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	if err := cli.Run([]string{"duel", "create", "--db", dbPath, "--length", "1", "--side2", "bot:2-ply"}); err == nil {
		t.Error("a bot side was created though nothing resolves it")
	}
}
