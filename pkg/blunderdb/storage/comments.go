package storage

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// CommentStore persists the free-text comments attached to positions. A
// position may carry several comment entries.
type CommentStore interface {
	// Add appends a comment the USER wrote to a position and returns its id.
	// It is AddFrom with domain.CommentOriginUser, spelled short because that
	// is what almost every caller means.
	Add(ctx context.Context, scope string, positionID int64, text string) (int64, error)

	// AddFrom appends a comment entry carrying its provenance (issue #263).
	// Importers call it with the origin of the file the note came out of, so a
	// per-move remark that arrived with a match can be told apart from a note
	// the user typed — which is what lets deleting the match spare the second
	// and not the first.
	AddFrom(ctx context.Context, scope string, positionID int64, text string, origin domain.CommentOrigin) (int64, error)

	// Update changes the text of the comment entry with the given id.
	Update(ctx context.Context, scope string, commentID int64, text string) error

	// Delete removes a single comment entry by its id.
	Delete(ctx context.Context, scope string, commentID int64) error

	// DeleteForPosition removes every comment entry of a position.
	DeleteForPosition(ctx context.Context, scope string, positionID int64) error

	// Upsert writes text as the position's primary comment — its oldest
	// non-empty entry — rewriting that entry in place when there is one and
	// appending a new entry otherwise. It returns the id of the entry
	// written. This is the single-comment view the desktop's tag commands
	// and importers edit: they read the primary entry back as the last item
	// of ByPosition, so an Upsert never duplicates the other entries of the
	// wall. It is NOT the inverse of Text, which joins every entry.
	Upsert(ctx context.Context, scope string, positionID int64, text string) (int64, error)

	// Text returns the concatenated comment text of a position (empty if none).
	Text(ctx context.Context, scope string, positionID int64) (string, error)

	// ByPositions returns the non-empty comments of the given positions, keyed
	// by position id and oldest first within a position — the order they were
	// written in, which is the order a copy of them should carry.
	ByPositions(ctx context.Context, scope string, positionIDs []int64) (map[int64][]*domain.CommentEntry, error)

	// ByPosition streams the comment entries of a position.
	ByPosition(ctx context.Context, scope string, positionID int64) iter.Seq2[*domain.CommentEntry, error]

	// ListAll streams every comment entry in the database.
	ListAll(ctx context.Context, scope string, opts ListOpts) iter.Seq2[*domain.CommentEntry, error]

	// Search streams comment entries whose text matches query.
	Search(ctx context.Context, scope string, query string) iter.Seq2[*domain.CommentEntry, error]

	// Tags returns the tag vocabulary of the tenant: every `#word` written in
	// a comment, with the number of POSITIONS carrying it, most used first and
	// alphabetically within a count (issue #265).
	//
	// It lives here, on the comments, because that is where a tag lives: a tag
	// is not a table and nothing declares one. The counts are over positions
	// and not comments — a tag written twice on the same position is one
	// position tagged, and the number shown beside a tag has to be the number
	// of positions clicking it will yield.
	Tags(ctx context.Context, scope string) ([]domain.TagCount, error)
}
