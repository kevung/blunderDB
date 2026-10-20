package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

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

// syncCursorPrefix namespaces the cursors in the metadata table. The key is
// never carried by an export: metadata leaves by allow-list
// (issuance.CarriedMetadataKeys), and this prefix is not on it.
const syncCursorPrefix = "sync_cursor:"

// DefaultSyncCursor is the cursor an incremental export uses when its caller
// names none — the daemon's case, which knows no destination folder.
const DefaultSyncCursor = "default"

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

// SyncAdvance is what an incremental export writes back to its source once
// the file is complete: the cursor's name and its new mark.
type SyncAdvance struct {
	Cursor string
	Mark   SyncMark
}

func syncCursorKey(cursor string) string {
	if cursor == "" {
		cursor = DefaultSyncCursor
	}
	return syncCursorPrefix + cursor
}

// LoadSyncMark reads a cursor of the scope; a cursor never advanced is the
// zero mark, so a first incremental export carries the whole library.
func LoadSyncMark(ctx context.Context, src storage.Storage, scope, cursor string) (SyncMark, error) {
	var mark SyncMark
	meta, err := src.Metadata().Load(ctx, scope)
	if err != nil {
		return mark, fmt.Errorf("ingest: read sync cursor: %w", err)
	}
	raw, ok := meta[syncCursorKey(cursor)]
	if !ok || raw == "" {
		return mark, nil
	}
	if err := json.Unmarshal([]byte(raw), &mark); err != nil {
		// A cursor that cannot be read is restarted rather than refused:
		// re-exporting rows the other side holds costs a few seconds of
		// idempotent import, refusing would leave the user with no way out.
		return SyncMark{}, nil
	}
	return mark, nil
}

// saveSyncMark records the cursor's new mark in the source's metadata.
func saveSyncMark(ctx context.Context, src storage.Storage, scope string, adv *SyncAdvance) error {
	raw, err := json.Marshal(adv.Mark)
	if err != nil {
		return err
	}
	if err := src.Metadata().Save(ctx, scope, map[string]string{syncCursorKey(adv.Cursor): string(raw)}); err != nil {
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
	since, err := LoadSyncMark(ctx, src, scope, cursor)
	if err != nil {
		return plan, err
	}
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
	plan.Sync = &SyncAdvance{Cursor: cursor, Mark: next}
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
