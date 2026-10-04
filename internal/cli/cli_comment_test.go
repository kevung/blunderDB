package cli

import (
	"encoding/json"
	"strconv"
	"testing"
)

// Two people annotate the same position: each comment keeps its author, and
// list --author narrows to one of them.
func TestCLI_CommentAddListByAuthor(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	_, ids := seedCollection(t, cli, "c", 1)
	pos := ids[0]

	for _, a := range [][2]string{{"Alice", "too early"}, {"Bob", "I double here"}} {
		captureStdout(t, func() {
			if err := cli.Run([]string{"comment", "add", "--db", dbPath, "--position", strconv.FormatInt(pos, 10), "--text", a[1], "--author", a[0]}); err != nil {
				t.Fatalf("comment add: %v", err)
			}
		})
	}

	out := captureStdout(t, func() {
		if err := cli.Run([]string{"comment", "list", "--db", dbPath, "--position", strconv.FormatInt(pos, 10), "--format", "json"}); err != nil {
			t.Fatalf("comment list: %v", err)
		}
	})
	var all []CommentEntry
	if err := json.Unmarshal([]byte(out), &all); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if len(all) != 2 {
		t.Fatalf("got %d comments, want 2", len(all))
	}

	out = captureStdout(t, func() {
		if err := cli.Run([]string{"comment", "list", "--db", dbPath, "--author", "Bob", "--format", "json"}); err != nil {
			t.Fatalf("comment list: %v", err)
		}
	})
	var bob []CommentEntry
	if err := json.Unmarshal([]byte(out), &bob); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(bob) != 1 || bob[0].Author != "Bob" || bob[0].Text != "I double here" {
		t.Errorf("--author Bob = %+v", bob)
	}
}
