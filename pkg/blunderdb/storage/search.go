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

	// Rank answers a query carrying the `like` token: the same filters as
	// Find, but ORDERED by how far each survivor stands from the query's
	// target, nearest first, with that distance attached (ADR-0043).
	//
	// It is a second entry point rather than a mode of Find for one reason:
	// the distance is part of the answer. A neighbour without its distance is
	// unreadable — it is the only thing that says whether one is looking at a
	// neighbour or at a coincidence — and domain.Position has no business
	// carrying a number that belongs to a comparison rather than to itself.
	//
	// The set it ranks is the filters INTERSECTED with the target's
	// equivalence class (ClassOf), so "the neighbours of 42 that I blundered"
	// is one query. opts bounds the RANKING, not the scan: every candidate has
	// to be seen before any can be called nearest, which is the exhaustive
	// scan P7 recommends and ADR-0043 keeps.
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
