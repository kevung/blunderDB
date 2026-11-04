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
	// ChangedAt only grows for an object written here: a write is dated now, or
	// a millisecond past the date the object holds when the clock has not passed
	// it, so a peer that holds the earlier write takes the later one. A burst of
	// more than a thousand writes of one object faster than the clock dates more
	// than a second ahead, past changeset's slack: two peers may then part for
	// good. No real path writes so; see tasks/BACKLOG.md.
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
//
// Family and the Read and End fields are a paged export in progress (package
// changeset): a caller stores the cursor whole and treats it as opaque.
type Cursor struct {
	Xid int64 `json:"xid,omitempty"`
	Seq int64 `json:"seq"`
	// Epoch names the PostgreSQL cluster and timeline whose transaction counter
	// Xid is on: a backup restored elsewhere restarts that counter and a
	// point-in-time recovery rewinds it, so no order holds between a cursor from
	// before and an entry from after (ErrCursorStale).
	Epoch   string `json:"epoch,omitempty"`
	Family  string `json:"family,omitempty"`
	ReadXid int64  `json:"read_xid,omitempty"`
	ReadSeq int64  `json:"read_seq,omitempty"`
	EndXid  int64  `json:"end_xid,omitempty"`
	EndSeq  int64  `json:"end_seq,omitempty"`
}

// IsStart says the cursor names no change: everything is after it.
func (c Cursor) IsStart() bool { return c.Xid == 0 && c.Seq == 0 && c.Family == "" }

type ChangeJournalStore interface {
	// Changes lists up to limit changes with seq > after, oldest first.
	Changes(ctx context.Context, scope string, after int64, limit int) ([]Change, error)
	// ChangesOf is Changes restricted to the given families (all when empty).
	ChangesOf(ctx context.Context, scope string, families []string, after Cursor, limit int) ([]Change, error)
	// Horizon is the last change of the given families a reader can see now
	// (the zero Cursor when none), with the journal's Epoch. An entry that
	// appears later sorts after it. ErrCursorStale when after cannot be on this
	// journal.
	Horizon(ctx context.Context, scope string, families []string, after Cursor) (Cursor, error)
	// Unjournaled lists up to limit uuids of a family's rows that have no
	// entry and whose id is past afterID, by id: the rows that existed before
	// the journal did, which no cursor reaches. The second result is the id
	// of the last row listed, to pass as afterID for the next page.
	Unjournaled(ctx context.Context, scope, family string, afterID int64, limit int) ([]string, int64, error)
	// Entry is the journal entry of one object, ErrNotFound when it has none.
	Entry(ctx context.Context, scope, uuid string) (Change, error)
	// Mark records an operation the triggers cannot see (a delete that went
	// through the trash is a trash, not a delete), or restamps an entry with
	// the time it had where it was made (at, as Change.ChangedAt; "" is now),
	// so a change that travelled is not newer than the one it carries.
	Mark(ctx context.Context, scope, family, uuid, op, at string) error
	// Adopt gives the objects an entry dated at as a create, where they have
	// none: an object edited meanwhile keeps its own.
	Adopt(ctx context.Context, scope, family string, uuids []string, at string) error
	// Applying says that what the current transaction writes comes from a peer
	// and is no change of this library. Until it is called with "" the triggers
	// date nothing now: a row they would journal as an update keeps its entry,
	// a row that goes is dated at, and a row that appears is dated below the
	// floor, so that its real record always wins.
	Applying(ctx context.Context, scope, at string) error
	// PartFact is the last fact known of a part of a position; false when
	// there has never been one.
	PartFact(ctx context.Context, scope string, positionID int64, part string) (PartFact, bool, error)
	// SetPartFact replaces the fact of f's part of f's position.
	SetPartFact(ctx context.Context, scope string, f PartFact) error
	// PendingLinks lists the derivation facts still waiting for their origin.
	PendingLinks(ctx context.Context, scope string) ([]PartFact, error)
}

// The parts of a position that are written apart from it, and so can differ
// between two libraries that both hold the position.
const (
	PartDrawing    = "drawing"
	PartDerivation = "derivation"
)

// PartFact dates the last write of a part of a position. No content with a fact
// is an erasure: it is what tells "erased" from "never heard of".
type PartFact struct {
	PositionID int64
	Part       string
	At         string
	// PendingOrigin is the board hash of an origin this library does not hold
	// yet, and PendingMove the move of that link (PartDerivation only).
	PendingOrigin, PendingMove string
}

// ChangeJournalProvider is implemented by the Stores of a backend that keeps
// the journal.
type ChangeJournalProvider interface {
	ChangeJournal() ChangeJournalStore
}
