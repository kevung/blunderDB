package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// action_label is tenant-scoped here: a registered label is text a tenant's
// import brought. Its code comes from a sequence, unique across tenants, so
// the shared read expression (sqlshared.ActionLabelSQL) needs no tenant filter;
// registration and lookup by label are filtered by tenant.
const (
	registerActionLabelSQL = `INSERT INTO action_label (tenant_id, label) VALUES ($1, $2)
ON CONFLICT (tenant_id, label) DO NOTHING`
	lookupActionLabelSQL = `SELECT code FROM action_label WHERE tenant_id = $1 AND label = $2`
)

// actionLabels registers and looks up a tenant's action labels
// (sqlshared.ActionCodeFor) on the caller's connection or transaction.
type actionLabels struct {
	ctx    context.Context
	db     execer
	tenant int64
}

func (r actionLabels) RegisterActionLabel(label string) error {
	_, err := r.db.Exec(r.ctx, registerActionLabelSQL, r.tenant, label)
	return err
}

func (r actionLabels) LookupActionLabel(label string) (int64, bool, error) {
	var code int64
	err := r.db.QueryRow(r.ctx, lookupActionLabelSQL, r.tenant, label).Scan(&code)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	return code, err == nil, err
}

// actionCode returns the action code that stores label for tenant.
func actionCode(ctx context.Context, db execer, tenant int64, label string) (int64, error) {
	return sqlshared.ActionCodeFor(actionLabels{ctx, db, tenant}, label)
}
