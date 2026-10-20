package gui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// importProgressEvent carries an ImportFiles call's progress, at most four
// times a second and once at the end, with the library group it imports into
// (importProgress).
const importProgressEvent = "import-files:progress"

// importProgress is one progress event: the pipeline's counts and the group.
type importProgress struct {
	ingest.BatchProgress
	Group int `json:"group"`
}

// ImportFileError is one file the import could not bring in.
type ImportFileError struct {
	File    string `json:"file"`
	Message string `json:"message"`
}

// ImportFilesSummary is what the frontend needs once a list of files has gone
// through the pipeline; the progress travels by event as it happens.
type ImportFilesSummary struct {
	Succeeded      int               `json:"succeeded"`
	Skipped        int               `json:"skipped"`
	Failed         int               `json:"failed"`
	Errors         []ImportFileError `json:"errors"`
	HadMatches     bool              `json:"hadMatches"`
	LastPositionID int64             `json:"lastPositionID"`
	Cancelled      bool              `json:"cancelled"`
	// Group is the library group the files went into. A background import
	// also closes its batch: Report and Journal are then what the batch
	// brought in, nil when no batch was open or reading it failed.
	Group   int                      `json:"group"`
	Report  *domain.ImportBatch      `json:"report"`
	Journal []domain.ImportFileEntry `json:"journal"`
	// BatchClosed says the import closed the batch it took over: the caller
	// must not close it again from the bound handle.
	BatchClosed bool `json:"batchClosed"`
}

// summarize folds the pipeline's outcomes into the frontend summary.
func summarize(out []ingest.FileOutcome) ImportFilesSummary {
	s := ImportFilesSummary{Errors: []ImportFileError{}}
	for _, o := range out {
		switch o.Status {
		case ingest.FileDuplicate:
			s.Skipped++
		case ingest.FileFailed:
			s.Failed++
			s.Errors = append(s.Errors, ImportFileError{File: o.Path, Message: o.Error})
		case ingest.FilePosition:
			s.Succeeded++
			s.LastPositionID = o.PositionID
		default:
			s.Succeeded++
			s.HadMatches = true
		}
	}
	return s
}

// ImportFiles imports a list of files in one call through the parallel
// pipeline, emitting its progress by event. CancelImport (bound on the
// Database) stops it; what was committed stays and is summarised.
//
// background runs it on a handle borrowed from the active library
// (borrowLibrary): the import batch the frontend opened moves to it and is
// closed at the end, and the import keeps writing that library when another
// library group comes on screen. CancelLibraryImport stops it.
func (a *App) ImportFiles(paths []string, background bool) (ImportFilesSummary, error) {
	if a.db == nil {
		return ImportFilesSummary{}, fmt.Errorf("no database is currently open")
	}
	if !background {
		group := a.activeGroup()
		out, err := a.db.ImportFiles(paths, database.ImportFilesOptions{
			OnProgress: func(p ingest.BatchProgress) { a.emitBatch(importProgressEvent, importProgress{p, group}) },
		})
		s, err := summarizeImport(out, len(paths), err)
		s.Group = group
		return s, err
	}
	group, lent, err := a.borrowLibrary(0)
	if err != nil {
		return ImportFilesSummary{Group: group}, err
	}
	database.HandOverImportBatch(a.db, lent)
	s, err := a.importOn(group, lent, paths)
	s.BatchClosed = true
	if err != nil {
		// The batch has moved and is closed: an error past this point is
		// part of the summary, so the caller never closes the batch again.
		s.Errors = append(s.Errors, ImportFileError{Message: err.Error()})
		s.Failed += len(paths) - s.Succeeded - s.Skipped - s.Failed
	}
	return s, nil
}

// ImportFilesIn imports paths into the library of group, on screen or not, on
// a borrowed handle and in a batch of its own named source: what a watched
// folder brings in for a group the user has left.
func (a *App) ImportFilesIn(group int, paths []string, source string) (ImportFilesSummary, error) {
	group, lent, err := a.borrowLibrary(group)
	if err != nil {
		return ImportFilesSummary{Group: group}, err
	}
	if _, err := lent.BeginImportBatch(source, "mixed"); err != nil {
		slog.Warn("could not open an import batch; importing without one", "err", err)
	}
	return a.importOn(group, lent, paths)
}

// importOn runs the import on lent, a borrowed handle, closes its batch,
// refreshes the planner statistics and lets the handle go.
func (a *App) importOn(group int, lent *database.Database, paths []string) (ImportFilesSummary, error) {
	defer lent.Close() // lets the borrowed library go; closes nothing
	done := a.imports.add(group, lent)
	defer done()
	out, err := lent.ImportFiles(paths, database.ImportFilesOptions{
		OnProgress: func(p ingest.BatchProgress) { a.emitBatch(importProgressEvent, importProgress{p, group}) },
	})
	s, err := summarizeImport(out, len(paths), err)
	s.Group = group
	if batchID := database.OpenImportBatch(lent); batchID != 0 {
		if ferr := lent.FinishImportBatch(batchID, domain.ImportReport{}); ferr != nil {
			slog.Warn("could not close the import batch", "batch", batchID, "err", ferr)
		} else {
			s.Report, _ = lent.ImportReport(batchID)
			s.Journal, _ = lent.ImportJournal(batchID)
		}
	}
	if s.Succeeded > 0 {
		lent.RefreshSearchStatistics()
	}
	return s, err
}

// summarizeImport folds an ImportFiles outcome into the summary; a cancelled
// import is a summary, not an error.
func summarizeImport(out []ingest.FileOutcome, asked int, err error) (ImportFilesSummary, error) {
	s := summarize(out)
	if err != nil {
		if len(out) < asked && errors.Is(err, context.Canceled) {
			s.Cancelled = true
			return s, nil
		}
		return s, err
	}
	return s, nil
}

// CancelLibraryImport stops the background imports into group's library (0:
// every group); CancelImport on the Database stops one on the bound handle.
func (a *App) CancelLibraryImport(group int) {
	a.imports.cancel(group)
}

// importsRunning counts the background imports in flight.
func (a *App) importsRunning() int {
	a.imports.mu.Lock()
	defer a.imports.mu.Unlock()
	return len(a.imports.running)
}

// backgroundImports are the imports running on borrowed handles, so that a
// group's close and the shutdown can stop them and wait.
type backgroundImports struct {
	mu      sync.Mutex
	running map[*database.Database]runningImport
}

type runningImport struct {
	group int
	done  chan struct{}
}

func (b *backgroundImports) add(group int, lent *database.Database) func() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.running == nil {
		b.running = map[*database.Database]runningImport{}
	}
	r := runningImport{group: group, done: make(chan struct{})}
	b.running[lent] = r
	return func() {
		b.mu.Lock()
		delete(b.running, lent)
		b.mu.Unlock()
		close(r.done)
	}
}

// cancel stops the imports into group (0: all) and returns a channel closed
// once they have all ended, nil when none ran.
func (b *backgroundImports) cancel(group int) <-chan struct{} {
	b.mu.Lock()
	var waits []chan struct{}
	for lent, r := range b.running {
		if group == 0 || r.group == group {
			lent.CancelImport()
			waits = append(waits, r.done)
		}
	}
	b.mu.Unlock()
	if len(waits) == 0 {
		return nil
	}
	all := make(chan struct{})
	go func() {
		for _, w := range waits {
			<-w
		}
		close(all)
	}()
	return all
}
