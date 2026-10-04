package sqlshared

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// PlayerContrast — see storage.StatsStore.
func (s *StatsStore) PlayerContrast(ctx context.Context, scope, playerA, playerB string, filter storage.StatsFilter) (*storage.PlayerContrast, error) {
	if playerA == "" || playerB == "" || playerA == playerB {
		return nil, fmt.Errorf("player contrast needs two distinct players: %w", storage.ErrInvalid)
	}
	aliases, err := aliasMap(ctx, s.DB, scope, storage.AliasPlayer)
	if err != nil {
		return nil, errf(s.DB, "PlayerContrast aliases", err)
	}
	if aliases.Canonical(playerA) == aliases.Canonical(playerB) {
		return nil, fmt.Errorf("player contrast needs two distinct players: %w", storage.ErrInvalid)
	}
	settings, err := librarySettings(ctx, s.DB, scope)
	if err != nil {
		return nil, errf(s.DB, "PlayerContrast settings", err)
	}
	rowsOf := func(name string) ([]storage.ContrastRow, error) {
		f := filter
		f.PlayerName, f.PlayerAliases = name, nil
		f, err := s.withPlayerAliases(ctx, scope, f)
		if err != nil {
			return nil, err
		}
		where, args := s.buildStatsWhereClause(scope, f)
		// The per-play error of the move, not the position's analysis column:
		// two players answering one position differently are scored each on
		// their own play.
		rows, err := s.DB.Query(ctx,
			`SELECT p.id, `+s.DB.Bigint("MAX(mv.error_mp)")+`, COUNT(*) `+statsBaseJoin+where+
				` AND mv.error_mp IS NOT NULL GROUP BY p.id ORDER BY p.id`, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []storage.ContrastRow
		for rows.Next() {
			var r storage.ContrastRow
			if err := rows.Scan(&r.PositionID, &r.WorstMP, &r.Times); err != nil {
				return nil, err
			}
			out = append(out, r)
		}
		return out, rows.Err()
	}
	a, err := rowsOf(playerA)
	if err != nil {
		return nil, errf(s.DB, "PlayerContrast "+playerA, err)
	}
	b, err := rowsOf(playerB)
	if err != nil {
		return nil, errf(s.DB, "PlayerContrast "+playerB, err)
	}
	common, positions := storage.ContrastPlayers(a, b, settings.ErrorThresholdMP)
	return &storage.PlayerContrast{PlayerA: playerA, PlayerB: playerB, ThresholdMP: settings.ErrorThresholdMP,
		CommonPositions: common, Positions: positions}, nil
}
