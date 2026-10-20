package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/report"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The Lesson family (ADR-0066) is an adapter over storage.LessonStore: every
// method takes d.mu and passes the desktop's implicit scope (""). The CLI and
// the daemon reach the same store, so the three modes share one rule.

func (d *Database) lessonStore() (storage.LessonStore, error) {
	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}
	return d.store.Lessons(), nil
}

func (d *Database) readLessons(fn func(storage.LessonStore) error) error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	ls, err := d.lessonStore()
	if err != nil {
		return err
	}
	return fn(ls)
}

func (d *Database) writeLessons(fn func(storage.LessonStore) error) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	ls, err := d.lessonStore()
	if err != nil {
		return err
	}
	return fn(ls)
}

// ListLessons returns every Lesson by name, with its step count.
func (d *Database) ListLessons() ([]*domain.Lesson, error) {
	var out []*domain.Lesson
	err := d.readLessons(func(ls storage.LessonStore) (err error) {
		out, err = ls.List(context.Background(), "")
		return err
	})
	return out, err
}

// GetLesson returns one Lesson with its steps in reading order.
func (d *Database) GetLesson(id int64) (*domain.Lesson, error) {
	var out *domain.Lesson
	err := d.readLessons(func(ls storage.LessonStore) (err error) {
		out, err = ls.Get(context.Background(), "", id)
		return err
	})
	return out, err
}

// CreateLesson creates an empty Lesson and returns its id.
func (d *Database) CreateLesson(name, description string) (int64, error) {
	var id int64
	err := d.writeLessons(func(ls storage.LessonStore) (err error) {
		id, err = ls.Create(context.Background(), "", name, description)
		return err
	})
	return id, err
}

// UpdateLesson renames a Lesson and rewrites its description.
func (d *Database) UpdateLesson(id int64, name, description string) error {
	return d.writeLessons(func(ls storage.LessonStore) error {
		return ls.Update(context.Background(), "", id, name, description)
	})
}

// DeleteLesson deletes a Lesson and its steps; what they showed stays.
func (d *Database) DeleteLesson(id int64) error {
	return d.writeLessons(func(ls storage.LessonStore) error {
		return ls.Delete(context.Background(), "", id)
	})
}

// AddLessonStep appends a step; collectionID and positionID are 0 when the
// step shows none.
func (d *Database) AddLessonStep(lessonID int64, title, text string, collectionID, positionID int64) (int64, error) {
	var id int64
	err := d.writeLessons(func(ls storage.LessonStore) (err error) {
		id, err = ls.AddStep(context.Background(), "", lessonID, domain.LessonStep{
			Title: title, Text: text, CollectionID: collectionID, PositionID: positionID,
		})
		return err
	})
	return id, err
}

// UpdateLessonStep rewrites a step's title, text, collection and position.
func (d *Database) UpdateLessonStep(stepID int64, title, text string, collectionID, positionID int64) error {
	return d.writeLessons(func(ls storage.LessonStore) error {
		return ls.UpdateStep(context.Background(), "", domain.LessonStep{
			ID: stepID, Title: title, Text: text, CollectionID: collectionID, PositionID: positionID,
		})
	})
}

// RemoveLessonStep removes one step.
func (d *Database) RemoveLessonStep(stepID int64) error {
	return d.writeLessons(func(ls storage.LessonStore) error {
		return ls.RemoveStep(context.Background(), "", stepID)
	})
}

// ReorderLessonSteps sets the order of a Lesson's steps; stepIDs names each
// of them once.
func (d *Database) ReorderLessonSteps(lessonID int64, stepIDs []int64) error {
	return d.writeLessons(func(ls storage.LessonStore) error {
		return ls.ReorderSteps(context.Background(), "", lessonID, stepIDs)
	})
}

// SetLessonStepDone records (done) or withdraws the reader's "step done"
// gesture on a step. It is the only writer of a Lesson's progress
// (ADR-0069): opening or reading a Lesson writes nothing.
func (d *Database) SetLessonStepDone(stepID int64, done bool) error {
	return d.writeLessons(func(ls storage.LessonStore) error {
		return ls.SetStepDone(context.Background(), "", stepID, done)
	})
}

// LessonDoneSteps returns the date of the "step done" gesture for each step
// of the Lesson marked done, keyed by step id.
func (d *Database) LessonDoneSteps(lessonID int64) (map[int64]string, error) {
	var out map[int64]string
	err := d.readLessons(func(ls storage.LessonStore) (err error) {
		out, err = ls.DoneSteps(context.Background(), "", lessonID)
		return err
	})
	return out, err
}

// LessonHTML renders the Lesson id as one self-contained HTML document (package
// report), written in lang (English when it has no catalogue). diagrams maps a
// position id to a ready SVG the caller drew itself, as the desktop app does
// with renderPositionSVG in the board palette; a position it does not cover
// gets the built-in diagram, and nil leaves every board to it. Nothing is
// written to the database.
func (d *Database) LessonHTML(id int64, lang string, diagrams map[int64]string) (string, error) {
	return d.LessonHTMLCtx(context.Background(), id, lang, diagrams)
}

// LessonHTMLCtx is LessonHTML under ctx.
func (d *Database) LessonHTMLCtx(ctx context.Context, id int64, lang string, diagrams map[int64]string) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var draw report.Diagrammer
	if len(diagrams) > 0 {
		draw = func(p *domain.Position) string {
			if svg, ok := diagrams[p.ID]; ok && svg != "" {
				return svg
			}
			return report.Diagram(p)
		}
	}
	return report.BuildLesson(ctx, d.store, "", id, lang, draw)
}

// LessonPositionIDs lists the positions the Lesson id shows, Steps' own and the
// first ones of their Collections, so a caller can draw their boards before
// asking for the document.
func (d *Database) LessonPositionIDs(id int64) ([]int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	ctx := context.Background()
	l, err := d.store.Lessons().Get(ctx, "", id)
	if err != nil {
		return nil, err
	}
	var ids []int64
	for _, st := range l.Steps {
		if st.CollectionID > 0 {
			got, err := d.store.Collections().PositionIDs(ctx, "", st.CollectionID, storage.ListOpts{Limit: report.MaxCollectionBoards})
			if err == nil {
				ids = append(ids, got...)
			}
		}
		if st.PositionID > 0 {
			ids = append(ids, st.PositionID)
		}
	}
	return ids, nil
}
