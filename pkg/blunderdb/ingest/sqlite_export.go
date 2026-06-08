package ingest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// SQLiteExporter serializes one tenant (scope) into a fresh, valid blunderDB
// SQLite file and streams its bytes to w. The result opens directly in blunderDB
// Desktop (`blunderdb --backend sqlite --db <file>`).
//
// It is backend-agnostic: it reads the source through the Storage interface and
// writes through a fresh SQLite Storage, so the same code exports a PostgreSQL
// tenant or snapshots any backend. The exported file is single-tenant — every
// row is written under the SQLite default (empty) scope.
//
// Match graphs (match/game/move) are not exported yet — parity with
// JSONExporter, which adds them once the Storage matchWriter lands. Anki review
// history is intentionally reset: an export is a fresh study copy (deck +
// positions), not a scheduler snapshot.
type SQLiteExporter struct{ S storage.Storage }

// destScope is the scope written into the exported SQLite file (single-tenant).
const destScope = ""

func (e SQLiteExporter) Export(ctx context.Context, scope string, w io.Writer, _ ExportOptions) error {
	tmp, err := os.CreateTemp("", "blunderdb-export-*.sqlite")
	if err != nil {
		return fmt.Errorf("ingest: temp file: %w", err)
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)

	dst, err := sqlite.Open(ctx, tmpPath, nil)
	if err != nil {
		return fmt.Errorf("ingest: open sqlite: %w", err)
	}
	if err := e.writeAll(ctx, scope, dst); err != nil {
		dst.Close()
		return err
	}
	if err := dst.Close(); err != nil {
		return fmt.Errorf("ingest: close sqlite: %w", err)
	}

	f, err := os.Open(tmpPath)
	if err != nil {
		return fmt.Errorf("ingest: reopen export: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(w, f); err != nil {
		return fmt.Errorf("ingest: stream export: %w", err)
	}
	return nil
}

// writeAll copies every supported family from the source tenant into a fresh
// SQLite Storage, inside a single transaction (all-or-nothing). Positions are
// re-saved (their ids change), so dependent rows are remapped via posMap/colMap.
func (e SQLiteExporter) writeAll(ctx context.Context, scope string, out storage.Storage) error {
	tx, err := out.BeginTx(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// Positions (+ analyses + comments). Drain the source list first so its
	// connection is released before the per-position reads (see JSONExporter).
	var positions []*domain.Position
	for p, err := range e.S.Positions().List(ctx, scope, storage.ListOpts{}) {
		if err != nil {
			return err
		}
		pc := *p
		positions = append(positions, &pc)
	}
	posMap := make(map[int64]int64, len(positions)) // source id → dest id
	for _, p := range positions {
		if err := ctx.Err(); err != nil {
			return err
		}
		src := p.ID
		p.ID = 0
		newID, err := tx.Positions().Save(ctx, destScope, p)
		if err != nil {
			return err
		}
		posMap[src] = newID

		a, err := e.S.Analyses().Load(ctx, scope, src)
		switch {
		case err == nil:
			if err := tx.Analyses().Save(ctx, destScope, newID, a); err != nil {
				return err
			}
		case errors.Is(err, storage.ErrNotFound):
			// no analysis for this position
		default:
			return err
		}
		for c, err := range e.S.Comments().ByPosition(ctx, scope, src) {
			if err != nil {
				return err
			}
			if _, err := tx.Comments().Add(ctx, destScope, newID, c.Text); err != nil {
				return err
			}
		}
	}

	// Collections (+ membership), remapping member position ids.
	type srcColl struct {
		id         int64
		name, desc string
	}
	var colls []srcColl
	for c, err := range e.S.Collections().List(ctx, scope) {
		if err != nil {
			return err
		}
		colls = append(colls, srcColl{c.ID, c.Name, c.Description})
	}
	colMap := make(map[int64]int64, len(colls))
	for _, c := range colls {
		newID, err := tx.Collections().Create(ctx, destScope, c.name, c.desc)
		if err != nil {
			return err
		}
		colMap[c.id] = newID
		var members []int64
		for p, err := range e.S.Collections().Positions(ctx, scope, c.id) {
			if err != nil {
				return err
			}
			if dstID, ok := posMap[p.ID]; ok {
				members = append(members, dstID)
			}
		}
		for _, pid := range members {
			if err := tx.Collections().AddPosition(ctx, destScope, newID, pid); err != nil {
				return err
			}
		}
	}

	// Tournaments (without match links — match graphs aren't exported yet).
	for tn, err := range e.S.Tournaments().List(ctx, scope) {
		if err != nil {
			return err
		}
		if _, err := tx.Tournaments().Create(ctx, destScope, tn.Name, tn.Date, tn.Location); err != nil {
			return err
		}
	}

	// Anki decks, rebuilt from their (remapped) member positions. Drain the deck
	// list before reading each deck's positions: nesting two source iterators
	// would deadlock the single-connection SQLite pool (as JSONExporter notes).
	type srcDeck struct {
		d      domain.AnkiDeck
		posIDs []int64
	}
	var decks []srcDeck
	for d, err := range e.S.Anki().ListDecks(ctx, scope) {
		if err != nil {
			return err
		}
		decks = append(decks, srcDeck{d: *d})
	}
	for i := range decks {
		for p, err := range e.S.Anki().DeckPositions(ctx, scope, decks[i].d.ID) {
			if err != nil {
				return err
			}
			if dstID, ok := posMap[p.ID]; ok {
				decks[i].posIDs = append(decks[i].posIDs, dstID)
			}
		}
	}
	for _, sd := range decks {
		sourceID := sd.d.SourceID
		sourceCmd := sd.d.SourceCommand
		switch sd.d.SourceType {
		case "collection":
			sourceID = colMap[sd.d.SourceID] // 0 if the collection was absent
		case "search":
			sourceCmd = remapIDList(sourceCmd, posMap)
		}
		newDeckID, err := tx.Anki().CreateDeck(ctx, destScope, sd.d.Name, sd.d.Description, sd.d.SourceType, sourceID, sourceCmd)
		if err != nil {
			return err
		}
		if len(sd.posIDs) > 0 {
			if err := tx.Anki().SyncWithPositions(ctx, destScope, newDeckID, sd.posIDs); err != nil {
				return err
			}
		}
		if err := tx.Anki().UpdateDeckParams(ctx, destScope, newDeckID, sd.d.RequestRetention, sd.d.MaximumInterval, sd.d.EnableFuzz); err != nil {
			return err
		}
	}

	// Saved filters (the filter library).
	for f, err := range e.S.Filters().List(ctx, scope) {
		if err != nil {
			return err
		}
		if _, err := tx.Filters().Save(ctx, destScope, f.Name, f.Command); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
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
