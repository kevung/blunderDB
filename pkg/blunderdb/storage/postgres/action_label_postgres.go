package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

var (
	registerActionLabelSQL = sqlshared.Rebind(sqlshared.RegisterActionLabelSQL)
	lookupActionLabelSQL   = sqlshared.Rebind(sqlshared.LookupActionLabelSQL)
)

// actionLabels registers and looks up action labels (sqlshared.ActionCodeFor)
// on the caller's connection or transaction. action_label is shared by the
// tenants: a label is a word of an analysis program, not tenant data.
type actionLabels struct {
	ctx context.Context
	db  execer
}

func (r actionLabels) RegisterActionLabel(label string) error {
	_, err := r.db.Exec(r.ctx, registerActionLabelSQL, label, label)
	return err
}

func (r actionLabels) LookupActionLabel(label string) (int64, bool, error) {
	var code int64
	err := r.db.QueryRow(r.ctx, lookupActionLabelSQL, label).Scan(&code)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	return code, err == nil, err
}

// actionCode returns the action code that stores label.
func actionCode(ctx context.Context, db execer, label string) (int64, error) {
	return sqlshared.ActionCodeFor(actionLabels{ctx, db}, label)
}
