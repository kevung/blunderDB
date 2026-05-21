package storage

import (
	"context"
	"iter"
)

// Filter is a named, saved search command (the filter library).
type Filter struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Command string `json:"command"`
}

// FilterStore persists the saved-filter library and the per-filter "edit
// position" scratch state.
type FilterStore interface {
	Save(ctx context.Context, scope string, name, command string) (int64, error)
	Update(ctx context.Context, scope string, id int64, name, command string) error
	Delete(ctx context.Context, scope string, id int64) error
	List(ctx context.Context, scope string) iter.Seq2[*Filter, error]

	// SaveEditPosition stores the in-progress edit position for a named filter.
	SaveEditPosition(ctx context.Context, scope string, filterName, editPosition string) error

	// LoadEditPosition returns the stored edit position for a named filter.
	LoadEditPosition(ctx context.Context, scope string, filterName string) (string, error)
}
