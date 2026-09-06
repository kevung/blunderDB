package cli

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// Tabular export of a database's contents (issue #280, fiche I.24).
//
// # Why CSV, and why not Parquet
//
// The fiche listed Parquet "as an option". It is not here, and the reason is
// measured rather than principled: a columnar library is several megabytes of
// dependency in a binary this project spends real effort keeping small (see
// tasks/taille-binaire-2026-09.md), and everything the export is FOR reads CSV
// in one line — `pd.read_csv`, `polars.read_csv`, R's `read.csv`, a
// spreadsheet. Parquet earns its keep on tens of millions of rows; a
// backgammon library that has been going for ten years holds a hundred
// thousand positions. If somebody one day has a database where the difference
// is measurable, that measurement is what should reopen this.
//
// # What the columns are
//
// A CONTRACT. A notebook, a script or a spreadsheet written against these
// names must keep working, so columns are added at the end and never renamed
// or reordered. Every equity is in millipoints (integer), because that is how
// they are stored and because a float in a CSV invites a locale to reformat
// it.

// positionColumns is the header of `list --type positions --format csv`.
var positionColumns = []string{
	"position_id", "xgid", "decision_type", "phase",
	"score_1", "score_2", "cube_value", "cube_owner", "player_on_roll",
	"pip_1", "pip_2", "flagged", "individually_imported",
	"best_cube_action", "cube_error_mp", "played_move_error_mp",
	"is_forced", "is_close_cube",
}

// moveColumns is the header of `list --type moves --format csv`. One row per
// recorded move: this is the table that answers "what did I actually do", the
// one the positions table cannot answer because a position is deduplicated
// across the matches it appears in.
var moveColumns = []string{
	"move_id", "game_id", "game_number", "match_id", "match_date",
	"player1", "player2", "match_length",
	"move_number", "move_type", "position_id", "player",
	"dice_1", "dice_2", "checker_move", "cube_action", "luck_mp",
}

// analysisColumns is the header of `list --type analyses --format csv`.
var analysisColumns = []string{
	"position_id", "engine", "depth", "analysis_type",
	"best_move", "best_move_equity_mp", "played_move_error_mp",
	"best_cube_action", "cube_error_mp",
	"player_win_rate", "player_gammon_rate", "player_backgammon_rate",
	"opponent_win_rate", "opponent_gammon_rate", "opponent_backgammon_rate",
}

// exportPositionsCSV writes one row per position.
func (cli *CLI) exportPositionsCSV(limit int) error {
	positions, err := cli.db.LoadAllPositions()
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()
	if err := w.Write(positionColumns); err != nil {
		return err
	}
	for i, pos := range positions {
		if limit > 0 && i >= limit {
			break
		}
		p := pos
		pipBlack, pipWhite := p.ComputePipCounts()
		pip1, pip2 := strconv.Itoa(pipBlack), strconv.Itoa(pipWhite)
		cols := engine.PopulateAnalysisColumns(nil, "", "")
		if a, err := cli.db.LoadAnalysis(p.ID); err == nil && a != nil {
			cols = engine.PopulateAnalysisColumns(a, firstPlayed(a.PlayedMoves), firstPlayed(a.PlayedCubeActions))
		}
		if err := w.Write([]string{
			strconv.FormatInt(p.ID, 10),
			xgidOf(&p),
			decisionTypeToken(p.DecisionType),
			engine.ClassifyGamePhase(&p).String(),
			strconv.Itoa(p.Score[0]), strconv.Itoa(p.Score[1]),
			strconv.Itoa(p.Cube.Value), colorToken(p.Cube.Owner),
			strconv.Itoa(p.PlayerOnRoll),
			pip1, pip2,
			boolToken(p.Flagged), boolToken(p.IndividuallyImported),
			cols.BestCubeAction,
			strconv.FormatInt(cols.CubeError, 10),
			strconv.FormatInt(cols.BestMoveEquityError, 10),
			strconv.FormatInt(cols.IsForced, 10),
			strconv.FormatInt(cols.IsCloseCube, 10),
		}); err != nil {
			return err
		}
	}
	return nil
}

