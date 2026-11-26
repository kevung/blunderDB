package printsheet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/arkqr"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// MatchOptions are the choices of the print dialog. The zero value is not the
// default: DefaultMatchOptions is.
type MatchOptions struct {
	// Language of the labels (Languages), English when it has no catalogue.
	Language string `json:"language,omitempty"`
	// Player names the player the sheet is printed for: their blunders are
	// numbered and detailed, their marked and commented positions listed.
	// Empty is the library's Main player when they played, else player 1.
	Player string `json:"player,omitempty"`
	// OpponentBlunders numbers and details the other player's blunders too.
	// Off, they stay marked ?? in the transcription and nothing more.
	OpponentBlunders bool `json:"opponentBlunders"`
	// Analysis writes the short analysis of every listed position.
	Analysis bool `json:"analysis"`
	// Comments prints the match's comment and the positions' comments, whole.
	Comments bool `json:"comments"`
	// Expanded writes the ArkIDs in their developed form.
	Expanded bool `json:"expanded"`
	// ReimportCode draws the match's paper archive code at the end of the
	// transcription and embeds it in the file.
	ReimportCode bool `json:"reimportCode"`
}

// DefaultMatchOptions is what the print dialog opens with.
func DefaultMatchOptions() MatchOptions {
	return MatchOptions{Analysis: true, Comments: true, ReimportCode: true}
}

// UnmarshalJSON starts from DefaultMatchOptions, so that a request naming only
// some choices keeps the defaults of the others in every mode.
func (o *MatchOptions) UnmarshalJSON(b []byte) error {
	type plain MatchOptions
	p := plain(DefaultMatchOptions())
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	*o = MatchOptions(p)
	return nil
}

// MaxMatches is the most matches one document prints: a batch is rendered in
// memory, then laid out by the browser, and beyond this neither stays usable.
const MaxMatches = 100

// SlowThinkMS is the decision time from which the transcription writes it
// beside the move: a long thought is worth seeing again.
const SlowThinkMS = 30_000

// Grades of a decision, at the library's thresholds.
const (
	GradeError   = storage.MoveGradeError
	GradeBlunder = storage.MoveGradeBlunder
)

// Kinds of a listed position.
const (
	KindBlunder = "blunder" // black number
	KindRemark  = "remark"  // framed number: marked or commented
)

// MatchSheet is what a match's sheet says, before any HTML.
type MatchSheet struct {
	MatchID  int64
	Players  [2]string
	Self     int // the player the sheet is printed for, 0 or 1
	Score    [2]int
	Length   int // 0 at money play
	Event    string
	Round    string
	Date     time.Time
	Comment  string
	Plies    int
	TotalMS  *int64
	Stats    [2]PlayerLine
	Games    []SheetGame
	Entries  []*Entry
	Code     []arkqr.Frame // the reimport code, data frames only
	CodeSize int           // bytes of the coded match
	CodeErr  string        // why there is no code although one was asked
}

// PlayerLine is a player's line of the header.
type PlayerLine struct {
	PR       *float64
	Errors   int // every Blunder is an Error
	Blunders int
	TimeMS   *int64
}

// SheetGame is one game of the transcription.
type SheetGame struct {
	Number   int
	Away     [2]int // 0 at money play
	Crawford bool
	Winner   int // 0, 1, or -1 unfinished
	Points   int
	TimeMS   *int64
	Rows     [][2]Cell
}

// Cell is one player's half of a numbered line.
type Cell struct {
	Parts []Part
}

// Part is one decision written in a cell.
type Part struct {
	Dice   string // "31", empty for a cube action
	Text   string // the move, or a cube action in Move.CubeAction's vocabulary
	Cube   int    // the value a Double turns the cube to
	Grade  string // "", GradeError or GradeBlunder
	Ref    int    // the number of its entry, 0 when not listed
	Kind   string // the entry's kind
	SlowMS int64  // the decision time when it is a long thought
}

// Entry is one position of the Positions block.
type Entry struct {
	Ref        int
	Kind       string
	ArkID      string
	Game, Row  int
	Player     int
	Cube       bool
	Played     string
	CostMP     *int
	Best       string
	Alt        string
	AltRank    int
	AltCostMP  int
	Comments   []string
	PositionID int64
	MoveID     int64
}

