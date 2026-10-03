package ingest

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The import journal records, for each file a batch met, what it was and what
// it gave, so a batch interrupted halfway can be resumed without parsing again
// what it already decided. A file counts as decided once its transaction has
// committed; a file that failed is not decided, a resumed batch tries it again.

// JournalTime spells a modification time the way the journal stores it: UTC,
// to the second (file systems differ below that).
func JournalTime(t time.Time) string {
	return t.UTC().Format("2006-01-02 15:04:05")
}

// JournalEntry is the journal line of a file's outcome.
func JournalEntry(o FileOutcome) domain.ImportFileEntry {
	e := domain.ImportFileEntry{
		Path: o.Path, Size: o.Size, MTime: o.ModTime, SHA256: o.SHA256, MatchID: o.MatchID,
	}
	switch o.Status {
	case FileDuplicate:
		e.Outcome = domain.JournalDuplicate
	case FileEnriched:
		e.Outcome = domain.JournalEnriched
	case FileFailed:
		e.Outcome, e.Error = domain.JournalError, o.Error
	default:
		e.Outcome = domain.JournalNew
	}
	return e
}

// RecordOutcomes appends a committed group to the batch's journal. A journal
// that cannot be written costs only the ability to resume: it is logged, never
// an import failure. batchID 0 (no batch) records nothing.
func RecordOutcomes(ctx context.Context, batches storage.ImportBatchStore, scope string, batchID int64, group []FileOutcome) {
	if batchID == 0 || len(group) == 0 {
		return
	}
	entries := make([]domain.ImportFileEntry, len(group))
	for i, o := range group {
		entries[i] = JournalEntry(o)
	}
	if err := batches.RecordFiles(ctx, scope, batchID, entries); err != nil {
		slog.Warn("import journal: recording files failed", "batch", batchID, "err", err)
	}
}

// Journal is a batch's journal loaded to decide what a resumed run skips.
type Journal struct {
	byPath map[journalKey]struct{}
	bySHA  map[string]int64
}

type journalKey struct {
	path  string
	size  int64
	mtime string
}

// LoadJournal reads the journal of batchID, or storage.ErrNotFound when the
// batch does not exist.
func LoadJournal(ctx context.Context, batches storage.ImportBatchStore, scope string, batchID int64) (*Journal, error) {
	files, err := batches.Files(ctx, scope, batchID)
	if err != nil {
		return nil, err
	}
	j := &Journal{byPath: map[journalKey]struct{}{}, bySHA: map[string]int64{}}
	for _, f := range files {
		if f.Outcome == domain.JournalError {
			continue
		}
		if f.MTime != "" {
			j.byPath[journalKey{f.Path, f.Size, f.MTime}] = struct{}{}
		}
		if f.SHA256 != "" {
			j.bySHA[f.SHA256] = f.MatchID
		}
	}
	return j, nil
}

// Pending returns the paths the journal does not already decide, in order: a
// file listed with the same path, size and modification time is skipped
// without being read. skipped counts the files left out.
func (j *Journal) Pending(paths []string) (todo []string, skipped int) {
	todo = make([]string, 0, len(paths))
	for _, p := range paths {
		fi, err := os.Stat(p)
		if err == nil {
			if _, ok := j.byPath[journalKey{p, fi.Size(), JournalTime(fi.ModTime())}]; ok {
				skipped++
				continue
			}
		}
		todo = append(todo, p)
	}
	return todo, skipped
}

// Known answers PipelineOptions.Known: a file with the digest of one the
// journal decided is not parsed again.
func (j *Journal) Known(sha string) (int64, bool) {
	id, ok := j.bySHA[sha]
	return id, ok
}
