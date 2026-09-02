package ingest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// This file is THE export of a blunderDB database to a file: the GUI's
// export dialog, the CLI's `export`, and the daemon's exports.sqlite all
// run ExportSQLite with a Selection that says what leaves and an
// ExportOptions that says how. Before it existed there were four copies of
// this walk (ExportDatabase, ExportCollections, ExportTournaments and the
// daemon's own), each with its own gaps — the daemon's carried no matches, no
// metadata and no watermark; the desktop's kept one comment per position and
// wrote analyses with their search columns empty.
//
// It reads the source through storage.Storage, so a PostgreSQL tenant exports
// exactly like a SQLite file, and writes a fresh SQLite database through the
// same schema and encoders a live database uses (sqlite.Bootstrap, the
// position and analysis writers), so the result opens in any blunderDB with
// dedup, indexes and SQL filters intact.

// Selection says which rows of the source leave in the export. Each family
// can be taken whole (All…), by identifier (…IDs), or not at all; the three
// closure flags pull in what a family reaches — a collection's members, a
// match's positions, a tournament's matches — so an export of "these
// tournaments" carries what a recipient needs to open them.
//
// Positions travel once whatever route brought them in, and a membership or
// a move that points at a position outside the selection is written without
// it (a collection row loses that member; a move keeps its dice and play but
// no position).
type Selection struct {
	// AllPositions exports every position of the scope. PositionIDs exports
	// the listed ones, in that order; Positions supplies them whole (the
	// caller already holds them) and wins over PositionIDs when set.
	AllPositions bool
	PositionIDs  []int64
	Positions    []*domain.Position

	AllCollections bool
	CollectionIDs  []int64
	// CollectionPositions adds the members of the selected collections to the
	// positions exported.
	CollectionPositions bool

	AllMatches bool
	MatchIDs   []int64
	// MatchPositions adds the positions reached by the selected matches.
	MatchPositions bool

	AllTournaments bool
	TournamentIDs  []int64
	// TournamentMatches adds the matches of the selected tournaments.
	TournamentMatches bool
}

// ExportReport counts what an export wrote. Skipped counts rows dropped along
// the way — a collection id that names nothing, a position that would not
// insert — each already logged where it happened; the export itself succeeds,
// because one stale identifier is not a reason to lose the rest.
type ExportReport struct {
	// Path is where the file landed: opts' path, or the .dbx container it was
	// wrapped into when a password was given.
	Path                                         string
	Positions, Analyses, Comments                int
	Matches, Games, Moves, MoveAnalyses          int
	Collections, Tournaments, Filters, AnkiDecks int
	Skipped                                      int
}

// WholeTenant is the export the daemon offers by default: everything the
// scope holds, every option on.
func WholeTenant(format Format) ExportOptions {
	return ExportOptions{
		Format: format,
		Selection: Selection{
			AllPositions: true, AllCollections: true, AllMatches: true, AllTournaments: true,
		},
		Analysis: true, Comments: true, PlayedMoves: true, FilterLibrary: true, AnkiDecks: true,
	}
}

// SealWatermark builds the signed document an export writes when the
// producer asked for a watermark, or "" when origin is empty. The identity is
// the caller's: the desktop's per-person key, the daemon's own.
func SealWatermark(identity *issuance.Identity, origin, note string) (string, error) {
	origin, note = strings.TrimSpace(origin), strings.TrimSpace(note)
	if origin == "" {
		return "", nil
	}
	env, err := issuance.Seal(identity, issuance.Watermark{Origin: origin, Note: note})
	if err != nil {
		return "", err
	}
	return issuance.EncodeEnvelope(env)
}

// SQLiteExporter serializes a selection of one scope into a fresh, valid
// blunderDB SQLite file and streams its bytes to w. It is ExportSQLite behind
// the Exporter interface, for the daemon; opts.Format is ignored.
type SQLiteExporter struct{ S storage.Storage }