// ReadMatch gathers what the sheet of match id says. It writes nothing.
func ReadMatch(ctx context.Context, s storage.Storage, scope string, id int64, o MatchOptions) (*MatchSheet, error) {
	m, err := s.Matches().Get(ctx, scope, id)
	if err != nil {
		return nil, err
	}
	var games []*domain.Game
	for g, err := range s.Matches().Games(ctx, scope, id) {
		if err != nil {
			return nil, err
		}
		games = append(games, g)
	}
	byGame := map[int64][]*domain.Move{}
	var posIDs []int64
	seenPos := map[int64]bool{}
	for mv, err := range s.Matches().MovesByMatch(ctx, scope, id) {
		if err != nil {
			return nil, err
		}
		byGame[mv.GameID] = append(byGame[mv.GameID], mv)
		if mv.PositionID != 0 && !seenPos[mv.PositionID] {
			seenPos[mv.PositionID] = true
			posIDs = append(posIDs, mv.PositionID)
		}
	}
	grades := map[int64]string{}
	gs, err := s.Stats().MatchMoveGrades(ctx, scope, id)
	if err != nil {
		return nil, err
	}
	for _, g := range gs {
		grades[g.MoveID] = g.Grade
	}
	positions := map[int64]*domain.Position{}
	if len(posIDs) > 0 {
		ps, err := s.Positions().LoadByIDs(ctx, scope, posIDs)
		if err != nil {
			return nil, err
		}
		for i := range ps {
			positions[ps[i].ID] = &ps[i]
		}
	}
	comments, err := s.Comments().ByPositions(ctx, scope, posIDs)
	if err != nil {
		return nil, err
	}

	ms := &MatchSheet{
		MatchID: m.ID, Players: [2]string{m.Player1Name, m.Player2Name},
		Length: int(m.MatchLength), Event: firstNonEmpty(m.TournamentName, m.Event), Round: m.Round,
		Date: m.MatchDate, Comment: m.Comment,
	}
	if ms.Self, err = selfOf(ctx, s, scope, m, o.Player); err != nil {
		return nil, err
	}
	if badges, err := s.Stats().MatchBadges(ctx, scope, []int64{id}); err == nil {
		if b, ok := badges[id]; ok {
			if b.PR > 0 {
				ms.Stats[0].PR = &b.PR
			}
			if b.PR2 > 0 {
				ms.Stats[1].PR = &b.PR2
			}
		}
	} else {
		return nil, err
	}

	b := &builder{ms: ms, o: o, grades: grades, positions: positions, comments: comments, listedPos: map[int64]bool{}}
	for _, g := range games {
		b.game(g, byGame[g.ID])
	}
	if n := len(games); n > 0 {
		last := games[n-1]
		ms.Score = [2]int{int(last.InitialScore[0]), int(last.InitialScore[1])}
		if w := side(last.Winner); w >= 0 {
			ms.Score[w] += int(last.PointsWon)
		}
	}
	if err := b.analyses(ctx, s, scope); err != nil {
		return nil, err
	}
	if o.ReimportCode {
		reimportCode(ctx, s, scope, ms)
	}
	return ms, nil
}

// reimportCode codes the match as its paper archive does. A match the codec
// cannot hold prints without a code, and says why.
func reimportCode(ctx context.Context, s storage.Storage, scope string, ms *MatchSheet) {
	am, _, err := ingest.ArchiveMatch(ctx, s, scope, ms.MatchID)
	var data []byte
	if err == nil {
		data, err = arkqr.Encode(am)
	}
	var frames []arkqr.Frame
	if err == nil {
		frames, _, err = arkqr.Frames([][]byte{data})
	}
	if err != nil {
		ms.CodeErr = err.Error()
		return
	}
	for _, f := range frames {
		if !f.Parity {
			ms.Code = append(ms.Code, f)
		}
	}
	ms.CodeSize = len(data)
}

