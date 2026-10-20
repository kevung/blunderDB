package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
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
//
// The report also counts the signed comments that no longer verify: a signature
// covers the position's hash (ADR-0099), so rehashing a position leaves the
// notes on it readable but unverified.
func (d *Database) RepairCrawfordSentinel() (storage.CrawfordRepair, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return storage.CrawfordRepair{}, fmt.Errorf("no database is currently open")
	}
	return storage.RepairCrawfordSentinel(context.Background(), d.store, "")
}
