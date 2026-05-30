package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"math"
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type searchStore struct{ db execer }

var _ storage.SearchStore = (*searchStore)(nil)

// statsErrExpr is the SQL CASE expression that selects the correct error
// column based on a position's decision type.
const statsErrExpr = "CASE WHEN p.decision_type = 1 THEN a.cube_error ELSE a.best_move_equity_error END"

// Find streams the positions matching f. It is a faithful port of the SQLite
// backend's search: the cheap predicates are pushed to SQL, the rest are
// evaluated in Go on the narrowed result set. Results are restricted to the
// scope's tenant.
func (s *searchStore) Find(ctx context.Context, scope string, f domain.SearchFilters) iter.Seq2[*domain.Position, error] {
	return func(yield func(*domain.Position, error) bool) {
		positions, err := s.find(ctx, tenantID(scope), f)
		if err != nil {
			yield(nil, err)
			return
		}
		for i := range positions {
			if !yield(&positions[i], nil) {
				return
			}
		}
	}
}

func (s *searchStore) find(ctx context.Context, tenant int64, f domain.SearchFilters) ([]domain.Position, error) {
	useSQLFilters := !f.MirrorFilter

	var where strings.Builder
	var args []any
	where.WriteString("p.tenant_id = ?")
	args = append(args, tenant)

	if f.MatchIDsFilter != "" || f.TournamentIDsFilter != "" {
		var allMatchIDs []int64
		if f.MatchIDsFilter != "" {
			if ids, err := parseFilterIDList(f.MatchIDsFilter); err == nil {
				allMatchIDs = append(allMatchIDs, ids...)
			}
		}
		if f.TournamentIDsFilter != "" {
			if tIDs, err := parseFilterIDList(f.TournamentIDsFilter); err == nil {
				for _, tID := range tIDs {
					if matchIDs, err := getMatchIDsForTournament(ctx, s.db, tID); err == nil {
						allMatchIDs = append(allMatchIDs, matchIDs...)
					}
				}
			}
		}
		if len(allMatchIDs) > 0 {
			placeholders := strings.Repeat("?,", len(allMatchIDs))
			placeholders = placeholders[:len(placeholders)-1]
			where.WriteString(
				" AND p.id IN (SELECT m.position_id FROM move m" +
					" WHERE m.game_id IN (SELECT id FROM game WHERE match_id IN (" + placeholders + ")))")
			for _, id := range allMatchIDs {
				args = append(args, id)
			}
		} else {
			where.WriteString(" AND 0=1")
		}
	}

	if f.RestrictToPositionIDs != "" {
		var ids []int64
		for _, idStr := range strings.Split(f.RestrictToPositionIDs, ",") {
			if id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64); err == nil {
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			placeholders := strings.Repeat("?,", len(ids))
			placeholders = placeholders[:len(placeholders)-1]
			where.WriteString(" AND p.id IN (" + placeholders + ")")
			for _, id := range ids {
				args = append(args, id)
			}
		} else {
			where.WriteString(" AND 0=1")
		}
	}

	var bitboardTight bool
	if useSQLFilters {
		if f.DecisionTypeFilter {
			where.WriteString(" AND p.decision_type = ? AND p.player_on_roll = ?")
			args = append(args, f.Filter.DecisionType, f.Filter.PlayerOnRoll)
		}
		if f.DiceRollFilter {
			if f.DiceRollMode == "first" {
				where.WriteString(" AND (p.dice_1 = ? OR p.dice_2 = ?) AND p.player_on_roll = ? AND p.decision_type = ?")
				args = append(args, f.Filter.Dice[0], f.Filter.Dice[0], f.Filter.PlayerOnRoll, f.Filter.DecisionType)
			} else {
				d1, d2 := f.Filter.Dice[0], f.Filter.Dice[1]
				if d1 == d2 {
					where.WriteString(" AND p.dice_1 = ? AND p.dice_2 = ? AND p.player_on_roll = ? AND p.decision_type = ?")
					args = append(args, d1, d2, f.Filter.PlayerOnRoll, f.Filter.DecisionType)
				} else {
					where.WriteString(" AND ((p.dice_1 = ? AND p.dice_2 = ?) OR (p.dice_1 = ? AND p.dice_2 = ?)) AND p.player_on_roll = ? AND p.decision_type = ?")
					args = append(args, d1, d2, d2, d1, f.Filter.PlayerOnRoll, f.Filter.DecisionType)
				}
			}
		}
		if f.IncludeCube {
			if f.Filter.Cube.Value == 0 {
				where.WriteString(" AND p.cube_value IS NULL")
			} else {
				where.WriteString(" AND p.cube_value = ? AND p.cube_owner = ?")
				args = append(args, f.Filter.Cube.Value, f.Filter.Cube.Owner)
			}
		}
		if f.IncludeScore {
			where.WriteString(" AND p.score_1 = ? AND p.score_2 = ?")
			args = append(args, f.Filter.Score[0], f.Filter.Score[1])
		}
		if f.NoContactFilter {
			where.WriteString(" AND p.no_contact IS TRUE")
		}

		pMin, pMax, pHasMin, pHasMax := parseIntFilterExpr(f.PipCountFilter, "p")
		appendIntRangeSQL("p.pip_diff", pMin, pMax, pHasMin, pHasMax, &where, &args)
		PMin, PMax, PHasMin, PHasMax := parseIntFilterExpr(f.Player1AbsolutePipCountFilter, "P")
		appendIntRangeSQL("p.pip_1", PMin, PMax, PHasMin, PHasMax, &where, &args)
		oMin, oMax, oHasMin, oHasMax := parseIntFilterExpr(f.Player1CheckerOffFilter, "o")
		appendIntRangeSQL("p.off_1", oMin, oMax, oHasMin, oHasMax, &where, &args)
		OMin, OMax, OHasMin, OHasMax := parseIntFilterExpr(f.Player2CheckerOffFilter, "O")
		appendIntRangeSQL("p.off_2", OMin, OMax, OHasMin, OHasMax, &where, &args)
		kMin, kMax, kHasMin, kHasMax := parseIntFilterExpr(f.Player1BackCheckerFilter, "k")
		appendIntRangeSQL("p.back_checkers_1", kMin, kMax, kHasMin, kHasMax, &where, &args)
		KMin, KMax, KHasMin, KHasMax := parseIntFilterExpr(f.Player2BackCheckerFilter, "K")
		appendIntRangeSQL("p.back_checkers_2", KMin, KMax, KHasMin, KHasMax, &where, &args)

		wMin, wMax, wHasMin, wHasMax := parseFloatFilterExpr(f.WinRateFilter, "w")
		appendIntRangeSQL("a.player1_win_rate", int(math.Round(wMin*100)), int(math.Round(wMax*100)), wHasMin, wHasMax, &where, &args)
		gMin, gMax, gHasMin, gHasMax := parseFloatFilterExpr(f.GammonRateFilter, "g")
		appendIntRangeSQL("a.player1_gammon_rate", int(math.Round(gMin*100)), int(math.Round(gMax*100)), gHasMin, gHasMax, &where, &args)
		bMin, bMax, bHasMin, bHasMax := parseFloatFilterExpr(f.BackgammonRateFilter, "b")
		appendIntRangeSQL("a.player1_backgammon_rate", int(math.Round(bMin*100)), int(math.Round(bMax*100)), bHasMin, bHasMax, &where, &args)
		WMin, WMax, WHasMin, WHasMax := parseFloatFilterExpr(f.Player2WinRateFilter, "W")
		appendIntRangeSQL("a.player2_win_rate", int(math.Round(WMin*100)), int(math.Round(WMax*100)), WHasMin, WHasMax, &where, &args)
		GMin, GMax, GHasMin, GHasMax := parseFloatFilterExpr(f.Player2GammonRateFilter, "G")
		appendIntRangeSQL("a.player2_gammon_rate", int(math.Round(GMin*100)), int(math.Round(GMax*100)), GHasMin, GHasMax, &where, &args)
		BMin, BMax, BHasMin, BHasMax := parseFloatFilterExpr(f.Player2BackgammonRateFilter, "B")
		appendIntRangeSQL("a.player2_backgammon_rate", int(math.Round(BMin*100)), int(math.Round(BMax*100)), BHasMin, BHasMax, &where, &args)

		if f.MoveErrorFilter != "" {
			eMin, eMax, eHasMin, eHasMax := parseFloatFilterExpr(f.MoveErrorFilter, "E")
			eqMin := int(math.Round(eMin))
			eqMax := int(math.Round(eMax))
			if eHasMin && eHasMax {
				where.WriteString(" AND " + statsErrExpr + " BETWEEN ? AND ?")
				args = append(args, eqMin, eqMax)
			} else if eHasMin {
				where.WriteString(" AND " + statsErrExpr + " >= ?")
				args = append(args, eqMin)
			} else if eHasMax {
				where.WriteString(" AND " + statsErrExpr + " <= ?")
				args = append(args, eqMax)
			}
		}

		if hasBoardFilter(f.Filter.Board) {
			occ1Req, pt1Req, occ2Req, pt2Req, tight := engine.CheckerStructureMasks(f.Filter)
			bitboardTight = tight
			where.WriteString(" AND (p.occupancy_1 & ?) = ? AND (p.point_mask_1 & ?) = ?")
			where.WriteString(" AND (p.occupancy_2 & ?) = ? AND (p.point_mask_2 & ?) = ?")
			args = append(args,
				int64(occ1Req), int64(occ1Req), int64(pt1Req), int64(pt1Req),
				int64(occ2Req), int64(occ2Req), int64(pt2Req), int64(pt2Req))
		}
	}

	query := `SELECT p.id, p.state,
		p.decision_type, p.player_on_roll, p.dice_1, p.dice_2,
		p.cube_value, p.cube_owner, p.score_1, p.score_2,
		p.has_jacoby, p.has_beaver,
		a.id, a.data
	FROM position p
	LEFT JOIN analysis a ON a.position_id = p.id
	WHERE ` + where.String() + ` ORDER BY p.id`

	rows, err := s.db.Query(ctx, rebind(query), args...)
	if err != nil {
		return nil, fmt.Errorf("postgres: search query: %w", err)
	}
	defer rows.Close()

	var positions []domain.Position

	for rows.Next() {
		var posID int64
		var posState string
		var pDT, pPOR, pD1, pD2, pCV, pCO, pS1, pS2 *int64
		var pHJ, pHB *bool
		var anaID *int64
		var anaData []byte

		if err := rows.Scan(
			&posID, &posState,
			&pDT, &pPOR, &pD1, &pD2, &pCV, &pCO, &pS1, &pS2, &pHJ, &pHB,
			&anaID, &anaData,
		); err != nil {
			return nil, fmt.Errorf("postgres: search scan: %w", err)
		}

		position := engine.ReconstructPosition(posID, posState,
			derefInt(pDT), derefInt(pPOR), derefInt(pD1), derefInt(pD2),
			derefInt(pCV), derefInt(pCO), derefInt(pS1), derefInt(pS2),
			boolToIntPtr(pHJ), boolToIntPtr(pHB))

		var ana *domain.PositionAnalysis
		if anaID != nil && len(anaData) > 0 {
			var a domain.PositionAnalysis
			if jsonErr := json.Unmarshal(anaData, &a); jsonErr == nil {
				ana = &a
			}
		}

		matchesGoFilters := func(pos domain.Position) bool {
			if hasBoardFilter(f.Filter.Board) {
				if !useSQLFilters || bitboardTight {
					if !pos.MatchesCheckerPosition(f.Filter) {
						return false
					}
				}
			}

			if !useSQLFilters {
				if !pos.MatchesCheckerPosition(f.Filter) {
					return false
				}
				if f.IncludeCube && !pos.MatchesCubePosition(f.Filter) {
					return false
				}
				if f.IncludeScore && !pos.MatchesScorePosition(f.Filter) {
					return false
				}
				if f.DecisionTypeFilter && !pos.MatchesDecisionType(f.Filter) {
					return false
				}
				if f.DiceRollFilter && !pos.MatchesDiceRollMode(f.Filter, f.DiceRollMode) {
					return false
				}
				if f.NoContactFilter && !pos.MatchesNoContact() {
					return false
				}
				if f.PipCountFilter != "" && !pos.MatchesPipCountFilter(f.PipCountFilter) {
					return false
				}
				if f.Player1AbsolutePipCountFilter != "" && !pos.MatchesPlayer1AbsolutePipCount(f.Player1AbsolutePipCountFilter) {
					return false
				}
				if f.Player1CheckerOffFilter != "" && !pos.MatchesPlayer1CheckerOff(f.Player1CheckerOffFilter) {
					return false
				}
				if f.Player2CheckerOffFilter != "" && !pos.MatchesPlayer2CheckerOff(f.Player2CheckerOffFilter) {
					return false
				}
				if f.Player1BackCheckerFilter != "" && !pos.MatchesPlayer1BackChecker(f.Player1BackCheckerFilter) {
					return false
				}
				if f.Player2BackCheckerFilter != "" && !pos.MatchesPlayer2BackChecker(f.Player2BackCheckerFilter) {
					return false
				}
				if f.WinRateFilter != "" {
					if ana == nil {
						return false
					}
					var wr float64
					if ana.DoublingCubeAnalysis != nil {
						wr = ana.DoublingCubeAnalysis.PlayerWinChances
					} else if ana.CheckerAnalysis != nil && len(ana.CheckerAnalysis.Moves) > 0 {
						wr = ana.CheckerAnalysis.Moves[0].PlayerWinChance
					} else {
						return false
					}
					if !analysisMatchesFloatFilter(f.WinRateFilter, "w", wr) {
						return false
					}
				}
				if f.GammonRateFilter != "" {
					if ana == nil {
						return false
					}
					var gr float64
					if ana.DoublingCubeAnalysis != nil {
						gr = ana.DoublingCubeAnalysis.PlayerGammonChances
					} else if ana.CheckerAnalysis != nil && len(ana.CheckerAnalysis.Moves) > 0 {
						gr = ana.CheckerAnalysis.Moves[0].PlayerGammonChance
					} else {
						return false
					}
					if !analysisMatchesFloatFilter(f.GammonRateFilter, "g", gr) {
						return false
					}
				}
				if f.BackgammonRateFilter != "" {
					if ana == nil {
						return false
					}
					var bgr float64
					if ana.DoublingCubeAnalysis != nil {
						bgr = ana.DoublingCubeAnalysis.PlayerBackgammonChances
					} else if ana.CheckerAnalysis != nil && len(ana.CheckerAnalysis.Moves) > 0 {
						bgr = ana.CheckerAnalysis.Moves[0].PlayerBackgammonChance
					} else {
						return false
					}
					if !analysisMatchesFloatFilter(f.BackgammonRateFilter, "b", bgr) {
						return false
					}
				}
				if f.Player2WinRateFilter != "" {
					if ana == nil {
						return false
					}
					var wr float64
					if ana.DoublingCubeAnalysis != nil {
						wr = ana.DoublingCubeAnalysis.OpponentWinChances
					} else if ana.CheckerAnalysis != nil && len(ana.CheckerAnalysis.Moves) > 0 {
						wr = ana.CheckerAnalysis.Moves[0].OpponentWinChance
					} else {
						return false
					}
					if !analysisMatchesFloatFilter(f.Player2WinRateFilter, "W", wr) {
						return false
					}
				}
				if f.Player2GammonRateFilter != "" {
					if ana == nil {
						return false
					}
					var gr float64
					if ana.DoublingCubeAnalysis != nil {
						gr = ana.DoublingCubeAnalysis.OpponentGammonChances
					} else if ana.CheckerAnalysis != nil && len(ana.CheckerAnalysis.Moves) > 0 {
						gr = ana.CheckerAnalysis.Moves[0].OpponentGammonChance
					} else {
						return false
					}
					if !analysisMatchesFloatFilter(f.Player2GammonRateFilter, "G", gr) {
						return false
					}
				}
				if f.Player2BackgammonRateFilter != "" {
					if ana == nil {
						return false
					}
					var bgr float64
					if ana.DoublingCubeAnalysis != nil {
						bgr = ana.DoublingCubeAnalysis.OpponentBackgammonChances
					} else if ana.CheckerAnalysis != nil && len(ana.CheckerAnalysis.Moves) > 0 {
						bgr = ana.CheckerAnalysis.Moves[0].OpponentBackgammonChance
					} else {
						return false
					}
					if !analysisMatchesFloatFilter(f.Player2BackgammonRateFilter, "B", bgr) {
						return false
					}
				}
				if f.MoveErrorFilter != "" && !matchesMoveErrorFilter(ctx, s.db, &pos, f.MoveErrorFilter) {
					return false
				}
			}

			if f.Player1CheckerInZoneFilter != "" && !pos.MatchesPlayer1CheckerInZone(f.Player1CheckerInZoneFilter) {
				return false
			}
			if f.Player2CheckerInZoneFilter != "" && !pos.MatchesPlayer2CheckerInZone(f.Player2CheckerInZoneFilter) {
				return false
			}
			if f.Player1OutfieldBlotFilter != "" && !pos.MatchesPlayer1OutfieldBlot(f.Player1OutfieldBlotFilter) {
				return false
			}
			if f.Player2OutfieldBlotFilter != "" && !pos.MatchesPlayer2OutfieldBlot(f.Player2OutfieldBlotFilter) {
				return false
			}
			if f.Player1JanBlotFilter != "" && !pos.MatchesPlayer1JanBlot(f.Player1JanBlotFilter) {
				return false
			}
			if f.Player2JanBlotFilter != "" && !pos.MatchesPlayer2JanBlot(f.Player2JanBlotFilter) {
				return false
			}
			if f.SearchText != "" && !matchesSearchText(ctx, s.db, &pos, f.SearchText) {
				return false
			}
			if f.DateFilter != "" && !matchesDateFilter(ctx, s.db, &pos, f.DateFilter) {
				return false
			}
			if f.EquityFilter != "" && !analysisMatchesEquityFilter(f.EquityFilter, ana) {
				return false
			}
			return true
		}

		addPosition := func(pos domain.Position) {
			if f.MoveErrorFilter != "" && pos.DecisionType == domain.CubeAction && isPlayer1TakePassCubeAction(ctx, s.db, &pos) {
				pos = pos.Mirror()
			}
			positions = append(positions, pos)
		}

		if matchesGoFilters(position) {
			if analysisMatchesMovePattern(f.MovePatternFilter, ana) {
				addPosition(position)
			}
		} else if f.MirrorFilter {
			mirrored := position.Mirror()
			if matchesGoFilters(mirrored) {
				if analysisMatchesMovePattern(f.MovePatternFilter, ana) {
					addPosition(mirrored)
				}
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: search rows: %w", err)
	}

	return positions, nil
}

type searchHistoryStore struct{ db execer }

var _ storage.SearchHistoryStore = (*searchHistoryStore)(nil)

func (*searchHistoryStore) Save(context.Context, string, string, string) error {
	return notImpl("SearchHistory", "Save")
}
func (*searchHistoryStore) List(context.Context, string) iter.Seq2[*storage.SearchHistory, error] {
	return errSeq2[storage.SearchHistory](notImpl("SearchHistory", "List"))
}
func (*searchHistoryStore) DeleteEntry(context.Context, string, int64) error {
	return notImpl("SearchHistory", "DeleteEntry")
}
