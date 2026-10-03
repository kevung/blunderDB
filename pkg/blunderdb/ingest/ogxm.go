package ingest

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"path/filepath"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/gnubgparser"
	"github.com/kevung/ogxmparser"
	"github.com/kevung/ogxmparser/ogid"
)

// OGXM (HedgeHog / OpenGammon) import. The file stores moves, not positions:
// ogxmparser replays them (spec 5.4) and every position below is the one its
// replay derives, which its corpus holds to HedgeHog's reference
// implementation ply by ply.
//
// Colours: OGXM's White is blunderDB's White, its Black our Black, on the
// same absolute board (point 0 White's bar, 25 Black's; White's checkers
// start on 1, 12, 17 and 19, as in domain's opening board). blunderDB's
// player 1 is Black, so Player1Name is the OGXM Black name.
//
// Analysis: the first analysis block is read. Checker equities are taken as
// the block states them (cubeful, on the normalised scale at a match score),
// like GNU Backgammon's. Cube equities in a match are match winning chances
// and are converted to normalised equity with the same MET conversion as
// GNU Backgammon's, so a cube decision reads alike from both sources. A
// block whose checker currency is match winning chances is not imported:
// converting a play's MWC needs the post-move score swing, which the file
// does not carry.

// MapOGXM maps an .ogxm file into a backend-independent MatchGraph.
func MapOGXM(path string) (*MatchGraph, error) {
	f, err := guardParse(func() (*ogxmparser.File, error) { return ogxmparser.ParseFile(path) })
	if err != nil {
		return nil, fmt.Errorf("ingest: parse ogxm file: %w", err)
	}
	if f.Match == nil {
		return nil, fmt.Errorf("ingest: %s: an analysis without its match (Analysis shape) cannot be imported", filepath.Base(path))
	}
	return mapOGXM(f, path)
}

func mapOGXM(f *ogxmparser.File, path string) (*MatchGraph, error) {
	m := f.Match
	if m.Variant != ogxmparser.Backgammon {
		return nil, fmt.Errorf("ingest: %s: variant %d is not backgammon", filepath.Base(path), m.Variant)
	}
	digest := f.MatchDigest()
	graph := &MatchGraph{
		Match: domain.Match{
			Player1Name:   m.BlackName,
			Player2Name:   m.WhiteName,
			Event:         m.Event,
			Location:      m.City,
			MatchLength:   int32(m.Length),
			FilePath:      path,
			GameCount:     len(f.Games),
			MatchHash:     hex.EncodeToString(digest[:]),
			CanonicalHash: ogxmCanonicalHash(f),
		},
	}
	if m.Stage != "" {
		graph.Match.Round = m.Stage
	} else if m.Round > 0 {
		graph.Match.Round = fmt.Sprint(m.Round)
	}
	if m.StartedAt != nil {
		graph.Match.MatchDate = *m.StartedAt
	}

	decisions := ogxmDecisions(f)
	replays := f.Replay()
	moveNumber := int32(0)
	for gi := range f.Games {
		g := &f.Games[gi]
		// A game that does not replay (spec M9) keeps the positions before
		// the failure; mapOGXMGame stops there.
		rp := replays[gi]
		gg := GameGraph{
			Game: domain.Game{
				GameNumber:   int32(gi + 1),
				InitialScore: [2]int32{int32(rp.ScoreBefore[ogxmparser.Black]), int32(rp.ScoreBefore[ogxmparser.White])},
				Winner:       domain.WinnerUnfinished,
				MoveCount:    len(g.Plies),
			},
		}
		if g.Winner != nil && *g.Winner <= ogxmparser.Black {
			gg.Game.Winner = domain.WinnerFromSide(ogxmColour(*g.Winner))
		}
		if g.PointsWon != nil {
			gg.Game.PointsWon = int32(*g.PointsWon)
		}
		moves, err := mapOGXMGame(f, g, rp, decisions, m.Length, &moveNumber)
		if err != nil {
			return nil, fmt.Errorf("ingest: %s: game %d: %w", filepath.Base(path), gi+1, err)
		}
		gg.Moves = moves
		graph.Games = append(graph.Games, gg)
	}
	return graph, nil
}

// ogxmColour maps an OGXM seat to blunderDB's colour index.
func ogxmColour(s ogxmparser.Seat) int {
	if s == ogxmparser.White {
		return domain.White
	}
	return domain.Black
}

