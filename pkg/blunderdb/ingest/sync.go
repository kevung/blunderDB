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

// On a backend whose identifiers can become visible out of order — on
// PostgreSQL a sequence value is taken when a row is inserted and seen when
// its transaction commits — an export reading at T may pass identifiers
// held by a transaction still running at T, which appear later. Its mark is
// therefore kept pending, and the exports that follow still start from the
// last trusted mark. A pending mark becomes trusted only through a later
// export that checked, before reading, that every transaction running when
// the mark was read has ended (SyncFence): that export saw whatever those
// transactions committed below the mark. No clock is trusted, so a
// transaction of any length delays the mark instead of losing its rows.

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
// mark below which every row has been sent. Pending holds, oldest first, the
// marks of the exports made since on a backend that does not commit
// identifiers in order, each waiting for its fence to clear.
type SyncCursor struct {
	Settled SyncMark      `json:"settled"`
	Pending []PendingMark `json:"pending,omitempty"`
}

// PendingMark is a mark read by an export, not yet trusted. Fence names the
// transactions running when it was read; At is informative only.
type PendingMark struct {
	Mark  SyncMark  `json:"mark"`
	Fence string    `json:"fence"`
	At    time.Time `json:"at"`
}

// SyncAdvance is what an incremental export writes back to its source once
// the file is complete: the cursor's name and its new state.
type SyncAdvance struct {
	Name   string
	Cursor SyncCursor
}

// OrderedIDs is implemented by a backend whose identifiers become visible in
// the order they are allocated — SQLite, whose single writer commits one
// transaction before the next can take an identifier. Its marks are trusted
// at once.
type OrderedIDs interface {
	IDsVisibleInOrder() bool
}

// SyncFence is implemented by a backend whose identifiers can become visible
// out of order but which can name the transactions running at a moment and
// later tell whether they have all ended — PostgreSQL. A backend that is
// neither ordered nor fenced never trusts a mark: each incremental export
// then carries everything since the zero mark, an overlap, never a hole.
type SyncFence interface {
	// SyncFence returns a token naming every transaction running now.
	SyncFence(ctx context.Context) (string, error)
	// SyncFenceCleared reports whether every transaction the token names
	// has ended. An unreadable token is not cleared.
	SyncFenceCleared(ctx context.Context, token string) (bool, error)
}

func idsVisibleInOrder(src storage.Storage) bool {
	o, ok := src.(OrderedIDs)
	return ok && o.IDsVisibleInOrder()
}

// clearedPrefix counts the pending marks, oldest first, whose fence has
// cleared. It stops at the first that has not: a newer mark is above an
// older one, so trusting it would pass the older one's late rows.
func (c SyncCursor) clearedPrefix(cleared func(token string) (bool, error)) (int, error) {
	for i, p := range c.Pending {
		ok, err := cleared(p.Fence)
		if err != nil {
			return 0, err
		}
		if !ok {
			return i, nil
		}
	}
	return len(c.Pending), nil
}

// advance records the mark read by an export that completed. The export
// started from c.Settled and checked, before reading, that the fences of
// the first n pending marks had cleared: it saw every row below them, so
// the newest of them is trusted. The mark it read waits behind fence; on an
// ordered backend it is trusted at once. fenced is false on a backend that
// can name no fence: nothing is ever trusted there.
func (c SyncCursor) advance(n int, mark SyncMark, fence string, now time.Time, ordered, fenced bool) SyncCursor {
	if ordered {
		return SyncCursor{Settled: mark}
	}
	if !fenced {
		return SyncCursor{Settled: c.Settled}
	}
	next := SyncCursor{Settled: c.Settled}
	if n > 0 {
		next.Settled = c.Pending[n-1].Mark
	}
	next.Pending = append(append([]PendingMark(nil), c.Pending[n:]...), PendingMark{Mark: mark, Fence: fence, At: now})
	return next
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
	ordered := idsVisibleInOrder(src)
	fence, fenced := src.(SyncFence)
	// The fences are checked before the read: a transaction that has
	// ended by now committed its rows before the read below sees them.
	cleared := 0
	if !ordered && fenced {
		if cleared, err = cur.clearedPrefix(func(token string) (bool, error) {
			return fence.SyncFenceCleared(ctx, token)
		}); err != nil {
			return plan, fmt.Errorf("ingest: check sync fence: %w", err)
		}
	}
	since := cur.Settled
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

	// The fence is taken after the read: it names every transaction that
	// could hold an identifier below the mark just read.
	var token string
	if !ordered && fenced {
		if token, err = fence.SyncFence(ctx); err != nil {
			return plan, fmt.Errorf("ingest: take sync fence: %w", err)
		}
	}
	plan.Selection = sel
	plan.Sync = &SyncAdvance{Name: cursor, Cursor: cur.advance(cleared, next, token, time.Now(), ordered, fenced)}
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
