package database

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// The session fuzzer drives one draft through ApplyTranscriptionGesture — undo
// and redo included — with saves and reopenings in the middle, and holds what
// the library promises around the draft:
//
//   - the row always holds the session's durable document;
//   - a draft reopened from its row is the same document;
//   - a draft produces at most ONE Match, whatever is saved, undone or redone
//     between two saves, and its match id never changes once posted.
//
// The rules of the gestures themselves are fuzzed in pkg/blunderdb/transcript.

const (
	sopTurn = iota
	sopTurn2
	sopTurn3
	sopOpening
	sopDouble
	sopTake
	sopPass
	sopResign
	sopBack
	sopForward
	sopDelete
	sopFlip
	sopSetLength
	sopSetScore
	sopUndo
	sopRedo
	sopSave
	sopReopen
	sopDie
	sopValidate
	sopCount
)

type sessionBytes struct {
	data []byte
	i    int
}

func (s *sessionBytes) next() (byte, bool) {
	if s.i >= len(s.data) {
		return 0, false
	}
	s.i++
	return s.data[s.i-1], true
}

func (s *sessionBytes) or(d byte) byte {
	if b, ok := s.next(); ok {
		return b
	}
	return d
}

type sessionRun struct {
	t       *testing.T
	db      *Database
	id      int64
	matchID int64
	matches int
	trace   []string
}

func (r *sessionRun) fail(format string, args ...any) {
	r.t.Helper()
	r.t.Fatalf("%s\ntrace: %v", fmt.Sprintf(format, args...), r.trace)
}

func (r *sessionRun) doc() transcript.Document {
	return r.db.transcriptSessions[r.id].Doc
}

// gesture applies one gesture; a refusal is an answer, not a failure.
func (r *sessionRun) gesture(g transcript.Gesture) {
	r.t.Helper()
	r.trace = append(r.trace, fmt.Sprintf("%s(%d,%d,%d,%v)", g.Kind, g.Die, g.Candidate, g.At, g.Score))
	if _, err := r.db.ApplyTranscriptionGesture(r.id, g); err != nil {
		return
	}
	r.check()
}

// check: the row is the session's durable document.
func (r *sessionRun) check() {
	r.t.Helper()
	want, err := durableJSON(r.doc())
	if err != nil {
		r.fail("durableJSON: %v", err)
	}
	row, err := r.db.loadTranscription(r.id)
	if err != nil {
		r.fail("loadTranscription: %v", err)
	}
	got, err := durableJSON(row)
	if err != nil {
		r.fail("durableJSON(row): %v", err)
	}
	if !bytes.Equal(got, want) {
		r.fail("the row drifted from the session:\n row %s\nsess %s", got, want)
	}
}

func (r *sessionRun) countMatches() int {
	r.t.Helper()
	var n int
	if err := r.db.db.QueryRow(`SELECT COUNT(*) FROM match`).Scan(&n); err != nil {
		r.fail("count matches: %v", err)
	}
	return n
}

func (r *sessionRun) save() {
	r.t.Helper()
	r.trace = append(r.trace, "SAVE")
	// Terminer releases the draft; the run goes on in the draft opened again
	// on the Match, which is how a correction continues after it.
	res, err := r.db.FinishTranscription(r.id)
	if err != nil {
		return
	}
	state, err := r.db.EditMatchTranscription(res.MatchID)
	if err != nil {
		r.fail("editing the Match %d just finished: %v", res.MatchID, err)
	}
	r.id = state.ID
	if r.matchID == 0 {
		r.matchID = res.MatchID
	} else if res.MatchID != r.matchID {
		r.fail("save produced Match %d, the draft had produced Match %d", res.MatchID, r.matchID)
	}
	if n := r.countMatches(); n != r.matches+1 {
		r.fail("%d Matches after saving one draft (%d before it)", n, r.matches)
	}
	if id := r.doc().Header.MatchID; id == nil || *id != r.matchID {
		r.fail("the draft carries match id %v after the save of Match %d", id, r.matchID)
	}
	r.check()
}