// ogxmDecisions indexes the first analysis block's decisions by ply_ref.
type ogxmPlyDecisions struct {
	checker *ogxmparser.CheckerDecision
	cube    *ogxmparser.CubeDecision
	roll    *ogxmparser.RollDecision
}

type ogxmAnalysis struct {
	block *ogxmparser.Analysis
	byRef map[int]*ogxmPlyDecisions
}

func ogxmDecisions(f *ogxmparser.File) *ogxmAnalysis {
	if len(f.Analyses) == 0 {
		return nil
	}
	a := &f.Analyses[0]
	out := &ogxmAnalysis{block: a, byRef: map[int]*ogxmPlyDecisions{}}
	for i := range a.Decisions {
		d := &a.Decisions[i]
		pd := out.byRef[d.PlyRef]
		if pd == nil {
			pd = &ogxmPlyDecisions{}
			out.byRef[d.PlyRef] = pd
		}
		switch {
		case d.Checker != nil:
			pd.checker = d.Checker
		case d.Cube != nil:
			pd.cube = d.Cube
		case d.Roll != nil:
			pd.roll = d.Roll
		}
	}
	return out
}

func mapOGXMGame(f *ogxmparser.File, g *ogxmparser.Game, rp ogxmparser.GameReplay, an *ogxmAnalysis, matchLength int, moveNumber *int32) ([]MoveGraph, error) {
	var out []MoveGraph
	for pi := range g.Plies {
		if rp.FailedAt >= 0 && pi >= rp.FailedAt {
			break
		}
		p := &g.Plies[pi]
		st := rp.Plies[pi]
		var pd *ogxmPlyDecisions
		if an != nil {
			pd = an.byRef[p.Ref]
		}
		player := ogxmColour(p.Seat)
		switch {
		case p.IsDiceAction():
			if st.UnplayedRoll {
				continue
			}
			pos := ogxmPosition(&st.Before, matchLength)
			pos.PlayerOnRoll = player
			pos.DecisionType = domain.CheckerAction
			dice := p.Dice()
			played := ogxmMoveString(p.Seat, p.Steps)
			mg := MoveGraph{
				Move: domain.Move{
					MoveNumber:  *moveNumber,
					MoveType:    "checker",
					Player:      blunderDBPlayerToXG(player),
					Dice:        [2]int32{int32(dice[0]), int32(dice[1])},
					CheckerMove: played,
				},
				Position: pos,
			}
			if pd != nil {
				if pd.roll != nil {
					mp := int32(math.Round(pd.roll.Luck * 1000))
					mg.Move.LuckMP = &mp
				}
				if a := ogxmCheckerAnalysis(an, pd.checker, p.Seat, played); a != nil {
					mg.Analyses = append(mg.Analyses, a)
				}
				if pd.cube != nil {
					if cube := ogxmCubeAnalysis(an, pd.cube, &st.Before, player, matchLength, rp.Crawford); cube != nil {
						mg.Analyses = append(mg.Analyses, &domain.PositionAnalysis{
							AnalysisType:          "CheckerMove",
							AnalysisEngineVersion: ogxmEngine(an.block),
							DoublingCubeAnalysis:  cube,
							CreationDate:          time.Now(),
							LastModifiedDate:      time.Now(),
						})
					}
				}
			}
			out = append(out, mg)
			*moveNumber++
		case p.Action == ogxmparser.ActionDouble:
			pos := ogxmPosition(&st.Before, matchLength)
			pos.PlayerOnRoll = player
			pos.DecisionType = domain.CubeAction
			pos.Dice = [2]int{}
			action := "Double/Pass"
			if pi+1 < len(g.Plies) {
				switch g.Plies[pi+1].Action {
				case ogxmparser.ActionTake, ogxmparser.ActionBeaver, ogxmparser.ActionRaccoon:
					action = "Double/Take"
				case ogxmparser.ActionDrop:
				default:
					action = "Double"
				}
			}
			mg := MoveGraph{
				Move: domain.Move{
					MoveNumber: *moveNumber,
					MoveType:   "cube",
					Player:     blunderDBPlayerToXG(player),
					CubeAction: action,
				},
				Position: pos,
			}
			if pd != nil && pd.cube != nil {
				if cube := ogxmCubeAnalysis(an, pd.cube, &st.Before, player, matchLength, rp.Crawford); cube != nil {
					var playedActions []string
					if action != "Double" {
						playedActions = []string{action}
					}
					mg.Analyses = append(mg.Analyses, &domain.PositionAnalysis{
						PlayedCubeActions:     playedActions,
						AnalysisType:          "DoublingCube",
						AnalysisEngineVersion: ogxmEngine(an.block),
						DoublingCubeAnalysis:  cube,
						CreationDate:          time.Now(),
						LastModifiedDate:      time.Now(),
					})
				}
			}
			out = append(out, mg)
			*moveNumber++
		}
	}
	return out, nil
}

