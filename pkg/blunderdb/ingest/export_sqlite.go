package ingest

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
	"github.com/kevung/blunderdb/pkg/blunderdb/mets"
	"github.com/kevung/blunderdb/pkg/blunderdb/searchquery"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/searchfilter"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// This file is THE export of a blunderDB database to a file: the GUI's
// export dialog, the CLI's `export`, and the daemon's exports.sqlite all
// run ExportSQLite with a Selection that says what leaves and an
// ExportOptions that says how. Keep it the only walk: separate copies drifted
// apart in what they carried.
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

	AllLessons bool
	LessonIDs  []int64
	// LessonContents adds what the selected lessons' steps show: their
	// collections, with their members, and their positions (ADR-0066).
	LessonContents bool

	// DeckIDs exports these Anki decks even when ExportOptions.AnkiDecks is
	// off — a deck handed to someone else, without the sender's other decks.
	// It travels without its review history unless ExportOptions.AnkiHistory.
	DeckIDs []int64
	// DeckPositions adds the members of the decks in DeckIDs to the positions
	// exported.
	DeckPositions bool
}

// ExportReport counts what an export wrote. Skipped counts rows dropped along
// the way — a collection id that names nothing, a position that would not
// insert — each already logged where it happened; the export itself succeeds,
// because one stale identifier is not a reason to lose the rest.
type ExportReport struct {
	// Path is where the file landed: opts' path, or the .dbx container it was
	// wrapped into when a password was given.
	Path           string `json:"path"`
	Positions      int    `json:"positions"`
	Analyses       int    `json:"analyses"`
	Comments       int    `json:"comments"`
	Matches        int    `json:"matches"`
	Games          int    `json:"games"`
	Moves          int    `json:"moves"`
	MoveAnalyses   int    `json:"moveAnalyses"`
	Collections    int    `json:"collections"`
	Tournaments    int    `json:"tournaments"`
	Filters        int    `json:"filters"`
	AnkiDecks      int    `json:"ankiDecks"`
	Transcriptions int    `json:"transcriptions"`
	Lessons        int    `json:"lessons"`
	Aliases        int    `json:"aliases"`
	Skipped        int    `json:"skipped"`
	// DegradedAnkiDecks counts the search decks that left with part of their
	// source dropped (searchSourceLost): their members travel, but their
	// command, payload or ids named rows the file does not hold.
	DegradedAnkiDecks int `json:"degradedAnkiDecks"`
	// Directions counts the tournament Directions that left with their
	// tournament (writeDirections).
	Directions int `json:"directions"`
}

