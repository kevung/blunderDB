package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// openExistingSQLite opens a database file that is only being read from —
// an import source, a file being inspected. The path is checked first (a
// directory is refused): sql.Open would create an empty file at a mistyped one.
func openExistingSQLite(path string) (*sql.DB, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("database file not found: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("database file not found: %s is a directory", path)
	}
	return sql.Open("sqlite", path)
}

// writeImportedMatch persists a mapped MatchGraph through the storage backend,
// shared by the format-specific Import* methods that delegate to the ingest
// pipeline. It preserves the GUI/CLI duplicate contract: an exact same-format
// re-import returns a *DuplicateMatchError, which is ErrDuplicateMatch (the
// ingest layer reports it as a silent skip), while a cross-format canonical duplicate is enriched in place and
// returns the existing match id without error. Callers must hold d.mu.
func (d *Database) writeImportedMatch(ctx context.Context, graph *ingest.MatchGraph) (int64, error) {
	// Stamp the import batch here, the one point every format passes through.
	graph.ImportBatchID = d.importBatchID
	graph.SkipDuplicates = d.skipDuplicates.Load()
	tx, err := d.store.BeginTx(ctx)
	if err != nil {
		return 0, err
	}
	res, err := ingest.WriteMatch(ctx, tx, "", graph, nil)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	d.importBatchCounts.AnalysesDropped += res.DroppedAnalyses
	if res.Skipped {
		// A duplicate still carries the study marks added in the source tool
		// since the first import (ADR-0006), and the analyses deeper than the
		// stored ones: they are committed, the rest of the file was never
		// written.
		if res.FlagsApplied > 0 || res.Deepened > 0 {
			if err := tx.Commit(); err != nil {
				return 0, err
			}
		} else {
			_ = tx.Rollback()
		}
		d.importBatchCounts.MatchesSkipped++
		if res.Deepened > 0 {
			d.importBatchCounts.MatchesDeepened++
			d.importBatchCounts.AnalysesDeepened += res.Deepened
		}
		return 0, &DuplicateMatchError{MatchID: res.MatchID, FlagsApplied: res.FlagsApplied, Deepened: res.Deepened}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	if res.ProbableDuplicate != nil {
		d.importBatchCounts.ProbableDuplicates = append(d.importBatchCounts.ProbableDuplicates, *res.ProbableDuplicate)
	}
	// Only the writing path knows written vs enriched; callers see an id either way.
	if res.Enriched {
		d.importBatchCounts.MatchesEnriched++
	} else {
		d.importBatchCounts.MatchesImported++
	}
	d.importBatchCounts.PositionsSaved += res.SavedPositions
	d.positionsSinceStats += res.SavedPositions
	return res.MatchID, nil
}

// writeImportedPosition persists the positions of a single-position import
// (XGP / BGF text) through the storage backend and returns the id of the first
// stored position. Positions dedup by Zobrist hash, so re-importing the same
// position returns its existing id. Callers must hold d.mu and pass the
// context from their own beginCancellableImport, so CancelImport reaches it.
func (d *Database) writeImportedPosition(ctx context.Context, graphs []ingest.PositionGraph) (int64, error) {
	if len(graphs) == 0 {
		return 0, fmt.Errorf("no position to import")
	}
	tx, err := d.store.BeginTx(ctx)
	if err != nil {
		return 0, err
	}
	var firstID int64
	for i := range graphs {
		id, err := ingest.WritePosition(ctx, tx, "", &graphs[i])
		if err != nil {
			_ = tx.Rollback()
			return 0, err
		}
		if i == 0 {
			firstID = id
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	d.positionsSinceStats += len(graphs)
	return firstID, nil
}

// ErrDuplicateMatch is returned when attempting to import a match that already exists
var ErrDuplicateMatch = fmt.Errorf("duplicate match: this match has already been imported")

// DuplicateMatchError is how an import reports a match already stored:
// errors.Is(err, ErrDuplicateMatch) holds. FlagsApplied tells "duplicate, N
// study marks delivered" (committed) from "duplicate, nothing to do" (0);
// Deepened counts the stored analyses the duplicate replaced with deeper ones.
type DuplicateMatchError struct {
	MatchID      int64
	FlagsApplied int
	Deepened     int
}

func (e *DuplicateMatchError) Error() string {
	var extra []string
	if e.FlagsApplied > 0 {
		extra = append(extra, fmt.Sprintf("%d study marks applied", e.FlagsApplied))
	}
	if e.Deepened > 0 {
		extra = append(extra, fmt.Sprintf("%d analyses deepened", e.Deepened))
	}
	if len(extra) == 0 {
		return ErrDuplicateMatch.Error()
	}
	return fmt.Sprintf("%s (%s)", ErrDuplicateMatch.Error(), strings.Join(extra, ", "))
}

// Is makes every DuplicateMatchError match ErrDuplicateMatch.
func (e *DuplicateMatchError) Is(target error) bool { return target == ErrDuplicateMatch }

// CheckMatchExists checks if a match with the given hash already exists in the database
// Returns the existing match ID if found, 0 otherwise
func (d *Database) CheckMatchExists(matchHash string) (int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	id, _, err := d.store.Matches().FindByHash(context.Background(), "", matchHash, "")
	if err != nil {
		return 0, fmt.Errorf("error checking for duplicate match: %w", err)
	}
	return id, nil
}

// SetSkipDuplicates chooses what the following imports do with an exact
// duplicate: by default (false) its analyses deeper than the stored ones
// replace them and the rest is skipped; true skips it outright, the behaviour
// of `import --skip-duplicates`.
func (d *Database) SetSkipDuplicates(skip bool) { d.skipDuplicates.Store(skip) }

// FindDuplicateMatches lists the pairs of stored matches the dice say are
// probably one (ingest.FindDuplicateSuspects), filling on the way the
// dice_hash of the matches imported before it existed. Nothing is merged.
func (d *Database) FindDuplicateMatches() ([]domain.DuplicateSuspect, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.store == nil {
		return nil, fmt.Errorf("find duplicate matches: %w", storage.ErrInternal)
	}
	out, _, err := ingest.FindDuplicateSuspects(context.Background(), d.store.Matches(), "")
	return out, err
}
