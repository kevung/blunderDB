package database

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// The recipe of T1.9, in two tests: a real multi-game document is saved,
// corrected and saved again — same Match, analyses kept where nothing moved,
// nothing through the trash — and the .mat a draft exports is the .mat the
// Match it produced exports.
//
// The document is a real .mat of the repository read back by transcript.FromMAT:
// four games, a cube, a Crawford game, dances and a resignation, which no
// hand-built fixture would cover at that price.

// matDraft reads a .mat fixture into a draft row and opens it, returning the
// row id.
func matDraft(t *testing.T, db *Database, path string) int64 {
	t.Helper()

	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	doc, err := transcript.FromMAT(string(text))
	if err != nil {
		t.Fatalf("FromMAT %s: %v", path, err)
	}
	id, err := db.saveTranscription(0, doc)
	if err != nil {
		t.Fatalf("saveTranscription: %v", err)
	}
	if _, err := db.OpenTranscription(id); err != nil {
		t.Fatalf("OpenTranscription: %v", err)
	}
	return id
}

// matchPositionIDs is the set of positions the match's moves stand on.
func matchPositionIDs(t *testing.T, db *Database, matchID int64) map[int64]bool {
	t.Helper()

	ids, err := queryInt64s(db.db, `
		SELECT DISTINCT p.id FROM position p
		  JOIN move mv ON mv.position_id = p.id
		  JOIN game g ON mv.game_id = g.id
		 WHERE g.match_id = ?`, matchID)
	if err != nil {
		t.Fatalf("match positions: %v", err)
	}
	out := map[int64]bool{}
	for _, id := range ids {
		out[id] = true
	}
	return out
}

func countTranscriptRows(t *testing.T, db *Database, query string, args ...any) int {
	t.Helper()

	var n int
	if err := db.db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return n
}

// lastPlayedAction is the index of the last Action that produced a Move — the
// one a correction is applied to here, because correcting the LAST play leaves
// every earlier position untouched, which is precisely the property the test
// is about.
func lastPlayedAction(t *testing.T, state *TranscriptionState) int {
	t.Helper()

	for i := len(state.Annotated.Actions) - 1; i >= 0; i-- {
		info := state.Annotated.Actions[i]
		if info.HasPosition && (info.Kind == transcript.KindChecker || info.Kind == transcript.KindDance) {
			return i
		}
	}
	t.Fatal("the document has no checker action")
	return -1
}

func TestSaveTranscriptionAsMatch_CreatesThenReplacesInPlace(t *testing.T) {
	db := newTestDB(t)
	id := matDraft(t, db, filepath.Join("testdata", "test.mat"))

	first, err := db.SaveTranscriptionAsMatch(id)
	if err != nil {
		t.Fatalf("SaveTranscriptionAsMatch: %v", err)
	}
	if first.MatchID == 0 {
		t.Fatal("the first save produced no match")
	}
	if first.Replaced {
		t.Error("the first save reports a replacement; it creates")
	}
	if first.Games < 2 || first.Moves == 0 || first.Positions == 0 {
		t.Fatalf("first save: %d games, %d moves, %d positions", first.Games, first.Moves, first.Positions)
	}

	// The match, its games and its moves are in the library.
	if got := countTranscriptRows(t, db, `SELECT COUNT(*) FROM game WHERE match_id = ?`, first.MatchID); got != first.Games {
		t.Errorf("%d game rows, the save reports %d", got, first.Games)
	}
	moveRows := countTranscriptRows(t, db, `SELECT COUNT(*) FROM move mv JOIN game g ON mv.game_id = g.id WHERE g.match_id = ?`, first.MatchID)
	if moveRows != first.Moves {
		t.Errorf("%d move rows, the save reports %d", moveRows, first.Moves)
	}

	// The match id is posted on the draft, and it is what makes the next save
	// a replacement rather than a second match.
	state, err := db.OpenTranscription(id)
	if err != nil {
		t.Fatalf("OpenTranscription: %v", err)
	}
	if state.Annotated.Document.Header.MatchID == nil || *state.Annotated.Document.Header.MatchID != first.MatchID {
		t.Fatalf("the draft's match id is %v, want %d", state.Annotated.Document.Header.MatchID, first.MatchID)
	}

	// Stand in for the analysis batch: every position of the match gets one,
	// so that "kept its analysis" is a statement about these rows and not
	// about an empty table.
	before := matchPositionIDs(t, db, first.MatchID)
	for posID := range before {
		if err := db.SaveAnalysis(posID, PositionAnalysis{AnalysisType: "CheckerMove"}); err != nil {
			t.Fatalf("SaveAnalysis(%d): %v", posID, err)
		}
	}

	// The correction: the last play changes camp. The Cursor is placed
	// directly rather than walked there with a hundred cursor_back gestures,
	// each of which is a full Replay and a row write; the gesture itself goes
	// through the ordinary path.
	at := lastPlayedAction(t, state)
	db.transcriptMu.Lock()
	db.transcriptSessions[id].Doc.Cursor = at
	db.transcriptMu.Unlock()
	if _, err := db.ApplyTranscriptionGesture(id, transcript.Gesture{Kind: transcript.GestureFlipSide}); err != nil {
		t.Fatalf("flip_side: %v", err)
	}

	second, err := db.SaveTranscriptionAsMatch(id)
	if err != nil {
		t.Fatalf("second SaveTranscriptionAsMatch: %v", err)
	}
	if second.MatchID != first.MatchID {
		t.Fatalf("the second save produced match %d, want the same match %d", second.MatchID, first.MatchID)
	}
	if !second.Replaced {
		t.Error("the second save reports a creation; it replaces")
	}
	if got := countTranscriptRows(t, db, `SELECT COUNT(*) FROM match`); got != 1 {
		t.Errorf("%d matches in the library, want 1", got)
	}
	if got := countTranscriptRows(t, db, `SELECT COUNT(*) FROM game WHERE match_id = ?`, first.MatchID); got != second.Games {
		t.Errorf("after the replacement, %d game rows for %d games", got, second.Games)
	}

	// What the correction did not touch is untouched: the same position rows,
	// with the analyses they were given.
	after := matchPositionIDs(t, db, first.MatchID)
	kept := 0
	for posID := range after {
		if !before[posID] {
			continue
		}
		kept++
		if n := countTranscriptRows(t, db, `SELECT COUNT(*) FROM analysis WHERE position_id = ?`, posID); n != 1 {
			t.Errorf("position %d survived the replacement without its analysis", posID)
		}
	}
	if kept < len(before)/2 {
		t.Fatalf("only %d of %d positions survived the replacement: the correction rewrote the match", kept, len(before))
	}

	// The corrected play stands on a position that is new, and therefore has
	// no analysis: that is exactly what the targeted batch is started on.
	fresh := 0
	for posID := range after {
		if !before[posID] {
			fresh++
		}
	}
	if fresh == 0 {
		t.Error("the correction produced no new position")
	}
	if second.ToAnalyze != fresh {
		t.Errorf("the save reports %d positions to analyse, %d are new", second.ToAnalyze, fresh)
	}

	// The position the correction left behind is purged by the ordinary
	// retention rule — and nothing at all went through the trash (ADR-0045 §3).
	for posID := range before {
		if after[posID] {
			continue
		}
		if n := countTranscriptRows(t, db, `SELECT COUNT(*) FROM position WHERE id = ?`, posID); n != 0 {
			t.Errorf("position %d is held by nothing and was not purged", posID)
		}
	}
	if n := countTranscriptRows(t, db, `SELECT COUNT(*) FROM trash`); n != 0 {
		t.Errorf("%d rows in the trash: a replacement is not a deletion", n)
	}
}

