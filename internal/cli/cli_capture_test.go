package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

const (
	captureClear  = "pkg/blunderdb/capture/testdata/xg-dice62.jpg"       // no doubt, score unread
	captureDoubts = "pkg/blunderdb/capture/testdata/xg-cube2-scaled.jpg" // a dice doubt, score unread
)

// runCaptureCLI runs the capture command on a fresh CLI, returning its output.
func runCaptureCLI(t *testing.T, args ...string) (*CLI, string, error) {
	t.Helper()
	cli := &CLI{db: NewDatabase()}
	t.Cleanup(func() { cli.db.Close() })
	var err error
	out := captureStdout(t, func() { err = cli.runCapture(args) })
	return cli, out, err
}

// The score is never on a capture: saving without --score is refused, and
// --force, which only vouches for the board, does not stand in for it.
func TestCLI_Capture_SaveNeedsTheScore(t *testing.T) {
	db := filepath.Join(tempDir(t), "c.db")
	for _, extra := range [][]string{nil, {"--force"}} {
		args := append([]string{"--db", db, "--save"}, extra...)
		_, _, err := runCaptureCLI(t, append(args, captureClear)...)
		if err == nil || !strings.Contains(err.Error(), "--score") {
			t.Fatalf("%v: got %v, want a refusal naming --score", extra, err)
		}
	}
}

func TestCLI_Capture_SaveWithTheScoreSet(t *testing.T) {
	db := filepath.Join(tempDir(t), "c.db")
	cli, out, err := runCaptureCLI(t, "--db", db, "--save", "--score", "2a/5a", captureClear)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Saved as position") || strings.Contains(out, "Not read: score") {
		t.Fatalf("output:\n%s", out)
	}
	pos, err := cli.db.LoadPosition(1)
	if err != nil {
		t.Fatal(err)
	}
	if pos.Score != [2]int{2, 5} {
		t.Errorf("score %v, want [2 5]", pos.Score)
	}
}

// A doubt is the user's to clear: --score settles what the image lacks, but
// only --force says the board was checked.
func TestCLI_Capture_DoubtsNeedForce(t *testing.T) {
	db := filepath.Join(tempDir(t), "c.db")
	_, _, err := runCaptureCLI(t, "--db", db, "--save", "--score", "money", captureDoubts)
	if err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("got %v, want a refusal naming --force", err)
	}
	if _, _, err := runCaptureCLI(t, "--db", db, "--save", "--score", "money", "--force", captureDoubts); err != nil {
		t.Fatal(err)
	}
}

func TestParseCaptureScore(t *testing.T) {
	for in, want := range map[string][2]int{"money": {-1, -1}, "2a/5a": {2, 5}, "1a/3a": {1, 3}, "0a/4a": {0, 4}, "7/7": {7, 7}} {
		if got, err := parseCaptureScore(in); err != nil || got != want {
			t.Errorf("%q: got %v, %v; want %v", in, got, err, want)
		}
	}
	for _, in := range []string{"", "2a", "2a-5a", "xa/3a", "-1a/3a", "0a/1a", "0a/0a"} {
		if _, err := parseCaptureScore(in); err == nil {
			t.Errorf("%q: accepted", in)
		}
	}
}