func (e SQLiteExporter) Export(ctx context.Context, scope string, w io.Writer, opts ExportOptions) error {
	tmp, err := os.CreateTemp("", "blunderdb-export-*.sqlite")
	if err != nil {
		return fmt.Errorf("ingest: temp file: %w", err)
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)

	report, err := ExportSQLite(ctx, e.S, scope, tmpPath, opts)
	if err != nil {
		return err
	}
	if report.Path != tmpPath {
		defer os.Remove(report.Path)
	}
	f, err := os.Open(report.Path)
	if err != nil {
		return fmt.Errorf("ingest: reopen export: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(w, f); err != nil {
		return fmt.Errorf("ingest: stream export: %w", err)
	}
	return nil
}

// ExportSQLite writes the selected part of scope into a new blunderDB
// database at path, replacing any file there. With a password the database is
// built next to path and wrapped into issuance.ProtectedPath(path); the
// unprotected intermediate is removed whether or not wrapping succeeds.
//
// Metadata travels by allow-list (issuance.Carried) and nothing else of the
// source's metadata does; the watermark, when given, is written verbatim (the
// signature is over those bytes); dateOfCreation defaults to today.
func ExportSQLite(ctx context.Context, src storage.Storage, scope, path string, opts ExportOptions) (ExportReport, error) {
	if err := ctx.Err(); err != nil {
		return ExportReport{}, err
	}
	finalPath := path
	if opts.Password != "" {
		// A protected export is named .dbx even when the caller asked for
		// .db: a file whose name says "database" but whose contents are
		// encrypted confuses every other tool.
		finalPath = issuance.ProtectedPath(path)
		path = finalPath + ".plain"
		defer func() {
			if _, statErr := os.Stat(path); statErr == nil {
				if rmErr := os.Remove(path); rmErr != nil {
					slog.Warn("removing the intermediate export", "path", path, "err", rmErr)
				}
			}
		}()
	}

	report, err := writeExport(ctx, src, scope, path, opts)
	if err != nil {
		return report, err
	}
	report.Path = finalPath

	if opts.Password != "" {
		env, err := issuance.DecodeEnvelope(opts.Watermark)
		if err != nil {
			return report, err
		}
		if err := issuance.WrapContainer(path, finalPath, env, opts.Password); err != nil {
			return report, err
		}
		slog.Info("protected the exported database", "path", finalPath)
	}
	if report.Skipped > 0 {
		slog.Warn("export completed with some rows skipped", "skipped", report.Skipped, "path", finalPath)
	}
	return report, nil
}

// writeExport builds the plain database at path.
func writeExport(ctx context.Context, src storage.Storage, scope, path string, opts ExportOptions) (ExportReport, error) {
	db, err := openExportTarget(ctx, path)
	if err != nil {
		return ExportReport{}, err
	}
	defer db.Close()

	// One transaction for the whole file: an export is all-or-nothing, and
	// autocommitting tens of thousands of rows one by one is what once made
	// large exports appear to hang.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return ExportReport{}, fmt.Errorf("ingest: begin export: %w", err)
	}
	defer tx.Rollback()

	e := &exporter{
		ctx: ctx, src: src, scope: scope, tx: tx, dst: sqlite.WrapTx(tx), opts: opts,
		posMap: make(map[int64]int64), written: make(map[int64]bool),
		collMap: make(map[int64]int64), tourMap: make(map[int64]int64), matchMap: make(map[int64]int64),
	}
	for _, step := range []func() error{
		e.writeMetadata,
		e.resolveSelection,
		e.writePositions,
		e.writeCollections,
		e.writeTournaments,
		e.writeMatches,
		e.writeFilters,
		e.writeAnkiDecks,
	} {
		if err := step(); err != nil {
			return e.report, err
		}
	}
	if err := tx.Commit(); err != nil {
		return e.report, fmt.Errorf("ingest: commit export: %w", err)
	}
	// The last pages may not have been flushed while the handle is open, and
	// on Windows a file with an open handle cannot be read wholesale.
	if err := db.Close(); err != nil {
		return e.report, fmt.Errorf("ingest: finalise export: %w", err)
	}
	slog.Info("exported positions", "count", e.report.Positions, "path", path)
	return e.report, nil
}

