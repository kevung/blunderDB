package database

import (
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// The recipe of T3.3: a transcribed match with gaps in its analysis offers to
// finish, a complete one offers nothing — and neither answer is read back from
// anywhere, since ADR-0045 §8 stores nothing at all. Every assertion below is
// therefore about a COUNT taken again, which is the whole mechanism.

// analyseEveryMatchPosition writes a placeholder analysis row for each of the
// match's positions, which is what the targeted batch would leave behind.
func analyseEveryMatchPosition(t *testing.T, db *Database, matchID int64) {
	t.Helper()

	_, err := db.db.Exec(`
		INSERT INTO analysis (position_id, data)
		SELECT DISTINCT p.id, '{}'
		  FROM position p
		  JOIN move mv ON mv.position_id = p.id
		  JOIN game g ON mv.game_id = g.id
		 WHERE g.match_id = ?`, matchID)
	if err != nil {
		t.Fatalf("filling the analyses of match %d: %v", matchID, err)
	}
}

func TestPendingTranscriptionAnalysis_OffersTheUnfinishedMatch(t *testing.T) {
	db := newTestDB(t)
	id := matDraft(t, db, filepath.Join("testdata", "test.mat"))

	saved, err := db.SaveTranscriptionAsMatch(id)
	if err != nil {
		t.Fatalf("SaveTranscriptionAsMatch: %v", err)
	}
	if saved.ToAnalyze == 0 {
		t.Fatal("the freshly saved match has nothing to analyse; the fixture cannot exercise a resume")
	}

	// Nothing has been analysed: reopening the library must offer to finish,
	// on THIS match and with the count the targeted batch would work through.
	resume, err := db.PendingTranscriptionAnalysis()
	if err != nil {
		t.Fatalf("PendingTranscriptionAnalysis: %v", err)
	}
	if resume == nil {
		t.Fatal("a transcribed match with no analysis at all offers no resume")
	}
	if resume.MatchID != saved.MatchID || resume.TranscriptionID != id {
		t.Errorf("resume names draft %d / match %d; the draft is %d and its match %d",
			resume.TranscriptionID, resume.MatchID, id, saved.MatchID)
	}
	if resume.ToAnalyze != saved.ToAnalyze {
		t.Errorf("resume counts %d positions to analyse, the save counted %d", resume.ToAnalyze, saved.ToAnalyze)
	}

	// Ignoring the offer stores nothing, so asking again asks the same thing.
	again, err := db.PendingTranscriptionAnalysis()
	if err != nil {
		t.Fatalf("PendingTranscriptionAnalysis, second call: %v", err)
	}
	if again == nil || again.ToAnalyze != resume.ToAnalyze {
		t.Errorf("the offer did not survive being ignored: %+v", again)
	}
}

func TestPendingTranscriptionAnalysis_SaysNothingOnAnAnalysedMatch(t *testing.T) {
	db := newTestDB(t)
	id := matDraft(t, db, filepath.Join("testdata", "test.mat"))

	saved, err := db.SaveTranscriptionAsMatch(id)
	if err != nil {
		t.Fatalf("SaveTranscriptionAsMatch: %v", err)
	}
	analyseEveryMatchPosition(t, db, saved.MatchID)

	resume, err := db.PendingTranscriptionAnalysis()
	if err != nil {
		t.Fatalf("PendingTranscriptionAnalysis: %v", err)
	}
	if resume != nil {
		t.Errorf("a fully analysed match still offers a resume: %+v", resume)
	}
}

func TestPendingTranscriptionAnalysis_IgnoresADraftThatProducedNoMatch(t *testing.T) {
	db := newTestDB(t)

	// A draft being typed has no match id, so there is nothing to finish — and
	// nothing to sweep the library for either, which is the failure mode this
	// guards: the offer must never become "analyse everything".
	if _, err := db.CreateTranscription(transcript.Header{MatchLength: 7}); err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}

	resume, err := db.PendingTranscriptionAnalysis()
	if err != nil {
		t.Fatalf("PendingTranscriptionAnalysis: %v", err)
	}
	if resume != nil {
		t.Errorf("an unsaved draft offers a resume: %+v", resume)
	}
}