// TestTranscriptionMAT_MatchesTheSavedMatchExport is the format gate of the
// save: what the draft exports and what the Match it produced exports are the
// same file. They travel by two entirely different roads — the document
// replayed in memory on one side, the game/move rows read back on the other —
// and the day they diverge, one of the two is lying about the match.
func TestTranscriptionMAT_MatchesTheSavedMatchExport(t *testing.T) {
	db := newTestDB(t)
	id := matDraft(t, db, filepath.Join("testdata", "test.mat"))

	// The one header a Match cannot carry: there is no match.transcriber
	// column (domain.Match says so), so a draft naming its transcriber exports
	// one line the saved Match cannot. Cleared here, and asserted below.
	db.transcriptMu.Lock()
	transcriber := db.transcriptSessions[id].Doc.Header.Transcriber
	db.transcriptSessions[id].Doc.Header.Transcriber = ""
	db.transcriptMu.Unlock()
	if transcriber == "" {
		t.Fatal("the fixture is meant to name a transcriber")
	}

	draftText, err := db.TranscriptionMAT(id)
	if err != nil {
		t.Fatalf("TranscriptionMAT: %v", err)
	}

	saved, err := db.SaveTranscriptionAsMatch(id)
	if err != nil {
		t.Fatalf("SaveTranscriptionAsMatch: %v", err)
	}

	out := filepath.Join(t.TempDir(), "match.mat")
	if err := db.ExportMatchMAT(saved.MatchID, out); err != nil {
		t.Fatalf("ExportMatchMAT: %v", err)
	}
	matchText, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read the exported .mat: %v", err)
	}
	if draftText != string(matchText) {
		t.Errorf("the draft and its Match export different .mat files\n--- draft ---\n%s\n--- match ---\n%s", draftText, matchText)
	}

	// And the draft's own export writes the very same bytes to the file the
	// user chose.
	fromDraft := filepath.Join(t.TempDir(), "draft.mat")
	if err := db.ExportTranscriptionMAT(id, fromDraft); err != nil {
		t.Fatalf("ExportTranscriptionMAT: %v", err)
	}
	written, err := os.ReadFile(fromDraft)
	if err != nil {
		t.Fatalf("read the draft's .mat: %v", err)
	}
	if string(written) != draftText {
		t.Error("ExportTranscriptionMAT wrote something other than TranscriptionMAT renders")
	}

	// The transcriber line is the draft's alone, and it is there when the
	// header names one.
	db.transcriptMu.Lock()
	db.transcriptSessions[id].Doc.Header.Transcriber = transcriber
	db.transcriptMu.Unlock()
	withTranscriber, err := db.TranscriptionMAT(id)
	if err != nil {
		t.Fatalf("TranscriptionMAT: %v", err)
	}
	if withTranscriber == draftText {
		t.Error("naming a transcriber changes nothing in the exported .mat")
	}

	name, err := db.SuggestTranscriptionMatFilename(id)
	if err != nil {
		t.Fatalf("SuggestTranscriptionMatFilename: %v", err)
	}
	if filepath.Ext(name) != ".mat" {
		t.Errorf("the suggested name is %q", name)
	}
}

// A draft with nothing in it is the one thing a save refuses: an empty Match
// would count in the statistics and answer searches with nothing at all.
func TestSaveTranscriptionAsMatch_RefusesAnEmptyDraft(t *testing.T) {
	db := newTestDB(t)

	state, err := db.CreateTranscription(transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	if _, err := db.SaveTranscriptionAsMatch(state.ID); err == nil {
		t.Fatal("saving an empty draft succeeded")
	}
	if n := countTranscriptRows(t, db, `SELECT COUNT(*) FROM match`); n != 0 {
		t.Errorf("%d matches written by a refused save", n)
	}
}