// ogxmPosition converts a replayed OGID to a Position: the board is absolute
// on both sides, so points map one to one.
func ogxmPosition(o *ogid.OGID, matchLength int) *domain.Position {
	pos := &domain.Position{}
	var onBoard [2]int
	for i, v := range o.Board {
		switch {
		case v > 0:
			pos.Board.Points[i] = domain.Point{Checkers: int(v), Color: domain.White}
			onBoard[domain.White] += int(v)
		case v < 0:
			pos.Board.Points[i] = domain.Point{Checkers: int(-v), Color: domain.Black}
			onBoard[domain.Black] += int(-v)
		default:
			pos.Board.Points[i] = domain.Point{Checkers: 0, Color: domain.None}
		}
	}
	pos.Board.Bearoff[domain.Black] = 15 - onBoard[domain.Black]
	pos.Board.Bearoff[domain.White] = 15 - onBoard[domain.White]
	pos.Cube.Value = o.CubeLog2
	switch o.CubeOwner {
	case 'W':
		pos.Cube.Owner = domain.White
	case 'B':
		pos.Cube.Owner = domain.Black
	default:
		pos.Cube.Owner = domain.None
	}
	pos.Dice = o.Dice
	pos.Score = [2]int{domain.Unlimited, domain.Unlimited}
	if matchLength > 0 {
		pos.Score = domain.AwayScoresWithCrawford(matchLength, o.Score[ogid.Black], o.Score[ogid.White], o.Crawford)
	}
	return pos
}

// ogxmMoveString writes steps in the mover's own numbering, as the GNU
// Backgammon path writes player-relative moves, so played moves and
// alternatives compare as strings.
func ogxmMoveString(seat ogxmparser.Seat, steps []ogxmparser.Step) string {
	move := [8]int{-1, -1, -1, -1, -1, -1, -1, -1}
	for i, s := range steps {
		if i >= 4 {
			break
		}
		own := s.From // Black counts its points as they are
		if seat == ogxmparser.White {
			own = 25 - s.From
		}
		to := own - s.Pips
		move[2*i] = own - 1 // 24 = the bar
		if to <= 0 {
			move[2*i+1] = -1
		} else {
			move[2*i+1] = to - 1
		}
	}
	return convertPlayerRelativeMoveToString(move)
}

func ogxmEngine(a *ogxmparser.Analysis) string {
	name := "HedgeHog"
	if a.ModelName != "" {
		name += " " + a.ModelName
	}
	return name
}

func ogxmDepth(l *ogxmparser.Level, cube bool) string {
	if l == nil {
		return ""
	}
	if l.Rollout != nil {
		return "Rollout"
	}
	ply := l.CheckerPly
	if cube {
		ply = l.CubePly
	}
	if ply != nil {
		return fmt.Sprintf("%d-ply", *ply)
	}
	if l.Preset != nil {
		return *l.Preset
	}
	return ""
}

func ogxmCheckerAnalysis(an *ogxmAnalysis, d *ogxmparser.CheckerDecision, seat ogxmparser.Seat, played string) *domain.PositionAnalysis {
	if d == nil || len(d.Alternatives) == 0 || an.block.Currency == ogxmparser.CubefulMatch {
		return nil
	}
	decisionLevel := d.Level.Effective(an.block.Level)
	best := d.Alternatives[0].Equity
	moves := make([]domain.CheckerMove, 0, len(d.Alternatives))
	for i, alt := range d.Alternatives {
		cm := domain.CheckerMove{
			Index:          i,
			AnalysisDepth:  ogxmDepth(alt.Level.Effective(decisionLevel), false),
			AnalysisEngine: "HedgeHog",
			Move:           ogxmMoveString(seat, alt.Steps),
			Equity:         alt.Equity,
		}
		if i > 0 {
			diff := best - alt.Equity
			cm.EquityError = &diff
		}
		if p := alt.Probs; p != nil {
			cm.PlayerWinChance = p.Win * 100
			cm.PlayerGammonChance = p.GammonWin * 100
			cm.PlayerBackgammonChance = p.BackgammonWin * 100
			cm.OpponentWinChance = (1 - p.Win) * 100
			cm.OpponentGammonChance = p.GammonLoss * 100
			cm.OpponentBackgammonChance = p.BackgammonLoss * 100
		}
		moves = append(moves, cm)
	}
	a := &domain.PositionAnalysis{
		AnalysisType:          "CheckerMove",
		AnalysisEngineVersion: ogxmEngine(an.block),
		CheckerAnalysis:       &domain.CheckerAnalysis{Moves: moves},
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}
	if played != "" {
		a.PlayedMoves = []string{played}
	}
	return a
}

