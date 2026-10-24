package sqlshared

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// pricedCubeActionCodes lists the codes of the fixed labels a cube error is
// read for: the active ones of activeCubeActionCodes and the no double.
var pricedCubeActionCodes = func() string {
	var codes []string
	for code, label := range domain.ActionLabels() {
		if isActiveCubeLabel(label) || engine.CanonicalCubeAction(label) == engine.CubeNoDouble {
			codes = append(codes, strconv.Itoa(code))
		}
	}
	return strings.Join(codes, ", ")
}()

// contrastErrExpr is a move's error as the contrast judges it:
// move.decision_error_mp, except on a cube decision whose recorded action
// names none of pricedCubeActionCodes (empty, or a label no importer could
// map). The per-move column prices those 0 for the statistics, which only
// count active cube actions; read here, that 0 would pass the player for
// having played the position well. They are unscored instead.
const contrastErrExpr = `CASE WHEN p.decision_type = 1 AND (mv.cube_action IS NULL OR mv.cube_action NOT IN (` + "%s" + `)) THEN NULL ELSE mv.decision_error_mp END`

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
	// unscored counts the analysed plays the contrast cannot judge: those
	// no candidate of the analysis names, and the cube decisions
	// contrastErrExpr leaves unpriced.
	unscored := 0
	errExpr := fmt.Sprintf(contrastErrExpr, pricedCubeActionCodes)
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
			`SELECT p.id, `+s.DB.Bigint("COALESCE(MAX("+errExpr+"), -1)")+`, COUNT(`+errExpr+`), `+
				`SUM(CASE WHEN (`+errExpr+`) IS NULL THEN 1 ELSE 0 END) `+statsBaseJoin+where+
				` GROUP BY p.id ORDER BY p.id`, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []storage.ContrastRow
		for rows.Next() {
			var r storage.ContrastRow
			var missing int
			if err := rows.Scan(&r.PositionID, &r.WorstMP, &r.Times, &missing); err != nil {
				return nil, err
			}
			unscored += missing
			if r.Times > 0 {
				out = append(out, r)
			}
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
		CommonPositions: common, Positions: positions, UnscoredMoves: unscored}, nil
}
