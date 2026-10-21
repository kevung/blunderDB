package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Walks over a whole library, for the tabular exports. Each reads in a few
// statements what a loop over the per-match or per-position getters reads in
// one statement per row, and hands the rows over as they come instead of
// holding the library in memory.

// positionWalkPage is how many positions EachPositionAnalysis reads at a
// time: large enough that the per-statement cost vanishes, small enough that
// a page of decoded analyses stays a few megabytes. A variable so that a
// test walks several pages of a small fixture.
var positionWalkPage = 2000

// EachLibraryMove calls fn for every recorded move of the library, as
// GetMovesByGame reads them, with the match and the game number it belongs
// to: matches in GetAllMatches's order (without its badges), then by game
// number, then by move number (storage.MatchStore.LibraryMoves). An error
// from fn stops the walk and is returned.
func (d *Database) EachLibraryMove(fn func(m *Match, gameNumber int32, mv *Move) error) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ctx := context.Background()
	matches := make(map[int64]*Match)
	for m, err := range d.store.Matches().List(ctx, "", storage.MatchListOpts{}) {
		if err != nil {
			return err
		}
		matches[m.ID] = m
	}
	for lm, err := range d.store.Matches().LibraryMoves(ctx, "") {
		if err != nil {
			return err
		}
		m, ok := matches[lm.MatchID]
		if !ok {
			return fmt.Errorf("move %d: match %d not in the library", lm.ID, lm.MatchID)
		}
		if err := fn(m, lm.GameNumber, &lm.Move); err != nil {
			return err
		}
	}
	return nil
}

// EachPositionAnalysis calls fn for every position, in LoadAllPositions's
// order, with its analysis as LoadAnalysis reads it — the played actions of
// every match folded in — or nil when it has none. Positions, analyses and
// recorded moves are read a page at a time (storage.AnalysisStore.LoadMany,
// storage.MatchStore.MovesByPositions), and fn runs outside the lock. An
// error from fn stops the walk and is returned.
func (d *Database) EachPositionAnalysis(fn func(pos *Position, a *PositionAnalysis) error) error {
	for offset := 0; ; offset += positionWalkPage {
		positions, analyses, err := d.positionAnalysisPage(offset)
		if err != nil {
			return err
		}
		for i := range positions {
			if err := fn(&positions[i], analyses[positions[i].ID]); err != nil {
				return err
			}
		}
		if len(positions) < positionWalkPage {
			return nil
		}
	}
}

// positionAnalysisPage reads the page of EachPositionAnalysis at offset.
func (d *Database) positionAnalysisPage(offset int) ([]Position, map[int64]*PositionAnalysis, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ctx := context.Background()
	var positions []Position
	for pos, err := range d.store.Positions().List(ctx, "", storage.ListOpts{Offset: offset, Limit: positionWalkPage}) {
		if err != nil {
			return nil, nil, err
		}
		positions = append(positions, *pos)
	}
	if len(positions) == 0 {
		return nil, nil, nil
	}
	ids := make([]int64, len(positions))
	for i := range positions {
		ids[i] = positions[i].ID
	}
	analyses, err := d.store.Analyses().LoadMany(ctx, "", ids)
	if err != nil {
		return nil, nil, err
	}
	played, err := d.store.Matches().MovesByPositions(ctx, "", ids)
	if err != nil {
		return nil, nil, err
	}
	for id, a := range analyses {
		var checkerMoves, cubeActions []string
		for _, mv := range played[id] {
			checkerMoves = append(checkerMoves, mv.CheckerMove)
			cubeActions = append(cubeActions, mv.CubeAction)
		}
		mergePlayedActions(a, checkerMoves, cubeActions)
		if len(a.AllCubeAnalyses) > 0 {
			sortCubeAnalysesByEngine(a.AllCubeAnalyses)
		}
	}
	return positions, analyses, nil
}
