package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/document"
)

// ImportDocumentReleve imports the Relevé of a document (package document,
// ADR-0117), given as its JSON text, into the Collection named collection —
// the document's title when empty. An entry neither sure nor verified refuses
// the import unless force; a refused import writes nothing.
func (d *Database) ImportDocumentReleve(releveJSON string, collection string, force bool) (document.ImportResult, error) {
	r, err := document.Decode([]byte(releveJSON))
	if err != nil {
		return document.ImportResult{}, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return document.ImportResult{}, fmt.Errorf("no database is currently open")
	}
	return document.Import(context.Background(), d.store, "", r, document.ImportOptions{Collection: collection, Force: force})
}
