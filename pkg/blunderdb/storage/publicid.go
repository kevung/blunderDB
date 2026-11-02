package storage

import "context"

// PublicIDStore reads and sets the public_id of a position: an opaque label a
// caller attaches (a link, a slug, a foreign key). The engine keeps it byte
// for byte and returns it; it never interprets, validates or searches it.
// It travels with the position in the change journal and the changeset.
type PublicIDStore interface {
	// PublicID is the label of the position, "" when none is set; ErrNotFound
	// when the position does not exist in the scope.
	PublicID(ctx context.Context, scope string, positionID int64) (string, error)
	// SetPublicID sets the label ("" clears it), or ErrNotFound. Writing it is
	// an edit of the position: the journal shows it.
	SetPublicID(ctx context.Context, scope string, positionID int64, publicID string) error
}

// PublicIDProvider is implemented by the Stores of a backend that keeps
// public_id (the remote client does not).
type PublicIDProvider interface {
	PublicIDs() PublicIDStore
}