// openExportTarget creates the SQLite file at path from scratch, on the
// current live schema (sqlite.Bootstrap). The handle is pinned to a single
// connection and configured with durability-relaxing PRAGMAs: the target is a
// throwaway file, rebuilt from scratch on any error, so mid-build durability
// buys nothing and an fsync per row is what made large exports crawl.
func openExportTarget(ctx context.Context, path string) (*sql.DB, error) {
	if _, err := os.Stat(path); err == nil {
		if err := os.Remove(path); err != nil {
			return nil, fmt.Errorf("ingest: cannot replace the existing export file: %w", err)
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("ingest: open export target: %w", err)
	}
	db.SetMaxOpenConns(1)
	for _, pragma := range []string{
		"PRAGMA journal_mode=OFF",
		"PRAGMA synchronous=OFF",
		"PRAGMA temp_store=MEMORY",
	} {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("ingest: configure export target: %w", err)
		}
	}
	if err := sqlite.Bootstrap(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("ingest: create export schema: %w", err)
	}
	return db, nil
}

// exporter carries one export's state: the resolved selection and the maps
// from source ids to the ids the new file assigned.
type exporter struct {
	ctx   context.Context
	src   storage.Storage
	scope string
	tx    *sql.Tx
	dst   storage.Tx // the same transaction, seen through the SQLite backend's writers
	opts  ExportOptions

	// Resolved selection, in the order rows are written.
	collectionIDs, tournamentIDs, matchIDs []int64
	extraPositionIDs                       []int64 // closure of collections/matches, beyond the explicit positions

	posMap   map[int64]int64 // source position id → exported id
	written  map[int64]bool  // exported position ids whose analysis/comments are done
	collMap  map[int64]int64
	tourMap  map[int64]int64
	matchMap map[int64]int64

	report ExportReport
}

func (e *exporter) skip(msg string, args ...any) {
	slog.Warn(msg, args...)
	e.report.Skipped++
}

// writeMetadata copies metadata by ALLOW-LIST, never by exclusion: an
// exported file is handed to someone else, and a document added to
// `metadata` next year must not travel by default. See ADR-0007.
// database_version is not written here: sqlite.Bootstrap stamped it.
func (e *exporter) writeMetadata() error {
	put := func(key, value string) error {
		_, err := e.tx.ExecContext(e.ctx, `INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`, key, value)
		if err != nil {
			return fmt.Errorf("ingest: write metadata %q: %w", key, err)
		}
		return nil
	}
	for key, value := range issuance.Carried(e.opts.Metadata) {
		if err := put(key, value); err != nil {
			return err
		}
	}
	if e.opts.Watermark != "" {
		if err := put(issuance.KeyWatermark, e.opts.Watermark); err != nil {
			return fmt.Errorf("cannot write the watermark into the exported file: %w", err)
		}
	}
	if e.opts.Metadata["dateOfCreation"] == "" {
		if err := put("dateOfCreation", time.Now().Format("2006-01-02")); err != nil {
			return err
		}
	}
	return nil
}

