package sqlshared

import (
	"context"
	"errors"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// PublicIDStore implements storage.PublicIDStore over position.public_id.
type PublicIDStore struct{ DB Execer }

var _ storage.PublicIDStore = (*PublicIDStore)(nil)

func (s *PublicIDStore) PublicID(ctx context.Context, scope string, id int64) (string, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	var v *string
	err := s.DB.QueryRow(ctx, `SELECT public_id FROM position WHERE `+tenant+` AND id = ?`, append(targs, id)...).Scan(&v)
	if errors.Is(err, ErrNoRows) {
		return "", errf(s.DB, "public id", storage.ErrNotFound)
	}
	if err != nil {
		return "", errf(s.DB, "public id", err)
	}
	if v == nil {
		return "", nil
	}
	return *v, nil
}

func (s *PublicIDStore) SetPublicID(ctx context.Context, scope string, id int64, publicID string) error {
	have, err := s.PublicID(ctx, scope, id)
	if err != nil {
		return err
	}
	// A write that changes nothing is no edit: the journal keeps its stamp.
	if have == publicID {
		return nil
	}
	var v any
	if publicID != "" {
		v = publicID
	}
	tenant, targs := s.DB.TenantFilter("", scope)
	if _, err := s.DB.Exec(ctx, `UPDATE position SET public_id = ? WHERE `+tenant+` AND id = ?`,
		append(append([]any{v}, targs...), id)...); err != nil {
		return errf(s.DB, "set public id", err)
	}
	return nil
}
