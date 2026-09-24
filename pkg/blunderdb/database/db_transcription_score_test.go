package database

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// TestDeclaredScore_TravelsIntoTheMatchAndTheMAT: a score declared on a game's
// opening (ADR-0053) is the score the game was played at — the draft keeps it
// across a reopening, the saved Match's game starts at it, the games after it
// follow it, and the .mat says it.
func TestDeclaredScore_TravelsIntoTheMatchAndTheMAT(t *testing.T) {
	db := newTestDB(t)
	id := matDraft(t, db, filepath.Join("testdata", "test.mat"))

	state, err := db.OpenTranscription(id)
	if err != nil {
		t.Fatal(err)
	}
	games := state.Annotated.Games
	if len(games) < 3 {
		t.Fatalf("the fixture has %d games", len(games))
	}
	derived := games[1].InitialScore
	declared := [2]int{derived[0] + 1, derived[1]}
	opening := games[1].First

	state, err = db.ApplyTranscriptionGesture(id, transcript.Gesture{Kind: transcript.GestureSetScore, At: opening, Score: &declared})
	if err != nil {
		t.Fatalf("set_score: %v", err)
	}
	flags := state.Annotated.Actions[opening].Inconsistencies
	if len(flags) != 1 || flags[0].Kind != transcript.ScoreMismatch {
		t.Fatalf("the opening carries %+v, want one score_mismatch", flags)
	}
	want := make([][2]int, len(state.Annotated.Games))
	for i, g := range state.Annotated.Games {
		want[i] = g.InitialScore
	}
	if want[1] != declared || want[2][0] != games[2].InitialScore[0]+1 {
		t.Fatalf("the games start at %v", want)
	}

	// The draft on disk carries it: a fresh session reads it back, as it would
	// after a restart.
	db.transcriptMu.Lock()
	delete(db.transcriptSessions, id)
	db.transcriptMu.Unlock()
	state, err = db.OpenTranscription(id)
	if err != nil {
		t.Fatal(err)
	}
	if doc := state.Annotated.Document; doc.FormatVersion != transcript.FormatVersion ||
		doc.Actions[opening].Score == nil || *doc.Actions[opening].Score != declared {
		t.Fatalf("the reopened draft lost the score: version %d, %+v", doc.FormatVersion, doc.Actions[opening])
	}

	saved, err := db.SaveTranscriptionAsMatch(id)
	if err != nil {
		t.Fatalf("SaveTranscriptionAsMatch: %v", err)
	}
	got := savedInitialScores(t, db, saved.MatchID)
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("the saved games start at %v, want %v", got, want)
	}

	out := filepath.Join(t.TempDir(), "match.mat")
	if err := db.ExportMatchMAT(saved.MatchID, out); err != nil {
		t.Fatal(err)
	}
	text, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	draftText, err := db.TranscriptionMAT(id)
	if err != nil {
		t.Fatal(err)
	}
	line := fmt.Sprintf(": %d", declared[0])
	if !strings.Contains(draftText, " Game 2\n") || !strings.Contains(sectionOf(draftText, " Game 2\n"), line) {
		t.Errorf("the draft's .mat does not write game 2 at %v:\n%s", declared, draftText)
	}
	if sectionOf(string(text), " Game 2\n") != sectionOf(draftText, " Game 2\n") {
		t.Errorf("the Match and its draft write game 2 differently")
	}
}

// sectionOf is the line after a .mat game header: the score line.
func sectionOf(text, header string) string {
	i := strings.Index(text, header)
	if i < 0 {
		return ""
	}
	rest := text[i+len(header):]
	if j := strings.IndexByte(rest, '\n'); j >= 0 {
		return rest[:j]
	}
	return rest
}

// savedInitialScores reads the initial score of every game of a saved match.
func savedInitialScores(t *testing.T, db *Database, matchID int64) [][2]int {
	t.Helper()
	rows, err := db.db.Query(`SELECT initial_score_1, initial_score_2 FROM game WHERE match_id = ? ORDER BY game_number`, matchID)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var out [][2]int
	for rows.Next() {
		var s [2]int
		if err := rows.Scan(&s[0], &s[1]); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}