// resolveSelection turns the Selection into ordered id lists, following the
// closure flags: tournaments → matches → positions, collections → positions.
// Every source iterator is drained before the next one starts — nesting two
// would hold two pooled connections at once, which deadlocks a
// single-connection source.
func (e *exporter) resolveSelection() error {
	sel := e.opts.Selection
	var err error

	if e.tournamentIDs, err = e.resolveIDs(sel.AllTournaments, sel.TournamentIDs, func(yield func(int64)) error {
		for t, err := range e.src.Tournaments().List(e.ctx, e.scope) {
			if err != nil {
				return err
			}
			yield(t.ID)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("ingest: list tournaments: %w", err)
	}

	if e.matchIDs, err = e.resolveIDs(sel.AllMatches, sel.MatchIDs, func(yield func(int64)) error {
		for m, err := range e.src.Matches().List(e.ctx, e.scope, storage.MatchListOpts{}) {
			if err != nil {
				return err
			}
			yield(m.ID)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("ingest: list matches: %w", err)
	}
	if sel.TournamentMatches && !sel.AllMatches {
		seen := toSet(e.matchIDs)
		for _, tid := range e.tournamentIDs {
			var ids []int64
			for m, err := range e.src.Tournaments().Matches(e.ctx, e.scope, tid) {
				if err != nil {
					return fmt.Errorf("ingest: list matches of tournament %d: %w", tid, err)
				}
				ids = append(ids, m.ID)
			}
			for _, id := range ids {
				if !seen[id] {
					seen[id] = true
					e.matchIDs = append(e.matchIDs, id)
				}
			}
		}
	}

	if e.collectionIDs, err = e.resolveIDs(sel.AllCollections, sel.CollectionIDs, func(yield func(int64)) error {
		for c, err := range e.src.Collections().List(e.ctx, e.scope) {
			if err != nil {
				return err
			}
			yield(c.ID)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("ingest: list collections: %w", err)
	}

	// Positions reached by the closure, minus the ones the caller listed;
	// sorted so the file is the same whatever order the closure found them.
	if sel.AllPositions {
		return nil
	}
	explicit := toSet(sel.PositionIDs)
	for _, p := range sel.Positions {
		explicit[p.ID] = true
	}
	extra := make(map[int64]bool)
	if sel.CollectionPositions {
		for _, cid := range e.collectionIDs {
			for p, err := range e.src.Collections().Positions(e.ctx, e.scope, cid) {
				if err != nil {
					return fmt.Errorf("ingest: list positions of collection %d: %w", cid, err)
				}
				if !explicit[p.ID] {
					extra[p.ID] = true
				}
			}
		}
	}
	if sel.MatchPositions {
		for _, mid := range e.matchIDs {
			for mv, err := range e.src.Matches().MovesByMatch(e.ctx, e.scope, mid) {
				if err != nil {
					return fmt.Errorf("ingest: list moves of match %d: %w", mid, err)
				}
				if mv.PositionID != 0 && !explicit[mv.PositionID] {
					extra[mv.PositionID] = true
				}
			}
		}
	}
	for id := range extra {
		e.extraPositionIDs = append(e.extraPositionIDs, id)
	}
	slices.Sort(e.extraPositionIDs)
	return nil
}

// resolveIDs returns every id of a family when all is set, else the listed
// ones, deduplicated in order.
func (e *exporter) resolveIDs(all bool, listed []int64, each func(yield func(int64)) error) ([]int64, error) {
	var ids []int64
	if all {
		if err := each(func(id int64) { ids = append(ids, id) }); err != nil {
			return nil, err
		}
		return ids, nil
	}
	seen := make(map[int64]bool, len(listed))
	for _, id := range listed {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func toSet(ids []int64) map[int64]bool {
	out := make(map[int64]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out
}

// positionBatch bounds the memory an export holds at once: analyses are the
// bulk of a database, and a whole one would be hundreds of megabytes.
const positionBatch = 1000

// writePositions walks the position set in batches, reading each batch's
// analyses, played moves and comments in one statement per family.
func (e *exporter) writePositions() error {
	sel := e.opts.Selection
	switch {
	case sel.AllPositions:
		for offset := 0; ; offset += positionBatch {
			var page []*domain.Position
			for p, err := range e.src.Positions().List(e.ctx, e.scope, storage.ListOpts{Limit: positionBatch, Offset: offset}) {
				if err != nil {
					return fmt.Errorf("ingest: list positions: %w", err)
				}
				pc := *p
				page = append(page, &pc)
			}
			if err := e.writeBatch(page); err != nil {
				return err
			}
			if len(page) < positionBatch {
				break
			}
		}
	case len(sel.Positions) > 0:
		for start := 0; start < len(sel.Positions); start += positionBatch {
			if err := e.writeBatch(sel.Positions[start:min(start+positionBatch, len(sel.Positions))]); err != nil {
				return err
			}
		}
	default:
		if err := e.writeByIDs(sel.PositionIDs); err != nil {
			return err
		}
	}
	return e.writeByIDs(e.extraPositionIDs)
}

func (e *exporter) writeByIDs(ids []int64) error {
	for start := 0; start < len(ids); start += positionBatch {
		loaded, err := e.src.Positions().LoadByIDs(e.ctx, e.scope, ids[start:min(start+positionBatch, len(ids))])
		if err != nil {
			return fmt.Errorf("cannot read the positions to export: %w", err)
		}
		batch := make([]*domain.Position, len(loaded))
		for i := range loaded {
			batch[i] = &loaded[i]
		}
		if err := e.writeBatch(batch); err != nil {
			return err
		}
	}
	return nil
}

func (e *exporter) writeBatch(positions []*domain.Position) error {
	if err := e.ctx.Err(); err != nil {
		return err
	}
	ids := make([]int64, 0, len(positions))
	for _, p := range positions {
		if p.ID > 0 {
			ids = append(ids, p.ID)
		}
	}
	var analyses map[int64]*domain.PositionAnalysis
	var moves map[int64][]*domain.Move
	var comments map[int64][]*domain.CommentEntry
	var err error
	if e.opts.Analysis {
		if analyses, err = e.src.Analyses().LoadMany(e.ctx, e.scope, ids); err != nil {
			return fmt.Errorf("cannot read analyses to export: %w", err)
		}
		if e.opts.PlayedMoves {
			if moves, err = e.src.Matches().MovesByPositions(e.ctx, e.scope, ids); err != nil {
				return fmt.Errorf("cannot read played moves to export: %w", err)
			}
		}
	}
	if e.opts.Comments {
		if comments, err = e.src.Comments().ByPositions(e.ctx, e.scope, ids); err != nil {
			return fmt.Errorf("cannot read comments to export: %w", err)
		}
	}

	for _, p := range positions {
		if _, done := e.posMap[p.ID]; done && p.ID > 0 {
			continue // listed twice
		}
		// Save dedups by Zobrist hash like a live database: a second position
		// normalising to the same board maps onto the row already written.
		pc := *p
		pc.ID = 0
		newID, err := e.dst.Positions().Save(e.ctx, "", &pc)
		if err != nil {
			e.skip("inserting position into export database", "positionID", p.ID, "err", err)
			continue
		}
		if p.ID > 0 {
			e.posMap[p.ID] = newID
		}
		if e.written[newID] {
			continue // the row already carries its analysis and comments
		}
		e.written[newID] = true
		e.report.Positions++

		if a := analyses[p.ID]; a != nil {
			ac := *a
			if e.opts.PlayedMoves {
				foldPlayedMoves(&ac, moves[p.ID])
			} else {
				ac.PlayedMove, ac.PlayedCubeAction = "", ""
				ac.PlayedMoves, ac.PlayedCubeActions = nil, nil
			}
			if err := sqlite.SaveAnalysisUncompressed(e.ctx, e.tx, newID, &ac); err != nil {
				e.skip("inserting analysis for position", "newID", newID, "oldID", p.ID, "err", err)
			} else {
				e.report.Analyses++
			}
		}
		for _, c := range comments[p.ID] {
			if _, err := e.tx.ExecContext(e.ctx,
				`INSERT INTO comment (position_id, text, created_at, modified_at)
				 VALUES (?, ?, COALESCE(NULLIF(?, ''), CURRENT_TIMESTAMP), NULLIF(?, ''))`,
				newID, c.Text, c.CreatedAt, c.ModifiedAt); err != nil {
				e.skip("inserting comment for position", "newID", newID, "oldID", p.ID, "err", err)
				continue
			}
			e.report.Comments++
		}
	}
	return nil
}

// foldPlayedMoves folds what the move table records against a position into
// the analysis's own played-move lists, deduplicated and sorted, so the
// exported analysis says everything that was ever played there. Checker moves
// are normalised (mergePlayedMoves); cube actions are kept as written.
func foldPlayedMoves(a *domain.PositionAnalysis, moves []*domain.Move) {
	var playedMoves, cubeActions []string
	if a.PlayedMove != "" {
		playedMoves = append(playedMoves, a.PlayedMove)
	}
	if a.PlayedCubeAction != "" {
		cubeActions = append(cubeActions, a.PlayedCubeAction)
	}
	for _, mv := range moves {
		if mv.CheckerMove != "" {
			playedMoves = append(playedMoves, mv.CheckerMove)
		}
		if mv.CubeAction != "" {
			cubeActions = append(cubeActions, mv.CubeAction)
		}
	}
	a.PlayedMoves = mergePlayedMoves(a.PlayedMoves, playedMoves)
	set := make(map[string]bool)
	for _, c := range append(a.PlayedCubeActions, cubeActions...) {
		if c != "" {
			set[c] = true
		}
	}
	a.PlayedCubeActions = make([]string, 0, len(set))
	for c := range set {
		a.PlayedCubeActions = append(a.PlayedCubeActions, c)
	}
	sort.Strings(a.PlayedCubeActions)
}

// writeCollections copies the selected collections and, for each, the
// memberships whose position was exported, in the collection's order.
func (e *exporter) writeCollections() error {
	for _, id := range e.collectionIDs {
		c, err := e.src.Collections().Get(e.ctx, e.scope, id)
		if err != nil {
			e.skip("reading collection", "collectionID", id, "err", err)
			continue
		}
		res, err := e.tx.ExecContext(e.ctx,
			`INSERT INTO collection (name, description, sort_order, created_at, updated_at)
			 VALUES (?, ?, ?, COALESCE(NULLIF(?, ''), CURRENT_TIMESTAMP), COALESCE(NULLIF(?, ''), CURRENT_TIMESTAMP))`,
			c.Name, c.Description, c.SortOrder, c.CreatedAt, c.UpdatedAt)
		if err != nil {
			e.skip("inserting collection", "collectionID", id, "err", err)
			continue
		}
		newID, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("ingest: collection id: %w", err)
		}
		e.collMap[id] = newID
		e.report.Collections++

		var members []int64
		for p, err := range e.src.Collections().Positions(e.ctx, e.scope, id) {
			if err != nil {
				return fmt.Errorf("ingest: list positions of collection %d: %w", id, err)
			}
			if dstID, ok := e.posMap[p.ID]; ok {
				members = append(members, dstID)
			}
		}
		for i, pid := range members {
			if _, err := e.tx.ExecContext(e.ctx,
				`INSERT OR IGNORE INTO collection_position (collection_id, position_id, sort_order) VALUES (?, ?, ?)`,
				newID, pid, i); err != nil {
				e.skip("inserting collection membership", "collectionID", id, "err", err)
			}
		}
	}
	if len(e.collectionIDs) > 0 {
		slog.Info("exported collections", "collections", e.report.Collections)
	}
	return nil
}

func (e *exporter) writeTournaments() error {
	for _, id := range e.tournamentIDs {
		t, err := e.src.Tournaments().Get(e.ctx, e.scope, id)
		if err != nil {
			e.skip("reading tournament", "tournamentID", id, "err", err)
			continue
		}
		res, err := e.tx.ExecContext(e.ctx,
			`INSERT INTO tournament (name, date, location, sort_order, comment, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, COALESCE(NULLIF(?, ''), CURRENT_TIMESTAMP), COALESCE(NULLIF(?, ''), CURRENT_TIMESTAMP))`,
			t.Name, t.Date, t.Location, t.SortOrder, t.Comment, t.CreatedAt, t.UpdatedAt)
		if err != nil {
			e.skip("inserting tournament", "tournamentID", id, "err", err)
			continue
		}
		newID, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("ingest: tournament id: %w", err)
		}
		e.tourMap[id] = newID
		e.report.Tournaments++
	}
	if len(e.tournamentIDs) > 0 {
		slog.Info("exported tournaments", "count", e.report.Tournaments)
	}
	return nil
}

// writeMatches copies each selected match with its games, moves and move
// analyses. A move whose position was not exported keeps everything but the
// position; a match whose tournament was not exported keeps everything but
// the link.
func (e *exporter) writeMatches() error {
	for _, id := range e.matchIDs {
		if err := e.ctx.Err(); err != nil {
			return err
		}
		m, err := e.src.Matches().Get(e.ctx, e.scope, id)
		if err != nil {
			e.skip("reading match", "matchID", id, "err", err)
			continue
		}
		var tournamentID any
		if m.TournamentID != nil {
			if newID, ok := e.tourMap[*m.TournamentID]; ok {
				tournamentID = newID
			}
		}
		res, err := e.tx.ExecContext(e.ctx,
			`INSERT INTO match (player1_name, player2_name, event, location, round, match_length,
			    match_date, import_date, file_path, game_count, match_hash, canonical_hash,
			    tournament_id, tournament_sort_order, last_visited_position, comment)
			 VALUES (?, ?, ?, ?, ?, ?, ?, COALESCE(?, CURRENT_TIMESTAMP), ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?, ?)`,
			m.Player1Name, m.Player2Name, m.Event, m.Location, m.Round, m.MatchLength,
			nullableTime(m.MatchDate), nullableTime(m.ImportDate), m.FilePath, m.GameCount,
			m.MatchHash, m.CanonicalHash,
			tournamentID, m.TournamentSortOrder, m.LastVisitedPosition, m.Comment)
		if err != nil {
			e.skip("inserting match", "matchID", id, "err", err)
			continue
		}
		newMatchID, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("ingest: match id: %w", err)
		}
		e.matchMap[id] = newMatchID
		e.report.Matches++

		var games []*domain.Game
		for g, err := range e.src.Matches().Games(e.ctx, e.scope, id) {
			if err != nil {
				return fmt.Errorf("ingest: list games of match %d: %w", id, err)
			}
			gc := *g
			games = append(games, &gc)
		}
		gameMap := make(map[int64]int64, len(games))
		for _, g := range games {
			gc := *g
			gc.MatchID = newMatchID
			newID, err := e.dst.Matches().CreateGame(e.ctx, "", &gc)
			if err != nil {
				e.skip("inserting game", "matchID", id, "gameID", g.ID, "err", err)
				continue
			}
			gameMap[g.ID] = newID
			e.report.Games++
		}

		var moves []*domain.Move
		for mv, err := range e.src.Matches().MovesByMatch(e.ctx, e.scope, id) {
			if err != nil {
				return fmt.Errorf("ingest: list moves of match %d: %w", id, err)
			}
			mc := *mv
			moves = append(moves, &mc)
		}
		moveMap := make(map[int64]int64, len(moves))
		for _, mv := range moves {
			newGameID, ok := gameMap[mv.GameID]
			if !ok {
				continue // its game was skipped
			}
			mc := *mv
			mc.GameID = newGameID
			mc.PositionID = e.posMap[mv.PositionID] // 0 → NULL when not exported
			newID, err := e.dst.Matches().CreateMove(e.ctx, "", &mc)
			if err != nil {
				e.skip("inserting move", "matchID", id, "moveID", mv.ID, "err", err)
				continue
			}
			moveMap[mv.ID] = newID
			e.report.Moves++
		}

		var analyses []*domain.MoveAnalysis
		for ma, err := range e.src.Matches().MoveAnalysesByMatch(e.ctx, e.scope, id) {
			if err != nil {
				return fmt.Errorf("ingest: list move analyses of match %d: %w", id, err)
			}
			ac := *ma
			analyses = append(analyses, &ac)
		}
		for _, ma := range analyses {
			newMoveID, ok := moveMap[ma.MoveID]
			if !ok {
				continue
			}
			ac := *ma
			ac.MoveID = newMoveID
			if _, err := e.dst.Matches().CreateMoveAnalysis(e.ctx, "", &ac); err != nil {
				e.skip("inserting move analysis", "matchID", id, "moveID", ma.MoveID, "err", err)
				continue
			}
			e.report.MoveAnalyses++
		}
	}
	if len(e.matchIDs) > 0 {
		slog.Info("exported matches", "matches", e.report.Matches, "games", e.report.Games,
			"moves", e.report.Moves, "moveAnalyses", e.report.MoveAnalyses)
	}
	return nil
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

// writeFilters copies the saved-filter library with each filter's edit
// position.
func (e *exporter) writeFilters() error {
	if !e.opts.FilterLibrary {
		return nil
	}
	var filters []*storage.Filter
	for f, err := range e.src.Filters().List(e.ctx, e.scope) {
		if err != nil {
			return fmt.Errorf("ingest: list filters: %w", err)
		}
		fc := *f
		filters = append(filters, &fc)
	}
	for _, f := range filters {
		editPosition, err := e.src.Filters().LoadEditPosition(e.ctx, e.scope, f.Name)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return fmt.Errorf("ingest: load edit position of filter %q: %w", f.Name, err)
		}
		if _, err := e.tx.ExecContext(e.ctx,
			`INSERT INTO filter_library (name, command, edit_position) VALUES (?, ?, ?)`,
			f.Name, f.Command, editPosition); err != nil {
			e.skip("inserting filter library entry", "name", f.Name, "err", err)
			continue
		}
		e.report.Filters++
	}
	return nil
}

// writeAnkiDecks rebuilds each deck from its exported member positions. The
// review history is intentionally left behind: an export is a fresh study
// copy, not a scheduler snapshot.
func (e *exporter) writeAnkiDecks() error {
	if !e.opts.AnkiDecks {
		return nil
	}
	type srcDeck struct {
		d      domain.AnkiDeck
		posIDs []int64
	}
	var decks []srcDeck
	for d, err := range e.src.Anki().ListDecks(e.ctx, e.scope) {
		if err != nil {
			return fmt.Errorf("ingest: list decks: %w", err)
		}
		decks = append(decks, srcDeck{d: *d})
	}
	for i := range decks {
		for p, err := range e.src.Anki().DeckPositions(e.ctx, e.scope, decks[i].d.ID) {
			if err != nil {
				return fmt.Errorf("ingest: list positions of deck %d: %w", decks[i].d.ID, err)
			}
			if dstID, ok := e.posMap[p.ID]; ok {
				decks[i].posIDs = append(decks[i].posIDs, dstID)
			}
		}
	}
	for _, sd := range decks {
		sourceID := sd.d.SourceID
		sourceCmd := sd.d.SourceCommand
		switch sd.d.SourceType {
		case "collection":
			sourceID = e.collMap[sd.d.SourceID] // 0 if the collection was not exported
		case "search":
			sourceCmd = remapIDList(sourceCmd, e.posMap)
		}
		newID, err := e.dst.Anki().CreateDeck(e.ctx, "", sd.d.Name, sd.d.Description, sd.d.SourceType, sourceID, sourceCmd)
		if err != nil {
			return fmt.Errorf("ingest: create deck %q: %w", sd.d.Name, err)
		}
		if len(sd.posIDs) > 0 {
			if err := e.dst.Anki().SyncWithPositions(e.ctx, "", newID, sd.posIDs); err != nil {
				return fmt.Errorf("ingest: fill deck %q: %w", sd.d.Name, err)
			}
		}
		if err := e.dst.Anki().UpdateDeckParams(e.ctx, "", newID, sd.d.RequestRetention, sd.d.MaximumInterval, sd.d.EnableFuzz, sd.d.SessionLimit); err != nil {
			return fmt.Errorf("ingest: deck %q parameters: %w", sd.d.Name, err)
		}
		e.report.AnkiDecks++
	}
	return nil
}

// remapIDList rewrites a comma-separated position-id list from source ids to
// dest ids, dropping any id without a mapping.
func remapIDList(csv string, m map[int64]int64) string {
	var out []string
	for _, p := range strings.Split(csv, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			continue
		}
		if dstID, ok := m[n]; ok {
			out = append(out, strconv.FormatInt(dstID, 10))
		}
	}
	return strings.Join(out, ",")
}
