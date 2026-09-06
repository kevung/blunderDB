package sqlshared

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// ReclassifyDerived rewrites the two DERIVED labels of every position whose
// stored value disagrees with what the classifiers say today — game_phase
// (issue #264, ADR-0035) and game_type (issue #291) — and returns how many
// rows changed.
//
// This is what makes both a derived label rather than stored data: change a
// classifier or one of its named thresholds, run this, and every row agrees
// with the new rule. Nothing else in the database reads the old value, so it
// can always be run and can always be run again.
//
// The phase reads only the board. The TYPE also reads the side on roll, since
// it names the plan of the player to move — hence the extra column in the
// query. A row whose state cannot be decoded classifies as unknown and is
// WRITTEN as such rather than skipped: "we do not know" is an answer, and it
// can then be counted.
//
// Written once here because both backends need it and the classification is
// the same in both; the dialect differences (the tenant filter, the
// placeholders) are the Execer's.
func ReclassifyDerived(ctx context.Context, db Execer, scope string) (int, error) {
	tenant, targs := db.TenantFilter("", scope)

	type todoRow struct {
		id       int64
		phase    domain.GamePhase
		gameType domain.GameType
	}
	var todo []todoRow
	if err := func() error {
		rows, err := db.Query(ctx,
			`SELECT id, state, player_on_roll, game_phase, game_type FROM position WHERE `+tenant+` ORDER BY id`, targs...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id int64
			var state string
			var onRoll, storedPhase, storedType *int64
			if err := rows.Scan(&id, &state, &onRoll, &storedPhase, &storedType); err != nil {
				return err
			}
			side := 0
			if onRoll != nil {
				side = int(*onRoll)
			}
			phase := PhaseOfState(state)
			gameType := TypeOfState(state, side)
			if int(phase) != derefInt64(storedPhase) || int(gameType) != derefInt64(storedType) {
				todo = append(todo, todoRow{id, phase, gameType})
			}
		}
		return rows.Err()
	}(); err != nil {
		return 0, errf(db, "classify the stored positions", err)
	}
	if len(todo) == 0 {
		return 0, nil
	}

	// One transaction: a crash mid-way leaves the database as it was, and the
	// next run picks the whole batch up again.
	err := db.Transact(ctx, func(tx Execer) error {
		ttenant, ttargs := tx.TenantFilter("", scope)
		for _, r := range todo {
			args := append([]any{int(r.phase), int(r.gameType), r.id}, ttargs...)
			if _, err := tx.Exec(ctx,
				`UPDATE position SET game_phase = ?, game_type = ? WHERE id = ? AND `+ttenant, args...); err != nil {
				return fmt.Errorf("position %d: %w", r.id, err)
			}
		}
		return nil
	})
	if err != nil {
		return 0, errf(db, "classify the stored positions", err)
	}
	return len(todo), nil
}

func derefInt64(p *int64) int {
	if p == nil {
		return 0
	}
	return int(*p)
}

// TypeOfState classifies the plan of play held by a stored `state` value and
// the side on roll, in either encoding the column has carried.
func TypeOfState(state string, playerOnRoll int) domain.GameType {
	p, ok := positionOfState(state)
	if !ok {
		return domain.TypeUnknown
	}
	p.PlayerOnRoll = playerOnRoll
	return engine.ClassifyGameType(&p)
}

func positionOfState(state string) (domain.Position, bool) {
	var p domain.Position
	switch {
	case engine.IsCompactState(state):
		p.Board = engine.DecodeBoardCompact(state)
	case state != "":
		if err := json.Unmarshal([]byte(state), &p); err != nil {
			return p, false
		}
	default:
		return p, false
	}
	return p, true
}

// PhaseOfState classifies the board held by a stored `state` value, in either
// encoding the column has carried: the compact array of 28 signed integers
// (2.2.0 onwards) and the legacy full-Position JSON before it.
func PhaseOfState(state string) domain.GamePhase {
	p, ok := positionOfState(state)
	if !ok {
		return domain.PhaseUnknown
	}
	return engine.ClassifyGamePhase(&p)
}
