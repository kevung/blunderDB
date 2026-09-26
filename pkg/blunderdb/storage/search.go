package storage

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// SearchHistory is one entry in the search history log.
type SearchHistory struct {
	ID       int    `json:"id"`
	Command  string `json:"command"`
	Position string `json:"position"`
	// ExcludePosition is the optional "Sauf" structure the search ran with:
	// the checkers a match must NOT have. "" when the search had none.
	ExcludePosition string `json:"excludePosition"`
	Timestamp       int64  `json:"timestamp"`
}

// SearchStore runs position searches.
type SearchStore interface {
	// Find streams the positions matching the given filters. opts bounds the
	// underlying SQL scan (LIMIT/OFFSET); a zero ListOpts means no limit. The
	// Go-only filters (mirror, loose structure mask, date/equity/move pattern)
	// run after, so a page may come back short: opts caps the SQL-matched
	// candidates, not the survivors.
	Find(ctx context.Context, scope string, f domain.SearchFilters, opts ListOpts) iter.Seq2[*domain.Position, error]

	// Rank answers a query carrying the `like` token: the same filters as
	// Find, but ORDERED by how far each survivor stands from the query's
	// target, nearest first, with that distance attached (ADR-0043).
	//
	// A separate entry point because the distance is part of the answer, and
	// domain.Position has no business carrying it.
	//
	// The set it ranks is the filters INTERSECTED with the target's
	// equivalence class (ClassOf). opts bounds the RANKING, not the scan:
	// every candidate must be seen before any is called nearest (ADR-0043).
	Rank(ctx context.Context, scope string, f domain.SearchFilters, opts ListOpts) ([]SimilarPosition, error)
}

// SearchHistoryStore persists the log of executed searches.
type SearchHistoryStore interface {
	// Save records an executed search: its command, the include position it
	// ran with and its optional "Sauf" exclusion structure ("" for none).
	Save(ctx context.Context, scope string, command, position, excludePosition string) error
	List(ctx context.Context, scope string) iter.Seq2[*SearchHistory, error]
	DeleteEntry(ctx context.Context, scope string, timestamp int64) error
}
