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

type freshPositionsKey struct{}

// WithFreshPositions is WithFreshPosition for every one of ids at once: the
// hint a bulk copy passes to AnalysisStore.SaveMany for the positions
// PositionStore.SaveMany just reported created.
func WithFreshPositions(ctx context.Context, ids []int64) context.Context {
	set := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return context.WithValue(ctx, freshPositionsKey{}, set)
}

// IsFreshPosition reports whether ctx carries WithFreshPosition or
// WithFreshPositions for positionID.
func IsFreshPosition(ctx context.Context, positionID int64) bool {
	if id, ok := ctx.Value(freshPositionKey{}).(int64); ok && id == positionID {
		return true
	}
	set, ok := ctx.Value(freshPositionsKey{}).(map[int64]struct{})
	if !ok {
		return false
	}
	_, fresh := set[positionID]
	return fresh
}
