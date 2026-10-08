package cli

import (
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func TestFormatMatchShowsSourceMetadata(t *testing.T) {
	elo, exp, yes, no := 1854.7, 205, true, false
	m := &Match{Player1Name: "Alice", Player2Name: "Bob"}
	m.Player1Elo, m.Player1Experience = &elo, &exp
	m.Transcriber, m.HasJacoby, m.HasBeaver = "Carol", &yes, &no
	m.EngineVersion = "eXtreme Gammon, file format 30"
	out, err := (&CLI{}).formatMatchSummary(m, nil, nil)
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
	out, err := (&CLI{}).formatMatchText(m, []MatchMovePosition{known, unknown}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(out, "Decision time"); got != 1 || !strings.Contains(out, "Decision time: 5.2 s") {
		t.Errorf("want one decision time of 5.2 s, none for the unknown:\n%s", out)
	}
}

func TestWriteDifficultySummary(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	var sb strings.Builder
	writeDifficultySummary(&sb, [2]string{"Alice", "Bob"}, []storage.DecisionLoss{
		{Player: 0, MWCLoss: f(0.03), Difficulty: f(0.006), Avoidable: true},
		{Player: 1, MWCLoss: f(0.001), Difficulty: f(0.002)},
	})
	want := "\nDifficulty:\n" +
		"  Alice: difficulty 0.60%, excess +2.40%, ratio 5.00, 1 avoidable errors\n" +
		"  Bob: difficulty 0.20%, excess -0.10%, ratio -, 0 avoidable errors\n"
	if sb.String() != want {
		t.Errorf("got %q, want %q", sb.String(), want)
	}
	sb.Reset()
	writeDifficultySummary(&sb, [2]string{"Alice", "Bob"}, []storage.DecisionLoss{{Player: 0, MWCLoss: f(0.01)}})
	if sb.String() != "" {
		t.Errorf("no difficulty, no section: got %q", sb.String())
	}
}
