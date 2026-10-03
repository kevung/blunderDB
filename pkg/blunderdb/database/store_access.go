package database

import "github.com/kevung/blunderdb/pkg/blunderdb/storage"

// CurrentStore returns the Storage over the file d has open and the generation
// it belongs to, or a nil Storage when no file is open. A host that serves the
// storage contract beside the GUI (the localhost MCP server) rebuilds its
// handler whenever the generation moves: its Storage borrows d's connection,
// which the next open closes.
// A function, not a method, so the Wails binding never exposes it.
func CurrentStore(d *Database) (storage.Storage, uint64) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil || d.store == nil {
		return nil, d.generation
	}
	return d.store, d.generation
}
