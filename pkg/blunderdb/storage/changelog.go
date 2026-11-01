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
	// Xid is the writer's transaction on PostgreSQL (Cursor), 0 on SQLite and
	// in Entry.
	Xid    int64  `json:"xid,omitempty"`
	Seq    int64  `json:"seq"`
	UUID   string `json:"uuid"`
	Family string `json:"family"`
	Op     string `json:"op"`
	// ChangedAt is UTC, RFC 3339 with milliseconds (the layout ChangedAtLayout
	// names). Anything that reads a ChangedAt from elsewhere normalises it
	// first: only that layout compares as text.
	ChangedAt string `json:"changed_at"`
}

// ChangedAtLayout is the layout of Change.ChangedAt.
const ChangedAtLayout = "2006-01-02T15:04:05.000Z"

// ChangeJournalStore reads the journal the triggers of the uuid-bearing
// tables write. A cursor is the seq of the last change a reader saw (0 for
// everything the journal holds). Seq is monotone, not gap-free, and a
// concurrent PostgreSQL writer can commit a lower seq after a higher one was
// read; PostgreSQL therefore lists an entry only once its writer's xid8 is
// below pg_snapshot_xmin, i.e. every transaction that started before it has
// ended. What remains: two writers whose seq draw order differs from their xid
// order (the older xid draws its seq second) can still be read out of order;
// a cursor on (xid, seq) would close it. SQLite has one writer at a time.
// Cursor is where a reader stopped: the last change it saw. On PostgreSQL the
// journal is read in the order of the writers' transactions (Xid), then of
// their seq, because seq alone is drawn out of commit order; with the guard
// that only entries behind pg_snapshot_xmin are listed, a reader never passes
// a writer still in flight. SQLite has one writer at a time: Xid stays 0 and
// seq orders.
type Cursor struct {
	Xid int64 `json:"xid,omitempty"`
	Seq int64 `json:"seq"`
}

type ChangeJournalStore interface {
	// Changes lists up to limit changes with seq > after, oldest first.
	Changes(ctx context.Context, scope string, after int64, limit int) ([]Change, error)
	// ChangesOf is Changes restricted to the given families (all when empty).
	ChangesOf(ctx context.Context, scope string, families []string, after Cursor, limit int) ([]Change, error)
	// Unjournaled lists the uuids of a family's rows that have no entry: the
	// rows that existed before the journal did, which no cursor reaches.
	Unjournaled(ctx context.Context, scope, family string) ([]string, error)
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
