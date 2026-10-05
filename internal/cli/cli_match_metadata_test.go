package cli

import (
	"strings"
	"testing"
)

func TestFormatMatchShowsSourceMetadata(t *testing.T) {
	elo, exp, yes, no := 1854.7, 205, true, false
	m := &Match{Player1Name: "Alice", Player2Name: "Bob"}
	m.Player1Elo, m.Player1Experience = &elo, &exp
	m.Transcriber, m.HasJacoby, m.HasBeaver = "Carol", &yes, &no
	m.EngineVersion = "eXtreme Gammon, file format 30"
	out, err := (&CLI{}).formatMatchSummary(m, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Rating Alice: 1855 (experience 205)", "Transcriber: Carol",
		"Jacoby: yes, Beaver: no", "Written by: eXtreme Gammon, file format 30",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("summary lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Rating Bob") {
		t.Errorf("a rating the file did not state is shown:\n%s", out)
	}
}

func TestFormatMatchTextLeavesAnUnknownDecisionTimeOut(t *testing.T) {
	d := int64(5200)
	m := &Match{Player1Name: "Alice", Player2Name: "Bob"}
	known := MatchMovePosition{GameNumber: 1, MoveNumber: 1, DecisionMS: &d}
	unknown := MatchMovePosition{GameNumber: 1, MoveNumber: 2}
	out, err := (&CLI{}).formatMatchText(m, []MatchMovePosition{known, unknown})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(out, "Decision time"); got != 1 || !strings.Contains(out, "Decision time: 5.2 s") {
		t.Errorf("want one decision time of 5.2 s, none for the unknown:\n%s", out)
	}
}
