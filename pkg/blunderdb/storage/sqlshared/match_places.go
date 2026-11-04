package sqlshared

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// MatchPlaceStore implements storage.MatchPlaceStore over match.
type MatchPlaceStore struct{ DB Execer }

var _ storage.MatchPlaceStore = (*MatchPlaceStore)(nil)

func (s *MatchPlaceStore) SetMatchOrder(ctx context.Context, scope string, matchID int64, order int) error {
	tenant, targs := s.DB.TenantFilter("", scope)
	if _, err := s.DB.Exec(ctx, `UPDATE match SET tournament_sort_order = ? WHERE id = ? AND `+tenant,
		append([]any{order, matchID}, targs...)...); err != nil {
		return fmt.Errorf("%s: order of match %d in its tournament: %w", s.DB.Name(), matchID, err)
	}
	return nil
}
