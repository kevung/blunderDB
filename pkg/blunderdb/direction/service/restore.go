package service

import (
	"context"
	"encoding/json"
	"fmt"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/trash"
)

// RestoreFromTrash puts one trash entry back and removes it from the trash (ADR-0036), and
// returns the id of what came back. A Rencontre is a gesture on its events: they join the room
// again under their locks, in one transaction with the room's creation and the entry's discard.
func (d *Service) RestoreFromTrash(ctx context.Context, trashID int64) (int64, error) {
	entry, err := d.st.Trash().Load(ctx, d.scope, trashID)
	if err != nil {
		return 0, err
	}
	if entry.Kind != domain.TrashRencontre {
		return trash.Restore(ctx, d.st, d.scope, trashID)
	}
	var payload domain.TrashRencontrePayload
	if err := json.Unmarshal(entry.Payload, &payload); err != nil {
		return 0, fmt.Errorf("trash entry %d: %w", trashID, err)
	}
	id, err := d.restoreRencontre(ctx, trashID, payload.Rencontre.TournamentIDs)
	if err != nil {
		return 0, err
	}
	ctx = context.WithoutCancel(ctx)
	d.publishGesture(ctx, 0, id, nil)
	d.writeRencontrePages(ctx, id)
	return id, nil
}

// restoreRencontre recreates the Rencontre of the trash entry trashID and attaches its events
// again, on the room's tables. Detached, an event played on its own tables: a running match on
// a table outside the rooms it had refuses the restore, naming the table (ADR-0058 §11), and
// the entry stays in the trash.
func (d *Service) restoreRencontre(ctx context.Context, trashID int64, members []int64) (id int64, err error) {
	d, release, err := d.lockMembers(ctx, members)
	if err != nil {
		return 0, err
	}
	defer release(&err)
	// Read again under the locks: a restore that ran meanwhile has discarded it.
	entry, err := d.st.Trash().Load(ctx, d.scope, trashID)
	if err != nil {
		return 0, err
	}
	if id, err = trash.RestoreRencontre(ctx, d.st, d.scope, entry); err != nil {
		return 0, err
	}
	r, err := d.st.Rencontres().Get(ctx, d.scope, id)
	if err != nil {
		return 0, err
	}
	if err := checkRunningStays(ctx, d.dirStore(), d.membersOf(ctx, d.st, r),
		func(int64) direction.TablePlan { return direction.TablePlan{} },
		func(tid int64) direction.TablePlan { return planIn(r, tid, tournoi.Config{}) }); err != nil {
		return 0, err
	}
	if err := d.realign(ctx, id); err != nil {
		return 0, err
	}
	return id, d.st.Trash().Discard(ctx, d.scope, trashID)
}