// selfOf is the seat of the player the sheet is printed for.
func selfOf(ctx context.Context, s storage.Storage, scope string, m *domain.Match, player string) (int, error) {
	aliases, err := s.Aliases().List(ctx, scope, storage.AliasPlayer)
	if err != nil {
		return 0, err
	}
	am := storage.NewAliasMap(aliases)
	name := strings.TrimSpace(player)
	if name == "" {
		if name, err = storage.MainPlayer(ctx, s, scope); err != nil {
			return 0, err
		}
	}
	name = am.Canonical(name)
	if name != "" && am.Canonical(m.Player2Name) == name && am.Canonical(m.Player1Name) != name {
		return 1, nil
	}
	return 0, nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// side is a Move.Player or Game.Winner as a seat, -1 for none.
func side(v int32) int {
	switch v {
	case domain.WinnerPlayer1:
		return 0
	case domain.WinnerPlayer2:
		return 1
	}
	return -1
}

func addMS(dst **int64, v *int64) {
	if v == nil {
		return
	}
	if *dst == nil {
		*dst = new(int64)
	}
	**dst += *v
}

type builder struct {
	ms        *MatchSheet
	o         MatchOptions
	grades    map[int64]string
	positions map[int64]*domain.Position
	comments  map[int64][]*domain.CommentEntry
	listedPos map[int64]bool
}

// cellOf is a cell before it is laid out in lines.
type cellOf struct {
	side    int
	parts   []Part
	entries []*Entry
}

func (b *builder) game(g *domain.Game, moves []*domain.Move) {
	ms := b.ms
	sg := SheetGame{Number: int(g.GameNumber), Winner: side(g.Winner), Points: int(g.PointsWon)}
	if ms.Length > 0 {
		sg.Away = [2]int{ms.Length - int(g.InitialScore[0]), ms.Length - int(g.InitialScore[1])}
	}
	if len(moves) > 0 {
		if p := b.positions[moves[0].PositionID]; p != nil && (p.Score[0] == domain.Crawford || p.Score[1] == domain.Crawford) {
			sg.Crawford = true
		}
	}
	var cells []cellOf
	var pending [2]*cellOf // a "No Double" worth printing, waiting for its roll
	flush := func(sd int) {
		if pending[sd] != nil {
			cells = append(cells, *pending[sd])
			pending[sd] = nil
		}
	}
	cube := 1
	// A combined "Double/Take" implies the answer; a source that records the
	// answer as a Move of its own replaces it.
	var answer *cellOf
	for _, mv := range moves {
		sd := side(mv.Player)
		if sd < 0 {
			continue
		}
		if answer != nil {
			if !(mv.MoveType == "cube" && sd == answer.side && mv.CubeAction == answer.parts[0].Text) {
				cells = append(cells, *answer)
				ms.Plies++
			}
			answer = nil
		}
		ms_ := msOf(mv)
		addMS(&sg.TimeMS, ms_)
		addMS(&ms.Stats[sd].TimeMS, ms_)
		grade := b.grades[mv.ID]
		if grade != "" {
			ms.Stats[sd].Errors++
			if grade == GradeBlunder {
				ms.Stats[sd].Blunders++
			}
		}
		part := Part{Grade: grade}
		if mv.DecisionMS != nil && *mv.DecisionMS >= SlowThinkMS {
			part.SlowMS = *mv.DecisionMS
		}
		entry := b.entry(mv, sd, grade)
		if entry != nil {
			part.Ref, part.Kind = entry.Ref, entry.Kind
		}
		attach := func(c *cellOf) {
			c.parts = append(c.parts, part)
			if entry != nil {
				c.entries = append(c.entries, entry)
			}
		}
		if mv.MoveType == "cube" {
			action := mv.CubeAction
			switch action {
			case "No Double", "":
				if grade == "" && entry == nil {
					continue
				}
				part.Text = "No Double"
				flush(sd)
				pending[sd] = &cellOf{side: sd}
				attach(pending[sd])
				ms.Plies++
				continue
			case "Double", "Double/Take", "Double/Pass":
				cube *= 2
				part.Text, part.Cube = "Double", cube
			default:
				part.Text = action
			}
			flush(0)
			flush(1)
			c := cellOf{side: sd}
			attach(&c)
			cells = append(cells, c)
			ms.Plies++
			if resp, ok := strings.CutPrefix(action, "Double/"); ok {
				answer = &cellOf{side: 1 - sd, parts: []Part{{Text: resp}}}
			}
			continue
		}
		part.Dice = fmt.Sprintf("%d%d", mv.Dice[0], mv.Dice[1])
		part.Text = mv.CheckerMove
		if part.Text == "Cannot Move" {
			part.Text = ""
		}
		c := pending[sd]
		pending[sd] = nil
		flush(1 - sd)
		if c == nil {
			c = &cellOf{side: sd}
		}
		attach(c)
		cells = append(cells, *c)
		ms.Plies++
	}
	if answer != nil {
		cells = append(cells, *answer)
		ms.Plies++
	}
	flush(0)
	flush(1)
	// Player 1 opens a line, player 2 joins the line player 1 just opened or
	// takes one of their own: the rule of a .mat transcript.
	for i := 0; i < len(cells); {
		var row [2]Cell
		var es []*Entry
		c := cells[i]
		row[c.side].Parts, es = c.parts, c.entries
		i++
		if c.side == 0 && i < len(cells) && cells[i].side == 1 {
			row[1].Parts = cells[i].parts
			es = append(es, cells[i].entries...)
			i++
		}
		sg.Rows = append(sg.Rows, row)
		for _, e := range es {
			e.Game, e.Row = sg.Number, len(sg.Rows)
		}
	}
	addMS(&ms.TotalMS, sg.TimeMS)
	ms.Games = append(ms.Games, sg)
}

func msOf(mv *domain.Move) *int64 {
	if mv.DecisionMS == nil && mv.CubeDecisionMS == nil {
		return nil
	}
	var t int64
	if mv.DecisionMS != nil {
		t += *mv.DecisionMS
	}
	if mv.CubeDecisionMS != nil {
		t += *mv.CubeDecisionMS
	}
	return &t
}

// entry lists a move when the sheet details it: a blunder of the player the
// sheet is for (of both, when asked), or a position of theirs they marked or
// commented, once however many times the match reaches it.
func (b *builder) entry(mv *domain.Move, sd int, grade string) *Entry {
	kind := ""
	switch {
	case grade == GradeBlunder && (sd == b.ms.Self || b.o.OpponentBlunders):
		kind = KindBlunder
	case sd == b.ms.Self && !b.listedPos[mv.PositionID] && b.remarkable(mv.PositionID):
		kind = KindRemark
	default:
		return nil
	}
	b.listedPos[mv.PositionID] = true
	e := &Entry{Ref: len(b.ms.Entries) + 1, Kind: kind, Player: sd, PositionID: mv.PositionID, MoveID: mv.ID}
	if mv.ErrorMP != nil {
		c := int(*mv.ErrorMP)
		e.CostMP = &c
	}
	if mv.MoveType == "cube" {
		e.Cube, e.Played, e.Best = true, mv.CubeAction, mv.BestCubeAction
	} else {
		e.Played, e.Best = mv.CheckerMove, mv.BestMove
	}
	if p := b.positions[mv.PositionID]; p != nil {
		var err error
		if b.o.Expanded {
			e.ArkID, err = domain.EncodeArkIDExpanded(p)
		} else {
			e.ArkID, err = domain.EncodeArkID(p)
		}
		if err != nil {
			e.ArkID = ""
		}
	}
	for _, c := range b.comments[mv.PositionID] {
		e.Comments = append(e.Comments, c.Text)
	}
	b.ms.Entries = append(b.ms.Entries, e)
	return e
}

func (b *builder) remarkable(posID int64) bool {
	p := b.positions[posID]
	return (p != nil && p.Flagged) || len(b.comments[posID]) > 0
}

// analyses fills the alternative move of the checker entries: the best
// candidate that is neither the first nor the one played.
func (b *builder) analyses(ctx context.Context, s storage.Storage, scope string) error {
	var ids []int64
	for _, e := range b.ms.Entries {
		if !e.Cube {
			ids = append(ids, e.PositionID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	as, err := s.Analyses().LoadMany(ctx, scope, ids)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return err
	}
	for _, e := range b.ms.Entries {
		a := as[e.PositionID]
		if e.Cube || a == nil || a.CheckerAnalysis == nil {
			continue
		}
		played := engine.NormalizeMove(e.Played)
		for i, c := range a.CheckerAnalysis.Moves {
			if i == 0 || strings.EqualFold(engine.NormalizeMove(c.Move), played) {
				continue
			}
			e.Alt, e.AltRank = c.Move, i+1
			if c.EquityError != nil {
				e.AltCostMP = int(math.Round(math.Abs(*c.EquityError) * 1000))
			}
			break
		}
	}
	return nil
}
