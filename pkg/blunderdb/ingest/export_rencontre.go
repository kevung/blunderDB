package ingest

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// rencontresOf returns, in id order, the Rencontres the given Tournaments
// play in. A Rencontre leaves as soon as one of its Tournaments does
// (ADR-0056); the Tournaments that stay behind are not attached to it in the
// file.
func rencontresOf(e *exporter, tournamentIDs []int64) ([]int64, error) {
	seen := map[int64]bool{}
	var out []int64
	for _, tid := range tournamentIDs {
		id, err := e.src.Rencontres().Of(e.ctx, e.scope, tid)
		if errors.Is(err, storage.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("ingest: reading the rencontre of tournament %d: %w", tid, err)
		}
		if id != 0 && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return out, nil
}

// writeRencontres copies the Rencontre of every exported Tournament: its name,
// dates and table count, its table properties, and for each exported member
// the rooms it may play in. The record travels field by field, an allow-list:
// OutputDir stays behind, a path on the producer's machine. The producer
// writes it into the file he makes; nothing is written on the recipient's
// side (ADR-0007).
func (e *exporter) writeRencontres() error {
	if len(e.tourMap) == 0 {
		return nil
	}
	srcTours := make([]int64, 0, len(e.tourMap))
	for id := range e.tourMap {
		srcTours = append(srcTours, id)
	}
	slices.Sort(srcTours)
	ids, err := rencontresOf(e, srcTours)
	if err != nil {
		return err
	}
	dst := e.dst.Rencontres()
	for _, id := range ids {
		if err := e.ctx.Err(); err != nil {
			return err
		}
		r, err := e.src.Rencontres().Get(e.ctx, e.scope, id)
		if err != nil {
			return fmt.Errorf("ingest: reading rencontre %d: %w", id, err)
		}
		newID, err := dst.Create(e.ctx, "", domain.Rencontre{
			Name: r.Name, StartsOn: r.StartsOn, EndsOn: r.EndsOn, Tables: r.Tables,
		})
		if err != nil {
			return fmt.Errorf("ingest: writing rencontre %d: %w", id, err)
		}
		if _, err := e.tx.ExecContext(e.ctx,
			`UPDATE rencontre SET created_at = COALESCE(NULLIF(?, ''), created_at),
			 updated_at = COALESCE(NULLIF(?, ''), updated_at) WHERE id = ?`,
			r.CreatedAt, r.UpdatedAt, newID); err != nil {
			return fmt.Errorf("ingest: keeping the dates of rencontre %d: %w", id, err)
		}
		if err := dst.SetTableSettings(e.ctx, "", newID, r.TableSettings); err != nil {
			return fmt.Errorf("ingest: writing the tables of rencontre %d: %w", id, err)
		}
		for _, srcTID := range r.TournamentIDs {
			tid, ok := e.tourMap[srcTID]
			if !ok {
				continue
			}
			if err := dst.Attach(e.ctx, "", tid, newID); err != nil {
				return fmt.Errorf("ingest: attaching tournament %d to rencontre %d: %w", tid, newID, err)
			}
			if rooms := r.EventRooms[srcTID]; len(rooms) > 0 {
				if err := dst.SetEventRooms(e.ctx, "", tid, rooms); err != nil {
					return fmt.Errorf("ingest: writing the rooms of tournament %d: %w", tid, err)
				}
			}
		}
		e.report.Rencontres++
	}
	if e.report.Rencontres > 0 {
		slog.Info("exported rencontres", "count", e.report.Rencontres)
	}
	return nil
}
