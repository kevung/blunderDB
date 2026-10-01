package postgres

import (
	"context"
	"fmt"
	"hash/fnv"
	"slices"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

var _ storage.GuardedBeginner = (*Storage)(nil)

// BeginGuardedTx opens a transaction that first takes a transaction-scoped advisory lock per
// key (pg_advisory_xact_lock): every instance over the database waits on the same locks, and
// they go with the transaction, at commit or rollback. Keys are taken in sorted order, so two
// transactions guarding overlapping sets cannot wait on each other.
func (s *Storage) BeginGuardedTx(ctx context.Context, keys ...string) (storage.Tx, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: guarded tx: %w", err)
	}
	sorted := slices.Clone(keys)
	slices.Sort(sorted)
	for _, k := range slices.Compact(sorted) {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, guardKey(k)); err != nil {
			_ = tx.Rollback(context.WithoutCancel(ctx))
			return nil, fmt.Errorf("postgres: guarded tx: lock %q: %w", k, err)
		}
	}
	return &txImpl{binder: binder{db: tx}, tx: tx, ctx: context.WithoutCancel(ctx)}, nil
}

// guardKey maps a key to the advisory lock's int64 space. A collision only makes two guards
// wait on each other, never lets two overlap.
func guardKey(k string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte("blunderdb guard|" + k))
	return int64(h.Sum64()) //nolint:gosec // a lock name: wrapping is harmless.
}
