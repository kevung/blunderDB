package cli

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/duel"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
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

// Two Bots play the whole match in the call that creates the Duel; a money
// session between them would never end and is refused.
func TestCLI_DuelOfTwoBots(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	var st duel.State
	out := captureStdout(t, func() {
		if err := cli.Run([]string{"duel", "create", "--db", dbPath, "--length", "1", "--side1", "bot:instant", "--side2", "bot:instant", "--format", "json"}); err != nil {
			t.Fatalf("duel create: %v", err)
		}
	})
	if err := json.Unmarshal([]byte(out), &st); err != nil {
		t.Fatal(err)
	}
	if st.Ended == nil || st.Ended.MatchID == 0 || st.Ended.DiceSeed == "" {
		t.Fatalf("a Duel of two Bots ends in the call that creates it: %+v", st.Ended)
	}
	err := cli.Run([]string{"duel", "create", "--db", dbPath, "--money", "--side1", "bot:instant", "--side2", "bot:instant"})
	if err == nil || !strings.Contains(err.Error(), "never ends") {
		t.Errorf("a money session between two Bots: %v", err)
	}
	if err := cli.Run([]string{"duel", "create", "--db", dbPath, "--length", "1", "--side1", "bot:bogus"}); err == nil {
		t.Error("an unknown bot level was accepted")
	}
}

// --revision refuses an Action on a Duel that moved since it was read.
func TestCLI_DuelStaleRevisionIsRefused(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	if err := cli.Run([]string{"duel", "create", "--db", dbPath, "--length", "3"}); err != nil {
		t.Fatal(err)
	}
	err := cli.Run([]string{"duel", "discard", "--db", dbPath, "--id", "1", "--revision", "99"})
	if err == nil || !strings.Contains(err.Error(), "revision") {
		t.Errorf("discard on a stale revision: %v", err)
	}
}

// `match --id` reads back the origin of a Match a Duel became: its seed gives
// the fingerprint published at creation, in every format.
func TestCLI_MatchShowsItsOrigin(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	var st duel.State
	out := captureStdout(t, func() {
		if err := cli.Run([]string{"duel", "create", "--db", dbPath, "--length", "1", "--side1", "bot:instant", "--side2", "bot:instant", "--format", "json"}); err != nil {
			t.Fatalf("duel create: %v", err)
		}
	})
	if err := json.Unmarshal([]byte(out), &st); err != nil || st.Ended == nil {
		t.Fatalf("duel create: %v, %+v", err, st.Ended)
	}
	id := strconv.FormatInt(st.Ended.MatchID, 10)

	var got struct {
		Origin *duel.Origin `json:"origin"`
	}
	out = captureStdout(t, func() {
		if err := cli.Run([]string{"match", "--db", dbPath, "--id", id, "--format", "json"}); err != nil {
			t.Fatalf("match json: %v", err)
		}
	})
	if err := json.Unmarshal([]byte(out), &got); err != nil || got.Origin == nil {
		t.Fatalf("match json origin: %v, %+v", err, got.Origin)
	}
	if fp, _ := duel.Fingerprint(got.Origin.DiceSeed); fp != st.Fingerprint || got.Origin.Fingerprint != st.Fingerprint {
		t.Errorf("seed read back gives %q, origin says %q, published %q", fp, got.Origin.Fingerprint, st.Fingerprint)
	}

	for _, format := range []string{"text", "summary"} {
		out = captureStdout(t, func() {
			if err := cli.Run([]string{"match", "--db", dbPath, "--id", id, "--format", format}); err != nil {
				t.Fatalf("match %s: %v", format, err)
			}
		})
		for _, want := range []string{"Origin: played here", "Dice seed: " + st.Ended.DiceSeed, "SHA-256 of the seed: " + st.Fingerprint, "Bot: level instant"} {
			if !strings.Contains(out, want) {
				t.Errorf("match --format %s lacks %q:\n%s", format, want, out)
			}
		}
	}
}

// A match lost on time is told apart from one stopped by hand, and a match
// not played here says so.
func TestCLI_MatchOriginEndings(t *testing.T) {
	m := &Match{Player1Name: "Alice", Player2Name: "Bob"}
	lose := &duel.Cadence{Reserve: 60, TimeOut: duel.TimeLoseMatch}
	cases := []struct {
		origin *duel.Origin
		want   string
		absent string
	}{
		{nil, "Origin: not played here", "Ended:"},
		{&duel.Origin{MatchOrigin: storage.MatchOrigin{StoppedEarly: true, OverTime: 2}, CadenceSettings: lose, LostOnTime: true},
			"Ended: lost on time (Bob)", "stopped before the end"},
		{&duel.Origin{MatchOrigin: storage.MatchOrigin{StoppedEarly: true, OverTime: 1}},
			"Ended: stopped before the end", "lost on time"},
	}
	for _, c := range cases {
		var sb strings.Builder
		writeOrigin(&sb, m, c.origin)
		if out := sb.String(); !strings.Contains(out, c.want) || strings.Contains(out, c.absent) {
			t.Errorf("origin %+v: got\n%s\nwant %q, not %q", c.origin, out, c.want, c.absent)
		}
	}
}