// exportMovesCSV writes one row per recorded move, with the match it belongs
// to spelt out on every row: a table that has to be joined before it says
// anything is a table nobody uses.
func (cli *CLI) exportMovesCSV(limit int) error {
	matches, err := cli.db.GetAllMatches()
	if err != nil {
		return fmt.Errorf("failed to get matches: %w", err)
	}
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()
	if err := w.Write(moveColumns); err != nil {
		return err
	}
	written := 0
	for _, m := range matches {
		games, err := cli.db.GetGamesByMatch(m.ID)
		if err != nil {
			return fmt.Errorf("match %d: %w", m.ID, err)
		}
		for _, g := range games {
			moves, err := cli.db.GetMovesByGame(g.ID)
			if err != nil {
				return fmt.Errorf("game %d: %w", g.ID, err)
			}
			for _, mv := range moves {
				if limit > 0 && written >= limit {
					return nil
				}
				luck := ""
				if mv.LuckMP != nil {
					luck = strconv.FormatInt(int64(*mv.LuckMP), 10)
				}
				if err := w.Write([]string{
					strconv.FormatInt(mv.ID, 10),
					strconv.FormatInt(mv.GameID, 10),
					strconv.FormatInt(int64(g.GameNumber), 10),
					strconv.FormatInt(m.ID, 10),
					m.MatchDate.Format("2006-01-02"),
					m.Player1Name, m.Player2Name,
					strconv.FormatInt(int64(m.MatchLength), 10),
					strconv.FormatInt(int64(mv.MoveNumber), 10),
					mv.MoveType,
					strconv.FormatInt(mv.PositionID, 10),
					strconv.FormatInt(int64(mv.Player), 10),
					strconv.FormatInt(int64(mv.Dice[0]), 10), strconv.FormatInt(int64(mv.Dice[1]), 10),
					mv.CheckerMove, mv.CubeAction, luck,
				}); err != nil {
					return err
				}
				written++
			}
		}
	}
	return nil
}

// exportAnalysesCSV writes one row per stored analysis.
func (cli *CLI) exportAnalysesCSV(limit int) error {
	positions, err := cli.db.LoadAllPositions()
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()
	if err := w.Write(analysisColumns); err != nil {
		return err
	}
	written := 0
	for _, pos := range positions {
		if limit > 0 && written >= limit {
			break
		}
		a, err := cli.db.LoadAnalysis(pos.ID)
		if err != nil || a == nil {
			continue
		}
		cols := engine.PopulateAnalysisColumns(a, firstPlayed(a.PlayedMoves), firstPlayed(a.PlayedCubeActions))

		bestMove, bestEquity := "", ""
		engineName, depth := "", ""
		if ca := a.CheckerAnalysis; ca != nil && len(ca.Moves) > 0 {
			bestMove = ca.Moves[0].Move
			bestEquity = strconv.FormatInt(int64(ca.Moves[0].Equity*1000), 10)
			engineName, depth = ca.Moves[0].AnalysisEngine, ca.Moves[0].AnalysisDepth
		}
		if dca := a.DoublingCubeAnalysis; dca != nil {
			if engineName == "" {
				engineName, depth = dca.AnalysisEngine, dca.AnalysisDepth
			}
		}
		if err := w.Write([]string{
			strconv.FormatInt(pos.ID, 10),
			engineName, depth, a.AnalysisType,
			bestMove, bestEquity,
			strconv.FormatInt(cols.BestMoveEquityError, 10),
			cols.BestCubeAction,
			strconv.FormatInt(cols.CubeError, 10),
			strconv.FormatInt(cols.Player1WinRate, 10),
			strconv.FormatInt(cols.Player1GammonRate, 10),
			strconv.FormatInt(cols.Player1BackgammonRate, 10),
			strconv.FormatInt(cols.Player2WinRate, 10),
			strconv.FormatInt(cols.Player2GammonRate, 10),
			strconv.FormatInt(cols.Player2BackgammonRate, 10),
		}); err != nil {
			return err
		}
		written++
	}
	return nil
}

// firstPlayed is the played-action convention the storage layer uses: the
// first non-empty entry (engine.PlayedActionsFor's primary half). Restated
// here rather than exported, because the export must show the SAME number the
// statistics do and reaching for a different rule would be how they drift.
func firstPlayed(ss []string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

func decisionTypeToken(t int) string {
	if t == domain.CubeAction {
		return "cube"
	}
	return "checker"
}

func colorToken(c int) string {
	switch c {
	case domain.Black:
		return "player1"
	case domain.White:
		return "player2"
	default:
		return "centre"
	}
}

func boolToken(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
