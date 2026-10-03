package postgres

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// requireOwned returns storage.ErrNotFound unless every id names a row of
// table held by tenant. A write that attaches a row to a parent calls it
// before writing: an INSERT … ON CONFLICT DO NOTHING on a unique index that
// carries no tenant_id answers "no conflict" or "conflict" before any
// foreign key is checked, which tells another tenant whether a row exists.
// One check, one message, whether the id is foreign, absent or a member.
// table is a constant of the caller, never user input.
func requireOwned(ctx context.Context, db execer, tenant int64, table string, ids ...int64) error {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	distinct := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			distinct = append(distinct, id)
		}
	}
	var n int
	if err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM `+table+` WHERE tenant_id = $1 AND id = ANY($2)`,
		tenant, distinct).Scan(&n); err != nil {
		return fmt.Errorf("postgres: check %s ownership: %w", table, err)
	}
	if n != len(distinct) {
		return fmt.Errorf("postgres: %s: %w", table, storage.ErrNotFound)
	}
	return nil
}
