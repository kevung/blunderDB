package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

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

	id, err := d.saveTranscription(0, doc)
	if err != nil {
		return nil, err
	}

	d.transcriptMu.Lock()
	defer d.transcriptMu.Unlock()
	d.openTranscript(id, doc)
	return &TranscriptionState{ID: id, Annotated: transcript.Replay(doc, 0)}, nil
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
		return &TranscriptionState{ID: id, Annotated: transcript.Replay(e.Doc, 0)}, nil
	}
	d.openTranscript(id, doc)
	return &TranscriptionState{ID: id, Annotated: transcript.Replay(doc, 0)}, nil
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
// the write: the row is rewritten whenever the persisted part of the document
// changed, which the dice being typed do not — Entry lives in the session and
// nowhere else, so a keystroke that only fills a die costs no write at all.
//
// The Replay starts at the Cursor the gesture left, so the Cursor comes back
// on the first Inconsistency at or after the Action just touched, which is
// what a correction wants to show next.
func (d *Database) ApplyTranscriptionGesture(id int64, g transcript.Gesture) (*TranscriptionState, error) {
	d.transcriptMu.Lock()
	defer d.transcriptMu.Unlock()

	ed, err := d.session(id)
	if err != nil {
		return nil, err
	}

	before, err := json.Marshal(ed.Doc)
	if err != nil {
		return nil, fmt.Errorf("encode transcription %d: %w", id, err)
	}
	if err := ed.Apply(g); err != nil {
		return nil, err
	}
	after, err := json.Marshal(ed.Doc)
	if err != nil {
		return nil, fmt.Errorf("encode transcription %d: %w", id, err)
	}
	if string(before) != string(after) {
		if _, err := d.saveTranscription(id, ed.Doc); err != nil {
			return nil, err
		}
	}
	return &TranscriptionState{ID: id, Annotated: transcript.Replay(ed.Doc, ed.Doc.Cursor)}, nil
}

// ── the session map ──────────────────────────────────────────────────
//
// d.transcriptMu guards it, and the lock ORDER is transcriptMu → d.mu, never
// the reverse: the storage helpers below take d.mu themselves, and they are
// called with transcriptMu held so that "read, apply, write" is one gesture
// and not three interleavable ones.

// openTranscript installs a session for id. Caller holds transcriptMu.
func (d *Database) openTranscript(id int64, doc transcript.Document) {
	if d.transcriptSessions == nil {
		d.transcriptSessions = make(map[int64]*transcript.Editor)
	}
	d.transcriptSessions[id] = transcript.NewEditor(doc)
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
	d.openTranscript(id, doc)
	return d.transcriptSessions[id], nil
}

// loadTranscription reads one row and decodes its document.
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
	return decodeTranscription(row)
}

// saveTranscription writes doc into row id (0 to insert) and returns the id
// the row has. The format version travels in the document AND in its own
// column: the column is what a future reader filters on without decoding
// every draft, the document is what makes the string true when the row is
// copied elsewhere.
func (d *Database) saveTranscription(id int64, doc transcript.Document) (int64, error) {
	blob, err := json.Marshal(doc)
	if err != nil {
		return 0, fmt.Errorf("encode transcription: %w", err)
	}

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
