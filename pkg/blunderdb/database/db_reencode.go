package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ReencodeResult reports a ReencodeAnalyses run. A struct because Wails v2
// binds at most (value, error).
type ReencodeResult struct {
	// Rewritten is how many legacy analysis blobs the run rewrote in the
	// binary format (ADR-0070).
	Rewritten int
}

// ReencodeAnalyses rewrites every analysis blob still in a legacy JSON format
// in the binary format, by batches of storage.ReencodeBatchSize rows, each its
// own transaction. Explicit only, never run at open: a legacy blob reads as
// well as a binary one, the pass only saves space and decoding time.
// Interrupted, it loses at most the batch in flight, and a new run skips in
// SQL what is already binary.
//
// d.mu is taken per batch, not for the whole run, so the GUI keeps reading a
// large library while it converts.
func (d *Database) ReencodeAnalyses() (ReencodeResult, error) {
	d.mu.RLock()
	open, store := d.db != nil, d.store
	d.mu.RUnlock()
	if !open {
		return ReencodeResult{}, fmt.Errorf("reencode analyses: no database open")
	}
	n, err := storage.ReencodeAllAnalyses(context.Background(), store.Analyses(), "", func() func() {
		d.mu.Lock()
		return d.mu.Unlock
	})
	return ReencodeResult{Rewritten: n}, err
}