// ogxmCubeAnalysis builds the doubler's cube analysis. It returns nil for a
// record without the three cubeful equities.
func ogxmCubeAnalysis(an *ogxmAnalysis, d *ogxmparser.CubeDecision, before *ogid.OGID, player, matchLength int, crawford bool) *domain.DoublingCubeAnalysis {
	if d.NoDoubleEquity == nil || d.DoubleTakeEquity == nil || d.DoublePassEquity == nil {
		return nil
	}
	currency := an.block.Currency
	if d.Currency != nil {
		currency = *d.Currency
	}
	nd, dt, dp := *d.NoDoubleEquity, *d.DoubleTakeEquity, *d.DoublePassEquity
	if matchLength > 0 && currency != ogxmparser.Cubeless {
		// At a match score the cube equities are match winning chances
		// whatever the label: convert them as GNU Backgammon's are.
		ca := gnubgparser.CubeAnalysis{CubefulNoDouble: nd, CubefulDoubleTake: dt, CubefulDoublePass: dp}
		convertGnuBGCubeMWCToEMG(&ca, before.Score[ogid.Black], before.Score[ogid.White], player, before.CubeValue(), matchLength, crawford)
		nd, dt, dp = ca.CubefulNoDouble, ca.CubefulDoubleTake, ca.CubefulDoublePass
	}
	params := cubeAnalysisParams{
		Depth:                   ogxmDepth(d.Level.Effective(an.block.Level), true),
		Engine:                  "HedgeHog",
		CubefulNoDoubleEquity:   nd,
		CubefulDoubleTakeEquity: dt,
		CubefulDoublePassEquity: dp,
	}
	if p := d.Probs; p != nil {
		params.PlayerWinChances = p.Win * 100
		params.PlayerGammonChances = p.GammonWin * 100
		params.PlayerBackgammonChances = p.BackgammonWin * 100
		params.OpponentWinChances = (1 - p.Win) * 100
		params.OpponentGammonChances = p.GammonLoss * 100
		params.OpponentBackgammonChances = p.BackgammonLoss * 100
	}
	cube := buildDoublingCubeAnalysis(params)
	return &cube
}

// ogxmCanonicalHash is the format-independent hash shared with XG, GNU
// Backgammon and BGBlitz imports of the same match (cross-format dedup).
func ogxmCanonicalHash(f *ogxmparser.File) string {
	games := make([][][2]int, len(f.Games))
	for gi, g := range f.Games {
		for _, p := range g.Plies {
			if p.IsDiceAction() {
				d := p.Dice()
				games[gi] = append(games[gi], [2]int{d[0], d[1]})
			}
		}
	}
	return CanonicalMatchHash(f.Match.BlackName, f.Match.WhiteName, f.Match.Length, games)
}

// OGXMImporter implements Importer for HedgeHog .ogxm files.
type OGXMImporter struct{ S storage.Storage }

func (im OGXMImporter) Import(ctx context.Context, scope string, src Source, prog func(Progress)) (Summary, error) {
	if src.Path == "" {
		return Summary{}, fmt.Errorf("ingest: ogxm import requires a file path")
	}
	graph, err := MapOGXM(src.Path)
	if err != nil {
		return Summary{}, err
	}
	graph.ImportBatchID = src.BatchID
	tx, err := im.S.BeginTx(ctx)
	if err != nil {
		return Summary{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	res, err := WriteMatch(ctx, tx, scope, graph, prog)
	if err != nil {
		return Summary{}, err
	}
	if err := ctx.Err(); err != nil {
		return Summary{}, err
	}
	if err := tx.Commit(); err != nil {
		return Summary{}, err
	}
	committed = true
	sum := Summary{SavedPositions: res.SavedPositions, Matches: 1, MatchID: res.MatchID, BatchID: src.BatchID}
	if res.Skipped {
		sum.SkippedDuplicates = 1
		sum.SavedPositions = 0
	}
	if res.Enriched {
		sum.Enriched = 1
	}
	return sum, nil
}
