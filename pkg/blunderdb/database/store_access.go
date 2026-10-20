package database

import "github.com/kevung/blunderdb/pkg/blunderdb/storage"

// CurrentStore returns the Storage over the file d has open and the epoch of
// that Storage, or a nil Storage when no file is open. A host that serves the
// storage contract beside the GUI (the localhost MCP server, the tournament
// display) rebuilds its handler whenever the epoch moves: its Storage borrows
// d's connection, which the next open — or a vacuum that swaps the file —
// closes.
// A function, not a method, so the Wails binding never exposes it.
func CurrentStore(d *Database) (storage.Storage, uint64) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil || d.store == nil {
		return nil, d.storeEpoch
	}
	return d.store, d.storeEpoch
}
