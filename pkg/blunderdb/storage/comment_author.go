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

type commentSignatureKey struct{}

// WithCommentSignature returns a context whose next comment Add stores
// signature verbatim beside the text: an import keeping the statement the
// producer sealed at export (ADR-0099). Nothing else writes one, and an
// edit erases it, since the words it covered are gone.
func WithCommentSignature(ctx context.Context, signature string) context.Context {
	return context.WithValue(ctx, commentSignatureKey{}, signature)
}

// CommentSignatureFromContext returns the signature set by
// WithCommentSignature, or "".
func CommentSignatureFromContext(ctx context.Context) string {
	v, _ := ctx.Value(commentSignatureKey{}).(string)
	return v
}

// CountVerifiedSignedComments counts the comments of scope whose signature
// still verifies. A pass that rehashes positions compares it before and after:
// a signature covers the position's Zobrist hash, so a note on a rehashed
// position stops verifying even though its words are untouched.
func CountVerifiedSignedComments(ctx context.Context, cs CommentStore, scope string) (int, error) {
	n := 0
	for c, err := range cs.ListAll(ctx, scope, ListOpts{}) {
		if err != nil {
			return 0, err
		}
		if c.Signer != nil && c.Signer.Verified {
			n++
		}
	}
	return n, nil
}

// CrawfordRepair is what RepairCrawfordSentinel changed: the positions it
// rehashed, and the signed comments that stopped verifying because the
// position they sit on was rehashed.
type CrawfordRepair struct {
	Rehashed             int `json:"rehashed"`
	SignaturesUnverified int `json:"signaturesUnverified"`
}

// RepairCrawfordSentinel runs PositionStore.RepairCrawfordSentinel over st and
// counts the signed comments it left unverified, so every mode can say so.
func RepairCrawfordSentinel(ctx context.Context, st Storage, scope string) (CrawfordRepair, error) {
	before, err := CountVerifiedSignedComments(ctx, st.Comments(), scope)
	if err != nil {
		return CrawfordRepair{}, err
	}
	var rep CrawfordRepair
	if rep.Rehashed, err = st.Positions().RepairCrawfordSentinel(ctx, scope); err != nil || rep.Rehashed == 0 {
		return rep, err
	}
	after, err := CountVerifiedSignedComments(ctx, st.Comments(), scope)
	if err != nil {
		return rep, err
	}
	rep.SignaturesUnverified = max(before-after, 0)
	return rep, nil
}
