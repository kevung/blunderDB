package printsheet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/searchquery"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// PositionsOptions are the choices of the position list's print dialog. The
// zero value is not the default: DefaultPositionsOptions is.
type PositionsOptions struct {
	// Language of the labels (Languages), English when it has no catalogue.
	Language string `json:"language,omitempty"`
	// Title heads every page; empty is the search's query, the collection's
	// name, or a generic title for a list of ids.
	Title string `json:"title,omitempty"`
	// Analysis writes each position's short analysis: best play, one other
	// and its loss, the chances of winning and of a gammon.
	Analysis bool `json:"analysis"`
	// Verso prints the analyses on the back of the positions, under the same
	// numbers: the front is a quiz. Without Analysis there is nothing to put
	// there and it is ignored.
	Verso bool `json:"verso"`
	// Comments prints the positions' comments, whole: on the back with Verso,
	// since they may give the answer away.
	Comments bool `json:"comments"`
	// Expanded writes the ArkIDs in their developed form.
	Expanded bool `json:"expanded"`
}

// DefaultPositionsOptions is what the print dialog opens with.
func DefaultPositionsOptions() PositionsOptions {
	return PositionsOptions{Analysis: true, Comments: true}
}

// UnmarshalJSON starts from DefaultPositionsOptions, so that a request naming
// only some choices keeps the defaults of the others in every mode.
func (o *PositionsOptions) UnmarshalJSON(b []byte) error {
	type plain PositionsOptions
	p := plain(DefaultPositionsOptions())
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	*o = PositionsOptions(p)
	return nil
}

// PositionSource names the list to print, exactly one of its fields set: the
// whole list, never the window a screen happens to have loaded.
type PositionSource struct {
	// IDs are the positions, in their order: a list the caller holds (the
	// browsed list, the Basket).
	IDs []int64 `json:"ids,omitempty"`
	// Query is a search in the command bar's grammar (package searchquery).
	Query string `json:"query,omitempty"`
	// Collection is a collection's id; a living one is resolved by its query.
	Collection int64 `json:"collection,omitempty"`
}

// MaxPositions is the most positions one document prints, about twenty
// sheets with the analysis: beyond it the list is refused, never cut.
const MaxPositions = 3000

// ListEntry is one line of a position list, before any HTML.
type ListEntry struct {
	N     int
	ArkID string
	// Pips are the pip counts of the player on roll, then of the opponent:
	// the ArkID is written from the player on roll's side.
	Pips [2]int
	Cube bool
	// Best is the best play or cube action; Others the other choices, with
	// their losses in millipoints on the stored scale (ADR-0019). A checker
	// play lists one other, the second best.
	Best   string
	Others []Choice
	// Win and Gammon are the player on roll's chances, in percent, after the
	// best choice; nil without an analysis.
	Win, Gammon *float64
	Comments    []string
}

// Choice is a play or a cube action and what it loses against the best.
type Choice struct {
	Text   string
	CostMP int
}

// PositionList is what a position list says, before any HTML.
type PositionList struct {
	Title   string
	Printed time.Time
	Entries []*ListEntry
}

// ResolvePositions reads the ids src names, in their order. It refuses with
// storage.ErrInvalid a source that names nothing, or more than one list, and
// a list longer than MaxPositions.
func ResolvePositions(ctx context.Context, s storage.Storage, scope string, src PositionSource) (ids []int64, title string, err error) {
	set := 0
	for _, on := range []bool{len(src.IDs) > 0, strings.TrimSpace(src.Query) != "", src.Collection != 0} {
		if on {
			set++
		}
	}
	if set != 1 {
		return nil, "", fmt.Errorf("name one list to print: ids, a query or a collection: %w", storage.ErrInvalid)
	}
	switch {
	case len(src.IDs) > 0:
		ids = src.IDs
	case src.Collection != 0:
		c, err := s.Collections().Get(ctx, scope, src.Collection)
		if err != nil {
			return nil, "", fmt.Errorf("collection %d: %w", src.Collection, err)
		}
		title = c.Name
		filters, _, living, err := storage.LivingFilters(ctx, s, scope, src.Collection)
		if err != nil {
			return nil, "", err
		}
		if living {
			ids, err = s.Search().FindIDs(ctx, scope, filters, storage.ListOpts{Limit: MaxPositions + 1})
		} else {
			ids, err = s.Collections().PositionIDs(ctx, scope, src.Collection, storage.ListOpts{Limit: MaxPositions + 1})
		}
		if err != nil {
			return nil, "", err
		}
	default:
		filters, diags := searchquery.Parse(src.Query)
		var unknown []string
		for _, d := range diags {
			if d.Kind == searchquery.DiagUnknown {
				unknown = append(unknown, d.Token)
			}
		}
		if len(unknown) > 0 {
			return nil, "", fmt.Errorf("unknown token(s) in the query: %s: %w", strings.Join(unknown, ", "), storage.ErrInvalid)
		}
		if err := searchquery.Refusal(diags); err != nil {
			return nil, "", fmt.Errorf("%v: %w", err, storage.ErrInvalid)
		}
		title = strings.TrimSpace(src.Query)
		if ids, err = s.Search().FindIDs(ctx, scope, filters, storage.ListOpts{Limit: MaxPositions + 1}); err != nil {
			return nil, "", err
		}
	}
	if len(ids) == 0 {
		return nil, "", fmt.Errorf("no position to print: %w", storage.ErrInvalid)
	}
	if len(ids) > MaxPositions {
		return nil, "", fmt.Errorf("more than %d positions to print: narrow the list: %w", MaxPositions, storage.ErrInvalid)
	}
	return ids, title, nil
}

