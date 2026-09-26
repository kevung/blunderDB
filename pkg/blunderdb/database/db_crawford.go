package database

import (
	"context"
	"fmt"
)

// RepairCrawfordSentinel rewrites the away score of every position that carries
// the ambiguous `1` although the game it was imported from is a post-Crawford
// one, and returns how many rows changed (ADR-0045 §7).
//
// The away score carries the Crawford rule INSIDE the number (CONTEXT.md,
// « Away score »): `1` is the Crawford game, `0` is post-Crawford. Rows
// written as matchLength − score read `1` for both, making the cube look dead.
//
// Never automatic: it rehashes positions, and opening a file must not rewrite
// stored identities.
func (d *Database) RepairCrawfordSentinel() (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return 0, fmt.Errorf("no database is currently open")
	}
	return d.store.Positions().RepairCrawfordSentinel(context.Background(), "")
}
