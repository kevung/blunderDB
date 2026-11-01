package sqlshared

import (
	"context"
	"errors"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ObjectUUIDStore implements storage.ObjectUUIDStore over the uuid columns.
type ObjectUUIDStore struct{ DB Execer }

var _ storage.ObjectUUIDStore = (*ObjectUUIDStore)(nil)

func objectUUIDTable(family string) (string, error) {
	for _, f := range ObjectUUIDFamilies {
		if f.Family == family {
			return f.Table, nil
		}
	}
	return "", fmt.Errorf("object family %q: %w", family, storage.ErrInvalid)
}

func (s *ObjectUUIDStore) ObjectUUID(ctx context.Context, scope, family string, id int64) (string, error) {
	table, err := objectUUIDTable(family)
	if err != nil {
		return "", errf(s.DB, "object uuid", err)
	}
	tenant, targs := s.DB.TenantFilter("", scope)
	var u *string
	err = s.DB.QueryRow(ctx, `SELECT CAST(uuid AS TEXT) FROM `+table+` WHERE `+tenant+` AND id = ?`,
		append(targs, id)...).Scan(&u)
	if errors.Is(err, ErrNoRows) {
		return "", errf(s.DB, "object uuid", storage.ErrNotFound)
	}
	if err != nil {
		return "", errf(s.DB, "object uuid", err)
	}
	if u == nil {
		return "", nil
	}
	return *u, nil
}

func (s *ObjectUUIDStore) SetObjectUUID(ctx context.Context, scope, family string, id int64, uuid string) error {
	table, err := objectUUIDTable(family)
	if err != nil {
		return errf(s.DB, "set object uuid", err)
	}
	if len(uuid) != 36 {
		return errf(s.DB, "set object uuid", fmt.Errorf("%q is not a UUID: %w", uuid, storage.ErrInvalid))
	}
	tenant, targs := s.DB.TenantFilter("", scope)
	otenant, oargs := s.DB.TenantFilter("o", scope)
	args := append([]any{uuid}, targs...)
	args = append(args, id)
	args = append(args, oargs...)
	args = append(args, uuid)
	if _, err := s.DB.Exec(ctx, `UPDATE `+table+` SET uuid = ? WHERE `+tenant+` AND id = ?
		AND NOT EXISTS (SELECT 1 FROM `+table+` o WHERE `+otenant+` AND o.uuid = ?)`, args...); err != nil {
		return errf(s.DB, "set object uuid", err)
	}
	return nil
}

func (s *ObjectUUIDStore) ObjectID(ctx context.Context, scope, family, uuid string) (int64, error) {
	table, err := objectUUIDTable(family)
	if err != nil {
		return 0, errf(s.DB, "object id", err)
	}
	tenant, targs := s.DB.TenantFilter("", scope)
	var id int64
	err = s.DB.QueryRow(ctx, `SELECT id FROM `+table+` WHERE `+tenant+` AND CAST(uuid AS TEXT) = ?`, append(targs, uuid)...).Scan(&id)
	if errors.Is(err, ErrNoRows) {
		return 0, errf(s.DB, "object id", storage.ErrNotFound)
	}
	if err != nil {
		return 0, errf(s.DB, "object id", err)
	}
	return id, nil
}
