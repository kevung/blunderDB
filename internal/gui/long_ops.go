package gui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/jobs"
	"github.com/kevung/blunderdb/pkg/blunderdb/progress"
)

// The long operations the user launches from a dialog — the export of a
// library, the paper archive both ways, the printed sheets — run as tasks of
// the panel of tasks rather than as calls the dialog waits on: the dialog
// closes at once and the user goes on working. Each Start* method returns
// the task's id and sends, on its event prefix,
//
//	<prefix>:progress {id, done, total}  at most about four a second
//	<prefix>:done     {id, result}
//	<prefix>:cancelled {id}
//	<prefix>:error    {id, message}
//
// exactly one of the last three. CancelJob(id) stops it; the work stops at
// its next batch and leaves nothing half written.

// LongOpProgress is the payload of <prefix>:progress.
type LongOpProgress struct {
	ID    int64 `json:"id"`
	Done  int   `json:"done"`
	Total int   `json:"total"`
}

// startLongOp registers a task of kind on the library on screen and runs
// work in the background on the bound handle itself, not on a borrowed one:
// the export and the sheets read one consistent cut of the library under its
// read lock, which only the bound handle's writers — the interface's — take
// too; and the archive import, like every direct import, is cancelled by a
// switch of library (beginCancellableImport). A switch also cancels the
// others first (cancelBoundLongOps), so it never waits a whole export out.
// A refusal (no library) still sends an error event, so the frontend reads
// every outcome in one place; its id is then 0.
func (a *App) startLongOp(kind jobs.Kind, label, prefix string, work func(ctx context.Context, db *database.Database) (any, error)) int64 {
	db := a.db
	if db == nil {
		a.emitBatch(prefix+":error", map[string]any{"id": int64(0), "message": "no database open"})
		return 0
	}
	ctx, cancel := context.WithCancel(a.batchCtx())
	task := a.jobs.Start(jobs.Spec{Kind: kind, Label: label, Library: a.shownGroup(), Priority: jobs.Requested, Cancel: cancel})
	id := task.ID()
	report := progress.Throttle(progress.Gap, func(done, total int) {
		task.Progress(done, total)
		a.emitBatch(prefix+":progress", LongOpProgress{ID: id, Done: done, Total: total})
	})
	go func() {
		defer task.Finish(errTaskPanicked) // after a panic: frees the library
		defer cancel()
		defer recoverBackground(a.ctx, string(kind))
		res, err := work(progress.With(ctx, report), db)
		switch {
		case err == nil:
			task.Finish(nil)
			a.emitBatch(prefix+":done", map[string]any{"id": id, "result": res})
		case errors.Is(err, context.Canceled) || ctx.Err() != nil:
			task.Finish(context.Canceled)
			a.emitBatch(prefix+":cancelled", map[string]any{"id": id})
		default:
			task.Finish(err)
			a.emitBatch(prefix+":error", map[string]any{"id": id, "message": err.Error()})
		}
	}()
	return id
}

// longOpKinds are the kinds startLongOp runs on the bound handle.
var longOpKinds = map[jobs.Kind]bool{jobs.Export: true, jobs.PaperArchive: true, jobs.PrintSheet: true}

// cancelBoundLongOps stops the long operations running on the bound handle,
// before it carries another library: the switch then waits for the current
// batch, not for the whole operation.
func (a *App) cancelBoundLongOps() {
	for _, t := range a.jobs.List() {
		if longOpKinds[t.Kind] && !t.State.Finished() {
			_ = a.jobs.Cancel(t.ID)
		}
	}
}

// StartExportDatabase is the export dialog's ExportDatabase as a task
// (export-db:*); its result is the ingest.ExportReport.
func (a *App) StartExportDatabase(opts database.ExportOptions) int64 {
	return a.startLongOp(jobs.Export, filepath.Base(opts.ExportPath), "export-db", func(ctx context.Context, db *database.Database) (any, error) {
		return db.ExportDatabaseCtx(ctx, opts)
	})
}

