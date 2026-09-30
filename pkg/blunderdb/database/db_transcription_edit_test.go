package database

import (
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// One draft per Match: editing a match that already has a draft open returns
// that draft, with what was typed into it.
func TestEditMatchTranscription_ReopensTheMatchsDraft(t *testing.T) {
	db := newTestDB(t)
	saved, err := db.FinishTranscription(matDraft(t, db, filepath.Join("testdata", "test.mat")))
	if err != nil {
		t.Fatalf("FinishTranscription: %v", err)
	}

	first := editDraft(t, db, saved.MatchID)
	applyGesture(t, db, first, transcript.Gesture{
		Kind:   transcript.GestureSetHeader,
		Header: transcript.Header{Player1: "Alice", Player2: "Bob"},
	})
	db.forgetTranscriptSessions()

	again, err := db.EditMatchTranscription(saved.MatchID)
	if err != nil {
		t.Fatalf("EditMatchTranscription, second time: %v", err)
	}
	if again.ID != first {
		t.Fatalf("a second edit opened draft %d, want the match's draft %d", again.ID, first)
	}
	if got := again.Annotated.Document.Header.Player1; got != "Alice" {
		t.Errorf("the reopened draft lost what was typed: player 1 is %q", got)
	}
	if n := countTranscriptRows(t, db, `SELECT COUNT(*) FROM transcription`); n != 1 {
		t.Errorf("%d drafts, want 1", n)
	}

	losses, err := db.MatchTranscriptionLosses(saved.MatchID)
	if err != nil {
		t.Fatalf("MatchTranscriptionLosses: %v", err)
	}
	if losses.DraftID != first {
		t.Errorf("losses name draft %d, want %d", losses.DraftID, first)
	}
}

// Abandoning a draft opened on a Match leaves the Match as it was.
func TestAbandonTranscription_LeavesTheEditedMatch(t *testing.T) {
	db := newTestDB(t)
	saved, err := db.FinishTranscription(matDraft(t, db, filepath.Join("testdata", "test.mat")))
	if err != nil {
		t.Fatalf("FinishTranscription: %v", err)
	}
	games := countTranscriptRows(t, db, `SELECT COUNT(*) FROM game WHERE match_id = ?`, saved.MatchID)

	id := editDraft(t, db, saved.MatchID)
	if err := db.AbandonTranscription(id); err != nil {
		t.Fatalf("AbandonTranscription: %v", err)
	}
	if n := countTranscriptRows(t, db, `SELECT COUNT(*) FROM transcription`); n != 0 {
		t.Errorf("%d drafts after abandoning, want 0", n)
	}
	if n := countTranscriptRows(t, db, `SELECT COUNT(*) FROM game WHERE match_id = ?`, saved.MatchID); n != games {
		t.Errorf("the match has %d games after the abandon, %d before", n, games)
	}
}

// Closing releases the session only: the draft stays in the list.
func TestCloseTranscription_KeepsTheDraft(t *testing.T) {
	db := newTestDB(t)
	id := matDraft(t, db, filepath.Join("testdata", "test.mat"))
	if err := db.CloseTranscription(id); err != nil {
		t.Fatalf("CloseTranscription: %v", err)
	}
	if n := countTranscriptRows(t, db, `SELECT COUNT(*) FROM transcription`); n != 1 {
		t.Fatalf("%d drafts after closing, want 1", n)
	}
	if _, err := db.OpenTranscription(id); err != nil {
		t.Fatalf("OpenTranscription after close: %v", err)
	}
}

// An imported match is counted: the analyses and comments of its positions
// are what a .mat cannot carry.
func TestMatchTranscriptionLosses_CountsAnImportedMatch(t *testing.T) {
	db := newTestDB(t)
	saved, err := db.FinishTranscription(matDraft(t, db, filepath.Join("testdata", "test.mat")))
	if err != nil {
		t.Fatalf("FinishTranscription: %v", err)
	}
	if _, err := db.db.Exec(`UPDATE match SET file_path = 'match.xg' WHERE id = ?`, saved.MatchID); err != nil {
		t.Fatal(err)
	}

	positions := matchPositionIDs(t, db, saved.MatchID)
	comments := 0
	for posID := range positions {
		if err := db.SaveAnalysis(posID, PositionAnalysis{AnalysisType: "CheckerMove"}); err != nil {
			t.Fatalf("SaveAnalysis(%d): %v", posID, err)
		}
		if comments < 3 {
			if _, err := db.db.Exec(`INSERT INTO comment (position_id, text, origin) VALUES (?, 'xg', 'xg')`, posID); err != nil {
				t.Fatal(err)
			}
			comments++
		}
	}

	losses, err := db.MatchTranscriptionLosses(saved.MatchID)
	if err != nil {
		t.Fatalf("MatchTranscriptionLosses: %v", err)
	}
	if !losses.Imported || !losses.Lossy() {
		t.Fatalf("an imported, analysed match is not lossy: %+v", losses)
	}
	if losses.Analyses != len(positions) || losses.Comments != comments {
		t.Errorf("losses = %d analyses, %d comments; want %d, %d", losses.Analyses, losses.Comments, len(positions), comments)
	}
}

// An edit draft whose Match was deleted is a draft that never produced one:
// Terminer creates a new Match instead of failing on the missing one.
func TestFinishTranscription_EditDraftOfADeletedMatchCreatesOne(t *testing.T) {
	db := newTestDB(t)
	saved, err := db.FinishTranscription(matDraft(t, db, filepath.Join("testdata", "test.mat")))
	if err != nil {
		t.Fatalf("FinishTranscription: %v", err)
	}
	id := editDraft(t, db, saved.MatchID)
	if err := db.DeleteMatch(saved.MatchID); err != nil {
		t.Fatalf("DeleteMatch: %v", err)
	}

	again, err := db.FinishTranscription(id)
	if err != nil {
		t.Fatalf("finishing the edit draft of a deleted match: %v", err)
	}
	if again.Replaced || again.MatchID == 0 || again.MatchID == saved.MatchID {
		t.Fatalf("finish: match %d replaced=%v, want a new match", again.MatchID, again.Replaced)
	}
	if n := countTranscriptRows(t, db, `SELECT COUNT(*) FROM transcription`); n != 0 {
		t.Fatalf("%d drafts after finishing, want 0", n)
	}
}

// Editing an imported match keeps the hashes of the file it came from: the
// same file imported again is still recognised as a duplicate.
func TestFinishTranscription_ImportedMatchKeepsItsHashes(t *testing.T) {
	db := newTestDB(t)
	path := filepath.Join("testdata", "test.mat")
	matchID, err := db.ImportGnuBGMatch(path)
	if err != nil {
		t.Fatalf("ImportGnuBGMatch: %v", err)
	}
	var hash, canonical string
	if err := db.db.QueryRow(`SELECT COALESCE(match_hash,''), COALESCE(canonical_hash,'') FROM match WHERE id = ?`, matchID).Scan(&hash, &canonical); err != nil {
		t.Fatal(err)
	}
	if hash == "" {
		t.Fatal("the import stored no match hash; the fixture cannot exercise the rule")
	}

	id := editDraft(t, db, matchID)
	applyGesture(t, db, id, transcript.Gesture{
		Kind:   transcript.GestureSetHeader,
		Header: transcript.Header{MatchLength: 7, Player1: "Alice", Player2: "Bob"},
	})
	if _, err := db.FinishTranscription(id); err != nil {
		t.Fatalf("FinishTranscription: %v", err)
	}

	var gotHash, gotCanonical string
	if err := db.db.QueryRow(`SELECT COALESCE(match_hash,''), COALESCE(canonical_hash,'') FROM match WHERE id = ?`, matchID).Scan(&gotHash, &gotCanonical); err != nil {
		t.Fatal(err)
	}
	if gotHash != hash || gotCanonical != canonical {
		t.Errorf("hashes after the edit = %q/%q, want the import's %q/%q", gotHash, gotCanonical, hash, canonical)
	}
	_, _ = db.ImportGnuBGMatch(path)
	if n := countTranscriptRows(t, db, `SELECT COUNT(*) FROM match`); n != 1 {
		t.Errorf("importing the same file again made %d matches: not taken for a duplicate", n)
	}
}
