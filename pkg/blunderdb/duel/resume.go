package duel

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Ensure makes the Duel the open one when it is not, so a caller that holds no
// memory of its own — a CLI whose every call is a process, a daemon that
// restarted — can act on a Duel in suspense. The revision the caller saw
// (0: none) is checked against the draft as it stands before opening, since
// opening a Duel stamps its clocks and moves the revision; the returned
// revision is the one the action that follows must name.
func (s *Service) Ensure(ctx context.Context, scope string, id, revision int64) (int64, error) {
	s.mu.Lock()
	open := s.open[scope] == id
	s.mu.Unlock()
	if open {
		return revision, nil
	}
	row, err := s.store.Duels().Get(ctx, scope, id)
	if err != nil {
		return 0, err
	}
	if revision != 0 && revision != row.Revision {
		return 0, fmt.Errorf("duel %d at revision %d, not %d: %w", id, row.Revision, revision, storage.ErrConflict)
	}
	st, err := s.Open(ctx, scope, id)
	if err != nil {
		return 0, err
	}
	if revision == 0 {
		return 0, nil
	}
	return st.Revision, nil
}

// Get reads a Duel in suspense as a State, without opening it or touching its
// clocks.
func (s *Service) Get(ctx context.Context, scope string, id int64) (*State, error) {
	row, g, err := s.load(ctx, scope, id)
	if err != nil {
		return nil, err
	}
	return state(row, g), nil
}
