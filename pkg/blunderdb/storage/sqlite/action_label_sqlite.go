package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// actionLabels registers and looks up action labels (sqlshared.ActionCodeFor)
// on the caller's connection or transaction.
type actionLabels struct {
	ctx context.Context
	db  execer
}

func (r actionLabels) RegisterActionLabel(label string) error {
	_, err := r.db.ExecContext(r.ctx, sqlshared.RegisterActionLabelSQL, label, label)
	return err
}

func (r actionLabels) LookupActionLabel(label string) (int64, bool, error) {
	var code int64
	err := r.db.QueryRowContext(r.ctx, sqlshared.LookupActionLabelSQL, label).Scan(&code)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	return code, err == nil, err
}

// actionCode returns the action code that stores label.
func actionCode(ctx context.Context, db execer, label string) (int64, error) {
	return sqlshared.ActionCodeFor(actionLabels{ctx, db}, label)
}
