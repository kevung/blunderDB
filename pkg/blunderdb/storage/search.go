package storage

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// SearchHistory is one entry in the search history log.
type SearchHistory struct {
	ID        int    `json:"id"`
	Command   string `json:"command"`
	Position  string `json:"position"`
	Timestamp int64  `json:"timestamp"`
}

// SearchStore runs position searches.
type SearchStore interface {
	// Find streams the positions matching the given filters.
	Find(ctx context.Context, scope string, f domain.SearchFilters) iter.Seq2[*domain.Position, error]
}

// SearchHistoryStore persists the log of executed searches.
type SearchHistoryStore interface {
	Save(ctx context.Context, scope string, command, position string) error
	List(ctx context.Context, scope string) iter.Seq2[*SearchHistory, error]
	DeleteEntry(ctx context.Context, scope string, timestamp int64) error
}
