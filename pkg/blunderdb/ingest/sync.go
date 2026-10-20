package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// An incremental export carries what entered the library since the previous
// incremental export to the same destination, so two libraries converge by
// exchanging files (ADR-0096). The cursor is the PRODUCER's: it lives in the
// exporting library's metadata and is advanced when the file is complete;
// the recipient imports an ordinary .db and records nothing (ADR-0007).
//
// The cursor is a high-water mark of row identifiers, one per family. Both
// backends allocate identifiers that only grow (AUTOINCREMENT, sequences), so
// "id above the mark" is "entered since", with no clock to trust and no
// schema change. What it cannot see is an edit in place — a comment
// rewritten, a deletion — and that is the stated contract: a library
// synchronises by addition only.
//
// Every doubt resolves into an overlap, never a hole: the import is
// idempotent, so a row sent twice costs a few seconds, a row never sent is
// lost without a word.

// syncCursorPrefix namespaces the cursors in the metadata table. The key is
// never carried by an export: metadata leaves by allow-list
// (issuance.CarriedMetadataKeys), and this prefix is not on it.
const syncCursorPrefix = "sync_cursor:"

// DefaultSyncCursor is the cursor an incremental export uses when its caller
// names none — the daemon's case, which knows no destination folder.
const DefaultSyncCursor = "default"

// SyncSettleDelay is how long a mark waits before it is trusted on a backend
// whose identifiers can become visible out of order. On PostgreSQL a
// sequence value is taken when a row is inserted and seen when its
// transaction commits: an import still running when the export reads holds
// identifiers below the highest one read, and they appear later. A mark
// read at T is trusted once no transaction open at T can still be running —
// a day, far beyond any import — and until then each export starts from the
// newest trusted mark, sending the last day's additions again.
const SyncSettleDelay = 24 * time.Hour

// SyncMark is the highest identifier of each family an incremental export
// has already carried. The zero mark carries everything.
type SyncMark struct {
	Position         int64 `json:"position"`
	Match            int64 `json:"match"`
	Tournament       int64 `json:"tournament"`
	Collection       int64 `json:"collection"`
	CollectionMember int64 `json:"collectionMember"`
	Lesson           int64 `json:"lesson"`
}

// SyncCursor is what the producer keeps per destination. Settled is the
// mark below which every row has been sent. Pending holds the marks of the
// exports made since, on a backend that does not commit identifiers in
// order; each settles once SyncSettleDelay has passed.
type SyncCursor struct {
	Settled SyncMark      `json:"settled"`
	Pending []PendingMark `json:"pending,omitempty"`
}

// PendingMark is a mark read by an export, not yet trusted.
type PendingMark struct {
	Mark SyncMark  `json:"mark"`
	At   time.Time `json:"at"`
}

// SyncAdvance is what an incremental export writes back to its source once
// the file is complete: the cursor's name and its new state.
type SyncAdvance struct {
	Name   string
	Cursor SyncCursor
}

// OrderedIDs is implemented by a backend whose identifiers become visible in
// the order they are allocated — SQLite, whose single writer commits one
// transaction before the next can take an identifier. A backend that does
// not say so is treated as unordered: its marks wait SyncSettleDelay.
type OrderedIDs interface {
	IDsVisibleInOrder() bool
}

func idsVisibleInOrder(src storage.Storage) bool {
	o, ok := src.(OrderedIDs)
	return ok && o.IDsVisibleInOrder()
}

// start is the mark an export reads from at now: the newest trusted one.
// The cursor returned has the marks that settled by now folded in.
func (c SyncCursor) start(now time.Time) (SyncMark, SyncCursor) {
	var keep []PendingMark
	for _, p := range c.Pending {
		if now.Sub(p.At) >= SyncSettleDelay {
			c.Settled = p.Mark
		} else {
			keep = append(keep, p)
		}
	}
	c.Pending = keep
	return c.Settled, c
}

// advance records a mark read at now by an export that completed.
func (c SyncCursor) advance(mark SyncMark, now time.Time, ordered bool) SyncCursor {
	if ordered {
		return SyncCursor{Settled: mark}
	}
	c.Pending = append(c.Pending, PendingMark{Mark: mark, At: now})
	return c
}

func syncCursorKey(cursor string) string {
	if cursor == "" {
		cursor = DefaultSyncCursor
	}
	return syncCursorPrefix + cursor
}

// LoadSyncCursor reads a cursor of the scope; a cursor never advanced is the
// zero cursor, so a first incremental export carries the whole library.
func LoadSyncCursor(ctx context.Context, src storage.Storage, scope, name string) (SyncCursor, error) {
	var cur SyncCursor
	meta, err := src.Metadata().Load(ctx, scope)
	if err != nil {
		return cur, fmt.Errorf("ingest: read sync cursor: %w", err)
	}
	raw, ok := meta[syncCursorKey(name)]
	if !ok || raw == "" {
		return cur, nil
	}
	if err := json.Unmarshal([]byte(raw), &cur); err != nil {
		// A cursor that cannot be read restarts from zero: an overlap the
		// import absorbs, where refusing would leave no way out.
		return SyncCursor{}, nil
	}
	return cur, nil
}

