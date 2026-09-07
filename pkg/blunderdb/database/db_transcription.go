package database

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// =====================================================================
// Transcriptions — drafts of matches being typed in (ADR-0045)
// =====================================================================
//
// This file is PLUMBING and nothing else. The rules of a transcription live
// in pkg/blunderdb/transcript (a pure package: a Document, the Gestures that
// change it, and the Replay that derives everything else); the persistence
// lives in storage.TranscriptionStore. What is written here is the join: read
// the row, hand the document to transcript, write the row back, return the
// annotated document.
//
// # Why a session is held in memory
//
// transcript.Document carries an Entry — the Action being typed, the dice as
// they come in one at a time — which is deliberately NOT serialised
// (`json:"-"`, ADR-0045 rule 1: a crash is allowed to lose it). Applying a
// gesture is therefore not a function of the stored row alone: two successive
// "enter a die" gestures, each a Wails round trip, need the same live
// Document. So an opened draft keeps a transcript.Editor in memory, which is
// also where its undo stack lives — the one the ADR says stays in memory.
//
// The session is a cache, never the truth: the row is rewritten after every
// gesture that changes the document, so losing every session (a crash, a
// close) loses only what a crash is allowed to lose.

// TranscriptionSummary is one line of the drafts list: what the panel shows
// without opening the draft. The header facts and the action count are read
// out of the document — the row holds it as one opaque string, and paying a
// JSON decode per draft is cheaper than a second, drifting copy of the same
// facts in columns of their own.
type TranscriptionSummary struct {
	ID            int64  `json:"id"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	FormatVersion string `json:"format_version"`
	// MatchID is the Match this draft has already produced, 0 while it has
	// produced none.
	MatchID int64  `json:"match_id"`
	Label   string `json:"label"`

	Player1 string `json:"player1"`
	Player2 string `json:"player2"`
	// MatchLength is 0 for a money session, as in transcript.Header.
	MatchLength int `json:"match_length"`
	ActionCount int `json:"action_count"`
}

// TranscriptionState is what every gesture binding hands back: the row's id
// and the whole annotated document. The panel is a client of the engine
// (ADR-0045 rule 9) — it never derives a score, a Crawford game or an
// Inconsistency of its own, it displays the ones that come back here.
type TranscriptionState struct {
	ID        int64                `json:"id"`
	Annotated transcript.Annotated `json:"annotated"`
	// CanUndo and CanRedo are the two sides of the session's stack, which is
	// in memory and nowhere else (ADR-0045 rule 1): they are false again the
	// moment the application is restarted, and that is the promise, not a gap.
	CanUndo bool `json:"can_undo"`
	CanRedo bool `json:"can_redo"`
}

// ListTranscriptions returns the library's drafts, most recently updated
// first. Never nil: the panel iterates it straight from JSON.
func (d *Database) ListTranscriptions() ([]TranscriptionSummary, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	out := []TranscriptionSummary{}
	for row, err := range d.store.Transcriptions().List(context.Background(), "") {
		if err != nil {
			return nil, err
		}
		out = append(out, summarize(row))
	}
	return out, nil
}

// CreateTranscription opens a new draft and returns it, ready to type into.
//
// The header is what the creation form (T1.2) collected; a zero header means
// a match of transcript.DefaultMatchLength, which is what the panel's bare
// "new transcription" button sends today. The draft is written immediately,
// before a single Action: a draft that exists only in the panel would be lost
// by the crash the whole table exists to survive.
func (d *Database) CreateTranscription(header transcript.Header) (*TranscriptionState, error) {
	doc := transcript.New(header.MatchLength)
	// Keep whatever else the form stated (players, event, rules), but never
	// let a caller post a match id on a draft that has produced no match.
	doc.Header = header
	doc.Header.MatchID = nil
	if doc.Header.MatchLength == 0 && !header.Jacoby && !header.Beaver {
		// A money draft is Jacoby by default (transcript.New); an all-zero
		// header is "unstated", not "Jacoby off".
		doc.Header.Jacoby = true
	}
	// The two fields the user is not asked for and would have to type anyway
	// (fonctionnel.md §1.1): today, and whoever this library says they are.
	// Defaults only — the metadata pane overwrites both, and overwriting the
	// transcriber there never touches the library's own `user` setting.
	if doc.Header.Date.IsZero() {
		doc.Header.Date = time.Now()
	}
	if strings.TrimSpace(doc.Header.Transcriber) == "" {
		doc.Header.Transcriber = d.metadataUser()
	}

	id, err := d.saveTranscription(0, doc)
	if err != nil {
		return nil, err
	}

	d.transcriptMu.Lock()
	defer d.transcriptMu.Unlock()
	return opened(id, d.openTranscript(id, doc)), nil
}

// metadataUser is the library's `user` metadata, trimmed, and "" when the
// library says nothing or is not open. It is the default transcriber of a new
// draft and nothing else: a draft carries the name it was given at creation,
// so renaming the user later does not rewrite the drafts already typed.
func (d *Database) metadataUser() string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return ""
	}
	meta, err := d.store.Metadata().Load(context.Background(), "")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(meta["user"])
}

// OpenTranscription loads a draft and returns it replayed in full: the
// position of every Action, the scores, the Crawford games, the
// Inconsistencies. The Cursor lands at the end of the document, since the
// Entry a correction was in the middle of is not persisted.
func (d *Database) OpenTranscription(id int64) (*TranscriptionState, error) {
	doc, err := d.loadTranscription(id)
	if err != nil {
		return nil, err
	}

	d.transcriptMu.Lock()
	defer d.transcriptMu.Unlock()
	// A draft already open keeps the session it has: reopening the tab must
	// not throw away an Entry or an undo stack the user still has in hand.
	if e := d.transcriptSessions[id]; e != nil {
		return opened(id, e), nil
	}
	return opened(id, d.openTranscript(id, doc)), nil
}

// TranscriptionMAT renders the open draft as the .mat text a Jellyfish or a
// gnubg reader would take. It is the SAME renderer the library's matches go
// through — transcript.MatchParts builds the graph, ingest.RenderMAT writes
// it — because the panel's ".mat text" pane is a view of the export and not a
// second opinion about it: a transcript written twice drifts, and the one the
// user reads must be the one the file will hold.
//
// Nothing is written: no row, no file, no Match. A draft with Inconsistencies
// renders all the same (ADR-0044 — nothing is refused); an illegal play goes
// out as played, which is what the export dialog warns about.
func (d *Database) TranscriptionMAT(id int64) (string, error) {
	d.transcriptMu.Lock()
	defer d.transcriptMu.Unlock()

	ed, err := d.session(id)
	if err != nil {
		return "", err
	}
	return ingest.RenderMAT(transcript.MatchParts(ed.Doc)), nil
}

// CloseTranscription ends a draft: the row is deleted and the session
// dropped. Nothing is snapshotted — a draft that was never saved has nothing
// saved to put back, and a draft that WAS saved leaves its Match behind
// (ADR-0045 rule 3). Leaving the tab is not a close; only the panel's explicit
// "close this draft", with its confirmation, calls this.
func (d *Database) CloseTranscription(id int64) error {
	d.transcriptMu.Lock()
	delete(d.transcriptSessions, id)
	d.transcriptMu.Unlock()

	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Transcriptions().Delete(context.Background(), "", id)
}

// ApplyTranscriptionGesture records one gesture and returns the draft as it
// became. The gesture itself is transcript's business; what happens here is
// the write: the row is rewritten, in its own transaction, whenever the
// durable part of the document changed — the header and the Actions.
//
// That is the whole of fonctionnel.md §3, read in both directions. A gesture
// that changes an Action (validating one, correcting it, inserting, deleting,
// changing a side, a length, the metadata) is on disk before the call
// returns. A gesture that only moves the Cursor, only fills a die or only
// picks a candidate writes nothing: the dice and the candidate live in the
// Entry, which is not serialised, and the Cursor is normalised away by
// durableJSON.
//
// The Replay starts at the Action the gesture TOUCHED (transcript.Editor.From),
// so the Cursor comes back on the first Inconsistency at or after it, which is
// what a correction wants to show next. That is not the Cursor: a correction in
// place sends the Cursor back where the user came from, and the Inconsistency it
// just created sits several Actions behind it.
//
// Undo and redo are the two gestures that do NOT go through transcript.Apply —
// a stack is state, and it lives in the Editor (transcript.ErrNotPure says so).
// They are routed here, and they write like any other gesture: undoing a
// validated Action must leave the row without it, or a crash would resurrect a
// gesture the user took back.
func (d *Database) ApplyTranscriptionGesture(id int64, g transcript.Gesture) (*TranscriptionState, error) {
	d.transcriptMu.Lock()
	defer d.transcriptMu.Unlock()

	ed, err := d.session(id)
	if err != nil {
		return nil, err
	}

	before, err := durableJSON(ed.Doc)
	if err != nil {
		return nil, fmt.Errorf("transcription %d: %w", id, err)
	}
	switch g.Kind {
	case transcript.GestureUndo:
		// A stack with nothing on it is not an error: Ctrl-Z at the start of a
		// session is a keystroke that does nothing, exactly as it is in every
		// editor, and an error dialog for it would be noise.
		ed.Undo()
	case transcript.GestureRedo:
		ed.Redo()
	default:
		if err := ed.Apply(g); err != nil {
			return nil, err
		}
	}
	after, err := durableJSON(ed.Doc)
	if err != nil {
		return nil, fmt.Errorf("transcription %d: %w", id, err)
	}
	if !bytes.Equal(before, after) {
		if _, err := d.writeTranscription(id, ed.Doc, after); err != nil {
			return nil, err
		}
	}
	return stateOf(id, ed, ed.From()), nil
}

// ── the session map ──────────────────────────────────────────────────
//
// d.transcriptMu guards it, and the lock ORDER is transcriptMu → d.mu, never
// the reverse: the storage helpers below take d.mu themselves, and they are
// called with transcriptMu held so that "read, apply, write" is one gesture
// and not three interleavable ones.

// openTranscript installs a session for id. Caller holds transcriptMu.
func (d *Database) openTranscript(id int64, doc transcript.Document) *transcript.Editor {
	if d.transcriptSessions == nil {
		d.transcriptSessions = make(map[int64]*transcript.Editor)
	}
	ed := transcript.NewEditor(doc)
	d.transcriptSessions[id] = ed
	return ed
}

// stateOf annotates a session and hands back what every binding returns.
//
// It is where fonctionnel.md §1.4's last sentence is made true — "after a Replay
// the Cursor jumps to the first Inconsistency". The Replay REPORTS where the
// Cursor should land, at or after `from`; if that is not where the session's
// document has it, the document is moved there and annotated again, so that the
// next `h` counts from the cell the user is looking at and not from the one the
// gesture happened to leave behind. The second Replay is the incremental one and
// costs nothing: the Actions did not change, only the Cursor did.
// opened is stateOf for a draft that has just been opened, which is the one
// case where the Cursor does NOT jump to an Inconsistency. A resumed draft
// continues after its last written Action (fonctionnel.md §3, and
// loadTranscription's own comment), and an Inconsistency the user has read and
// chosen to keep must not drag them back to it at every open.
func opened(id int64, ed *transcript.Editor) *TranscriptionState {
	return stateOf(id, ed, len(ed.Doc.Actions))
}

func stateOf(id int64, ed *transcript.Editor, from int) *TranscriptionState {
	ann := ed.Replay(from)
	if ann.Cursor != ed.Doc.Cursor {
		ed.SeekCursor(ann.Cursor)
		ann = ed.Replay(ed.Doc.Cursor)
	}
	return &TranscriptionState{ID: id, Annotated: ann, CanUndo: ed.CanUndo(), CanRedo: ed.CanRedo()}
}

// session returns the live editor for id, loading the row when the draft was
// never opened in this run of the application (a CLI call, or a gesture that
// reaches a draft the GUI restored from a previous session). Caller holds
// transcriptMu.
func (d *Database) session(id int64) (*transcript.Editor, error) {
	if e := d.transcriptSessions[id]; e != nil {
		return e, nil
	}
	doc, err := d.loadTranscription(id)
	if err != nil {
		return nil, err
	}
	return d.openTranscript(id, doc), nil
}

// loadTranscription reads one row and decodes its document — the resumption
// of fonctionnel.md §3, whether it follows a crash or simply a tab the user
// had left. The Cursor is put back at the END of the document rather than
// wherever the row happens to say: the Action the user was in the middle of
// correcting lived in the Entry, which is not persisted, so the only place a
// resumed draft can honestly continue from is after its last written Action.
// (durableJSON already writes it there; doing it again here is what makes the
// rule true of any row, including one written by another build.)
func (d *Database) loadTranscription(id int64) (transcript.Document, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var doc transcript.Document
	if d.db == nil {
		return doc, fmt.Errorf("no database is currently open")
	}
	row, err := d.store.Transcriptions().Get(context.Background(), "", id)
	if err != nil {
		return doc, err
	}
	doc, err = decodeTranscription(row)
	if err != nil {
		return doc, err
	}
	doc.Cursor = len(doc.Actions)
	return doc, nil
}

// durableJSON encodes the part of a document that a crash must not lose: the
// header and the Actions. It is what the row holds and what "has the draft
// changed?" is asked of.
//
// The Cursor is normalised to the end of the document rather than serialised
// as it stands, and the two reasons are the same one. It is not a fact of the
// match — moving it changes no Action — so a gesture that only walks the
// Transcript must cost no write (fonctionnel.md §3); and a resumed draft
// continues after its last written Action anyway, since the Entry a
// correction was in the middle of is not persisted. The row therefore states
// the Cursor a resumption will actually use.
//
// Entry, the correction's return position and the pending board are `json:"-"`
// in transcript.Document: a crash is allowed to lose the dice half typed and
// the undo stack, never a validated Action (ADR-0045 rule 1).
func durableJSON(doc transcript.Document) ([]byte, error) {
	doc.Cursor = len(doc.Actions)
	blob, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("encode transcription: %w", err)
	}
	return blob, nil
}

// saveTranscription writes doc into row id (0 to insert) and returns the id
// the row has.
func (d *Database) saveTranscription(id int64, doc transcript.Document) (int64, error) {
	blob, err := durableJSON(doc)
	if err != nil {
		return 0, err
	}
	return d.writeTranscription(id, doc, blob)
}

// writeTranscription is the write itself: one statement, and therefore one
// transaction of its own (ADR-0045 rule 1 — the store issues a single INSERT
// or UPDATE in autocommit, so a gesture is committed or it is not, and the
// next gesture cannot be riding in the same transaction as this one).
//
// What "committed" buys is a crash of the application: the row is in the
// write-ahead log, and the next open recovers it. SQLite runs with
// synchronous=NORMAL (storage/sqlite/sqlite.go), so a power cut in the same
// instant may still cost the last commits — the trade the whole database
// makes, not one this file may quietly change for itself.
//
// The format version travels in the document AND in its own column: the
// column is what a future reader filters on without decoding every draft, the
// document is what makes the string true when the row is copied elsewhere.
func (d *Database) writeTranscription(id int64, doc transcript.Document, blob []byte) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return 0, fmt.Errorf("no database is currently open")
	}
	row := &storage.Transcription{
		ID:            id,
		FormatVersion: strconv.Itoa(doc.FormatVersion),
		Label:         labelOf(doc.Header),
		Document:      string(blob),
	}
	if doc.Header.MatchID != nil {
		row.MatchID = *doc.Header.MatchID
	}
	return d.store.Transcriptions().Save(context.Background(), "", row)
}

// decodeTranscription reads the JSON of a stored draft. A document whose
// format version this build does not know is refused rather than half-read:
// the version exists precisely so an older binary can say so instead of
// silently dropping the Actions it cannot parse.
func decodeTranscription(row *storage.Transcription) (transcript.Document, error) {
	var doc transcript.Document
	if err := json.Unmarshal([]byte(row.Document), &doc); err != nil {
		return doc, fmt.Errorf("transcription %d: %w", row.ID, err)
	}
	if doc.FormatVersion > transcript.FormatVersion {
		return doc, fmt.Errorf("transcription %d was written in document format %d, this version reads up to %d",
			row.ID, doc.FormatVersion, transcript.FormatVersion)
	}
	return doc, nil
}

// summarize reads a row's document for the handful of facts the list shows. A
// document that cannot be decoded still yields a line — with the label and
// the dates the row itself carries — because a draft the panel cannot list is
// a draft the user cannot delete either.
func summarize(row *storage.Transcription) TranscriptionSummary {
	s := TranscriptionSummary{
		ID:            row.ID,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
		FormatVersion: row.FormatVersion,
		MatchID:       row.MatchID,
		Label:         row.Label,
		MatchLength:   -1,
		ActionCount:   -1,
	}
	doc, err := decodeTranscription(row)
	if err != nil {
		return s
	}
	s.Player1 = doc.Header.Player1
	s.Player2 = doc.Header.Player2
	s.MatchLength = doc.Header.MatchLength
	s.ActionCount = len(doc.Actions)
	return s
}

// labelOf is the free text the list falls back on and a future search reads:
// the two players when they are known, the event otherwise, nothing at all
// for a draft that has not said who is playing yet.
func labelOf(h transcript.Header) string {
	p1, p2 := strings.TrimSpace(h.Player1), strings.TrimSpace(h.Player2)
	switch {
	case p1 != "" && p2 != "":
		return p1 + " vs " + p2
	case p1 != "":
		return p1
	case p2 != "":
		return p2
	default:
		return strings.TrimSpace(h.Event)
	}
}

// forgetTranscriptSessions drops every open draft. It is called when the
// database handle is replaced or closed, BEFORE d.mu is taken — the lock
// order is transcriptMu → mu, and a session keyed by a row id of the previous
// library would otherwise answer for a row id of the next one.
func (d *Database) forgetTranscriptSessions() {
	d.transcriptMu.Lock()
	defer d.transcriptMu.Unlock()
	d.transcriptSessions = nil
}
