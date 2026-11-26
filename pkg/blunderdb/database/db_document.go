package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/document"
)

// ImportDocumentReleve imports the Relevé of a document (package document,
// ADR-0117), given as its JSON text, into the Collection named collection —
// the document's title when empty. An entry neither sure nor verified refuses
// the import unless force, or acceptComplete for an entry of fifteen checkers
// a side read with no doubt whose context is certified, or fallback for an
// entry whose checkers are certified and whose context is to verify,
// imported under the context its position carries; a refused import writes
// nothing.
func (d *Database) ImportDocumentReleve(releveJSON string, collection string, acceptComplete, fallback, force bool) (document.ImportResult, error) {
	r, err := document.Decode([]byte(releveJSON))
	if err != nil {
		return document.ImportResult{}, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return document.ImportResult{}, fmt.Errorf("no database is currently open")
	}
	return document.Import(context.Background(), d.store, "", r, document.ImportOptions{Collection: collection, AcceptComplete: acceptComplete, Fallback: fallback, Force: force})
}

// AnalyzeDocument reads the diagrams of the document at path (package
// document, ADR-0117) — a PDF, a page image, or a folder of page images —
// and returns their Relevé as JSON text, the form ImportDocumentReleve takes
// back. title and author replace the document's own metadata when not empty. It reads and writes no database: a document is
// analysed on the person's machine, then its Relevé is verified and imported.
func (d *Database) AnalyzeDocument(path, title, author string) (string, error) {
	// A scanned book takes minutes to decode: CancelImport stops it.
	ctx, done := d.beginCancellableImport()
	defer done()
	r, err := document.AnalyzeFile(path, document.AnalyzeOptions{Title: title, Author: author, Context: ctx})
	if err != nil {
		return "", err
	}
	out, err := r.Encode()
	return string(out), err
}

// SummarizeDocumentReleve counts the entries of a Relevé by certainty and
// names the ranks an import refuses or accepts under a fallback, as
// `document check` does. It reads and writes no database.
func (d *Database) SummarizeDocumentReleve(releveJSON string) (document.Summary, error) {
	r, err := document.Decode([]byte(releveJSON))
	if err != nil {
		return document.Summary{}, err
	}
	return document.Summarize(r), nil
}

// ReviewDocumentReleve lists the entries of a Relevé an import refuses
// without a decision, each with the fields to check against its diagram,
// and the ranks the import with these options (acceptComplete, fallback)
// still refuses: the one list Import itself applies.
func (d *Database) ReviewDocumentReleve(releveJSON string, acceptComplete, fallback bool) (document.ReviewResult, error) {
	r, err := document.Decode([]byte(releveJSON))
	if err != nil {
		return document.ReviewResult{}, err
	}
	return document.Review(r, document.ImportOptions{AcceptComplete: acceptComplete, Fallback: fallback}), nil
}

// ConfirmDocumentReleve returns the Relevé with the document's defaults and
// its direction of play marked confirmed, once: the gesture of setting
// "confirmed": true by hand in the JSON file.
func (d *Database) ConfirmDocumentReleve(releveJSON string) (string, error) {
	r, err := document.Decode([]byte(releveJSON))
	if err != nil {
		return "", err
	}
	if err := document.ConfirmDefaults(r); err != nil {
		return "", err
	}
	out, err := r.Encode()
	return string(out), err
}
