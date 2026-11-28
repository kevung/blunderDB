package database

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/arkqr"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/progress"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// PaperArchiveExport is what ExportPaperArchive wrote.
type PaperArchiveExport = ingest.ArchiveExport

// PaperArchiveSkip is a match an archive could not hold, and why.
type PaperArchiveSkip = ingest.ArchiveSkip

// ExportPaperArchive writes matches as a printable paper archive (ADR-0119):
// an HTML page of A4 sheets of QR codes. No ids means the whole library.
// origin, when set, is the producer's watermark line, printed at the foot of
// each sheet.
func (d *Database) ExportPaperArchive(matchIDs []int64, outPath, origin string) (*PaperArchiveExport, error) {
	page, res, err := PaperArchivePage(d, matchIDs, origin)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(outPath, page, 0o644); err != nil {
		return nil, err
	}
	return res, nil
}

// PaperArchivePage is the page ExportPaperArchive writes, for a caller that
// prints it rather than saving it. A function, not a method: the interface
// binds every method of Database, and a page of bytes is not for the webview.
func PaperArchivePage(d *Database, matchIDs []int64, origin string) ([]byte, *PaperArchiveExport, error) {
	return PaperArchivePageCtx(context.Background(), d, matchIDs, origin)
}

// PaperArchivePageCtx is PaperArchivePage under ctx: cancelling it stops the
// encoding between two matches, and a progress.Func in ctx hears one unit a
// match encoded.
func PaperArchivePageCtx(ctx context.Context, d *Database, matchIDs []int64, origin string) ([]byte, *PaperArchiveExport, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	page, res, err := ingest.ExportArchive(ctx, d.store, "", matchIDs, origin)
	if err != nil {
		return nil, nil, err
	}
	return page, &res, nil
}

// PaperArchiveImport is what ImportPaperArchive read and wrote.
type PaperArchiveImport struct {
	// Codes counts the archive codes read, Repaired those rebuilt from parity.
	Codes    int `json:"codes"`
	Repaired int `json:"repaired"`
	// Imported lists the new matches; Duplicates counts those the library
	// already held, from this archive or from another format.
	Imported   []int64 `json:"imported"`
	Duplicates int     `json:"duplicates"`
	// Positions counts the positions a printed list's reimport code brought
	// that the library did not hold, KnownPositions the others: those it
	// already held, and a position the list prints twice from its second
	// line on. Each is saved on its own, as a position the user brings.
	Positions      int `json:"positions"`
	KnownPositions int `json:"knownPositions"`
	// Missing lists the ranks (from 1) of the codes neither read nor rebuilt.
	Missing []int `json:"missing,omitempty"`
	// Failed lists the matches that could not be read or written, and why.
	Failed []string `json:"failed,omitempty"`
}

// ImportPaperArchive reads paper archives — the exported HTML page, whose
// embedded matches need no code reading, or scans and photographs (PNG,
// JPEG) of its sheets, any number and in any order — and imports their
// matches through the import path of every format: an import batch when the
// caller opened one, and the duplicate check. The reimport codes of a printed
// position list, read the same way, bring their positions back one by one.
// Nothing about the archive or its producer is recorded (ADR-0007).
func (d *Database) ImportPaperArchive(paths []string) (*PaperArchiveImport, error) {
	return ImportPaperArchiveCtx(context.Background(), d, paths)
}

// ImportPaperArchiveCtx is ImportPaperArchive under ctx, which stops it like
// CancelImport does: between two files read, or two matches written. A
// progress.Func in ctx hears one unit a file read, then one a match written.
// Each match is written under its own lock, so the library stays readable
// (and writable) between two of them; a match is never written half.
func ImportPaperArchiveCtx(parent context.Context, d *Database, paths []string) (*PaperArchiveImport, error) {
	// Reading scans is the long part: it is cancellable like the writes.
	ctx, done := d.beginCancellableImportFrom(parent)
	defer done()
	var frames []arkqr.Frame
	for i, p := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		progress.Report(ctx, i, len(paths))
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		fs, err := ingest.ReadArchive(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		frames = append(frames, fs...)
	}
	res := &PaperArchiveImport{Codes: len(frames)}
	if len(frames) == 0 {
		return res, fmt.Errorf("no archive code found: %w", storage.ErrInvalid)
	}
	graphs, asm, failed := ingest.ArchiveGraphs(frames)
	positions, pfailed := ingest.ArchivePositions(asm)
	res.Repaired, res.Failed = asm.Repaired, append(failed, pfailed...)
	for _, ranks := range asm.Missing {
		for _, r := range ranks {
			res.Missing = append(res.Missing, r+1)
		}
	}

	total := len(paths) + len(graphs)
	progress.Report(ctx, len(paths), total)
	var err error
	d.mu.Lock()
	res.Positions, res.KnownPositions, err = ingest.SaveArchivePositions(ctx, d.store.Positions(), "", positions)
	d.mu.Unlock()
	if err != nil {
		return res, err
	}
	for i, graph := range graphs {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		d.mu.Lock()
		id, enriched, err := d.writeImportedMatchEnriched(ctx, graph)
		d.mu.Unlock()
		progress.Report(ctx, len(paths)+i+1, total)
		switch {
		case errors.Is(err, ErrDuplicateMatch), err == nil && enriched:
			res.Duplicates++
		case err != nil:
			if ctx.Err() != nil {
				return res, err
			}
			res.Failed = append(res.Failed, fmt.Sprintf("match %d: %v", i+1, err))
		default:
			res.Imported = append(res.Imported, id)
		}
	}
	return res, nil
}

// IsPaperArchivePage reports whether a file is an exported archive page, or
// a PDF printed by blunderDB (a sheet, a list, an archive), so the ordinary
// import can hand it over. Any other PDF is not one: the import says so.
func IsPaperArchivePage(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".html", ".htm":
		data, err := os.ReadFile(path)
		return err == nil && arkqr.IsPage(data)
	case ".pdf":
		return true
	}
	return false
}
