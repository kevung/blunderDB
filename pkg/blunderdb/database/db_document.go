package database

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

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
	// CancelImport stops the import, which then writes nothing.
	ctx, done := d.beginCancellableImport()
	defer done()
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return document.ImportResult{}, fmt.Errorf("no database is currently open")
	}
	return document.Import(ctx, d.store, "", r, document.ImportOptions{Collection: collection, AcceptComplete: acceptComplete, Fallback: fallback, Force: force})
}

// AnalyzeDocument reads the diagrams of the document at path (package
// document, ADR-0117) — a PDF, a page image, or a folder of page images —
// and returns their Relevé as JSON text, the form ImportDocumentReleve takes
// back. title and author replace the document's own metadata when not empty. It reads and writes no database: a document is
// analysed on the person's machine, then its Relevé is verified and imported.
func (d *Database) AnalyzeDocument(path, title, author string) (string, error) {
	return tracked(d, TaskImport, path, d.CancelImport, func() (string, error) {
		// A scanned book takes minutes to decode: CancelImport stops it.
		ctx, done := d.beginCancellableImport()
		defer done()
		opts := document.AnalyzeOptions{Title: title, Author: author, Context: ctx}
		if hook := d.documentProgress.Load(); hook != nil {
			opts.Progress = document.ThrottledProgress(*hook, 250*time.Millisecond)
		}
		r, err := document.AnalyzeFile(path, opts)
		if err != nil {
			return "", err
		}
		out, err := r.Encode()
		return string(out), err
	})
}

// SetDocumentProgress registers where AnalyzeDocument reports how far it has
// read (nil removes it): at most four times a second, the last page always.
// A function, not a method, as SetTaskTracker is.
func SetDocumentProgress(d *Database, f func(document.Progress)) {
	if f == nil {
		d.documentProgress.Store(nil)
		return
	}
	d.documentProgress.Store(&f)
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

// DocumentSheet lists every entry of a Relevé for the verification sheet:
// page, rectangle of its diagram, position read, state, whether a person
// verified it and the fields to compare. It reads and writes no database.
func (d *Database) DocumentSheet(releveJSON string) ([]document.SheetEntry, error) {
	r, err := document.Decode([]byte(releveJSON))
	if err != nil {
		return nil, err
	}
	return document.Sheet(r), nil
}

// LoadDocumentReleve returns the Relevé kept for what the person opened at
// path — beside a document, or the Relevé file itself — as JSON text, ""
// when there is none or it was read from another version of the document:
// a book is verified over several sittings. It reads no database.
func (d *Database) LoadDocumentReleve(path string) (string, error) {
	if document.IsReleveFile(path) {
		return "", nil
	}
	r, err := document.LoadSidecar(path)
	if err != nil || r == nil {
		return "", err
	}
	out, err := r.Encode()
	return string(out), err
}

// SaveDocumentReleve writes the Relevé where LoadDocumentReleve finds it
// again and returns that file's path. Nothing goes into the database: the
// progress of a verification lives with the document (ADR-0117).
func (d *Database) SaveDocumentReleve(path, releveJSON string) (string, error) {
	r, err := document.Decode([]byte(releveJSON))
	if err != nil {
		return "", err
	}
	dst := document.ReleveFileFor(path)
	return dst, document.WriteFile(dst, r)
}

// VerifyDocumentEntry records a person's verdict on the entry of the given
// rank — verified or not, with position as the correction of what was read
// when not empty — and returns the Relevé as JSON text, also written where
// SaveDocumentReleve writes it when path is not empty.
func (d *Database) VerifyDocumentEntry(path, releveJSON string, rank int, verified bool, position string) (string, error) {
	r, err := document.Decode([]byte(releveJSON))
	if err != nil {
		return "", err
	}
	if err := document.SetVerified(r, rank, verified, position); err != nil {
		return "", err
	}
	out, err := r.Encode()
	if err != nil {
		return "", err
	}
	if path != "" {
		if err := document.WriteFile(document.ReleveFileFor(path), r); err != nil {
			return string(out), fmt.Errorf("the verdict is kept until the import but could not be saved: %w", err)
		}
	}
	return string(out), nil
}

// DocumentDiagramImage crops the diagram at the rectangle (fractions of the
// page) of the given page of what the person opened at path, from its
// document, and returns it as a PNG data URL at most 480 pixels wide. It is
// redrawn from the file each time and never stored.
func (d *Database) DocumentDiagramImage(path string, page int, x, y, w, h float64) (string, error) {
	src := document.SourceFor(path)
	if src == "" {
		return "", document.ErrNoPageImage
	}
	png, err := document.DiagramImage(context.Background(), src, page, document.Rect{X: x, Y: y, W: w, H: h}, 480)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), nil
}
