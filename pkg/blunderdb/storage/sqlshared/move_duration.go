package sqlshared

import "database/sql"

// NullableMS reads a move's millisecond column — a decision duration
// (ADR-0073) or a video Repère (ADR-0082): NULL stays nil, the unknown value,
// which must never read as zero.
func NullableMS(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	ms := v.Int64
	return &ms
}

// MSArg is a move's millisecond value as a query argument: nil is written
// NULL.
func MSArg(ms *int64) any {
	if ms == nil {
		return nil
	}
	return *ms
}