// reopen drops the session and opens the row again, as a restart does.
func (r *sessionRun) reopen() {
	r.t.Helper()
	r.trace = append(r.trace, "REOPEN")
	before, err := durableJSON(r.doc())
	if err != nil {
		r.fail("durableJSON: %v", err)
	}
	r.db.transcriptMu.Lock()
	delete(r.db.transcriptSessions, r.id)
	r.db.transcriptMu.Unlock()
	st, err := r.db.OpenTranscription(r.id)
	if err != nil {
		r.fail("OpenTranscription: %v", err)
	}
	after, err := durableJSON(st.Annotated.Document)
	if err != nil {
		r.fail("durableJSON: %v", err)
	}
	if !bytes.Equal(before, after) {
		r.fail("reopening changed the draft:\nbefore %s\n after %s", before, after)
	}
	if st.CanUndo || st.CanRedo {
		r.fail("a reopened draft has an undo stack")
	}
}

func (r *sessionRun) turn(s *sessionBytes) {
	r.gesture(transcript.Gesture{Kind: transcript.GestureEnterDie, Die: 1 + int(s.or(0))%6})
	r.gesture(transcript.Gesture{Kind: transcript.GestureEnterDie, Die: 1 + int(s.or(1))%6})
	cands := transcript.Candidates(r.doc())
	if len(cands) == 0 {
		r.gesture(transcript.Gesture{Kind: transcript.GestureValidate})
		r.gesture(transcript.Gesture{Kind: transcript.GestureDance})
		return
	}
	r.gesture(transcript.Gesture{Kind: transcript.GestureSelectCandidate, Candidate: int(s.or(0)) % len(cands)})
	r.gesture(transcript.Gesture{Kind: transcript.GestureValidate})
}

func runSessionScenario(t *testing.T, db *Database, data []byte) {
	src := &sessionBytes{data: data}
	st, err := db.CreateTranscription(transcript.Header{
		MatchLength: int(src.or(7)) % 12, Player1: "Ana", Player2: "Bea",
	})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	r := &sessionRun{t: t, db: db, id: st.ID}
	r.matches = r.countMatches()
	defer func() {
		if err := db.CloseTranscription(r.id); err != nil {
			t.Errorf("CloseTranscription: %v", err)
		}
	}()

	for steps := 0; steps < 120; steps++ {
		b, ok := src.next()
		if !ok {
			break
		}
		switch int(b) % sopCount {
		case sopTurn, sopTurn2, sopTurn3:
			r.turn(src)
		case sopOpening:
			r.gesture(transcript.Gesture{Kind: transcript.GestureEnterDie, Die: 1 + int(src.or(0))%6})
			r.gesture(transcript.Gesture{Kind: transcript.GestureEnterDie, Die: 1 + int(src.or(1))%6})
			r.gesture(transcript.Gesture{Kind: transcript.GestureValidate})
		case sopDouble:
			r.gesture(transcript.Gesture{Kind: transcript.GestureDouble})
		case sopTake:
			r.gesture(transcript.Gesture{Kind: transcript.GestureTake})
		case sopPass:
			r.gesture(transcript.Gesture{Kind: transcript.GesturePass})
		case sopResign:
			r.gesture(transcript.Gesture{Kind: transcript.GestureResign, Level: 1 + int(src.or(0))%3})
		case sopBack:
			r.gesture(transcript.Gesture{Kind: transcript.GestureCursorBack})
		case sopForward:
			r.gesture(transcript.Gesture{Kind: transcript.GestureCursorForward})
		case sopDelete:
			r.gesture(transcript.Gesture{Kind: transcript.GestureDelete})
		case sopFlip:
			r.gesture(transcript.Gesture{Kind: transcript.GestureFlipSide})
		case sopSetLength:
			r.gesture(transcript.Gesture{Kind: transcript.GestureSetLength, MatchLength: int(src.or(5)) % 12, HasLength: true})
		case sopSetScore:
			sc := [2]int{int(src.or(0)) % 15, int(src.or(0)) % 15}
			r.gesture(transcript.Gesture{Kind: transcript.GestureSetScore, At: int(src.or(0)) % (len(r.doc().Actions) + 1), Score: &sc})
		case sopUndo:
			for k := int(src.or(0)) % 5; k >= 0; k-- {
				r.gesture(transcript.Gesture{Kind: transcript.GestureUndo})
			}
		case sopRedo:
			for k := int(src.or(0)) % 5; k >= 0; k-- {
				r.gesture(transcript.Gesture{Kind: transcript.GestureRedo})
			}
		case sopSave:
			r.save()
		case sopReopen:
			r.reopen()
		case sopDie:
			r.gesture(transcript.Gesture{Kind: transcript.GestureEnterDie, Die: int(src.or(0)) % 8})
		case sopValidate:
			r.gesture(transcript.Gesture{Kind: transcript.GestureValidate})
		}
	}
	r.save()
	r.save()
}