// PaperArchiveExportResult is what StartPaperArchiveExport wrote, and where.
type PaperArchiveExportResult struct {
	*database.PaperArchiveExport
	Path string `json:"path"`
}

// StartPaperArchiveExport writes the paper archive of matchIDs (none: the
// whole library) at path, printed to PDF when path ends in .pdf, as the HTML
// page otherwise (paper-archive:*). The PDF render is the task's last step.
func (a *App) StartPaperArchiveExport(matchIDs []int64, path string) int64 {
	return a.startLongOp(jobs.PaperArchive, "export", "paper-archive", func(ctx context.Context, db *database.Database) (any, error) {
		pdf, err := checkSheetPath(path)
		if err != nil {
			return nil, err
		}
		page, res, err := database.PaperArchivePageCtx(ctx, db, matchIDs, "")
		if err != nil {
			return nil, err
		}
		if err := writeSheet(ctx, path, page, pdf); err != nil {
			return nil, err
		}
		return PaperArchiveExportResult{PaperArchiveExport: res, Path: path}, nil
	})
}

// StartPaperArchiveImport is ImportPaperArchive as a task
// (paper-archive:*); its result is the database.PaperArchiveImport.
func (a *App) StartPaperArchiveImport(paths []string) int64 {
	return a.startLongOp(jobs.PaperArchive, "import", "paper-archive", func(ctx context.Context, db *database.Database) (any, error) {
		return database.ImportPaperArchiveCtx(ctx, db, paths)
	})
}

// SheetResult is where a printed sheet went.
type SheetResult struct {
	Path string `json:"path"`
}

// StartMatchSheet prints the sheets of matchIDs to path, a PDF or an HTML
// page by its extension (print-sheet:*).
func (a *App) StartMatchSheet(matchIDs []int64, opts database.MatchSheetOptions, path string) int64 {
	return a.startLongOp(jobs.PrintSheet, filepath.Base(path), "print-sheet", func(ctx context.Context, db *database.Database) (any, error) {
		pdf, err := checkSheetPath(path)
		if err != nil {
			return nil, err
		}
		page, err := database.MatchSheetHTMLCtx(ctx, db, matchIDs, opts)
		if err != nil {
			return nil, err
		}
		return SheetResult{Path: path}, writeSheet(ctx, path, []byte(page), pdf)
	})
}

// StartPositionSheet prints the position list src names to path, a PDF or
// an HTML page by its extension (print-sheet:*).
func (a *App) StartPositionSheet(src database.PositionSheetSource, opts database.PositionSheetOptions, path string) int64 {
	return a.startLongOp(jobs.PrintSheet, filepath.Base(path), "print-sheet", func(ctx context.Context, db *database.Database) (any, error) {
		pdf, err := checkSheetPath(path)
		if err != nil {
			return nil, err
		}
		page, err := database.PositionSheetHTMLCtx(ctx, db, src, opts)
		if err != nil {
			return nil, err
		}
		return SheetResult{Path: path}, writeSheet(ctx, path, []byte(page), pdf)
	})
}

// checkSheetPath accepts an absolute .pdf or .html path, and says which.
func checkSheetPath(path string) (pdf bool, err error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".pdf":
		return true, checkPDFPath(path)
	case ".html", ".htm":
		if !filepath.IsAbs(path) {
			return false, newGUIError(CodeInvalid, "page path must be absolute: "+path)
		}
		return false, nil
	}
	return false, newGUIError(CodeInvalid, "not a PDF or HTML path: "+path)
}

// writeSheet writes page at path, rendered to PDF when pdf; a cancel that
// came before the write leaves no file.
func writeSheet(ctx context.Context, path string, page []byte, pdf bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if pdf {
		return writeSheetPDF(ctx, path, page)
	}
	return os.WriteFile(path, page, 0o644)
}
