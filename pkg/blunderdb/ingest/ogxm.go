package ingest

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math"
	"path/filepath"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
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
// Analysis: the first analysis block is read; a later one is logged and
// left out. Only a block in cubeful money currency (9.10, value 1) is
// imported: blunderDB stores cubeful equities, so a cubeless block, or one
// whose currency is not recorded, would be presented as what it is not, and
// a block in match winning chances cannot be converted play by play (it
// needs each play's score swing, which the file does not carry). Checker
// equities and luck are then taken as stated: cubeful, normalised to the
// current cube at a match score, as GNU Backgammon's are (ADR-0019).
//
// Cube equities are the exception. HedgeHog writes them, at a match score,
// as match winning chances under the block's cubeful-money label (its own
// JSON projection then normalises them with its MET). They are normalised
// here on the file's own anchors: the doubler's double/pass equity is its MWC
// on winning the current cube, so it scales to +1 exactly, and the losing
// anchor is 1 minus the opponent's double/pass at the same score and cube,
// which a match file nearly always holds. blunderDB's MET (Kazaross-XG2)
// stands in only where the opponent never faced that cube at that score. A
// double/pass that strays from the MET's winning anchor means the block does
// not hold MWC as assumed: its cube analysis is then left out whole.

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
	decisions.indexCubeAnchors(f, replays)
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
	decisions.warnMETFallbacks()
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
	// winAnchor holds, at a match score, each player's MWC on winning the
	// current cube, read from the file's double/pass equities.
	winAnchor map[ogxmAnchorKey]float64
	// noCube leaves the block's cube analysis out: its double/pass
	// equities are not the MWC they are read as.
	noCube bool
	// metFallbacks counts the cube decisions normalised on blunderDB's MET
	// for want of the opponent's double/pass.
	metFallbacks int
}

// warnMETFallbacks reports the cube decisions whose losing anchor came from
// blunderDB's MET while the file declares another table: they carry the
// difference between the two.
func (an *ogxmAnalysis) warnMETFallbacks() {
	if an == nil || an.metFallbacks == 0 || an.block.METID == ogxmMETID {
		return
	}
	slog.Warn("ogxm import: some cube equities normalised against blunderDB's MET, not the file's",
		"file_met", an.block.METID, "met", ogxmMETID, "decisions", an.metFallbacks)
}

// ogxmAnchorKey is what a double/pass anchor depends on: who wins the cube,
// the score, the cube's value and the Crawford rule.
type ogxmAnchorKey struct {
	player, scoreBlack, scoreWhite, cube int
	crawford                             bool
}

func ogxmAnchorKeyAt(o *ogid.OGID, player int, crawford bool) ogxmAnchorKey {
	return ogxmAnchorKey{player, o.Score[ogid.Black], o.Score[ogid.White], o.CubeValue(), crawford}
}

// ogxmPlausibleMWC bounds how far a double/pass may stray from the MET's
// winning anchor: two published tables differ by a few thousandths.
const ogxmPlausibleMWC = 0.01

// indexCubeAnchors reads the winning anchors of a match file's cube
// decisions and checks each against blunderDB's MET.
func (an *ogxmAnalysis) indexCubeAnchors(f *ogxmparser.File, replays []ogxmparser.GameReplay) {
	if an == nil || f.Match.Length <= 0 {
		return
	}
	an.winAnchor = map[ogxmAnchorKey]float64{}
	length := f.Match.Length
	for gi := range f.Games {
		g, rp := &f.Games[gi], replays[gi]
		for pi := range g.Plies {
			if rp.FailedAt >= 0 && pi >= rp.FailedAt {
				break
			}
			p := &g.Plies[pi]
			pd := an.byRef[p.Ref]
			if pd == nil || pd.cube == nil || !ogxmCubeful(pd.cube) || pd.cube.DoublePassEquity == nil {
				continue
			}
			if !p.IsDiceAction() && p.Action != ogxmparser.ActionDouble {
				continue
			}
			before := &rp.Plies[pi].Before
			player := ogxmColour(p.Seat)
			dp := *pd.cube.DoublePassEquity
			met := engine.GnuBGGetME(before.Score[ogid.Black], before.Score[ogid.White], length,
				player, before.CubeValue(), player, rp.Crawford)
			if math.Abs(dp-met) > ogxmPlausibleMWC {
				slog.Warn("ogxm import: cube analysis not imported, a double/pass equity is not the match winning chance it should be",
					"game", gi+1, "ply", p.Ref, "double_pass", dp, "met", met)
				an.noCube = true
				return
			}
			k := ogxmAnchorKeyAt(before, player, rp.Crawford)
			if _, ok := an.winAnchor[k]; !ok {
				an.winAnchor[k] = dp
			}
		}
	}
}

