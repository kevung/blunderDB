package storage

import "context"

// ObjectUUIDStore reads and sets the uuid of an object by its integer id.
// Families are the sqlshared.ObjectUUIDFamilies names ("position", "comment",
// "collection", ...). Not part of Stores: a backend that keeps no uuid
// (the remote client) does not offer it, and a caller that needs it asks
// ObjectUUIDProvider.
type ObjectUUIDStore interface {
	// ObjectUUID is the uuid of the object, "" when the row holds none.
	ObjectUUID(ctx context.Context, scope, family string, id int64) (string, error)
	// SetObjectUUID gives the object a uuid it held before (a restore from
	// the trash). A uuid another row of the scope now holds is left alone:
	// the object keeps the one it has.
	SetObjectUUID(ctx context.Context, scope, family string, id int64, uuid string) error
}

// ObjectUUIDProvider is implemented by the Stores of a backend that keeps
// object uuids.
type ObjectUUIDProvider interface {
	ObjectUUIDs() ObjectUUIDStore
}