func sessionSeeds() [][]byte {
	return [][]byte{
		{7, sopOpening, 6, 3, sopTurn, 1, 2, 0, sopSave, sopTurn, 5, 5, 1, sopSave, sopReopen, sopSave},
		{5, sopOpening, 4, 2, sopTurn, 6, 6, 0, sopSave, sopUndo, 4, sopSave, sopRedo, 4, sopSave},
		{3, sopOpening, 3, 1, sopTurn, 6, 5, 0, sopDouble, sopPass, sopSave, sopBack, sopBack, sopDelete, sopSave},
		{0, sopOpening, 5, 2, sopTurn, 4, 4, 0, sopResign, 1, sopSave, sopSetLength, 3, sopSave, sopReopen},
	}
}

func openFuzzDB(tb testing.TB) *Database {
	tb.Helper()
	db := NewDatabase()
	if err := db.SetupDatabase(filepath.Join(tempDir(tb), "fuzz.db")); err != nil {
		tb.Fatalf("SetupDatabase: %v", err)
	}
	closeOnCleanup(tb, db)
	return db
}

func FuzzTranscriptionSession(f *testing.F) {
	for _, s := range sessionSeeds() {
		f.Add(s)
	}
	db := openFuzzDB(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 300 {
			return
		}
		runSessionScenario(t, db, data)
	})
}

// TestTranscriptionSessionSeeded runs the session scenario on pseudo-random
// programs from fixed seeds, so the properties hold on every push.
func TestTranscriptionSessionSeeded(t *testing.T) {
	n := 16
	if testing.Short() {
		n = 4
	}
	db := newTestDB(t)
	for seed := 0; seed < n; seed++ {
		data := make([]byte, 120)
		x := uint32(seed*2246822519 + 7)
		for i := range data {
			x ^= x << 13
			x ^= x >> 17
			x ^= x << 5
			data[i] = byte(x)
		}
		t.Run(fmt.Sprint(seed), func(t *testing.T) { runSessionScenario(t, db, data) })
	}
}

// TestTranscriptionUndoAfterSaveKeepsTheMatch: the match id is a fact of the
// library, not a gesture, so undoing the gesture before a save must not take it
// away — or the next save would create a SECOND Match.
func TestTranscriptionUndoAfterSaveKeepsTheMatch(t *testing.T) {
	db := newTestDB(t)
	st, err := db.CreateTranscription(transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range []transcript.Gesture{
		{Kind: transcript.GestureEnterDie, Die: 6}, {Kind: transcript.GestureEnterDie, Die: 3},
		{Kind: transcript.GestureValidate},
		{Kind: transcript.GestureEnterDie, Die: 2}, {Kind: transcript.GestureEnterDie, Die: 1},
		{Kind: transcript.GestureSelectCandidate}, {Kind: transcript.GestureValidate},
	} {
		if _, err := db.ApplyTranscriptionGesture(st.ID, g); err != nil {
			t.Fatalf("%s: %v", g.Kind, err)
		}
	}
	first, err := db.FinishTranscription(st.ID)
	if err != nil {
		t.Fatal(err)
	}
	edited, err := db.EditMatchTranscription(first.MatchID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ApplyTranscriptionGesture(edited.ID, transcript.Gesture{
		Kind: transcript.GestureSetHeader, Header: transcript.Header{MatchLength: 7, Player1: "Alice"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ApplyTranscriptionGesture(edited.ID, transcript.Gesture{Kind: transcript.GestureUndo}); err != nil {
		t.Fatal(err)
	}
	second, err := db.FinishTranscription(edited.ID)
	if err != nil {
		t.Fatal(err)
	}
	if second.MatchID != first.MatchID || !second.Replaced {
		t.Fatalf("the second save wrote Match %d (replaced=%v), the draft had produced Match %d",
			second.MatchID, second.Replaced, first.MatchID)
	}
}
