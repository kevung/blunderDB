package storage

import "context"

// Journal operations. Create and update are both an upsert for a reader; delete
// is a removal for good, trash a removal the owner can still undo.
const (
	ChangeCreate = "create"
	ChangeUpdate = "update"
	ChangeDelete = "delete"
	ChangeTrash  = "trash"
)

// Change is the latest thing that happened to one object: the journal keeps
// one entry per uuid, moved to the end each time the object changes, so it
// stays as small as the library and a reader past a cursor sees each object
// once, in its final state.
type Change struct {
	Seq    int64  `json:"seq"`
	UUID   string `json:"uuid"`
	Family string `json:"family"`
	Op     string `json:"op"`
	// ChangedAt is UTC, RFC 3339 with milliseconds: it compares as text.
	ChangedAt string `json:"changed_at"`
}

// ChangeJournalStore reads the journal the triggers of the uuid-bearing
// tables write. A cursor is the seq of the last change a reader saw (0 for
// everything the journal holds). Seq is monotone, not gap-free, and a
// concurrent PostgreSQL writer can commit a lower seq after a higher one was
// read: a reader that must not miss it re-reads a short window behind its cursor.
type ChangeJournalStore interface {
	// Changes lists up to limit changes with seq > after, oldest first.
	Changes(ctx context.Context, scope string, after int64, limit int) ([]Change, error)
	// Entry is the journal entry of one object, ErrNotFound when it has none.
	Entry(ctx context.Context, scope, uuid string) (Change, error)
	// Mark records an operation the triggers cannot see (a delete that went
	// through the trash is a trash, not a delete), or restamps an entry with
	// the time it had where it was made (at, as Change.ChangedAt; "" is now),
	// so a change that travelled is not newer than the one it carries.
	Mark(ctx context.Context, scope, family, uuid, op, at string) error
}

// ChangeJournalProvider is implemented by the Stores of a backend that keeps
// the journal.
type ChangeJournalProvider interface {
	ChangeJournal() ChangeJournalStore
}
