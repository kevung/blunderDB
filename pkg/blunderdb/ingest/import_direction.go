package ingest

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// mergeDirections reads back the Direction an export carried (ADR-0047) into
// the Tournament each source Tournament merged into: the record without its
// output folder, its own tables, the journal event for event, the pairs, and
// the Slot each imported Match fills. Run once res.MatchMap and
// res.TournamentMap are known.
//
// A target Tournament that already has a Direction keeps it untouched: two
// journals cannot be merged, and the one here is the recipient's. Importing
// the same file twice therefore writes nothing the second time.
//
// The journal names its players by Participant ids local to the Tournament,
// never by a row of the library: it travels as is. Only the Matches filling
// its Slots are remapped.
func mergeDirections(ctx context.Context, tx storage.Stores, scope string, source storage.Stores, res *MatchMerge) error {
	if len(res.TournamentMap) == 0 {
		return nil
	}
	recs, err := source.Directions().List(ctx, scope)
	if err != nil {
		res.unreadable("directions", err)
		recs = nil
	}
	dst := tx.Directions()
	// kept holds the target Tournaments whose Direction is the recipient's.
	kept := map[int64]bool{}
	created := map[int64]bool{}
	for _, rec := range recs {
		if err := ctx.Err(); err != nil {
			return err
		}
		tid, ok := res.TournamentMap[rec.TournamentID]
		if !ok {
			continue
		}
		// Two source Tournaments of one name merge into one target
		// Tournament: the first Direction read back is its own.
		if created[tid] {
			res.DirectionsKnown++
			continue
		}
		if _, err := dst.Get(ctx, scope, tid); err == nil {
			kept[tid] = true
			res.DirectionsKnown++
			continue
		} else if !errors.Is(err, direction.ErrNoDirection) && !errors.Is(err, storage.ErrNotFound) {
			return fmt.Errorf("ingest: reading the direction of tournament %d: %w", tid, err)
		}
		if err := copyDirection(ctx, tx, scope, source, rec, tid, res); err != nil {
			return err
		}
		created[tid] = true
		res.Directions++
	}
	return mergeRencontres(ctx, tx, scope, source, res, kept)
}

func copyDirection(ctx context.Context, tx storage.Stores, scope string, source storage.Stores, rec direction.Record, tid int64, res *MatchMerge) error {
	src, dst := source.Directions(), tx.Directions()
	srcTID := rec.TournamentID
	events, err := src.LoadEvents(ctx, scope, srcTID)
	if err != nil {
		return fmt.Errorf("ingest: reading the journal of source tournament %d: %w", srcTID, err)
	}
	// A source older than pairs (schema < 2.25.0) or than table properties
	// (< 2.28.0) holds a Direction without them: it is read back without them.
	pairs, err := src.Pairs(ctx, scope, srcTID)
	if err != nil {
		res.unreadableOnce("direction pairs", err)
		pairs = nil
	}
	slots, err := src.FilledSlots(ctx, scope, srcTID)
	if err != nil {
		return fmt.Errorf("ingest: reading the slots of source tournament %d: %w", srcTID, err)
	}
	tables, err := source.Rencontres().TournamentTableSettings(ctx, scope, srcTID)
	if err != nil {
		res.unreadableOnce("table settings", err)
		tables = nil
	}

	// Field by field: OutputDir is a path on the producer's machine.
	if err := dst.Create(ctx, scope, direction.Record{
		TournamentID: tid, FormatVersion: rec.FormatVersion, EngineVersion: rec.EngineVersion,
		State: rec.State, Config: rec.Config,
	}); err != nil {
		return fmt.Errorf("ingest: import the direction of tournament %d: %w", tid, err)
	}
	if len(tables) > 0 {
		if err := tx.Rencontres().SetTournamentTableSettings(ctx, scope, tid, tables); err != nil {
			return fmt.Errorf("ingest: import the tables of tournament %d: %w", tid, err)
		}
	}
	// In sequence order: a journal is replayed from its first event.
	for _, ev := range events {
		if err := dst.AppendEvent(ctx, scope, tid, ev); err != nil {
			return fmt.Errorf("ingest: import event %d of tournament %d: %w", ev.Seq, tid, err)
		}
	}
	for _, id := range slices.Sorted(maps.Keys(pairs)) {
		if err := dst.SetPair(ctx, scope, tid, id, pairs[id]); err != nil {
			return fmt.Errorf("ingest: import pair %s of tournament %d: %w", id, tid, err)
		}
	}
	for _, f := range slots {
		matchID, ok := res.MatchMap[f.MatchID]
		if !ok {
			continue
		}
		free, err := slotFree(ctx, tx, scope, tid, matchID)
		if err != nil {
			return err
		}
		if !free {
			continue
		}
		if err := dst.AttachSlot(ctx, scope, tid, f.SlotID, matchID); err != nil {
			return fmt.Errorf("ingest: import slot %s of tournament %d: %w", f.SlotID, tid, err)
		}
		res.Slots++
	}
	// Last: appending to the journal and filling a Slot stamp the record.
	if err := dst.KeepDates(ctx, scope, tid, exportedStamp(rec.CreatedAt), exportedStamp(rec.UpdatedAt)); err != nil {
		return fmt.Errorf("ingest: import the dates of the direction of tournament %d: %w", tid, err)
	}
	return nil
}

