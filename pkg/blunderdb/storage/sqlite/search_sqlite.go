package sqlite

import (
	"context"
	"database/sql"
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

// Find streams the positions matching f. It is a faithful port of the
// Database wrapper's LoadPositionsByFiltersCore: the cheap predicates are
// pushed to SQL, the rest are evaluated in Go on the narrowed result set.
func (s *searchStore) Find(ctx context.Context, scope string, f domain.SearchFilters) iter.Seq2[*domain.Position, error] {
	return func(yield func(*domain.Position, error) bool) {
		positions, err := s.find(ctx, f)
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

func (s *searchStore) find(ctx context.Context, f domain.SearchFilters) ([]domain.Position, error) {
	useSQLFilters := !f.MirrorFilter

	// The decoded analysis is consumed only by the move-pattern filter and the
	// Go-side analysis re-checks of mirror search; every other analysis filter
	// runs on the denormalised SQL columns. So decode the (zlib-compressed) blob
	// per row only when one of those paths needs it — a no-move-pattern,
	// non-mirror search skips the decompress+unmarshal of every result row.
	needAnalysis := f.MovePatternFilter != "" || f.MirrorFilter

	// On points shared with the exclusion structure, "Except" wins over "At least":
	// clear those points from the include filter so the two are not contradictory.
	effInclude := domain.EffectiveIncludeFilter(f.Filter, f.ExcludeFilter)

	var where strings.Builder
	var args []any
	where.WriteString("1=1")

	// Provenance is a property of the row, not of the board, so mirroring a
	// position cannot change it: this one filter stays in SQL even in mirror
	// search, where every board filter falls back to the Go phase.
	if f.IndividuallyImportedFilter {
		where.WriteString(" AND p.individually_imported = 1")
	}

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

	// Player filter: keep positions that occur in any match where the named
	// player sat at either seat. LIKE (no wildcards) gives case-insensitive
	// exact matching for ASCII names, mirroring the match-id subquery shape.
	if f.PlayerFilter != "" {
		where.WriteString(
			" AND p.id IN (SELECT mv.position_id FROM move mv" +
				" JOIN game g ON mv.game_id = g.id" +
				" JOIN match mt ON g.match_id = mt.id" +
				" WHERE mt.player1_name LIKE ? OR mt.player2_name LIKE ?)")
		args = append(args, f.PlayerFilter, f.PlayerFilter)
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

	// User-facing position-id filter (command-line token `id`). Uses the same
	// list/range semantics as the match/tournament filters (e.g. "2,7" is the
	// range 2..7; ";"-joined values are an explicit list).
	if f.PositionIDsFilter != "" {
		ids, err := parseFilterIDList(f.PositionIDsFilter)
		if err == nil && len(ids) > 0 {
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
			// Cube sub-type: distinguish double/no-double from take/pass responses.
			if f.Filter.DecisionType == domain.CubeAction {
				switch f.CubeResponseFilter {
				case "double":
					where.WriteString(" AND p.is_cube_response = 0")
				case "takepass":
					where.WriteString(" AND p.is_cube_response = 1")
				}
			}
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
		// Except-dice (xD65): exclude positions rolled with any of the listed rolls,
		// each in either order. Unscoped by on-roll/decision-type — a roll is a roll
		// whoever holds it; cube decisions (dice 0-0) never match, so they survive.
		for _, pair := range domain.ParseExceptDice(f.ExceptDiceFilter) {
			where.WriteString(" AND NOT ((p.dice_1 = ? AND p.dice_2 = ?) OR (p.dice_1 = ? AND p.dice_2 = ?))")
			args = append(args, pair[0], pair[1], pair[1], pair[0])
		}
		if f.IncludeCube {
			if f.Filter.Cube.Value == 0 {
				where.WriteString(" AND p.cube_value IS NULL")
			} else if f.DecisionTypeFilter && f.CubeResponseFilter == "takepass" {
				// A take/pass offered cube is always centered (owner -1); the board
				// can't build a centered value>1 cube, so match the centered owner.
				where.WriteString(" AND p.cube_value = ? AND p.cube_owner = -1")
				args = append(args, f.Filter.Cube.Value)
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
			where.WriteString(" AND p.no_contact = 1")
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
			errExpr := statsErrExpr
			if eHasMin && eHasMax {
				where.WriteString(" AND " + errExpr + " BETWEEN ? AND ?")
				args = append(args, eqMin, eqMax)
			} else if eHasMin {
				where.WriteString(" AND " + errExpr + " >= ?")
				args = append(args, eqMin)
			} else if eHasMax {
				where.WriteString(" AND " + errExpr + " <= ?")
				args = append(args, eqMax)
			}
		}

		if hasBoardFilter(effInclude.Board) {
			occ1Req, pt1Req, occ2Req, pt2Req, tight := engine.CheckerStructureMasks(effInclude)
			bitboardTight = tight
			where.WriteString(" AND (p.occupancy_1 & ?) = ? AND (p.point_mask_1 & ?) = ?")
			where.WriteString(" AND (p.occupancy_2 & ?) = ? AND (p.point_mask_2 & ?) = ?")
			args = append(args,
				int64(occ1Req), int64(occ1Req), int64(pt1Req), int64(pt1Req),
				int64(occ2Req), int64(occ2Req), int64(pt2Req), int64(pt2Req))
		}

		// Exclusion structure ("Sauf"): drop positions that contain ANY of the
		// excluded elements (OR semantics across points). Keep a position only when
		// none of its points match an excluded element. Template points with >2
		// checkers are not representable as bitmasks and are left to the Go-side
		// check (Position.ContainsAnyCheckerOf) below.
		if hasBoardFilter(f.ExcludeFilter.Board) {
			eSingle1, eMade1, eSingle2, eMade2 := engine.ExclusionMasks(f.ExcludeFilter)
			where.WriteString(" AND (p.occupancy_1 & ?) = 0 AND (p.point_mask_1 & ?) = 0")
			where.WriteString(" AND (p.occupancy_2 & ?) = 0 AND (p.point_mask_2 & ?) = 0")
			args = append(args,
				int64(eSingle1), int64(eMade1), int64(eSingle2), int64(eMade2))
		}
	}

	query := `SELECT p.id, p.state,
		p.decision_type, p.player_on_roll, p.dice_1, p.dice_2,
		p.cube_value, p.cube_owner, p.score_1, p.score_2,
		p.has_jacoby, p.has_beaver, p.is_cube_response,
		a.id, a.data,
		a.cube_error, a.best_move_equity_error,
		a.player1_win_rate, a.player1_gammon_rate, a.player1_backgammon_rate,
		a.player2_win_rate, a.player2_gammon_rate, a.player2_backgammon_rate,
		a.best_cube_action
	FROM position p
	LEFT JOIN analysis a ON a.position_id = p.id
	WHERE ` + where.String() + ` ORDER BY ` + domain.SearchOrderByClause(f.Sort)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite: search query: %w", err)
	}
	defer rows.Close()

	var positions []domain.Position

	for rows.Next() {
		var posID int64
		var posJSON string
		var pDT, pPOR, pD1, pD2, pCV, pCO, pS1, pS2, pHJ, pHB sql.NullInt64
		var pICR sql.NullInt64
		var anaID sql.NullInt64
		var anaJSON sql.NullString
		var cubeError, moveError sql.NullFloat64
		var p1Win, p1Gammon, p1BG, p2Win, p2Gammon, p2BG sql.NullFloat64
		var bestCubeAction sql.NullString

		if err := rows.Scan(
			&posID, &posJSON,
			&pDT, &pPOR, &pD1, &pD2, &pCV, &pCO, &pS1, &pS2, &pHJ, &pHB, &pICR,
			&anaID, &anaJSON,
			&cubeError, &moveError,
			&p1Win, &p1Gammon, &p1BG,
			&p2Win, &p2Gammon, &p2BG,
			&bestCubeAction,
		); err != nil {
			return nil, fmt.Errorf("sqlite: search scan: %w", err)
		}

		position := engine.ReconstructPosition(posID, posJSON,
			int(pDT.Int64), int(pPOR.Int64), int(pD1.Int64), int(pD2.Int64),
			int(pCV.Int64), int(pCO.Int64), int(pS1.Int64), int(pS2.Int64),
			int(pHJ.Int64), int(pHB.Int64))

		var ana *domain.PositionAnalysis
		if needAnalysis && anaID.Valid && anaJSON.Valid && anaJSON.String != "" {
			// a.data is stored zlib-compressed (engine.EncodeAnalysisForStorage),
			// so it must be decompressed before unmarshalling — a plain
			// json.Unmarshal of the raw bytes silently fails (leaving ana nil),
			// which broke the analysis-dependent Go-side filters (move pattern,
			// and the win/gammon/equity fallbacks used by mirror search).
			if a, decErr := engine.DecodeAnalysisFromStorage([]byte(anaJSON.String)); decErr == nil {
				ana = &a
			}
		}

		matchesGoFilters := func(pos domain.Position) bool {
			if hasBoardFilter(effInclude.Board) {
				if !useSQLFilters || bitboardTight {
					if !pos.MatchesCheckerPosition(effInclude) {
						return false
					}
				}
			}

			// Exclusion structure: reject positions that contain ANY excluded element
			// (authoritative; also covers template counts >2 the SQL mask skips).
			if hasBoardFilter(f.ExcludeFilter.Board) {
				if pos.ContainsAnyCheckerOf(f.ExcludeFilter) {
					return false
				}
			}

			if !useSQLFilters {
				if !pos.MatchesCheckerPosition(effInclude) {
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
				// Cube sub-type (take/pass vs double/no-double) lives in the
				// is_cube_response column, scanned separately above.
				if f.DecisionTypeFilter && f.Filter.DecisionType == domain.CubeAction {
					isResp := pICR.Int64 == 1
					if f.CubeResponseFilter == "double" && isResp {
						return false
					}
					if f.CubeResponseFilter == "takepass" && !isResp {
						return false
					}
				}
				if f.DiceRollFilter && !pos.MatchesDiceRollMode(f.Filter, f.DiceRollMode) {
					return false
				}
				if f.ExceptDiceFilter != "" && !pos.MatchesExceptDice(domain.ParseExceptDice(f.ExceptDiceFilter)) {
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
		return nil, fmt.Errorf("sqlite: search rows: %w", err)
	}

	return positions, nil
}
