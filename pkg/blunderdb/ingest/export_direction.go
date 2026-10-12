package ingest

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// writeDirections carries the Direction of every exported tournament
// (ADR-0047): its record, its journal, the persons behind its pairs, and the
// Slot each exported match fills. The producer writes them into the file he
// makes; nothing is written on the recipient's side (ADR-0007).
//
// The record travels field by field, an allow-list like the match columns:
// OutputDir stays behind, a path on the producer's machine that would make the
// recipient's first refresh fail.
//
// A match whose tournament stays behind leaves without its Slot: its
// direction_match_id names a pairing of a journal the file does not hold.
func (e *exporter) writeDirections() error {
	if len(e.tourMap) == 0 {
		return nil
	}
	src, dst := e.src.Directions(), e.dst.Directions()
	for _, srcTID := range slices.Sorted(maps.Keys(e.tourMap)) {
		if err := e.ctx.Err(); err != nil {
			return err
		}
		rec, err := src.Get(e.ctx, e.scope, srcTID)
		if errors.Is(err, direction.ErrNoDirection) || errors.Is(err, storage.ErrNotFound) {
			continue
		}
		if err != nil {
			return fmt.Errorf("ingest: reading the direction of tournament %d: %w", srcTID, err)
		}
		events, err := src.LoadEvents(e.ctx, e.scope, srcTID)
		if err != nil {
			return fmt.Errorf("ingest: reading the journal of tournament %d: %w", srcTID, err)
		}
		pairs, err := src.Pairs(e.ctx, e.scope, srcTID)
		if err != nil {
			return fmt.Errorf("ingest: reading the pairs of tournament %d: %w", srcTID, err)
		}
		slots, err := src.FilledSlots(e.ctx, e.scope, srcTID)
		if err != nil {
			return fmt.Errorf("ingest: reading the slots of tournament %d: %w", srcTID, err)
		}

		tid := e.tourMap[srcTID]
		if err := dst.Create(e.ctx, "", direction.Record{
			TournamentID: tid, FormatVersion: rec.FormatVersion, EngineVersion: rec.EngineVersion,
			State: rec.State, Config: rec.Config,
		}); err != nil {
			return fmt.Errorf("ingest: writing the direction of tournament %d: %w", tid, err)
		}
		e.keepDirectionDates(tid, rec)
		// In sequence order: a journal is replayed from its first event, and a
		// hole or a swap would replay to another tournament.
		for _, ev := range events {
			if err := dst.AppendEvent(e.ctx, "", tid, ev); err != nil {
				return fmt.Errorf("ingest: writing event %d of tournament %d: %w", ev.Seq, tid, err)
			}
		}
		for _, id := range slices.Sorted(maps.Keys(pairs)) {
			if err := dst.SetPair(e.ctx, "", tid, id, pairs[id]); err != nil {
				return fmt.Errorf("ingest: writing pair %s of tournament %d: %w", id, tid, err)
			}
		}
		for _, f := range slots {
			matchID, ok := e.matchMap[f.MatchID]
			if !ok {
				continue
			}
			if err := dst.AttachSlot(e.ctx, "", tid, f.SlotID, matchID); err != nil {
				e.skip("attaching a match to its slot", "matchID", f.MatchID, "slot", f.SlotID, "err", err)
			}
		}
		e.report.Directions++
	}
	if e.report.Directions > 0 {
		slog.Info("exported directions", "count", e.report.Directions)
	}
	return nil
}

// keepDirectionDates gives the copied record the dates of its source: Create
// stamps the moment of the export.
func (e *exporter) keepDirectionDates(tid int64, rec direction.Record) {
	stamp := func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.UTC().Format(time.DateTime)
	}
	if _, err := e.tx.ExecContext(e.ctx,
		`UPDATE direction SET created_at = COALESCE(NULLIF(?, ''), created_at),
		 updated_at = COALESCE(NULLIF(?, ''), updated_at) WHERE tournament_id = ?`,
		stamp(rec.CreatedAt), stamp(rec.UpdatedAt), tid); err != nil {
		e.skip("keeping direction dates", "tournamentID", tid, "err", err)
	}
}
