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

// Transcriptions (ADR-0045): plumbing only. The rules live in
// pkg/blunderdb/transcript, persistence in storage.TranscriptionStore; this
// file reads the row, applies the gesture, writes the row back.
//
// An opened draft keeps a transcript.Editor in memory because the Entry (the
// Action being typed) and the undo stack are deliberately not serialised
// (ADR-0045 rule 1), yet successive gestures need them. The session is a
// cache, never the truth: the row is rewritten after every durable change.

// TranscriptionSummary is one line of the drafts list. Its facts are decoded
// from the document rather than duplicated in columns that could drift.
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
// and the annotated document. The panel derives nothing itself (ADR-0045
// rule 9).
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
// A zero header means a match of transcript.DefaultMatchLength. The draft is
// written immediately, before any Action, so a crash cannot lose it.
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
	// Defaults only (today, the library's user); the metadata pane overwrites
	// both without touching the library's `user` setting.
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

// metadataUser is the library's `user` metadata, trimmed, "" when absent or
// not open: only the default transcriber of a new draft.
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

// TranscriptionMAT renders the open draft as .mat text through the SAME
// renderer as the library's matches (transcript.MatchParts + ingest.RenderMAT),
// so the pane shows exactly what the export holds. Writes nothing. A draft
// with Inconsistencies renders all the same (ADR-0044); an illegal play goes
// out as played.
func (d *Database) TranscriptionMAT(id int64) (string, error) {
	d.transcriptMu.Lock()
	defer d.transcriptMu.Unlock()

	ed, err := d.session(id)
	if err != nil {
		return "", err
	}
	return ingest.RenderMAT(transcript.MatchParts(ed.Doc)), nil
}

// CloseTranscription deletes the row and drops the session. Nothing is
// snapshotted: a saved draft leaves its Match behind (ADR-0045 rule 3).
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

// ApplyTranscriptionGesture records one gesture and returns the draft. The row
// is rewritten, in its own transaction, whenever the header or the Actions
// changed; a gesture that only moves the Cursor, fills a die or picks a
// candidate writes nothing (Entry is not serialised, durableJSON normalises
// the Cursor).
//
// The Replay starts at the Action the gesture TOUCHED (Editor.From), not at
// the Cursor: a correction in place returns the Cursor elsewhere, while the
// Inconsistency it created lies behind.
//
// Undo and redo bypass transcript.Apply (the stack lives in the Editor) but
// write like any gesture, or a crash would resurrect an undone Action.
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

// The session map: d.transcriptMu guards it, lock ORDER transcriptMu → d.mu,
// never the reverse. The storage helpers take d.mu under transcriptMu so
// "read, apply, write" is one gesture, not three interleavable ones.

// openTranscript installs a session for id. Caller holds transcriptMu.
func (d *Database) openTranscript(id int64, doc transcript.Document) *transcript.Editor {
	if d.transcriptSessions == nil {
		d.transcriptSessions = make(map[int64]*transcript.Editor)
	}
	ed := transcript.NewEditor(doc)
	d.transcriptSessions[id] = ed
	return ed
}

// opened is stateOf for a freshly opened draft, the one case where the Cursor
// does NOT jump to an Inconsistency: a resumed draft continues after its last
// Action, and a kept Inconsistency must not drag the user back at every open.
func opened(id int64, ed *transcript.Editor) *TranscriptionState {
	return stateOf(id, ed, len(ed.Doc.Actions))
}

// stateOf annotates a session and hands back what every binding returns. After
// a Replay the Cursor jumps to the first Inconsistency at or after `from`; the
// document is moved there and replayed again (incremental, free) so the next
// `h` counts from the cell the user sees.
func stateOf(id int64, ed *transcript.Editor, from int) *TranscriptionState {
	ann := ed.Replay(from)
	if ann.Cursor != ed.Doc.Cursor {
		ed.SeekCursor(ann.Cursor)
		ann = ed.Replay(ed.Doc.Cursor)
	}
	return &TranscriptionState{ID: id, Annotated: ann, CanUndo: ed.CanUndo(), CanRedo: ed.CanRedo()}
}

// session returns the live editor for id, loading the row when the draft was
// not opened in this run. Caller holds transcriptMu.
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

// loadTranscription reads one row and decodes its document, putting the
// Cursor at the END: the Entry is not persisted, so a resumed draft continues
// after its last Action. Redone here (durableJSON already does it) so the
// rule holds for rows written by any build.
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

// durableJSON encodes what a crash must not lose — header and Actions — and is
// what "has the draft changed?" compares. The Cursor is normalised to the end:
// it is not a fact of the match, so moving it must cost no write, and a
// resumption starts there anyway. Entry and the undo stack are `json:"-"`
// (ADR-0045 rule 1).
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

// writeTranscription is one autocommit statement, so one gesture is one
// transaction (ADR-0045 rule 1). It survives an application crash; under
// synchronous=NORMAL a power cut may still cost the last commits, the trade
// the whole database makes. The format version is in the document (true when
// copied) AND a column (filterable without decoding).
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

// decodeTranscription reads a stored draft's JSON. An unknown format version
// is refused rather than half-read, which would drop Actions silently.
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

// summarize reads a row's document for the list. An undecodable document
// still yields a line from the row's own columns, or the user could not
// delete it.
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

// forgetTranscriptSessions drops every open draft when the handle is replaced
// or closed, BEFORE d.mu is taken (order transcriptMu → mu); otherwise a row
// id of the previous library would answer for the next one.
func (d *Database) forgetTranscriptSessions() {
	d.transcriptMu.Lock()
	defer d.transcriptMu.Unlock()
	d.transcriptSessions = nil
}
