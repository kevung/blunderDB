package gui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
)

// The split screen's second pane is a read-only viewer of one library group
// (ADR-0101): the front keeps one set of stores, photographed per group, so a
// second live group would write into the stores of the first. The viewer
// reads its group through these calls only, each naming the group, and never
// writes: nothing it does reaches the bound handle's stores or any library.

// ViewerSource names what the viewer walks through in its group: the whole
// library ("library"), a match replayed move by move ("match", ID the match)
// or a collection ("collection", ID the collection).
type ViewerSource struct {
	Kind string `json:"kind"`
	ID   int64  `json:"id"`
}

// ViewerStep is one stop of the walk: the position and, in a match, the
// move played from it and its place in the match.
type ViewerStep struct {
	PositionID int64  `json:"positionId"`
	GameNumber int32  `json:"gameNumber,omitempty"`
	MoveNumber int32  `json:"moveNumber,omitempty"`
	Player     int32  `json:"player,omitempty"`
	Played     string `json:"played,omitempty"`
}

// ViewerEntry is a match or a collection the viewer can open.
type ViewerEntry struct {
	ID    int64  `json:"id"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

// ViewerCatalog is what a group offers the viewer besides its library.
type ViewerCatalog struct {
	Matches     []ViewerEntry `json:"matches"`
	Collections []ViewerEntry `json:"collections"`
}

// ViewerItem is one position as the viewer draws it.
type ViewerItem struct {
	Position database.Position          `json:"position"`
	Analysis *database.PositionAnalysis `json:"analysis"`
	Comment  string                     `json:"comment"`
}

// readGroup runs read on the handle carrying group's library (0: the active
// one). The group lock is held throughout, so no switch or close moves the
// library under the read.
func (a *App) readGroup(group int, read func(d *database.Database) error) error {
	if a.db == nil {
		return errors.New("no database handle")
	}
	a.libs.mu.Lock()
	defer a.libs.mu.Unlock()
	a.libs.init()
	carrier := a.db
	if group != 0 && group != a.libs.active {
		carrier = a.libs.parked[group]
	}
	if carrier == nil {
		return fmt.Errorf("no library group %d", group)
	}
	if st, _ := database.CurrentStore(carrier); st == nil {
		return errors.New("no database open")
	}
	return read(carrier)
}

// ViewerSteps lists the stops of src in group, in order.
func (a *App) ViewerSteps(group int, src ViewerSource) ([]ViewerStep, error) {
	steps := []ViewerStep{}
	err := a.readGroup(group, func(d *database.Database) error {
		switch src.Kind {
		case "match":
			moves, err := d.GetMatchMovePositions(src.ID)
			if err != nil {
				return err
			}
			for _, m := range moves {
				played := m.CheckerMove
				if m.MoveType == "cube" {
					played = m.CubeAction
				}
				steps = append(steps, ViewerStep{PositionID: m.Position.ID, GameNumber: m.GameNumber, MoveNumber: m.MoveNumber, Player: m.PlayerOnRoll, Played: strings.TrimSpace(played)})
			}
			return nil
		case "collection":
			ids, err := d.ListCollectionPositionIDs(src.ID, 0, 0)
			steps = appendIDs(steps, ids)
			return err
		case "library", "":
			ids, err := d.ListPositionIDs(0, 0)
			steps = appendIDs(steps, ids)
			return err
		default:
			return fmt.Errorf("unknown viewer source %q", src.Kind)
		}
	})
	return steps, err
}

func appendIDs(steps []ViewerStep, ids []int64) []ViewerStep {
	for _, id := range ids {
		steps = append(steps, ViewerStep{PositionID: id})
	}
	return steps
}

// ViewerShow reads position id of group with its analysis and comment.
func (a *App) ViewerShow(group int, id int64) (*ViewerItem, error) {
	var item *ViewerItem
	err := a.readGroup(group, func(d *database.Database) error {
		pos, err := d.LoadPosition(int(id))
		if err != nil {
			return err
		}
		view, err := d.LoadPositionView(id)
		if err != nil {
			return err
		}
		item = &ViewerItem{Position: *pos, Analysis: view.Analysis, Comment: view.Comment}
		return nil
	})
	return item, err
}

// ViewerCatalogOf lists the matches and collections of group.
func (a *App) ViewerCatalogOf(group int) (ViewerCatalog, error) {
	cat := ViewerCatalog{Matches: []ViewerEntry{}, Collections: []ViewerEntry{}}
	err := a.readGroup(group, func(d *database.Database) error {
		matches, err := d.GetAllMatches()
		if err != nil {
			return err
		}
		for _, m := range matches {
			label := m.Player1Name + " – " + m.Player2Name
			if m.MatchLength > 0 {
				label += fmt.Sprintf(" (%d)", m.MatchLength)
			}
			if !m.MatchDate.IsZero() {
				label += " · " + m.MatchDate.Format("2006-01-02")
			}
			cat.Matches = append(cat.Matches, ViewerEntry{ID: m.ID, Label: label, Count: m.GameCount})
		}
		cols, err := d.GetAllCollections()
		if err != nil {
			return err
		}
		for _, c := range cols {
			cat.Collections = append(cat.Collections, ViewerEntry{ID: c.ID, Label: c.Name, Count: c.PositionCount})
		}
		return nil
	})
	return cat, err
}
