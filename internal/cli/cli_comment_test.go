package cli

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Two people annotate the same position: each comment keeps its author, and
// list --author narrows to one of them.
func TestCLI_CommentAddListByAuthor(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	_, ids := seedCollection(t, cli, "c", 1)
	pos := ids[0]

	for _, a := range [][2]string{{"Alice", "too early"}, {"Bob", "I double here"}, {"Élodie", "pass"}} {
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
	if len(all) != 3 {
		t.Fatalf("got %d comments, want 3", len(all))
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

	// Any case, accents included, as the au"…" search token compares it.
	for _, q := range []string{"élodie", " ÉLODIE "} {
		out = captureStdout(t, func() {
			if err := cli.Run([]string{"comment", "list", "--db", dbPath, "--author", q, "--format", "json"}); err != nil {
				t.Fatalf("comment list: %v", err)
			}
		})
		var got []CommentEntry
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("json: %v", err)
		}
		if len(got) != 1 || got[0].Author != "Élodie" {
			t.Errorf("--author %q = %+v", q, got)
		}
	}
}

// TestSignerLabel: a signature that cannot be read says so, instead of an
// empty name and fingerprint.
func TestSignerLabel(t *testing.T) {
	for _, tc := range []struct {
		in   *domain.CommentSigner
		want string
	}{
		{nil, ""},
		{&domain.CommentSigner{}, "(unreadable signature)"},
		{&domain.CommentSigner{Name: "Coach", Fingerprint: "21FA-ECC9-77E0-EC62", Verified: true}, "Coach [21FA-ECC9-77E0-EC62]"},
		{&domain.CommentSigner{Name: "Coach", Fingerprint: "21FA-ECC9-77E0-EC62"}, "Coach [21FA-ECC9-77E0-EC62] (unverified)"},
	} {
		if got := signerLabel(tc.in); got != tc.want {
			t.Errorf("signerLabel(%+v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
