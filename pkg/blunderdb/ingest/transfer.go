package ingest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// A transfer copies part of one library into another (ADR-0102): the
// selection of an export, read from the source, merged into the destination
// by the native import. It is the export and the import the user could run by
// hand, chained without a file the user handles, so that the three modes copy
// exactly what an export carries and merge exactly as an import of it does:
// positions dedup by Zobrist hash, analyses and comments merge, matches,
// collections, lessons and decks follow MergeMatches / MergeCollections /
// MergeLessons / MergeDecks. A → B then B → A therefore creates nothing the
// first copy did not.
//
// Nothing is written that says where the copy came from: no watermark, no
// import batch, no metadata of the source. The intermediate file holds only
// issuance.CarriedMetadataKeys, and the native import reads none of it. Nor
// does the source receive anything: an incremental export, which advances a
// cursor in its source, is refused.

// TransferEnd is one side of a transfer: a library and the scope within it
// (a tenant on PostgreSQL, "" on a SQLite file).
type TransferEnd struct {
	S     storage.Storage
	Scope string
}

// TransferReport counts what a transfer read and what it merged.
type TransferReport struct {
	// Exported counts what the selection carried out of the source.
	Exported ExportReport `json:"exported"`
	// Imported counts what the destination took of it; a row it already
	// held is counted as a duplicate, not added.
	Imported Summary `json:"imported"`
}

// ErrSelfTransfer refuses a transfer whose source and destination are the
// same library: it would copy nothing and still hold both ends.
var ErrSelfTransfer = errors.New("a library cannot be transferred into itself")

// Transfer copies what opts selects in from into to. opts are the export
// options of the desktop, the CLI and the daemon, read by OptionsFor; their
// watermark, password, incremental and metadata fields are refused, each
// being a statement about a file or a write on the source.
//
// Each side is read or written under a context scoped to its own tenant, so a
// caller holding the destination tenant's context reads the source under the
// source's (row-level security on PostgreSQL).
func Transfer(ctx context.Context, from, to TransferEnd, opts domain.ExportOptions, prog func(Progress)) (TransferReport, error) {
	if from.S == to.S && from.Scope == to.Scope {
		return TransferReport{}, fmt.Errorf("%w: %w", storage.ErrInvalid, ErrSelfTransfer)
	}
	switch {
	case opts.Incremental:
		return TransferReport{}, fmt.Errorf("%w: a transfer is not incremental: it writes no cursor in its source", storage.ErrInvalid)
	case opts.Watermark != "" || opts.WatermarkNote != "":
		return TransferReport{}, fmt.Errorf("%w: a transfer carries no watermark: nothing records where a copy came from", storage.ErrInvalid)
	case opts.Password != "":
		return TransferReport{}, fmt.Errorf("%w: a transfer writes no file to protect", storage.ErrInvalid)
	case len(opts.Metadata) > 0:
		return TransferReport{}, fmt.Errorf("%w: a transfer carries no metadata: the destination keeps its own", storage.ErrInvalid)
	}
	srcCtx, err := scopedContext(ctx, from.Scope)
	if err != nil {
		return TransferReport{}, err
	}
	dstCtx, err := scopedContext(ctx, to.Scope)
	if err != nil {
		return TransferReport{}, err
	}

	if err := checkNamed(srcCtx, from, opts); err != nil {
		return TransferReport{}, err
	}
	plan, err := OptionsFor(srcCtx, from.S, from.Scope, opts)
	if err != nil {
		return TransferReport{}, err
	}
	if err := transferSelection(srcCtx, from, opts, &plan); err != nil {
		return TransferReport{}, err
	}

	dir, err := os.MkdirTemp("", "blunderdb-transfer-*")
	if err != nil {
		return TransferReport{}, fmt.Errorf("ingest: transfer: create a working folder: %w", err)
	}
	defer func() {
		if rmErr := os.RemoveAll(dir); rmErr != nil {
			slog.Warn("transfer: removing the working folder", "dir", dir, "err", rmErr)
		}
	}()
	path := filepath.Join(dir, "transfer.db")

	var rep TransferReport
	if rep.Exported, err = ExportSQLite(srcCtx, from.S, from.Scope, path, plan); err != nil {
		return rep, fmt.Errorf("ingest: transfer: read the source: %w", err)
	}
	rep.Exported.Path = ""
	if rep.Imported, err = (DBImporter{S: to.S}).Import(dstCtx, to.Scope, Source{Format: FormatNativeDB, Path: path}, prog); err != nil {
		return rep, fmt.Errorf("ingest: transfer: merge into the destination: %w", err)
	}
	return rep, nil
}

