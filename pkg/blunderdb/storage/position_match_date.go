package storage

import (
	"context"
	"time"
)

type positionMatchDateKey struct{}

// WithPositionMatchDate returns a context under which SaveMany dates the
// positions it creates with t (the date of the match being imported), so the
// row is born with the date the moves would otherwise lower it to. A zero t
// leaves them undated. A store that dates positions another way ignores it.
func WithPositionMatchDate(ctx context.Context, t time.Time) context.Context {
	return context.WithValue(ctx, positionMatchDateKey{}, t)
}

// PositionMatchDateFromContext returns the date set by WithPositionMatchDate.
func PositionMatchDateFromContext(ctx context.Context) time.Time {
	t, _ := ctx.Value(positionMatchDateKey{}).(time.Time)
	return t
}
