package sqlshared

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// LoadTargetPosition reads the position a ranking is taken against. Only its
// checkers matter to the distance, so only the board is read.
func LoadTargetPosition(ctx context.Context, db Execer, scope string, id int64) (*domain.Position, error) {
	tenant, args := db.TenantFilter("p", scope)
	args = append(args, id)
	rows, err := db.Query(ctx,
		`SELECT p.id, p.state FROM position p WHERE `+tenant+` AND p.id = ?`, args...)
	if err != nil {
		return nil, errf(db, "load the position to rank against", err)
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, errf(db, "load the position to rank against", err)
		}
		return nil, storage.ErrNotFound
	}
	var (
		pid   int64
		state []byte
	)
	if err := rows.Scan(&pid, &state); err != nil {
		return nil, errf(db, "load the position to rank against", err)
	}
	p, ok := positionOfState(string(state))
	if !ok {
		return nil, storage.ErrNotFound
	}
	p.ID = pid
	return &p, nil
}
