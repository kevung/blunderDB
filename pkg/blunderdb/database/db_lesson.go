package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
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
