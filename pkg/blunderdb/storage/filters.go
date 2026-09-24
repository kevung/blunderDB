package storage

import (
	"context"
	"iter"
)

// Filter is a named, saved search command (the filter library).
//
// Pinned marks a favourite: the interface keeps the pinned filters within one
// gesture (a chip, Alt+1…9), in library order. It is a reading habit of the
// scope, not part of the filter: renaming keeps it, deleting drops it, and an
// export of the library does not carry it.
type Filter struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Command string `json:"command"`
	Pinned  bool   `json:"pinned"`
}

// FilterStore persists the saved-filter library and the per-filter "edit
// position" and "exclude position" scratch state.
type FilterStore interface {
	Save(ctx context.Context, scope string, name, command string) (int64, error)
	Update(ctx context.Context, scope string, id int64, name, command string) error
	Delete(ctx context.Context, scope string, id int64) error
	List(ctx context.Context, scope string) iter.Seq2[*Filter, error]

	// SetPinned pins or unpins a filter, or reports ErrNotFound for an id the
	// scope does not hold. Pinning twice, or unpinning a filter that is not
	// pinned, is not an error.
	SetPinned(ctx context.Context, scope string, id int64, pinned bool) error

	// SaveEditPosition stores the in-progress edit position for a named filter.
	SaveEditPosition(ctx context.Context, scope string, filterName, editPosition string) error

	// LoadEditPosition returns the stored edit position for a named filter.
	LoadEditPosition(ctx context.Context, scope string, filterName string) (string, error)

	// SaveExcludePosition stores the "Sauf" (exclusion) structure of a named
	// filter — the checkers the search must NOT find — or reports ErrNotFound.
	SaveExcludePosition(ctx context.Context, scope string, filterName, excludePosition string) error

	// LoadExcludePosition returns the stored exclusion structure of a named
	// filter, or "" when the filter is unknown or carries none.
	LoadExcludePosition(ctx context.Context, scope string, filterName string) (string, error)
}
