package database

import (
	"context"
	"fmt"
)

// RepairCrawfordSentinel rewrites the away score of every position that carries
// the ambiguous `1` although the game it was imported from is a post-Crawford
// one, and returns how many rows changed (issue #338, ADR-0045 §7).
//
// The away score carries the Crawford rule INSIDE the number (CONTEXT.md,
// « Away score »): `1` is "one point to go, and this IS the Crawford game", `0`
// is "one point to go, Crawford behind us". The three importers computed only
// matchLength − score and wrote `1` for both, so every post-Crawford position
// already imported reads as a Crawford one — cube dead — where the trailer in
// fact doubles at the first opportunity. Fixing the importers does nothing for
// the rows already written, which is what this repairs.
//
// It is `blunderdb repair`'s third pass, next to the analysis columns and the
// derived phase, and never automatic: it rehashes positions, and a tool does
// not rewrite the identity of stored rows on the mere act of opening a file.
func (d *Database) RepairCrawfordSentinel() (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return 0, fmt.Errorf("no database is currently open")
	}
	return d.store.Positions().RepairCrawfordSentinel(context.Background(), "")
}
