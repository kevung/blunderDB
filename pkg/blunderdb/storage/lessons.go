package storage

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// LessonStore persists Lessons and their Steps (ADR-0066). A Step's order is
// its rank in Lesson.Steps; AddStep appends, ReorderSteps rewrites the order.
// A Step that names a Collection or a Position of another tenant, or one that
// does not exist, is ErrInvalid and writes nothing.
type LessonStore interface {
	Create(ctx context.Context, scope string, name, description string) (int64, error)
	// Get returns the Lesson with its Steps in order, or ErrNotFound.
	Get(ctx context.Context, scope string, id int64) (*domain.Lesson, error)
	// List returns every Lesson by name, with StepCount and no Steps.
	List(ctx context.Context, scope string) ([]*domain.Lesson, error)
	Update(ctx context.Context, scope string, id int64, name, description string) error
	// Delete removes the Lesson and its Steps; the Collections and Positions
	// they showed stay.
	Delete(ctx context.Context, scope string, id int64) error

	// AddStep appends step to the Lesson and returns the Step's id.
	AddStep(ctx context.Context, scope string, lessonID int64, step domain.LessonStep) (int64, error)
	// UpdateStep rewrites the Step step.ID: title, text, collection, position.
	UpdateStep(ctx context.Context, scope string, step domain.LessonStep) error
	RemoveStep(ctx context.Context, scope string, stepID int64) error
	// ReorderSteps sets the order of the Lesson's Steps. stepIDs must name
	// every Step of the Lesson exactly once, or it is ErrInvalid.
	ReorderSteps(ctx context.Context, scope string, lessonID int64, stepIDs []int64) error
}
