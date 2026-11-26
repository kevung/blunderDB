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

// AnalyzeDocument reads the diagrams of the document at path (package
// document, ADR-0117) — a PDF, a page image, or a folder of page images —
// and returns their Relevé as JSON text, the form ImportDocumentReleve takes
// back. title and author replace the document's own metadata when not empty. It reads and writes no database: a document is
// analysed on the person's machine, then its Relevé is verified and imported.
func (d *Database) AnalyzeDocument(path, title, author string) (string, error) {
	r, err := document.AnalyzeFile(path, document.AnalyzeOptions{Title: title, Author: author})
	if err != nil {
		return "", err
	}
	out, err := r.Encode()
	return string(out), err
}
