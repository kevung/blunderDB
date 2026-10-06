package duel

import "context"

// Get reads a Duel as a State, without opening it or touching its
// clocks.
func (s *Service) Get(ctx context.Context, scope string, id int64) (*State, error) {
	row, g, err := s.load(ctx, scope, id)
	if err != nil {
		return nil, err
	}
	return state(row, g), nil
}
