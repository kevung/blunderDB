package domain

import "encoding/json"

// The trash (issue #285, ADR-0036).
//
// blunderDB destroyed things on a confirmation dialog and nothing else. A
// collection, a comment, an Anki card, a position: all final the instant the
// dialog was accepted, and the only recovery a backup the user may not have.
//
// The trash is a SNAPSHOT, not a soft-delete flag. The delete really happens,
// and a JSON copy of what was deleted is written first — so not one of the
// fifty search filters, neither statistics backend, and neither copy of the
// retention predicate has to know that a row might be "deleted but present".
// ADR-0036 has the full reasoning; the short version is that a `deleted_at`
// column would have to be honoured in every one of those places, and the one
// that forgot would not fail loudly.

// TrashKind names what a trash entry holds.
type TrashKind string

const (
	// TrashPosition — a position, with enough of it to be re-saved. Restoring
	// re-Saves it, so the Zobrist deduplication decides where it lands: onto
	// the row that already holds the position if one came back meanwhile,
	// onto a NEW row otherwise. The id is not preserved — the old row is
	// gone and SQLite's AUTOINCREMENT does not reuse it — and that is the
	// honest consequence of the trash being a snapshot rather than a
	// deleted_at flag. What is preserved is the position, its analysis and
	// its comments; what is not is a number nothing outside the database
	// refers to.
	TrashPosition TrashKind = "position"
	// TrashCollection — a collection with its membership, so restoring gives
	// back the list and not just the name. A member position that has since
	// been deleted is simply absent on restore.
	TrashCollection TrashKind = "collection"
	// TrashComment — one comment entry, with its position and its origin.
	TrashComment TrashKind = "comment"
	// TrashAnkiCard — one card's scheduling state, so restoring puts it back
	// where FSRS had it rather than as new.
	TrashAnkiCard TrashKind = "anki_card"
)

// TrashEntry is one deleted thing, kept so it can be put back.
type TrashEntry struct {
	ID   int64     `json:"id"`
	Kind TrashKind `json:"kind"`
	// Label is what the trash list shows: "Position 412", a collection's name.
	// Built when the entry is written, because the thing it describes is gone
	// by the time anyone reads it.
	Label string `json:"label"`
	// DeletedAt is when it was deleted, as the backend spells a timestamp.
	DeletedAt string `json:"deletedAt"`
	// Payload is everything needed to put it back, as JSON. Its shape depends
	// on Kind and is this package's business — see TrashPositionPayload and
	// its siblings.
	Payload json.RawMessage `json:"payload"`
}

// TrashPositionPayload is what a deleted position keeps.
//
// The analyses and comments that cascaded off the position are carried too:
// restoring a position without what the user had written on it would be a
// restore in name only. What is NOT carried is the position's membership of
// collections and Anki decks — those are the collection's and the deck's own
// rows, and a position deleted out from under them was already removed from
// them by the cascade.
type TrashPositionPayload struct {
	Position Position          `json:"position"`
	Analysis *PositionAnalysis `json:"analysis,omitempty"`
	Comments []CommentEntry    `json:"comments,omitempty"`
}

// TrashCollectionPayload is what a deleted collection keeps: enough of the
// collection to recreate it, and the ids of the positions that were in it, in
// order.
//
// The fields are spelled out here rather than reusing storage.Collection: this
// package has no dependency beyond the standard library, and a payload that
// travelled through a persistence type would be a persistence type in a
// domain document.
type TrashCollectionPayload struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	SortOrder   int     `json:"sortOrder"`
	PositionIDs []int64 `json:"positionIds,omitempty"`
}

// TrashCommentPayload is what a deleted comment entry keeps.
type TrashCommentPayload struct {
	Comment CommentEntry `json:"comment"`
}

// TrashAnkiCardPayload is what a deleted Anki card keeps: the card with its
// scheduling state, and the deck it belonged to.
type TrashAnkiCardPayload struct {
	Card   AnkiCard `json:"card"`
	DeckID int64    `json:"deckId"`
}

// TrashRetentionDays is how long a deleted thing stays recoverable.
// `blunderdb vacuum` drops what is older; nothing purges on open.
const TrashRetentionDays = 30
