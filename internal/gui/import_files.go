package gui

import (
	"context"
	"errors"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// importFileEvent carries one decided file of an ImportFiles call.
const importFileEvent = "import-files:file"

// ImportFileError is one file the import could not bring in.
type ImportFileError struct {
	File    string `json:"file"`
	Message string `json:"message"`
}

// ImportFilesSummary is what the frontend needs once a list of files has gone
// through the pipeline; the per-file detail travels by event as it happens.
type ImportFilesSummary struct {
	Succeeded      int               `json:"succeeded"`
	Skipped        int               `json:"skipped"`
	Failed         int               `json:"failed"`
	Errors         []ImportFileError `json:"errors"`
	HadMatches     bool              `json:"hadMatches"`
	LastPositionID int64             `json:"lastPositionID"`
	Cancelled      bool              `json:"cancelled"`
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
// pipeline, emitting one event per decided file. CancelImport (bound on the
// Database) stops it; what was committed stays and is summarised.
func (a *App) ImportFiles(paths []string) (ImportFilesSummary, error) {
	if a.db == nil {
		return ImportFilesSummary{}, fmt.Errorf("no database is currently open")
	}
	out, err := a.db.ImportFiles(paths, database.ImportFilesOptions{
		OnFile: func(o ingest.FileOutcome) { a.emitBatch(importFileEvent, o) },
	})
	s := summarize(out)
	if err != nil {
		if len(out) < len(paths) && errors.Is(err, context.Canceled) {
			s.Cancelled = true
			return s, nil
		}
		return s, err
	}
	return s, nil
}
