package storage

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// ImportBatchStore persists the import batches — one row per import the user
// launched — and measures what each of them brought in.
//
// The counts only the running import sees (imported, skipped, enriched,
// failed) are written by the importer. Everything else is MEASURED afterwards
// over the batch's matches, so it can be recomputed from a batch id and does
// not depend on an import that may have been cancelled halfway.
type ImportBatchStore interface {
	// Begin opens a batch for an import that is starting and returns its id.
	// source is shown to the user verbatim (a file path, a folder); format is
	// the import format, or "mixed" for a folder holding several.
	Begin(ctx context.Context, scope string, source, format string) (int64, error)

	// Finish stamps the batch as done and stores the counts the import
	// observed. It does not measure anything: Report does that.
	Finish(ctx context.Context, scope string, batchID int64, counts domain.ImportReport) error

	// Load returns a batch with its stored counts, or ErrNotFound.
	Load(ctx context.Context, scope string, batchID int64) (*domain.ImportBatch, error)

	// List returns the batches, most recent first, bounded by opts.
	List(ctx context.Context, scope string, opts ListOpts) ([]*domain.ImportBatch, error)

	// Report returns the batch's stored counts completed by what can be
	// measured over its matches now: flagged positions, positions no engine
	// has judged, the batch's own error rate, and its worst decisions.
	//
	// Recomputed on every call, not cached: positions can be analysed later.
	//
	// players names whose decisions the error rate is about (the reference
	// player and their alternate spellings). Empty scores BOTH seats, and the
	// report says so (ImportReport.Player).
	Report(ctx context.Context, scope string, batchID int64, players []string) (*domain.ImportBatch, error)

	// StudyQueue returns the batch's positions worth a second look, in the
	// order they should be walked: the decisions that cost something, worst
	// first; then the positions the source tool had marked for study; then the
	// close cube decisions.
	//
	// A position appears ONCE, under the first reason that claims it.
	//
	// It is measured, never stored: nothing records that a position was seen.
	//
	// players is read as it is by Report: empty scores both seats.
	StudyQueue(ctx context.Context, scope string, batchID int64, players []string, limit int) ([]domain.StudyQueueEntry, error)

	// StudyBacklog returns, across every batch and every match, the
	// positions of players whose played decision cost at least the library's
	// error threshold and that nothing has dealt with yet: no comment, no
	// Anki card, no collection, no study mark. Costliest first, at most limit
	// (0 means domain.MaxStudyQueue), each position once.
	//
	// players is read as it is by Report: empty scores both seats.
	StudyBacklog(ctx context.Context, scope string, players []string, limit int) ([]domain.StudyQueueEntry, error)

	// SetStudied writes (studied true) or withdraws the user's "studied" mark
	// on a position, which removes it from StudyBacklog. Idempotent; ErrNotFound
	// when the position does not exist. The mark is the user's own data: it is
	// never exported and nothing but this call writes it (ADR-0007).
	SetStudied(ctx context.Context, scope string, positionID int64, studied bool) error

	// RecordFiles appends lines to the batch's journal: one per file the
	// import decided. Called after the file's transaction committed, so a
	// journaled file is a decided file.
	RecordFiles(ctx context.Context, scope string, batchID int64, files []domain.ImportFileEntry) error

	// Files returns the batch's journal, in the order it was written, or
	// ErrNotFound when the batch does not exist.
	Files(ctx context.Context, scope string, batchID int64) ([]domain.ImportFileEntry, error)

	// StudyMark returns when the position was marked studied (Unix seconds),
	// 0 when it is not.
	StudyMark(ctx context.Context, scope string, positionID int64) (int64, error)
	// RestoreStudyMark puts back a mark StudyMark read, with its own date,
	// for a position restored from the trash. A mark already there is kept.
	RestoreStudyMark(ctx context.Context, scope string, positionID, markedAt int64) error

	// FilesOfMatch returns the ids of the journal lines that name the match.
	FilesOfMatch(ctx context.Context, scope string, matchID int64) ([]int64, error)
	// RelinkFiles names matchID again on the journal lines of fileIDs that
	// name no match: a match restored from the trash finds its import's
	// journal again. A line that names another match is left as it is.
	RelinkFiles(ctx context.Context, scope string, fileIDs []int64, matchID int64) error
}