// WholeTenant is the export the daemon offers by default: everything the
// scope holds, every option on.
func WholeTenant(format Format) ExportOptions {
	return ExportOptions{
		Format: format,
		Selection: Selection{
			AllPositions: true, AllCollections: true, AllMatches: true, AllTournaments: true,
			AllLessons: true,
		},
		Analysis: true, Comments: true, PlayedMoves: true, FilterLibrary: true, AnkiDecks: true,
		Transcriptions: true, Aliases: true, AnkiHistory: true,
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
		livingEvals: make(map[int64]*storage.CollectionEvaluation),
		addedAt:     make(map[int64]map[int64]string),
	}
	for _, step := range []func() error{
		e.writeMetadata,
		e.resolveSelection,
		e.writePositions,
		e.writeCollections,
		e.writeLessons,
		e.writeTournaments,
		e.writeMatches,
		e.writeDirections,
		e.writeLivingQueries,
		e.writeMatchOrigins,
		e.writeAliases,
		// No Duel in suspense: its seed is every roll to come, and it leaves
		// by no route before the end (ADR-0072 rule 11). A finished Duel
		// travels as its Match and that Match's origin, seed revealed.
		e.writeTranscriptions,
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
	lessonIDs                              []int64
	deckIDs                                []int64
	lessonCollIDs                          []int64 // collections the lessons' steps show
	lessonPosIDs                           []int64 // positions the lessons' steps show
	extraPositionIDs                       []int64 // closure of collections/matches, beyond the explicit positions

	posMap   map[int64]int64 // source position id → exported id
	written  map[int64]bool  // exported position ids whose analysis/comments are done
	collMap  map[int64]int64
	tourMap  map[int64]int64
	matchMap map[int64]int64
	// matchNames holds, per alias kind, the names the exported matches write.
	matchNames map[storage.AliasKind]map[string]bool
	// living holds the LIVING collections written without their query, and
	// livingEvals each one's evaluation at the source, read once.
	living      []livingCollection
	livingEvals map[int64]*storage.CollectionEvaluation
	// addedAt holds, per source collection, when each of its rows was added:
	// a membership keeps its date in the file, as the collection keeps its own.
	addedAt map[int64]map[int64]string
	tables  *mets.Carrier // the tables the exported analyses cite, built on first use

	report ExportReport
}

// carrier returns the export's table carrier: the source's tables, copied
// into the file the first time an exported analysis cites one (ADR-0068,
// rule 5). They arrive without their current flag.
func (e *exporter) carrier() (*mets.Carrier, error) {
	if e.tables == nil {
		tables, err := mets.ReadTables(e.ctx, e.src.MatchEquityTables(), e.scope)
		if err != nil {
			return nil, fmt.Errorf("cannot read match equity tables to export: %w", err)
		}
		e.tables = mets.NewCarrier("export", e.dst.MatchEquityTables(), "", tables)
	}
	return e.tables, nil
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
		for t, err := range e.src.Tournaments().List(e.ctx, e.scope, storage.ListOpts{}) {
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

	if err := e.resolveLessons(); err != nil {
		return err
	}
	e.deckIDs, _ = e.resolveIDs(false, sel.DeckIDs, nil)

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
	for _, pid := range e.lessonPosIDs {
		if !explicit[pid] {
			extra[pid] = true
		}
	}
	reach := e.lessonCollIDs
	if sel.CollectionPositions {
		reach = e.collectionIDs
	}
	for _, cid := range reach {
		ids, err := e.collectionMembers(cid)
		if err != nil {
			return err
		}
		for _, pid := range ids {
			if !explicit[pid] {
				extra[pid] = true
			}
		}
	}
	if sel.DeckPositions {
		// DeckPositions reads the scope's own decks only: an id another
		// tenant owns lists nothing.
		for _, did := range e.deckIDs {
			for p, err := range e.src.Anki().DeckPositions(e.ctx, e.scope, did) {
				if err != nil {
					return fmt.Errorf("ingest: list positions of deck %d: %w", did, err)
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
	var analysisMET map[int64]int64
	var moves map[int64][]*domain.Move
	var comments map[int64][]*domain.CommentEntry
	var err error
	if e.opts.Analysis {
		if analyses, err = e.src.Analyses().LoadMany(e.ctx, e.scope, ids); err != nil {
			return fmt.Errorf("cannot read analyses to export: %w", err)
		}
		if analysisMET, err = e.src.MatchEquityTables().OfAnalyses(e.ctx, e.scope, ids); err != nil {
			return fmt.Errorf("cannot read the analyses' match equity tables to export: %w", err)
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
				if err := e.carryTable(analysisMET[p.ID], newID); err != nil {
					return err
				}
			}
		}
		for _, c := range comments[p.ID] {
			cols, marks, args := carriedComment(c)
			if _, err := e.tx.ExecContext(e.ctx,
				`INSERT INTO comment (position_id, `+strings.Join(cols, ", ")+`) VALUES (?, `+strings.Join(marks, ", ")+`)`,
				append([]any{newID}, args...)...); err != nil {
				e.skip("inserting comment for position", "newID", newID, "oldID", p.ID, "err", err)
				continue
			}
			e.report.Comments++
		}
	}
	return nil
}

// carryTable records that the exported analysis of newID was valued with the
// source table srcMET. A failure fails the export: an analysis written without
// its table would be read as Kazaross-XG2 by whoever opens the file.
func (e *exporter) carryTable(srcMET, newID int64) error {
	if srcMET == 0 {
		return nil
	}
	c, err := e.carrier()
	if err != nil {
		return err
	}
	met, err := c.Target(e.ctx, srcMET)
	if err != nil {
		return fmt.Errorf("cannot export a match equity table: %w", err)
	}
	if met == 0 {
		return nil
	}
	if err := e.dst.MatchEquityTables().TagAnalyses(e.ctx, "", met, []int64{newID}); err != nil {
		return fmt.Errorf("cannot export an analysis's match equity table: %w", err)
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

// collectionMembers lists the positions a collection shows: the rows of a
// hand-made list, or what the query of a LIVING one selects now, so that the
// recipient's evaluation of the copied query finds them in the file.
func (e *exporter) collectionMembers(cid int64) ([]int64, error) {
	c, err := e.src.Collections().Get(e.ctx, e.scope, cid)
	if err != nil {
		return nil, fmt.Errorf("ingest: read collection %d: %w", cid, err)
	}
	if strings.TrimSpace(c.FilterQuery) != "" {
		return e.evaluateLiving(cid).PositionIDs, nil
	}
	ids, err := e.src.Collections().PositionIDs(e.ctx, e.scope, cid, storage.ListOpts{})
	if err != nil {
		return nil, fmt.Errorf("ingest: list positions of collection %d: %w", cid, err)
	}
	return ids, nil
}

// evaluateLiving evaluates a LIVING collection once per export, under the
// ceiling every evaluation obeys (storage.LivingCollectionCap). A query the
// source cannot evaluate selects nothing here, as it opens nothing there.
func (e *exporter) evaluateLiving(cid int64) *storage.CollectionEvaluation {
	if ev, ok := e.livingEvals[cid]; ok {
		return ev
	}
	ev, err := storage.EvaluateCollection(e.ctx, e.src, e.scope, cid, 0)
	if err != nil {
		e.skip("evaluating living collection", "collectionID", cid, "err", err)
		ev = &storage.CollectionEvaluation{CollectionID: cid}
	}
	if ev.Truncated {
		slog.Warn("export: living collection holds more positions than an evaluation reads; the first ones travel",
			"collectionID", cid, "total", ev.Total, "cap", ev.Cap)
	}
	e.livingEvals[cid] = ev
	return ev
}

// writeCollections copies the selected collections and, for each, the
// memberships whose position was exported, in the collection's order. A
// LIVING collection is settled later, by writeLivingQueries: its query may
// name matches and tournaments, which are numbered after it.
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

		var hand []int64
		added := map[int64]string{}
		for m, err := range e.src.Collections().Members(e.ctx, e.scope, id) {
			if err != nil {
				return fmt.Errorf("ingest: list positions of collection %d: %w", id, err)
			}
			hand = append(hand, m.PositionID)
			added[m.PositionID] = m.AddedAt
		}
		e.addedAt[id] = added
		if query := strings.TrimSpace(c.FilterQuery); query != "" {
			e.living = append(e.living, livingCollection{src: id, dst: newID, query: query, hand: hand})
			continue
		}
		e.writeMembers(id, newID, hand)
	}
	if len(e.collectionIDs) > 0 {
		slog.Info("exported collections", "collections", e.report.Collections)
	}
	return nil
}

// livingCollection is a LIVING collection written without its query yet.
type livingCollection struct {
	src, dst int64
	query    string
	hand     []int64 // the rows it kept from before it was living
}

// writeMembers writes the memberships of an exported collection, in the
// order given, for the positions the file holds.
func (e *exporter) writeMembers(srcID, dstID int64, positions []int64) {
	order := 0
	for _, pid := range positions {
		to, ok := e.posMap[pid]
		if !ok {
			continue
		}
		if _, err := e.tx.ExecContext(e.ctx,
			`INSERT OR IGNORE INTO collection_position (collection_id, position_id, sort_order, added_at)
			 VALUES (?, ?, ?, COALESCE(NULLIF(?, ''), CURRENT_TIMESTAMP))`,
			dstID, to, order, e.addedAt[srcID][pid]); err != nil {
			e.skip("inserting collection membership", "collectionID", srcID, "err", err)
		}
		order++
	}
}

// writeLivingQueries settles each LIVING collection once positions, matches
// and tournaments have their numbers in the file. A query naming none of them
// travels as written; one naming them travels rewritten onto the file's
// numbers. When one of those has no number in the file, the query cannot say
// in the file what it said at the source — dropping the token would widen it,
// keeping it would aim at another row — so the collection leaves FROZEN: what
// its query selects now, as an ordinary list.
func (e *exporter) writeLivingQueries() error {
	for _, lc := range e.living {
		query, ok := carriedLivingQuery(lc.query, e.posMap, e.matchMap, e.tourMap)
		if !ok {
			slog.Warn("export: living collection names rows the file does not hold; it leaves frozen",
				"collectionID", lc.src)
			e.writeMembers(lc.src, lc.dst, e.evaluateLiving(lc.src).PositionIDs)
			continue
		}
		if _, err := e.tx.ExecContext(e.ctx,
			`UPDATE collection SET filter_query = ? WHERE id = ?`, query, lc.dst); err != nil {
			e.skip("writing living collection query", "collectionID", lc.src, "err", err)
			continue
		}
		e.writeMembers(lc.src, lc.dst, lc.hand)
	}
	return nil
}

// carriedLivingQuery rewrites a living query's row ids — `like<n>`, `id<n>`,
// `ma<n>`, `tn<n>` — from the source's numbers to the file's. ok is false
// when the rewritten query could not select in the file what it selects at
// the source, or when the query also holds a token the parser does not read:
// rewriting goes through Parse and Format, which would drop it.
func carriedLivingQuery(query string, positions, matches, tournaments map[int64]int64) (string, bool) {
	f, diags := searchquery.Parse(query)
	if f.LikeTargetID == 0 && f.PositionIDsFilter == "" && f.MatchIDsFilter == "" && f.TournamentIDsFilter == "" {
		return query, true
	}
	for _, d := range diags {
		if d.Kind == searchquery.DiagUnknown {
			return "", false
		}
	}
	if f.LikeTargetID != 0 {
		to, ok := positions[f.LikeTargetID]
		if !ok {
			return "", false
		}
		f.LikeTargetID = to
	}
	// A position the file does not hold is selected by nothing in the file,
	// so its id drops from an `id` list without changing what the list
	// selects there — as long as one id remains, for an empty list selects
	// everything. A match or a tournament left behind may still have
	// positions in the file, which its id selected at the source: it freezes.
	for _, field := range []struct {
		list     *string
		m        map[int64]int64
		dropMiss bool
	}{{&f.PositionIDsFilter, positions, true}, {&f.MatchIDsFilter, matches, false}, {&f.TournamentIDsFilter, tournaments, false}} {
		if *field.list == "" {
			continue
		}
		ids, err := searchfilter.ParseFilterIDList(*field.list)
		if err != nil {
			return "", false
		}
		// An explicit list: a range of source ids is no range of file ids.
		out := make([]string, 0, len(ids))
		for _, id := range ids {
			to, ok := field.m[id]
			if !ok && !field.dropMiss {
				return "", false
			}
			if ok {
				out = append(out, strconv.FormatInt(to, 10))
			}
		}
		if len(out) == 0 {
			return "", false
		}
		*field.list = strings.Join(out, ";")
	}
	return searchquery.Format(f), true
}

// resolveLessons lists the selected lessons and, under LessonContents, adds
// the collections their steps show to the collections exported.
func (e *exporter) resolveLessons() error {
	sel := e.opts.Selection
	var err error
	if e.lessonIDs, err = e.resolveIDs(sel.AllLessons, sel.LessonIDs, func(yield func(int64)) error {
		lessons, err := e.src.Lessons().List(e.ctx, e.scope)
		if err != nil {
			return err
		}
		for _, l := range lessons {
			yield(l.ID)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("ingest: list lessons: %w", err)
	}
	if !sel.LessonContents {
		return nil
	}
	colls := toSet(e.collectionIDs)
	for _, id := range e.lessonIDs {
		l, err := e.src.Lessons().Get(e.ctx, e.scope, id)
		if err != nil {
			e.skip("reading lesson", "lessonID", id, "err", err)
			continue
		}
		for _, st := range l.Steps {
			if st.CollectionID != 0 {
				e.lessonCollIDs = append(e.lessonCollIDs, st.CollectionID)
				if !colls[st.CollectionID] {
					colls[st.CollectionID] = true
					e.collectionIDs = append(e.collectionIDs, st.CollectionID)
				}
			}
			if st.PositionID != 0 {
				e.lessonPosIDs = append(e.lessonPosIDs, st.PositionID)
			}
		}
	}
	return nil
}

// writeLessons copies the selected lessons with their steps. A step keeps
// its text; its collection or position travels only when exported too.
func (e *exporter) writeLessons() error {
	dst := e.dst.Lessons()
	for _, id := range e.lessonIDs {
		l, err := e.src.Lessons().Get(e.ctx, e.scope, id)
		if err != nil {
			e.skip("reading lesson", "lessonID", id, "err", err)
			continue
		}
		newID, err := dst.Create(e.ctx, "", l.Name, l.Description)
		if err != nil {
			e.skip("inserting lesson", "lessonID", id, "err", err)
			continue
		}
		e.report.Lessons++
		for _, st := range l.Steps {
			st.CollectionID = e.collMap[st.CollectionID]
			st.PositionID = e.posMap[st.PositionID]
			if _, err := dst.AddStep(e.ctx, "", newID, st); err != nil {
				return fmt.Errorf("ingest: write step of lesson %d: %w", id, err)
			}
		}
		e.keepDates("lesson", newID, l.CreatedAt, l.UpdatedAt)
	}
	return nil
}

// keepDates gives a row the file just wrote the dates its source row bears:
// the store's writers stamp the moment of the export, which would date a
// lesson, a draft or a deck from the day it was handed over. An empty date
// leaves the stamp in place.
func (e *exporter) keepDates(table string, id int64, created, updated string) {
	if _, err := e.tx.ExecContext(e.ctx,
		`UPDATE `+table+` SET created_at = COALESCE(NULLIF(?, ''), created_at),
		 updated_at = COALESCE(NULLIF(?, ''), updated_at) WHERE id = ?`,
		created, updated, id); err != nil {
		e.skip("keeping dates", "table", table, "id", id, "err", err)
	}
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
// exportedMatchColumns are the match columns an export carries, in the
// order writeMatches binds them: an allow-list, like issuance.Carried for
// the metadata keys (ADR-0007). The match's own note and its author travel
// by issuance.CarriedMatchCommentColumns instead, like a position's
// comments, and are appended after these. A column added to the match table is not
// exported until it is named here — or in notExportedMatchColumns, with the
// reason — and TestExportMatchColumnsClassified fails until one of the
// lists says so. The source metadata (ratings, experience, transcriber, session
// rules, engine version) is what the file said of the match: it travels
// with it (ADR-0067). The source file travels by its name alone
// (issuance.CarriedFileName, ADR-0067 rule 3); a local video by its name, an
// http(s) one as its URL (issuance.CarriedVideoSource, ADR-0082).
var exportedMatchColumns = []string{
	"player1_name", "player2_name", "event", "location", "round", "match_length",
	"match_date", "import_date", "file_path", "game_count", "match_hash", "canonical_hash",
	"tournament_id", "tournament_sort_order", "last_visited_position",
	"player1_elo", "player2_elo", "player1_experience", "player2_experience",
	"transcriber", "has_jacoby", "has_beaver", "engine_version", "video_source",
}

// notExportedMatchColumns are the match columns an export leaves behind, each
// with why.
var notExportedMatchColumns = map[string]string{
	"id":                 "the export numbers its own rows",
	"import_batch_id":    "names an import of the source database, not of the export",
	"dice_hash":          "derived from the games; the recipient computes it again",
	"direction_match_id": "written by writeDirections, and only when the match's tournament Direction leaves with it",
}

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
		noteCols, noteArgs := carriedMatchComment(m)
		res, err := e.tx.ExecContext(e.ctx,
			`INSERT INTO match (`+strings.Join(append(slices.Clone(exportedMatchColumns), noteCols...), ", ")+`)
			 VALUES (?, ?, ?, ?, ?, ?, ?, COALESCE(?, CURRENT_TIMESTAMP), ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?,
			         ?, ?, ?, ?, ?, ?, ?, ?, ?`+strings.Repeat(", ?", len(noteArgs))+`)`,
			append([]any{m.Player1Name, m.Player2Name, m.Event, m.Location, m.Round, m.MatchLength,
				nullableTime(m.MatchDate), nullableTime(m.ImportDate), issuance.CarriedFileName(m.FilePath), m.GameCount,
				m.MatchHash, m.CanonicalHash,
				tournamentID, m.TournamentSortOrder, m.LastVisitedPosition,
				m.Player1Elo, m.Player2Elo, m.Player1Experience, m.Player2Experience,
				m.Transcriber, m.HasJacoby, m.HasBeaver, m.EngineVersion,
				issuance.CarriedVideoSource(m.VideoSource)}, noteArgs...)...)
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
		e.noteMatchNames(m)

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

// noteMatchNames records the player and event names m writes, which decide
// the aliases that leave with it.
func (e *exporter) noteMatchNames(m *domain.Match) {
	if e.matchNames == nil {
		e.matchNames = map[storage.AliasKind]map[string]bool{
			storage.AliasPlayer: {}, storage.AliasEvent: {},
		}
	}
	for kind, names := range map[storage.AliasKind][]string{
		storage.AliasPlayer: {m.Player1Name, m.Player2Name},
		storage.AliasEvent:  {m.Event},
	} {
		for _, n := range names {
			if n = strings.TrimSpace(n); n != "" {
				e.matchNames[kind][n] = true
			}
		}
	}
}

// writeAliases copies, under opts.Aliases, the player and event aliases
// whose spelling or canonical name an exported match writes: the recipient's
// stats then read a merged player as one, as the producer's do. An alias
// that touches no exported match stays home — it names someone the file
// says nothing about.
func (e *exporter) writeAliases() error {
	if !e.opts.Aliases || len(e.matchNames) == 0 {
		return nil
	}
	for _, kind := range []storage.AliasKind{storage.AliasPlayer, storage.AliasEvent} {
		aliases, err := e.src.Aliases().List(e.ctx, e.scope, kind)
		if err != nil {
			return fmt.Errorf("ingest: list %s aliases: %w", kind, err)
		}
		names := e.matchNames[kind]
		for _, a := range aliases {
			if !names[a.Alias] && !names[a.Canonical] {
				continue
			}
			if err := e.dst.Aliases().Set(e.ctx, "", kind, a.Alias, a.Canonical); err != nil {
				e.skip("writing alias", "kind", kind, "alias", a.Alias, "err", err)
				continue
			}
			if a.CreatedAt != "" {
				if _, err := e.tx.ExecContext(e.ctx,
					`UPDATE `+string(kind)+`_alias SET created_at = ? WHERE alias = ?`, a.CreatedAt, a.Alias); err != nil {
					e.skip("keeping alias date", "kind", kind, "alias", a.Alias, "err", err)
				}
			}
			e.report.Aliases++
		}
	}
	return nil
}

// writeFilters copies the saved-filter library with each filter's edit
// position. A filter that names rows by id — `like<n>`, `id<n>`, `ma<n>`,
// `tn<n>` — has them renumbered to the file's (carriedLivingQuery); one that
// names a row the export leaves behind does not leave, since in the file it
// would select something else or nothing.
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
		command, ok := carriedLivingQuery(f.Command, e.posMap, e.matchMap, e.tourMap)
		if !ok {
			e.skip("saved filter names a row the export leaves behind", "name", f.Name)
			continue
		}
		if command != f.Command && strings.HasPrefix(strings.TrimSpace(f.Command), "ss ") {
			command = "s" + command // Format writes the plain verb; keep the narrowing one
		}
		editPosition, err := e.src.Filters().LoadEditPosition(e.ctx, e.scope, f.Name)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return fmt.Errorf("ingest: load edit position of filter %q: %w", f.Name, err)
		}
		if _, err := e.tx.ExecContext(e.ctx,
			`INSERT INTO filter_library (name, command, edit_position) VALUES (?, ?, ?)`,
			f.Name, command, editPosition); err != nil {
			e.skip("inserting filter library entry", "name", f.Name, "err", err)
			continue
		}
		e.report.Filters++
	}
	return nil
}

// writeTranscriptions copies the drafts (ADR-0045) under opts.Transcriptions:
// a draft travels with the match it produced, the link going through
// matchMap. A draft that belongs to no exported match — never saved, or its
// match left behind — travels only when every match and every position do,
// and then without a link: a narrower export cannot tell whose it is.
func (e *exporter) writeTranscriptions() error {
	if !e.opts.Transcriptions {
		return nil
	}
	sel := e.opts.Selection
	unattached := sel.AllMatches && sel.AllPositions
	var drafts []*storage.Transcription
	for tr, err := range e.src.Transcriptions().List(e.ctx, e.scope) {
		if err != nil {
			return fmt.Errorf("ingest: list transcriptions: %w", err)
		}
		d := *tr
		drafts = append(drafts, &d)
	}
	for _, d := range drafts {
		match, exported := e.matchMap[d.MatchID]
		if !exported && !unattached {
			continue
		}
		d.ID = 0
		d.MatchID = match // 0 when the match did not export: the typing travels, the link does not
		doc, err := carriedTranscriptDocument(d.Document)
		if err != nil {
			e.skip("reading transcription", "label", d.Label, "err", err)
			continue
		}
		d.Document = doc
		newID, err := e.dst.Transcriptions().Save(e.ctx, "", d)
		if err != nil {
			e.skip("inserting transcription", "label", d.Label, "err", err)
			continue
		}
		e.keepDates("transcription", newID, d.CreatedAt, d.UpdatedAt)
		e.report.Transcriptions++
	}
	return nil
}

// carriedTranscriptDocument is a draft's document as it leaves: its header's
// video source goes through issuance.CarriedTranscriptVideoSource, every other
// field as written. The document is otherwise opaque here, so only that one key
// is rewritten and the rest is passed through unread.
func carriedTranscriptDocument(document string) (string, error) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal([]byte(document), &doc); err != nil {
		return "", err
	}
	var header map[string]json.RawMessage
	if raw, ok := doc["header"]; !ok || json.Unmarshal(raw, &header) != nil {
		return document, nil
	}
	var source string
	if raw, ok := header["video_source"]; !ok || json.Unmarshal(raw, &source) != nil {
		return document, nil
	}
	carried, err := json.Marshal(issuance.CarriedTranscriptVideoSource(source))
	if err != nil {
		return "", err
	}
	header["video_source"] = carried
	if doc["header"], err = json.Marshal(header); err != nil {
		return "", err
	}
	out, err := json.Marshal(doc)
	return string(out), err
}

// writeMatchOrigins copies the origin of every exported Match played here,
// whatever the selection: it is a fact of the Match, and without it the copy
// would write a Start's Match as a .mat and reopen it as a Transcription.
func (e *exporter) writeMatchOrigins() error {
	for oldID, newID := range e.matchMap {
		o, err := e.src.Duels().Origin(e.ctx, e.scope, oldID)
		if errors.Is(err, storage.ErrNotFound) {
			continue
		}
		if err != nil {
			return fmt.Errorf("ingest: origin of match %d: %w", oldID, err)
		}
		o.MatchID = newID
		if err := e.dst.Duels().SetOrigin(e.ctx, "", o); err != nil {
			return fmt.Errorf("ingest: origin of match %d: %w", oldID, err)
		}
	}
	return nil
}

// writeAnkiDecks rebuilds each deck from its exported member positions. By
// default the review history stays behind — an export is a fresh study copy —
// and ExportOptions.AnkiHistory carries it: a backup, or one's own decks moved
// to another machine.
func (e *exporter) writeAnkiDecks() error {
	if !e.opts.AnkiDecks && len(e.deckIDs) == 0 {
		return nil
	}
	picked := toSet(e.deckIDs)
	type srcDeck struct {
		d      domain.AnkiDeck
		posIDs []int64
	}
	var decks []srcDeck
	for d, err := range e.src.Anki().ListDecks(e.ctx, e.scope) {
		if err != nil {
			return fmt.Errorf("ingest: list decks: %w", err)
		}
		if e.opts.AnkiDecks || picked[d.ID] {
			decks = append(decks, srcDeck{d: *d})
		}
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
		case domain.AnkiSourceCollection:
			sourceID = e.collMap[sd.d.SourceID] // 0 if the collection was not exported
		case domain.AnkiSourceSearch:
			sourceCmd = carriedSearchSource(sourceCmd, e.posMap, e.matchMap, e.tourMap)
			if searchSourceLost(sd.d.SourceCommand, sourceCmd) {
				slog.Warn("export: search deck leaves with part of its source dropped", "deck", sd.d.Name)
				e.report.DegradedAnkiDecks++
			}
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
		// A deck of scores holds no position and its cards do not travel:
		// Sync states the 36 score cards here as on any machine (ADR-0042 rule 2).
		if sd.d.SourceType == domain.AnkiSourceScores {
			if err := e.dst.Anki().Sync(e.ctx, "", newID); err != nil {
				return fmt.Errorf("ingest: fill deck %q: %w", sd.d.Name, err)
			}
		}
		if err := e.dst.Anki().UpdateDeckParams(e.ctx, "", newID, sd.d.RequestRetention, sd.d.MaximumInterval, sd.d.EnableFuzz, sd.d.SessionLimit); err != nil {
			return fmt.Errorf("ingest: deck %q parameters: %w", sd.d.Name, err)
		}
		if e.opts.AnkiHistory {
			if err := e.writeAnkiHistory(sd.d.ID, newID); err != nil {
				return fmt.Errorf("ingest: deck %q history: %w", sd.d.Name, err)
			}
		}
		e.keepDates("anki_deck", newID, sd.d.CreatedAt, sd.d.UpdatedAt)
		e.report.AnkiDecks++
	}
	return nil
}

// writeAnkiHistory gives the file's cards of a deck the scheduling state of
// their source cards, and the file the review events of those cards. A card
// is matched on its question — kind and key, a position key renumbered to the
// file's position — since the file's deck was rebuilt from its members; a
// card whose position stayed behind has no counterpart, and its events none
// either.
func (e *exporter) writeAnkiHistory(srcDeck, dstDeck int64) error {
	cardMap := make(map[int64]int64)
	for c, err := range e.src.Anki().DeckCards(e.ctx, e.scope, srcDeck) {
		if err != nil {
			return err
		}
		key, _, ok := e.ankiKey(c.Kind, c.Key, c.PositionID)
		if !ok {
			continue
		}
		var dstID int64
		err := e.tx.QueryRowContext(e.ctx,
			`UPDATE anki_card SET due = ?, stability = ?, difficulty = ?, elapsed_days = ?, scheduled_days = ?,
			 reps = ?, lapses = ?, state = ?, last_review = ?, suspended = ?, buried_until = ?
			 WHERE deck_id = ? AND kind = ? AND key = ? RETURNING id`,
			c.Due, c.Stability, c.Difficulty, c.ElapsedDays, c.ScheduledDays,
			c.Reps, c.Lapses, c.State, c.LastReview, c.Suspended, nullIfEmpty(c.BuriedUntil),
			dstDeck, c.Kind, key).Scan(&dstID)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		cardMap[c.ID] = dstID
	}
	var logs []*domain.AnkiReviewLog
	for l, err := range e.src.Anki().ReviewLog(e.ctx, e.scope, srcDeck, 0) {
		if err != nil {
			return err
		}
		logs = append(logs, l)
	}
	// The source streams most recent first; the file keeps the order of events.
	for i := len(logs) - 1; i >= 0; i-- {
		l := logs[i]
		card, ok := cardMap[l.CardID]
		if !ok {
			continue
		}
		key, posID, ok := e.ankiKey(l.Kind, l.Key, l.PositionID)
		if !ok {
			continue
		}
		var position any
		if posID != 0 {
			position = posID
		}
		if _, err := e.tx.ExecContext(e.ctx,
			`INSERT INTO anki_review_log (card_id, deck_id, kind, key, position_id, rating, state,
			 stability, difficulty, elapsed_days, scheduled_days, reviewed_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			card, dstDeck, l.Kind, key, position, l.Rating, l.State,
			l.Stability, l.Difficulty, l.ElapsedDays, l.ScheduledDays, l.ReviewedAt); err != nil {
			return err
		}
	}
	return nil
}

// ankiKey renumbers a card's question onto the file: a position card names
// the file's position, any other card keeps its key. ok is false for a
// position the file does not hold.
func (e *exporter) ankiKey(kind, key string, positionID int64) (string, int64, bool) {
	if kind != domain.AnkiKindPosition {
		return key, 0, true
	}
	to, ok := e.posMap[positionID]
	if !ok {
		return "", 0, false
	}
	return strconv.FormatInt(to, 10), to, true
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

// carriedMatchComment spells the match's own note from the
// issuance.CarriedMatchCommentColumns allow-list, as carriedComment does for
// a position's comments.
func carriedMatchComment(m *domain.Match) (cols []string, args []any) {
	for _, col := range issuance.CarriedMatchCommentColumns {
		switch col {
		case "comment":
			args = append(args, m.Comment)
		case "comment_author":
			args = append(args, m.CommentAuthor)
		default:
			panic("issuance.CarriedMatchCommentColumns names " + col + ", which the export does not know how to copy")
		}
		cols = append(cols, col)
	}
	return cols, args
}

// carriedComment spells the INSERT of one exported comment from the
// issuance.CarriedCommentColumns allow-list: a column it does not name is not
// copied, and a name it adds without a value here fails the export loudly.
func carriedComment(c *domain.CommentEntry) (cols, marks []string, args []any) {
	for _, col := range issuance.CarriedCommentColumns {
		switch col {
		case "text":
			marks, args = append(marks, "?"), append(args, c.Text)
		case "created_at":
			marks, args = append(marks, "COALESCE(NULLIF(?, ''), CURRENT_TIMESTAMP)"), append(args, c.CreatedAt)
		case "modified_at":
			marks, args = append(marks, "NULLIF(?, '')"), append(args, c.ModifiedAt)
		case "author":
			marks, args = append(marks, "?"), append(args, c.Author)
		default:
			panic("issuance.CarriedCommentColumns names " + col + ", which the export does not know how to copy")
		}
		cols = append(cols, col)
	}
	return cols, marks, args
}

// nullIfEmpty writes "" as NULL: a nullable column reads NULL as "not set".
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
