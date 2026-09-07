package storage

import (
	"context"
	"iter"
)

// Transcription is a DRAFT of a match: what the user is typing in, before it
// is worth a row in match/game/move (ADR-0045 §1). Everything the drafting
// itself needs lives in Document, one opaque JSON string this package never
// looks inside; the other fields are the handful of facts the library list
// shows without parsing it.
//
// FormatVersion versions the DOCUMENT, not the database. A change to the
// document's shape is a version of the document, read and upgraded by
// whoever understands it — never a DatabaseVersion migration, which is the
// whole point of storing the draft as one JSON string.
type Transcription struct {
	ID int64 `json:"id"`
	// CreatedAt and UpdatedAt are the backend's timestamps, as text ("" when
	// the backend left them NULL). They are written by the store, never by
	// the caller.
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	// FormatVersion is the version of the Document's own format.
	FormatVersion string `json:"format_version"`
	// MatchID is the Match this draft has already produced, 0 while it has
	// produced none. Deleting that match sets the column back to NULL rather
	// than destroying the typing that produced it: the draft simply becomes
	// one that was never saved.
	MatchID int64 `json:"match_id"`
	// Label is what the list shows — the players, the event, whatever the
	// writer put there. Free text, possibly empty.
	Label string `json:"label"`
	// Document is the draft itself, opaque to storage.
	Document string `json:"document"`
}

// TranscriptionStore persists the drafts. A draft is written after every
// gesture the user records (ADR-0045 §1), so Save is the hot path: it inserts
// when the Transcription carries no id and rewrites the row in place
// otherwise, touching updated_at either way.
type TranscriptionStore interface {
	// List streams the scope's drafts, most recently updated first.
	List(ctx context.Context, scope string) iter.Seq2[*Transcription, error]

	// Get returns one draft, or ErrNotFound.
	Get(ctx context.Context, scope string, id int64) (*Transcription, error)

	// Save writes t and returns its id: an insert when t.ID is 0 (t.ID is
	// left untouched — the caller reads the returned id), an in-place rewrite
	// otherwise, which reports ErrNotFound when the row is gone. A MatchID
	// that names no match of the scope reports ErrNotFound.
	Save(ctx context.Context, scope string, t *Transcription) (int64, error)

	// Delete removes a draft, or reports ErrNotFound. Nothing is snapshotted:
	// closing a draft that was never saved has nothing saved to put back
	// (ADR-0045 §3).
	Delete(ctx context.Context, scope string, id int64) error
}