// ogxmCubeful reports whether a cube decision's own currency, when it
// overrides the block's, still states cubeful equities.
func ogxmCubeful(d *ogxmparser.CubeDecision) bool {
	return d.Currency == nil || *d.Currency == ogxmparser.CubefulMoney || *d.Currency == ogxmparser.CubefulMatch
}

// ogxmMETID names blunderDB's MET (engine/met.go) the way OGXM's met_id does.
const ogxmMETID = "kazaross-xg2"

func ogxmDecisions(f *ogxmparser.File) *ogxmAnalysis {
	if len(f.Analyses) == 0 {
		return nil
	}
	if len(f.Analyses) > 1 {
		slog.Warn("ogxm import: analysis blocks after the first are not imported", "blocks", len(f.Analyses))
	}
	a := &f.Analyses[0]
	if a.Currency != ogxmparser.CubefulMoney {
		slog.Warn("ogxm import: analysis not imported, its currency is not cubeful money", "currency", int(a.Currency))
		return nil
	}
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
			pos, err := ogxmPosition(&st.Before, matchLength)
			if err != nil {
				return nil, fmt.Errorf("ply %d: %w", p.Ref, err)
			}
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
			pos, err := ogxmPosition(&st.Before, matchLength)
			if err != nil {
				return nil, fmt.Errorf("ply %d: %w", p.Ref, err)
			}
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
func ogxmPosition(o *ogid.OGID, matchLength int) (*domain.Position, error) {
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
	// The replay writes W, B or N only (ogxmparser's ownerChar); a dead or
	// unknown cube ('D', '?') comes from a pasted OGID, never from a file.
	switch o.CubeOwner {
	case 'W':
		pos.Cube.Owner = domain.White
	case 'B':
		pos.Cube.Owner = domain.Black
	case 'N':
		pos.Cube.Owner = domain.None
	default:
		return nil, fmt.Errorf("cube owner %q is neither a player nor centred", o.CubeOwner)
	}
	pos.Dice = o.Dice
	pos.Score = [2]int{domain.Unlimited, domain.Unlimited}
	if matchLength > 0 {
		pos.Score = domain.AwayScoresWithCrawford(matchLength, o.Score[ogid.Black], o.Score[ogid.White], o.Crawford)
	}
	return pos, nil
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
	if d == nil || len(d.Alternatives) == 0 {
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
	if !ogxmCubeful(d) || an.noCube {
		return nil
	}
	if matchLength == 0 && d.Currency != nil && *d.Currency == ogxmparser.CubefulMatch {
		return nil // match winning chances have no meaning at money play
	}
	nd, dt, dp := *d.NoDoubleEquity, *d.DoubleTakeEquity, *d.DoublePassEquity
	if matchLength > 0 {
		win := dp // the file's own MWC for the doubler winning the cube
		var lose float64
		if opp, ok := an.winAnchor[ogxmAnchorKeyAt(before, 1-player, crawford)]; ok {
			lose = 1 - opp
		} else {
			an.metFallbacks++
			lose = engine.GnuBGGetME(before.Score[ogid.Black], before.Score[ogid.White], matchLength,
				player, before.CubeValue(), 1-player, crawford)
		}
		if win-lose < 1e-7 {
			return nil
		}
		norm := func(mwc float64) float64 { return (2*mwc - win - lose) / (win - lose) }
		nd, dt, dp = norm(nd), norm(dt), 1
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
	graph.SkipDuplicates = src.SkipDuplicates
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
		sum.FlagsApplied = res.FlagsApplied
		sum.Deepened = res.Deepened
		sum.SavedPositions = 0
	}
	if res.Enriched {
		sum.Enriched = 1
	}
	return sum, nil
}