// namesSome reports whether opts narrow a family to listed ids.
func namesSome(opts domain.ExportOptions) bool {
	return len(opts.CollectionIDs) > 0 || len(opts.MatchIDs) > 0 || len(opts.TournamentIDs) > 0 ||
		len(opts.LessonIDs) > 0 || len(opts.DeckIDs) > 0
}

// transferSelection turns the export's plan into what a transfer copies. An
// export of the whole library keeps every position even when its families are
// narrowed, so that the file stays a library; a transfer that names ids is a
// pick, and the destination must receive what was picked and nothing more —
// least of all the comments of positions nobody chose. So:
//
//   - once a family is narrowed to listed ids, the positions are those the
//     families kept reach (members, moves, lesson steps, deck cards);
//   - a collection always travels with its members, never empty;
//   - a match named by id brings its tournament's record, not the
//     tournament's other matches, unless tournaments are named or left out.
func transferSelection(ctx context.Context, from TransferEnd, opts domain.ExportOptions, plan *ExportOptions) error {
	sel := &plan.Selection
	sel.CollectionPositions = true
	if opts.WholeLibrary && namesSome(opts) {
		sel.AllPositions = false
		sel.MatchPositions = true
		if plan.AnkiDecks && len(sel.DeckIDs) == 0 {
			for d, err := range from.S.Anki().ListDecks(ctx, from.Scope) {
				if err != nil {
					return fmt.Errorf("ingest: transfer: list decks: %w", err)
				}
				sel.DeckIDs = append(sel.DeckIDs, d.ID)
			}
		}
		sel.DeckPositions = len(sel.DeckIDs) > 0
	}
	if len(opts.MatchIDs) > 0 && opts.IncludeTournaments && len(opts.TournamentIDs) == 0 {
		var tids []int64
		seen := make(map[int64]bool)
		for _, id := range opts.MatchIDs {
			m, err := from.S.Matches().Get(ctx, from.Scope, id)
			if err != nil {
				return fmt.Errorf("ingest: transfer: read match %d: %w", id, err)
			}
			if m.TournamentID != nil && !seen[*m.TournamentID] {
				seen[*m.TournamentID] = true
				tids = append(tids, *m.TournamentID)
			}
		}
		sel.AllTournaments, sel.TournamentIDs, sel.TournamentMatches = false, tids, false
	}
	return nil
}

// checkNamed refuses an id the source does not hold. An export skips a stale
// id and goes on; a transfer would then copy the rest of the family, or all of
// it, for what is most often a typo or an id read in the other library.
func checkNamed(ctx context.Context, from TransferEnd, opts domain.ExportOptions) error {
	check := func(family string, ids []int64, get func(int64) error) error {
		for _, id := range ids {
			err := get(id)
			if errors.Is(err, storage.ErrNotFound) {
				return fmt.Errorf("%w: %s %d is not in the source library", storage.ErrInvalid, family, id)
			}
			if err != nil {
				return fmt.Errorf("ingest: transfer: read %s %d: %w", family, id, err)
			}
		}
		return nil
	}
	s, scope := from.S, from.Scope
	if !opts.WholeLibrary && len(opts.Positions) == 0 {
		if err := check("position", opts.PositionIDs, func(id int64) error {
			_, err := s.Positions().Load(ctx, scope, id)
			return err
		}); err != nil {
			return err
		}
	}
	if err := check("collection", opts.CollectionIDs, func(id int64) error {
		_, err := s.Collections().Get(ctx, scope, id)
		return err
	}); err != nil {
		return err
	}
	if err := check("match", opts.MatchIDs, func(id int64) error {
		_, err := s.Matches().Get(ctx, scope, id)
		return err
	}); err != nil {
		return err
	}
	if err := check("tournament", opts.TournamentIDs, func(id int64) error {
		_, err := s.Tournaments().Get(ctx, scope, id)
		return err
	}); err != nil {
		return err
	}
	if err := check("lesson", opts.LessonIDs, func(id int64) error {
		_, err := s.Lessons().Get(ctx, scope, id)
		return err
	}); err != nil {
		return err
	}
	if len(opts.DeckIDs) == 0 {
		return nil
	}
	decks := make(map[int64]bool)
	for d, err := range s.Anki().ListDecks(ctx, scope) {
		if err != nil {
			return fmt.Errorf("ingest: transfer: list decks: %w", err)
		}
		decks[d.ID] = true
	}
	return check("Anki deck", opts.DeckIDs, func(id int64) error {
		if !decks[id] {
			return storage.ErrNotFound
		}
		return nil
	})
}

// scopedContext scopes ctx to tenant scope; "" (a SQLite file) leaves it as is.
func scopedContext(ctx context.Context, scope string) (context.Context, error) {
	if scope == "" {
		return ctx, nil
	}
	n, err := storage.ParseTenant(scope)
	if err != nil {
		return nil, err
	}
	return storage.WithTenant(ctx, n), nil
}
