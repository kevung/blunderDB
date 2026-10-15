package storage

import (
	"errors"
	"fmt"
)

// Typed sentinel errors returned by Storage implementations. They map onto a
// closed set of categories so callers (the HTTP layer included) can branch
// without inspecting backend-specific error strings.
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

// The refusals of MatchStore.SwapPlayers. Both match ErrInvalid; their
// messages are part of the contract — the GUI recognises them to translate
// them.
var (
	// ErrSwapDraftOpen — a Transcription draft is open on the match: its
	// Finish would replace the match in the order the draft holds, and
	// silently undo the swap.
	ErrSwapDraftOpen = fmt.Errorf("%w: a transcription draft is open on the match", ErrInvalid)

	// ErrSwapPlayedHere — the match was played here, in a Duel: its origin
	// (the Start XGID, the seat that ran over time, the declared Bots, the
	// seed contributions) names the seats as they were played (ADR-0072
	// rule 5).
	ErrSwapPlayedHere = fmt.Errorf("%w: the match was played here, under arbitration", ErrInvalid)
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

// ErrNoReviewToUndo — AnkiStore.UndoReview found no review of the card whose
// prior state was kept: the card was never graded, every grade was already
// undone, or the remaining one predates the snapshot (an older library, an
// imported history). It matches ErrConflict; its message is part of the
// contract — the GUI recognises it to translate it.
var ErrNoReviewToUndo = fmt.Errorf("%w: the card has no review to undo", ErrConflict)
