package storage

import "errors"

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
