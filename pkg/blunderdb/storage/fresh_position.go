package storage

import "context"

type freshPositionKey struct{}

// WithFreshPosition tells the AnalysisStore that positionID was created by
// the caller's own transaction a moment ago: no other session can see it, so
// it has no analysis, no move and no match reaching it yet. A bulk importer
// that just got created=true from PositionStore.SaveCreated passes it to
// Merge, which then skips the reads that could only find nothing (existing
// analysis, ownership, matches whose stats to invalidate, moves to rescore).
// It is a hint about one id: any other position is handled as usual, and a
// store may ignore it.
func WithFreshPosition(ctx context.Context, positionID int64) context.Context {
	return context.WithValue(ctx, freshPositionKey{}, positionID)
}

// IsFreshPosition reports whether ctx carries WithFreshPosition for positionID.
func IsFreshPosition(ctx context.Context, positionID int64) bool {
	id, ok := ctx.Value(freshPositionKey{}).(int64)
	return ok && id == positionID
}
