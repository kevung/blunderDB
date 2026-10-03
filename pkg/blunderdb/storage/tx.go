package storage

import "context"

// Tx is a storage transaction. It exposes the same per-family accessors as
// Storage; every operation performed through them is part of the transaction
// and becomes durable only on Commit.
//
//	tx, err := s.BeginTx(ctx)
//	if err != nil { ... }
//	defer tx.Rollback() // no-op once committed
//	if _, err := tx.Positions().Save(ctx, scope, p); err != nil { ... }
//	return tx.Commit()
type Tx interface {
	Stores

	// Commit makes the transaction's changes durable.
	Commit() error

	// Rollback discards the transaction. It is safe to call after Commit.
	Rollback() error
}

// GuardedBeginner is implemented by a backend that can open a transaction holding, from its
// first statement to its end, a lock other processes on the same database also respect: the
// guard a read-check-write sequence needs when the check (a version a client stated) must
// still hold at the write. keys name what is guarded; two transactions guarding a common key
// never overlap. A backend may guard more than the keys — SQLite takes its one write lock.
type GuardedBeginner interface {
	BeginGuardedTx(ctx context.Context, keys ...string) (Tx, error)
}

// BeginGuarded opens a guarded transaction on keys when st can, and a plain
// one otherwise.
func BeginGuarded(ctx context.Context, st Storage, keys ...string) (Tx, error) {
	if gb, ok := st.(GuardedBeginner); ok {
		return gb.BeginGuardedTx(ctx, keys...)
	}
	return st.BeginTx(ctx)
}
