package storage

import "context"

// SessionState is the persisted UI session: the last search and the view tabs.
type SessionState struct {
	LastSearchCommand  string  `json:"lastSearchCommand"`
	LastSearchPosition string  `json:"lastSearchPosition"`
	LastPositionIndex  int     `json:"lastPositionIndex"`
	LastPositionIDs    []int64 `json:"lastPositionIds"`
	HasActiveSearch    bool    `json:"hasActiveSearch"`
	ViewsJSON          string  `json:"viewsJSON"`
}

// SessionStore persists the UI session state. The scope argument identifies
// the session once session scoping lands in P4; it is empty until then.
type SessionStore interface {
	Save(ctx context.Context, scope string, state SessionState) error
	Load(ctx context.Context, scope string) (*SessionState, error)
	Clear(ctx context.Context, scope string) error
}
