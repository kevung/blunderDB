package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// The recipe of T3.4. Two properties and nothing else: --check says what the
// replay found and NEVER refuses the file for it (ADR-0044), and --render
// writes a .mat that replays back into the same transcription.

// illegalMat is a two-move .mat whose second play moves a checker off a point
// its owner does not hold — the injected illegal play. The rest is a plain
// gnubg transcript so the parser has no other reason to complain.
const illegalMat = ` 7 point match

 Game 1
 A : 0                          B : 0
  1)                             41: 13/9 24/23 
  2) 31: 2/1 6/3                 41: 6/5 9/5 
`

// runTranscribeText runs the command and returns everything it printed. The
// error is returned too: the point of most of these tests is that it is nil.
func runTranscribeText(t *testing.T, args ...string) (string, error) {
	t.Helper()

	var err error
	out := captureStdout(t, func() { err = (&CLI{db: NewDatabase()}).runTranscribe(args) })
	return out, err
}

func TestTranscribeCheck_ACleanMatHasNothingToSay(t *testing.T) {
	out, err := runTranscribeText(t, "--mat", filepath.Join("testdata", "charlot1-charlot2_7p_2025-11-08-2305.mat"), "--check")
	if err != nil {
		t.Fatalf("transcribe --check on a sound .mat: %v", err)
	}
	if !strings.Contains(out, "Inconsistencies: none") {
		t.Errorf("a sound .mat is reported as inconsistent:\n%s", out)
	}
	if !strings.Contains(out, "7 point match") {
		t.Errorf("the report does not state the match length:\n%s", out)
	}
}

func TestTranscribeCheck_NamesAnIllegalPlayAndStillSucceeds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "illegal.mat")
	if err := os.WriteFile(path, []byte(illegalMat), 0o644); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}

	out, err := runTranscribeText(t, "--mat", path, "--check", "--format", "json")
	// The whole decision of this command: an inconsistency is a finding, not a
	// refusal. A non-zero status here would be indistinguishable from a file
	// the parser could not read.
	if err != nil {
		t.Fatalf("an inconsistency made the command fail: %v", err)
	}

	var result transcribeResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("decoding the json report: %v\n%s", err, out)
	}
	if len(result.Inconsistencies) == 0 {
		t.Fatalf("the injected illegal play was not reported:\n%s", out)
	}

	var found *transcribeInconsistency
	for i := range result.Inconsistencies {
		if result.Inconsistencies[i].Type == string(transcript.IllegalMove) {
			found = &result.Inconsistencies[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("no illegal play among the findings: %+v", result.Inconsistencies)
	}
	if found.Game != 1 {
		t.Errorf("the illegal play is placed in game %d; the fixture has one game", found.Game)
	}
	if found.Action < 0 || found.Action >= result.Actions {
		t.Errorf("the illegal play is placed at action %d, outside the %d actions of the document", found.Action, result.Actions)
	}
}

func TestTranscribeRender_ReplaysBackIntoTheSameTranscription(t *testing.T) {
	src := filepath.Join("testdata", "charlot1-charlot2_7p_2025-11-08-2305.mat")
	out := filepath.Join(t.TempDir(), "round-trip.mat")

	if _, err := runTranscribeText(t, "--mat", src, "--render", out); err != nil {
		t.Fatalf("transcribe --render: %v", err)
	}

	before := replayFile(t, src)
	after := replayFile(t, out)

	if len(after.Games) != len(before.Games) || len(after.Actions) != len(before.Actions) {
		t.Fatalf("the round trip changed the document: %d games / %d actions became %d / %d",
			len(before.Games), len(before.Actions), len(after.Games), len(after.Actions))
	}
	if after.Score != before.Score || after.Finished != before.Finished {
		t.Errorf("the round trip changed the outcome: %v/%v became %v/%v",
			before.Score, before.Finished, after.Score, after.Finished)
	}
	for i := range before.Actions {
		if after.Actions[i].Kind != before.Actions[i].Kind || after.Actions[i].Side != before.Actions[i].Side {
			t.Fatalf("action %d became %s/side %d, it was %s/side %d",
				i, after.Actions[i].Kind, after.Actions[i].Side, before.Actions[i].Kind, before.Actions[i].Side)
		}
		if len(after.Actions[i].Inconsistencies) != len(before.Actions[i].Inconsistencies) {
			t.Errorf("action %d gained or lost an inconsistency in the round trip", i)
		}
	}
}

func replayFile(t *testing.T, path string) transcript.Annotated {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	doc, err := transcript.FromMAT(string(raw))
	if err != nil {
		t.Fatalf("replaying %s: %v", path, err)
	}
	return transcript.Replay(doc, 0)
}

func TestTranscribe_RefusesAnAmbiguousOrMissingSource(t *testing.T) {
	// Not a finding about a transcription: the command was not told what to
	// replay, which is the kind of failure that DOES deserve a status.
	for _, args := range [][]string{
		{"--check"},
		{"--mat", "a.mat", "--match", "5", "--db", "x.db"},
		{"--match", "5"},
	} {
		if _, err := runTranscribeText(t, args...); err == nil {
			t.Errorf("transcribe %v was accepted", args)
		}
	}
}