// ReadPositions gathers what the list of ids says. It writes nothing.
func ReadPositions(ctx context.Context, s storage.Storage, scope string, ids []int64, o PositionsOptions) (*PositionList, error) {
	ps, err := s.Positions().LoadByIDs(ctx, scope, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]*domain.Position, len(ps))
	for i := range ps {
		byID[ps[i].ID] = &ps[i]
	}
	var as map[int64]*domain.PositionAnalysis
	if o.Analysis {
		if as, err = s.Analyses().LoadMany(ctx, scope, ids); err != nil && !errors.Is(err, storage.ErrNotFound) {
			return nil, err
		}
	}
	var cms map[int64][]*domain.CommentEntry
	if o.Comments {
		if cms, err = s.Comments().ByPositions(ctx, scope, ids); err != nil {
			return nil, err
		}
	}
	l := &PositionList{Printed: time.Now()}
	for _, id := range ids {
		p := byID[id]
		if p == nil {
			return nil, fmt.Errorf("position %d: %w", id, storage.ErrNotFound)
		}
		e := &ListEntry{N: len(l.Entries) + 1, Pips: OnRollPips(p), Cube: p.Dice[0] == 0}
		if o.Expanded {
			e.ArkID, err = domain.EncodeArkIDExpanded(p)
		} else {
			e.ArkID, err = domain.EncodeArkID(p)
		}
		if err != nil {
			return nil, fmt.Errorf("position %d: %w", id, err)
		}
		if a := as[id]; a != nil {
			if e.Cube {
				cubeChoices(e, a.DoublingCubeAnalysis)
			} else {
				checkerChoices(e, a.CheckerAnalysis)
			}
		}
		for _, c := range cms[id] {
			e.Comments = append(e.Comments, c.Text)
		}
		l.Entries = append(l.Entries, e)
	}
	return l, nil
}

// OnRollPips is the pip count of the player on roll, then of the opponent.
func OnRollPips(p *domain.Position) [2]int {
	a, b := p.ComputePipCounts()
	if p.PlayerOnRoll == 1 {
		a, b = b, a
	}
	return [2]int{a, b}
}

func milli(v float64) int { return int(math.Round(math.Abs(v) * 1000)) }

func checkerChoices(e *ListEntry, a *domain.CheckerAnalysis) {
	if a == nil || len(a.Moves) == 0 {
		return
	}
	best := a.Moves[0]
	e.Best = best.Move
	win, gammon := best.PlayerWinChance, best.PlayerGammonChance
	e.Win, e.Gammon = &win, &gammon
	if len(a.Moves) > 1 {
		alt := Choice{Text: a.Moves[1].Move}
		if a.Moves[1].EquityError != nil {
			alt.CostMP = milli(*a.Moves[1].EquityError)
		} else {
			alt.CostMP = milli(best.Equity - a.Moves[1].Equity)
		}
		e.Others = []Choice{alt}
	}
}

// Cube actions, in the vocabulary of the stored analyses.
const (
	cubeNoDouble   = "No Double"
	cubeDoubleTake = "Double, Take"
	cubeDoublePass = "Double, Pass"
)

func cubeChoices(e *ListEntry, d *domain.DoublingCubeAnalysis) {
	if d == nil {
		return
	}
	all := []Choice{
		{cubeNoDouble, milli(d.CubefulNoDoubleError)},
		{cubeDoubleTake, milli(d.CubefulDoubleTakeError)},
		{cubeDoublePass, milli(d.CubefulDoublePassError)},
	}
	best := 0
	for i, c := range all {
		if c.CostMP < all[best].CostMP {
			best = i
		}
	}
	e.Best = d.BestCubeAction
	if e.Best == "" {
		e.Best = all[best].Text
	}
	for i, c := range all {
		if i != best {
			e.Others = append(e.Others, c)
		}
	}
	win, gammon := d.PlayerWinChances, d.PlayerGammonChances
	e.Win, e.Gammon = &win, &gammon
}
