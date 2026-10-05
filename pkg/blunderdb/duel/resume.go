package duel

import "context"

// Ensure makes the Duel the open one when it is not, so a caller that holds no
// memory of its own — a CLI whose every call is a process, a daemon that
// restarted — can act on a Duel in suspense. The revision the caller saw
// (0: none) is checked under the Service's lock, against the draft as it
// stands before opening, since opening moves it; the returned revision is the
// one the action that follows must name. A Duel already open is left as it
// is: the action that follows checks the revision itself.
func (s *Service) Ensure(ctx context.Context, scope string, id, revision int64) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.open[scope] == id {
		return revision, nil
	}
	st, err := s.openLocked(ctx, scope, id, revision)
	if err != nil || revision == 0 {
		return 0, err
	}
	return st.Revision, nil
}

// OpenAt is Open refusing a draft that is no longer at the revision the
// caller saw (0: none), the check and the opening being one step.
func (s *Service) OpenAt(ctx context.Context, scope string, id, revision int64) (*State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.openLocked(ctx, scope, id, revision)
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
