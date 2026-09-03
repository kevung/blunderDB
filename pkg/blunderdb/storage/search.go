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
	// underlying SQL scan (LIMIT/OFFSET); a zero ListOpts means no limit, from
	// the start — today's behaviour. Because filters that can only be
	// evaluated in Go (mirror search, checker structure on a non-tight mask,
	// date/equity/move-pattern) still run on the page opts.Limit bounded, a
	// caller paging through a search using one of those may see short pages:
	// opts caps how many SQL-matched candidates are considered, not how many
	// of them survive.
	Find(ctx context.Context, scope string, f domain.SearchFilters, opts ListOpts) iter.Seq2[*domain.Position, error]
}

// SearchHistoryStore persists the log of executed searches.
type SearchHistoryStore interface {
	// Save records an executed search: its command, the include position it
	// ran with and its optional "Sauf" exclusion structure ("" for none).
	Save(ctx context.Context, scope string, command, position, excludePosition string) error
	List(ctx context.Context, scope string) iter.Seq2[*SearchHistory, error]
	DeleteEntry(ctx context.Context, scope string, timestamp int64) error
}
