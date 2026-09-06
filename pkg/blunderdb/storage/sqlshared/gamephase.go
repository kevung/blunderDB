package sqlshared

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// ReclassifyPhases rewrites position.game_phase for every row whose stored
// value disagrees with what engine.ClassifyGamePhase says today, and returns
// how many rows changed (issue #264, ADR-0035).
//
// This is what makes the phase a DERIVED label rather than stored data: change
// the classifier or its threshold, run this, and every row agrees with the new
// rule. Nothing else in the database reads the old value, so it can always be
// run and can always be run again.
//
// Only the board is read, out of the compact `state` encoding: the phase does
// not depend on the cube, the score, the dice or the side on roll. A row whose
// state cannot be decoded classifies as unknown and is WRITTEN as such rather
// than skipped — "we do not know" is a phase, and it can then be counted.
//
// Written once here because both backends need it and the classification is
// the same in both; the dialect differences (the tenant filter, the
// placeholders) are the Execer's.
func ReclassifyPhases(ctx context.Context, db Execer, scope string) (int, error) {
	tenant, targs := db.TenantFilter("", scope)

	type todoRow struct {
		id    int64
		phase domain.GamePhase
	}
	var todo []todoRow
	if err := func() error {
		rows, err := db.Query(ctx,
			`SELECT id, state, game_phase FROM position WHERE `+tenant+` ORDER BY id`, targs...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id int64
			var state string
			var stored int
			if err := rows.Scan(&id, &state, &stored); err != nil {
				return err
			}
			if want := PhaseOfState(state); int(want) != stored {
				todo = append(todo, todoRow{id, want})
			}
		}
		return rows.Err()
	}(); err != nil {
		return 0, errf(db, "classify the phase of the stored positions", err)
	}
	if len(todo) == 0 {
		return 0, nil
	}

	// One transaction: a crash mid-way leaves the database as it was, and the
	// next run picks the whole batch up again.
	err := db.Transact(ctx, func(tx Execer) error {
		ttenant, ttargs := tx.TenantFilter("", scope)
		for _, r := range todo {
			args := append([]any{int(r.phase), r.id}, ttargs...)
			if _, err := tx.Exec(ctx,
				`UPDATE position SET game_phase = ? WHERE id = ? AND `+ttenant, args...); err != nil {
				return fmt.Errorf("position %d: %w", r.id, err)
			}
		}
		return nil
	})
	if err != nil {
		return 0, errf(db, "classify the phase of the stored positions", err)
	}
	return len(todo), nil
}

// PhaseOfState classifies the board held by a stored `state` value, in either
// encoding the column has carried: the compact array of 28 signed integers
// (2.2.0 onwards) and the legacy full-Position JSON before it.
func PhaseOfState(state string) domain.GamePhase {
	var p domain.Position
	switch {
	case engine.IsCompactState(state):
		p.Board = engine.DecodeBoardCompact(state)
	case state != "":
		if err := json.Unmarshal([]byte(state), &p); err != nil {
			return domain.PhaseUnknown
		}
	default:
		return domain.PhaseUnknown
	}
	return engine.ClassifyGamePhase(&p)
}
