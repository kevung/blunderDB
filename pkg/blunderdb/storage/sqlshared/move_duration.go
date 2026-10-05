package sqlshared

import "database/sql"

// NullableMS reads a move's decision duration: NULL stays nil, the unknown
// duration, which must never read as zero (ADR-0073).
func NullableMS(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	ms := v.Int64
	return &ms
}

// MSArg is a decision duration as a query argument: nil is written NULL.
func MSArg(ms *int64) any {
	if ms == nil {
		return nil
	}
	return *ms
}