// slotFree says whether a target Match may fill a Slot of tournament tid: a
// Match the target already held stays where the recipient filed it — in
// another Tournament, or in a Slot already.
func slotFree(ctx context.Context, tx storage.Stores, scope string, tid, matchID int64) (bool, error) {
	if _, slot, err := tx.Directions().SlotOf(ctx, scope, matchID); err != nil {
		return false, fmt.Errorf("ingest: reading the slot of match %d: %w", matchID, err)
	} else if slot != "" {
		return false, nil
	}
	m, err := tx.Matches().Get(ctx, scope, matchID)
	if err != nil {
		return false, fmt.Errorf("ingest: reading match %d: %w", matchID, err)
	}
	return m.TournamentID == nil || *m.TournamentID == tid, nil
}

// mergeRencontres reads back the Rencontre of every imported Tournament
// (ADR-0056). The Rencontre of the same name in the target receives them,
// else it is created with the source's dates, tables and table properties. A
// Tournament already in a Rencontre here stays in it, and one whose Direction
// is the recipient's (kept) is left as the recipient set it.
func mergeRencontres(ctx context.Context, tx storage.Stores, scope string, source storage.Stores, res *MatchMerge, kept map[int64]bool) error {
	srcRencontres, err := source.Rencontres().List(ctx, scope)
	if err != nil {
		res.unreadable("rencontres", err)
		return nil
	}
	known, err := tx.Rencontres().List(ctx, scope)
	if err != nil {
		return fmt.Errorf("ingest: reading the rencontres: %w", err)
	}
	byName := map[string]int64{}
	for _, r := range known {
		byName[r.Name] = r.ID
	}
	slices.SortFunc(srcRencontres, func(a, b *domain.Rencontre) int { return cmp.Compare(a.ID, b.ID) })
	for _, r := range srcRencontres {
		var members []int64
		seen := map[int64]bool{}
		for _, srcTID := range r.TournamentIDs {
			tid, ok := res.TournamentMap[srcTID]
			if !ok || kept[tid] || seen[tid] {
				continue
			}
			seen[tid] = true
			if in, err := tx.Rencontres().Of(ctx, scope, tid); err != nil && !errors.Is(err, storage.ErrNotFound) {
				return fmt.Errorf("ingest: reading the rencontre of tournament %d: %w", tid, err)
			} else if in != 0 {
				continue
			}
			members = append(members, srcTID)
		}
		if len(members) == 0 {
			continue
		}
		id, existed := byName[r.Name]
		if !existed {
			if id, err = createRencontre(ctx, tx, scope, r); err != nil {
				return err
			}
			byName[r.Name] = id
			res.Rencontres++
		}
		for _, srcTID := range members {
			tid := res.TournamentMap[srcTID]
			if err := tx.Rencontres().Attach(ctx, scope, tid, id); err != nil {
				return fmt.Errorf("ingest: attaching tournament %d to rencontre %d: %w", tid, id, err)
			}
			if rooms := r.EventRooms[srcTID]; len(rooms) > 0 {
				if err := tx.Rencontres().SetEventRooms(ctx, scope, tid, rooms); err != nil {
					return fmt.Errorf("ingest: import the rooms of tournament %d: %w", tid, err)
				}
			}
		}
		// Last: attaching a member may stamp the Rencontre.
		if !existed {
			if err := tx.Rencontres().KeepDates(ctx, scope, id, r.CreatedAt, r.UpdatedAt); err != nil {
				return fmt.Errorf("ingest: import the dates of rencontre %q: %w", r.Name, err)
			}
		}
	}
	return nil
}

// createRencontre writes a source Rencontre field by field: OutputDir stays
// behind, a path on the producer's machine.
func createRencontre(ctx context.Context, tx storage.Stores, scope string, r *domain.Rencontre) (int64, error) {
	rs := tx.Rencontres()
	id, err := rs.Create(ctx, scope, domain.Rencontre{Name: r.Name, StartsOn: r.StartsOn, EndsOn: r.EndsOn, Tables: r.Tables})
	if err != nil {
		return 0, fmt.Errorf("ingest: import rencontre %q: %w", r.Name, err)
	}
	if len(r.TableSettings) > 0 {
		if err := rs.SetTableSettings(ctx, scope, id, r.TableSettings); err != nil {
			return 0, fmt.Errorf("ingest: import the tables of rencontre %q: %w", r.Name, err)
		}
	}
	return id, nil
}

// unreadableOnce is unreadable for a family read once per Tournament: it is
// named once however many Tournaments lack it.
func (res *MatchMerge) unreadableOnce(family string, err error) {
	if !slices.Contains(res.Unreadable, family) {
		res.unreadable(family, err)
	}
}
