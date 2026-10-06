package transcription

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/events"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// Summary is one line of the drafts list. Its facts are decoded from the
// document rather than duplicated in columns that could drift.
type Summary struct {
	ID            int64  `json:"id"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
	FormatVersion string `json:"formatVersion"`
	// MatchID is the Match the draft was opened from, 0 for a new match.
	MatchID  int64  `json:"matchId"`
	Label    string `json:"label"`
	Revision int64  `json:"revision"`
	Player1  string `json:"player1"`
	Player2  string `json:"player2"`
	// MatchLength and ActionCount are -1 when the document cannot be read.
	MatchLength int `json:"matchLength"`
	ActionCount int `json:"actionCount"`
}

// List returns the scope's drafts, most recently updated first.
func (s *Service) List(ctx context.Context, scope string) ([]Summary, error) {
	out := []Summary{}
	for row, err := range s.store.Transcriptions().List(ctx, scope) {
		if err != nil {
			return nil, err
		}
		out = append(out, summarize(row))
	}
	return out, nil
}

// Create writes a new draft before any Action, so a crash cannot lose it, and
// opens its session. A zero header means a match of
// transcript.DefaultMatchLength; the transcriber defaults to the library's
// `user` metadata.
func (s *Service) Create(ctx context.Context, scope string, header transcript.Header) (*State, error) {
	doc := transcript.New(header.MatchLength)
	doc.Header = s.draftHeader(ctx, scope, header)
	return s.insert(ctx, scope, doc)
}

// draftHeader is the header a new Transcription starts from: whatever the form
// stated (players, event, rules), with the defaults a typed draft and a
// materialised one share, so the two write the same Match.
func (s *Service) draftHeader(ctx context.Context, scope string, header transcript.Header) transcript.Header {
	// A caller never posts a match id on a draft that has produced no match.
	header.MatchID = nil
	if header.MatchLength == 0 && !header.Jacoby && !header.Beaver {
		// A money draft is Jacoby by default (transcript.New); an all-zero
		// header is "unstated", not "Jacoby off".
		header.Jacoby = true
	}
	if header.Date.IsZero() {
		header.Date = time.Now()
	}
	if strings.TrimSpace(header.Transcriber) == "" {
		if meta, err := s.store.Metadata().Load(ctx, scope); err == nil {
			header.Transcriber = strings.TrimSpace(meta["user"])
		}
	}
	return header
}

// insert writes doc as a new row and opens its session.
func (s *Service) insert(ctx context.Context, scope string, doc transcript.Document) (*State, error) {
	blob, err := DurableJSON(doc)
	if err != nil {
		return nil, err
	}
	row := rowOf(0, doc, blob)
	id, err := s.store.Transcriptions().Save(ctx, scope, row)
	if err != nil {
		return nil, err
	}
	s.published(scope, id, events.Event{Revision: row.Revision})
	ss := s.install(scope, id, doc, row.Revision)
	ss.mu.Lock()
	defer ss.mu.Unlock()
	return opened(id, ss), nil
}

// Open returns a draft replayed in full and the session to type in: the live
// one when the draft is already open, so reopening a tab keeps the Entry and
// the undo stack, otherwise a new one with the Cursor at the end of the
// document.
func (s *Service) Open(ctx context.Context, scope string, id int64) (*State, error) {
	ss, err := s.acquire(ctx, scope, id, "")
	if err != nil {
		return nil, err
	}
	defer ss.mu.Unlock()
	return opened(id, ss), nil
}

// Get reads a draft from its row, replayed, without opening a session: a
// read, which also answers "what is the draft now" after a conflict.
func (s *Service) Get(ctx context.Context, scope string, id int64) (*State, error) {
	doc, rev, err := s.read(ctx, scope, id)
	if err != nil {
		return nil, err
	}
	ed := transcript.NewEditor(doc)
	return annotate(id, ed, rev, "", len(ed.Doc.Actions)), nil
}

// Revision is a draft's current revision, read from its column alone.
func (s *Service) Revision(ctx context.Context, scope string, id int64) (int64, error) {
	row, err := s.store.Transcriptions().Get(ctx, scope, id)
	if err != nil {
		return 0, err
	}
	return row.Revision, nil
}

// Apply records one gesture and returns the draft. The row is rewritten, and
// the revision advances, only when the durable document — header and
// Actions — changed: a die typed in the Entry or a Cursor moved lives in the
// session alone, writes nothing and leaves the revision where it was, so it
// costs no write per keystroke and works on a library opened read-only.
//
// The Replay starts at the Action the gesture TOUCHED (Editor.From), not at
// the Cursor: a correction in place returns the Cursor elsewhere, while the
// Inconsistency it created lies behind. Undo and redo bypass transcript.Apply
// (the stack lives in the Editor) but write like any gesture, or a crash
// would resurrect an undone Action.
func (s *Service) Apply(ctx context.Context, scope string, id int64, exp Expect, g transcript.Gesture) (*State, error) {
	ss, err := s.acquire(ctx, scope, id, exp.Session)
	if err != nil {
		return nil, err
	}
	defer ss.mu.Unlock()
	if exp.Revision != 0 && exp.Revision != ss.rev {
		return nil, staleSession(id, ss)
	}

	before, err := DurableJSON(ss.ed.Doc)
	if err != nil {
		return nil, fmt.Errorf("transcription %d: %w", id, err)
	}
	switch g.Kind {
	case transcript.GestureUndo:
		// An empty stack is not an error: Ctrl-Z at the start of a session
		// is a keystroke that does nothing, as in every editor.
		ss.ed.Undo()
	case transcript.GestureRedo:
		ss.ed.Redo()
	default:
		if err := ss.ed.Apply(g); err != nil {
			return nil, err
		}
	}
	after, err := DurableJSON(ss.ed.Doc)
	if err != nil {
		return nil, fmt.Errorf("transcription %d: %w", id, err)
	}
	if !bytes.Equal(before, after) {
		if err := s.write(ctx, scope, id, ss, after); err != nil {
			return nil, err
		}
	}
	return annotate(id, ss.ed, ss.rev, ss.id, ss.ed.From()), nil
}

// Abandon deletes the draft without a Match, under the revision the caller
// names (0: none). Nothing is snapshotted, and a Match the draft was opened
// from is left untouched (ADR-0045 §3).
func (s *Service) Abandon(ctx context.Context, scope string, id int64, exp Expect) error {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	if _, err := tx.Transcriptions().Touch(ctx, scope, id, exp.Revision); err != nil {
		_ = tx.Rollback()
		return s.stale(ctx, scope, id, err)
	}
	if err := tx.Transcriptions().Delete(ctx, scope, id); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.published(scope, id, events.Event{Removed: true})
	_ = s.Close(scope, id, "")
	return nil
}

// MAT renders the draft as .mat text through the SAME renderer as the
// library's matches, so a pane shows exactly what the export holds: the live
// session's document when the draft is open, the row's otherwise. A draft
// with Inconsistencies renders all the same (ADR-0044). Writes nothing.
func (s *Service) MAT(ctx context.Context, scope string, id int64) (string, error) {
	doc, err := s.current(ctx, scope, id)
	if err != nil {
		return "", err
	}
	return ingest.RenderMAT(transcript.MatchParts(doc)), nil
}

// SuggestMATFilename is the file name an export of the draft defaults to.
func (s *Service) SuggestMATFilename(ctx context.Context, scope string, id int64) (string, error) {
	doc, err := s.current(ctx, scope, id)
	if err != nil {
		return "", err
	}
	m, _, _ := transcript.MatchParts(doc)
	return ingest.SuggestMATFilename(m), nil
}

// current is the draft's document as its live session holds it, or as the
// row does when no session is open. Opens nothing.
func (s *Service) current(ctx context.Context, scope string, id int64) (transcript.Document, error) {
	var doc transcript.Document
	if s.WithEditor(scope, id, func(ed *transcript.Editor) { doc = ed.Doc }) {
		return doc, nil
	}
	doc, _, err := s.read(ctx, scope, id)
	return doc, err
}

// write rewrites the row from ss under ss's revision; a refusal goes through
// failed (the undo stack of a conflicting session described a document that
// is no longer there). Caller holds ss.mu.
func (s *Service) write(ctx context.Context, scope string, id int64, ss *session, blob []byte) error {
	if blob == nil {
		var err error
		if blob, err = DurableJSON(ss.ed.Doc); err != nil {
			return err
		}
	}
	row := rowOf(id, ss.ed.Doc, blob)
	row.Revision = ss.rev
	if _, err := s.store.Transcriptions().Save(ctx, scope, row); err != nil {
		return s.failed(ctx, scope, id, ss, err)
	}
	ss.rev = row.Revision
	s.published(scope, id, events.Event{Revision: row.Revision})
	return nil
}

// failed handles a write the store refused. A conflict reloads the session
// from the row and reports it fresh; any other failure drops the session,
// whose Editor no longer agrees with the row. Caller holds ss.mu.
func (s *Service) failed(ctx context.Context, scope string, id int64, ss *session, err error) error {
	if errors.Is(err, storage.ErrConflict) {
		if lerr := s.load(ctx, scope, id, ss); lerr == nil {
			return staleSession(id, ss)
		}
	}
	s.drop(scope, id, ss)
	return err
}

// staleSession is the conflict a session answers with: its own state, which
// is the row's. It is read off a copy replayed from the end, so that neither
// the shared session's Cursor nor the one handed back jumps to an
// Inconsistency. Caller holds ss.mu.
func staleSession(id int64, ss *session) *StaleError {
	view := transcript.NewEditor(ss.ed.Doc)
	st := &State{ID: id, Revision: ss.rev, SessionID: ss.id, Annotated: view.Replay(len(view.Doc.Actions)),
		CanUndo: ss.ed.CanUndo(), CanRedo: ss.ed.CanRedo()}
	return &StaleError{Revision: ss.rev, State: st}
}

// stale turns a refused conditional write outside any session into a
// StaleError carrying the draft as its row now holds it; any other error
// passes through.
func (s *Service) stale(ctx context.Context, scope string, id int64, err error) error {
	if !errors.Is(err, storage.ErrConflict) {
		return err
	}
	st, gerr := s.Get(ctx, scope, id)
	if gerr != nil {
		return gerr
	}
	return &StaleError{Revision: st.Revision, State: st}
}

// read loads a row and decodes its document, Cursor at the END: the Entry is
// not persisted, so a resumed draft continues after its last Action.
func (s *Service) read(ctx context.Context, scope string, id int64) (transcript.Document, int64, error) {
	row, err := s.store.Transcriptions().Get(ctx, scope, id)
	if err != nil {
		return transcript.Document{}, 0, err
	}
	doc, err := Decode(row)
	if err != nil {
		return doc, 0, err
	}
	doc.Cursor = len(doc.Actions)
	return doc, row.Revision, nil
}

// opened is annotate for a freshly opened draft, the one case where the
// Cursor does NOT jump to an Inconsistency: a resumed draft continues after
// its last Action, and a kept Inconsistency must not drag the user back at
// every open. Caller holds ss.mu.
func opened(id int64, ss *session) *State {
	return annotate(id, ss.ed, ss.rev, ss.id, len(ss.ed.Doc.Actions))
}

// annotate replays the Editor. After a Replay the Cursor jumps to the first
// Inconsistency at or after `from`; the document is moved there and replayed
// again (incremental, free) so the next `h` counts from the cell the user
// sees.
func annotate(id int64, ed *transcript.Editor, rev int64, sessionID string, from int) *State {
	ann := ed.Replay(from)
	if ann.Cursor != ed.Doc.Cursor {
		ed.SeekCursor(ann.Cursor)
		ann = ed.Replay(ed.Doc.Cursor)
	}
	return &State{ID: id, Revision: rev, SessionID: sessionID, Annotated: ann, CanUndo: ed.CanUndo(), CanRedo: ed.CanRedo()}
}

// DurableJSON encodes what a crash must not lose — header and Actions — and
// is what "has the draft changed?" compares. The Cursor is normalised to the
// end: it is not a fact of the match, so moving it costs no rewrite of the
// document, and a resumption starts there anyway. Entry and the undo stack
// are `json:"-"` (ADR-0045 rule 1).
func DurableJSON(doc transcript.Document) ([]byte, error) {
	doc.Cursor = len(doc.Actions)
	if doc.Actions == nil {
		// No Action yet, however the slice came to be: one encoding, or an
		// untouched draft would compare as changed.
		doc.Actions = []transcript.Action{}
	}
	blob, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("encode transcription: %w", err)
	}
	return blob, nil
}

// Decode reads a stored draft's JSON. An unknown format version is refused
// rather than half-read, which would drop Actions silently; an older one is
// converted to the current shape.
func Decode(row *storage.Transcription) (transcript.Document, error) {
	var doc transcript.Document
	if err := json.Unmarshal([]byte(row.Document), &doc); err != nil {
		return doc, fmt.Errorf("transcription %d: %w", row.ID, err)
	}
	if doc.FormatVersion > transcript.FormatVersion {
		return doc, fmt.Errorf("transcription %d was written in document format %d, this version reads up to %d",
			row.ID, doc.FormatVersion, transcript.FormatVersion)
	}
	return transcript.Upgrade(doc), nil
}

// rowOf is the row a document is written as. The format version is in the
// document (true when copied) AND a column (filterable without decoding).
func rowOf(id int64, doc transcript.Document, blob []byte) *storage.Transcription {
	row := &storage.Transcription{
		ID:            id,
		FormatVersion: strconv.Itoa(doc.FormatVersion),
		Label:         Label(doc.Header),
		Document:      string(blob),
	}
	if doc.Header.MatchID != nil {
		row.MatchID = *doc.Header.MatchID
	}
	return row
}

// summarize reads a row's document for the list. An undecodable document
// still yields a line from the row's own columns, or it could not be deleted.
func summarize(row *storage.Transcription) Summary {
	out := Summary{
		ID:            row.ID,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
		FormatVersion: row.FormatVersion,
		MatchID:       row.MatchID,
		Label:         row.Label,
		Revision:      row.Revision,
		MatchLength:   -1,
		ActionCount:   -1,
	}
	doc, err := Decode(row)
	if err != nil {
		return out
	}
	out.Player1 = doc.Header.Player1
	out.Player2 = doc.Header.Player2
	out.MatchLength = doc.Header.MatchLength
	out.ActionCount = len(doc.Actions)
	return out
}

// Label is the free text the list falls back on and a search reads: the two
// players when they are known, the event otherwise, nothing at all for a
// draft that has not said who is playing yet.
func Label(h transcript.Header) string {
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
