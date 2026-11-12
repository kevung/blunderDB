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
	// RESULTS (LIMIT/OFFSET over the survivors); a zero ListOpts means no
	// limit. Without a Go-only filter (mirror, loose structure mask, comment
	// text, zone, date/equity/move pattern) the window goes to SQL; with one,
	// candidates are scanned a chunk at a time until the window is full, so
	// memory stays bounded by the window, not by the library. Each position
	// yielded is the caller's: the store never reuses or alters it.
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

	// FindIDs is Find answering ids only, in the same order and window: a
	// projection on the id when no Go-only filter is left, so a list of
	// results is browsed without reconstructing any position.
	FindIDs(ctx context.Context, scope string, f domain.SearchFilters, opts ListOpts) ([]int64, error)

	// FindPage is Find read by cursor: up to opts.Limit positions after
	// opts.Cursor (the start when empty), and the cursor of the page that
	// follows ("" after the last). A page costs its own rows whatever its
	// depth, where an offset costs every row before it. Rows saved between
	// two pages neither repeat nor hide a row already ordered before them.
	FindPage(ctx context.Context, scope string, f domain.SearchFilters, opts PageOpts) (positions []domain.Position, nextCursor string, err error)

	// FindIDsPage is FindPage answering ids only, with the same cursors.
	FindIDsPage(ctx context.Context, scope string, f domain.SearchFilters, opts PageOpts) (ids []int64, nextCursor string, err error)

	// Count is the length of Find's unbounded result.
	Count(ctx context.Context, scope string, f domain.SearchFilters) (int, error)

	// CountUpTo is Count bounded: it stops at max and reports capped when the
	// result is longer, so that a result of any size costs no more than max rows.
	// A count the context cuts short reports what it reached as capped ("at least n").
	CountUpTo(ctx context.Context, scope string, f domain.SearchFilters, max int) (n int, capped bool, err error)

	// IndexOf is the rank of id in Find's unbounded result; found is false
	// when the search does not find it.
	IndexOf(ctx context.Context, scope string, f domain.SearchFilters, id int64) (index int, found bool, err error)
}

// PageOpts bounds a page read by cursor. Limit is required. Cursor is opaque:
// it is only good for the search (scope, filters, sort) that issued it, and
// any other is ErrInvalid. Offset, for a client still counting rows, cuts the
// first page only: an offset with a cursor is ErrInvalid.
type PageOpts struct {
	Limit  int
	Offset int
	Cursor string
}

// SearchHistoryStore persists the log of executed searches.
type SearchHistoryStore interface {
	// Save records an executed search: its command, the include position it
	// ran with and its optional "Sauf" exclusion structure ("" for none).
	Save(ctx context.Context, scope string, command, position, excludePosition string) error
	List(ctx context.Context, scope string) iter.Seq2[*SearchHistory, error]
	DeleteEntry(ctx context.Context, scope string, timestamp int64) error
}
