package storage

import (
	"context"
	"strings"
)

// The author of a comment travels in the context rather than in every
// CommentStore signature: each caller knows it once (the desktop's setting,
// the CLI's --author, the daemon's X-User-Name, an importer's source), and
// every write below it — add, upsert, edit, restore — stamps the same name.

type commentAuthorKey struct{}

// MaxCommentAuthorLen bounds an author name: it is free text from a header or
// a setting, shown beside each comment.
const MaxCommentAuthorLen = 120

// WithCommentAuthor returns a context whose comment writes are signed by
// author. Surrounding blanks are trimmed and the name is cut to
// MaxCommentAuthorLen runes; an empty name leaves the comments unsigned.
func WithCommentAuthor(ctx context.Context, author string) context.Context {
	return context.WithValue(ctx, commentAuthorKey{}, NormalizeCommentAuthor(author))
}

// CommentAuthorFromContext returns the author set by WithCommentAuthor, or "".
func CommentAuthorFromContext(ctx context.Context) string {
	v, _ := ctx.Value(commentAuthorKey{}).(string)
	return v
}

// NormalizeCommentAuthor trims author and cuts it to MaxCommentAuthorLen runes.
func NormalizeCommentAuthor(author string) string {
	author = strings.TrimSpace(author)
	if r := []rune(author); len(r) > MaxCommentAuthorLen {
		author = strings.TrimSpace(string(r[:MaxCommentAuthorLen]))
	}
	return author
}
