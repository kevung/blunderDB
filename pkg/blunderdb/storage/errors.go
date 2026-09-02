package storage

import (
	"errors"
	"fmt"
)

// Typed sentinel errors returned by Storage implementations. They map onto a
// closed set of categories so callers (and the future HTTP layer in P6) can
// branch without inspecting backend-specific error strings.
//
// Use errors.Is to test them; implementations should wrap them with %w to add
// context.
var (
	// ErrNotFound — the requested row does not exist.
	ErrNotFound = errors.New("storage: not found")

	// ErrConflict — the operation violates a uniqueness or referential
	// constraint (e.g. a duplicate Zobrist hash, a duplicate match import).
	ErrConflict = errors.New("storage: conflict")

	// ErrInvalid — the request is malformed or fails validation before it
	// reaches the backend.
	ErrInvalid = errors.New("storage: invalid argument")

	// ErrInternal — an unexpected backend failure.
	ErrInternal = errors.New("storage: internal error")
)

// DuplicatePositionError is what PositionStore.Update returns when the edited
// position has become identical (same Zobrist hash) to another stored one.
// Positions are identified by that hash, so the update would create a second
// row for one position; it is refused and the row is left as it was. It
// matches ErrConflict under errors.Is; ExistingID names the row the user can
// go to instead. The message is part of the contract — the GUI reads the id
// out of it to translate it.
type DuplicatePositionError struct {
	ExistingID int64
}

func (e *DuplicatePositionError) Error() string {
	return fmt.Sprintf("this position already exists (id %d)", e.ExistingID)
}

// Is makes errors.Is(err, ErrConflict) true for a duplicate position.
func (e *DuplicatePositionError) Is(target error) bool { return target == ErrConflict }