// saveSyncCursor records the cursor's new state in the source's metadata.
func saveSyncCursor(ctx context.Context, src storage.Storage, scope string, adv *SyncAdvance) error {
	raw, err := json.Marshal(adv.Cursor)
	if err != nil {
		return err
	}
	if err := src.Metadata().Save(ctx, scope, map[string]string{syncCursorKey(adv.Name): string(raw)}); err != nil {
		return fmt.Errorf("ingest: advance sync cursor: %w", err)
	}
	return nil
}

// incrementalOptions is OptionsFor's reading of an incremental export: every
// row above the cursor's mark, each family still subject to its Include…
// flag, with the closures that make each row usable on arrival — a match its
// positions, a tournament its matches, a collection its members, a lesson
// what its steps show. Rows the closure pulls in that the other side already
// holds are merged away by its import.
func incrementalOptions(ctx context.Context, src storage.Storage, scope string, opts domain.ExportOptions) (ExportOptions, error) {
	plan := ExportOptions{
		Format:         FormatSQLite,
		Analysis:       opts.IncludeAnalysis,
		Comments:       opts.IncludeComments,
		PlayedMoves:    opts.IncludePlayedMoves,
		Transcriptions: opts.IncludeTranscriptions,
		Aliases:        opts.IncludeAliases,
	}
	cursor := opts.SyncCursor
	if cursor == "" {
		cursor = DefaultSyncCursor
	}
	cur, err := LoadSyncCursor(ctx, src, scope, cursor)
	if err != nil {
		return plan, err
	}
	now := time.Now()
	since, cur := cur.start(now)
	next := since
	sel := Selection{MatchPositions: true, CollectionPositions: true, TournamentMatches: true, LessonContents: true}

	ids, err := src.Positions().ListIDs(ctx, scope, storage.ListOpts{})
	if err != nil {
		return plan, fmt.Errorf("ingest: list positions: %w", err)
	}
	sel.PositionIDs, next.Position = above(ids, since.Position)

	if opts.IncludeMatches {
		var all []int64
		for m, err := range src.Matches().List(ctx, scope, storage.MatchListOpts{}) {
			if err != nil {
				return plan, fmt.Errorf("ingest: list matches: %w", err)
			}
			all = append(all, m.ID)
		}
		sel.MatchIDs, next.Match = above(all, since.Match)
	}

	if opts.IncludeTournaments {
		var all []int64
		for t, err := range src.Tournaments().List(ctx, scope, storage.ListOpts{}) {
			if err != nil {
				return plan, fmt.Errorf("ingest: list tournaments: %w", err)
			}
			all = append(all, t.ID)
		}
		sel.TournamentIDs, next.Tournament = above(all, since.Tournament)
	}

	if opts.IncludeCollections {
		// Drain the listing before reading members: nesting two iterators
		// holds two pooled connections, which deadlocks a one-connection
		// source.
		var all []int64
		for c, err := range src.Collections().List(ctx, scope) {
			if err != nil {
				return plan, fmt.Errorf("ingest: list collections: %w", err)
			}
			all = append(all, c.ID)
		}
		next.Collection = maxID(all, since.Collection)
		for _, cid := range all {
			grown := cid > since.Collection
			for m, err := range src.Collections().Members(ctx, scope, cid) {
				if err != nil {
					return plan, fmt.Errorf("ingest: list members of collection %d: %w", cid, err)
				}
				if m.ID > since.CollectionMember {
					grown = true
				}
				next.CollectionMember = max(next.CollectionMember, m.ID)
			}
			// A member added to an old collection travels with the
			// collection: the import merges collections by name.
			if grown {
				sel.CollectionIDs = append(sel.CollectionIDs, cid)
			}
		}
	}

	if opts.IncludeLessons {
		lessons, err := src.Lessons().List(ctx, scope)
		if err != nil {
			return plan, fmt.Errorf("ingest: list lessons: %w", err)
		}
		all := make([]int64, 0, len(lessons))
		for _, l := range lessons {
			all = append(all, l.ID)
		}
		sel.LessonIDs, next.Lesson = above(all, since.Lesson)
	}

	plan.Selection = sel
	plan.Sync = &SyncAdvance{Name: cursor, Cursor: cur.advance(next, now, idsVisibleInOrder(src))}
	return plan, nil
}

// above returns the ids greater than mark, sorted, and the new mark.
func above(ids []int64, mark int64) ([]int64, int64) {
	var out []int64
	for _, id := range ids {
		if id > mark {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return out, maxID(ids, mark)
}

func maxID(ids []int64, mark int64) int64 {
	for _, id := range ids {
		mark = max(mark, id)
	}
	return mark
}
