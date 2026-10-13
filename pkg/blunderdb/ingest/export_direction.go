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
		if err := e.writeOwnTables(srcTID, tid); err != nil {
			return err
		}
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
		for _, f := range exportableSlots(slots, e.matchMap) {
			matchID := e.matchMap[f.MatchID]
			if err := dst.AttachSlot(e.ctx, "", tid, f.SlotID, matchID); err != nil {
				// Counted by Skipped and, by its absence from Slots, by the
				// plan's mismatch: the file holds a pairing without its match.
				e.skip("attaching a match to its slot", "matchID", f.MatchID, "slot", f.SlotID, "err", err)
				continue
			}
			e.report.Slots++
		}
		// Last: appending to the journal and filling a Slot stamp the record.
		if err := e.keepDirectionDates(tid, rec); err != nil {
			return err
		}
		e.report.Directions++
	}
	if e.report.Directions > 0 {
		slog.Info("exported directions", "count", e.report.Directions)
	}
	return nil
}

// exportableSlots keeps the filled Slots whose match leaves with the file:
// the planner counts the same ones.
func exportableSlots(slots []storage.FilledSlot, matchMap map[int64]int64) []storage.FilledSlot {
	var out []storage.FilledSlot
	for _, f := range slots {
		if _, ok := matchMap[f.MatchID]; ok {
			out = append(out, f)
		}
	}
	return out
}

// exportedStamp writes a date at the precision it was read: the fractional
// seconds are dropped only when they are zero.
func exportedStamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02 15:04:05.999999999")
}

// keepDirectionDates gives the copied record the dates of its source: Create
// stamps the moment of the export.
func (e *exporter) keepDirectionDates(tid int64, rec direction.Record) error {
	if _, err := e.tx.ExecContext(e.ctx,
		`UPDATE direction SET created_at = COALESCE(NULLIF(?, ''), created_at),
		 updated_at = COALESCE(NULLIF(?, ''), updated_at) WHERE tournament_id = ?`,
		exportedStamp(rec.CreatedAt), exportedStamp(rec.UpdatedAt), tid); err != nil {
		return fmt.Errorf("ingest: keeping the dates of the direction of tournament %d: %w", tid, err)
	}
	return nil
}

// writeOwnTables copies the table properties a Tournament owns to run on its
// own (ADR-0058 §3): they leave with its Direction even when it plays in a
// Rencontre, where they are kept and ignored.
func (e *exporter) writeOwnTables(srcTID, tid int64) error {
	settings, err := e.src.Rencontres().TournamentTableSettings(e.ctx, e.scope, srcTID)
	if err != nil {
		return fmt.Errorf("ingest: reading the tables of tournament %d: %w", srcTID, err)
	}
	if len(settings) == 0 {
		return nil
	}
	if err := e.dst.Rencontres().SetTournamentTableSettings(e.ctx, "", tid, settings); err != nil {
		return fmt.Errorf("ingest: writing the tables of tournament %d: %w", tid, err)
	}
	return nil
}
