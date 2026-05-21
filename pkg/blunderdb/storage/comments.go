package storage

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// CommentStore persists the free-text comments attached to positions. A
// position may carry several comment entries.
type CommentStore interface {
	// Add appends a new comment entry to a position and returns its id.
	Add(ctx context.Context, scope string, positionID int64, text string) (int64, error)

	// Update changes the text of the comment entry with the given id.
	Update(ctx context.Context, scope string, commentID int64, text string) error

	// Delete removes a single comment entry by its id.
	Delete(ctx context.Context, scope string, commentID int64) error

	// DeleteForPosition removes every comment entry of a position.
	DeleteForPosition(ctx context.Context, scope string, positionID int64) error

	// Text returns the concatenated comment text of a position (empty if none).
	Text(ctx context.Context, scope string, positionID int64) (string, error)

	// ByPosition streams the comment entries of a position.
	ByPosition(ctx context.Context, scope string, positionID int64) iter.Seq2[*domain.CommentEntry, error]

	// ListAll streams every comment entry in the database.
	ListAll(ctx context.Context, scope string) iter.Seq2[*domain.CommentEntry, error]

	// Search streams comment entries whose text matches query.
	Search(ctx context.Context, scope string, query string) iter.Seq2[*domain.CommentEntry, error]
}
