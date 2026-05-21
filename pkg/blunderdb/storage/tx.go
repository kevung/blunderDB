package storage

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
