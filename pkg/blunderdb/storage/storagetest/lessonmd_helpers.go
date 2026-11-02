package storagetest

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func domainStep(title, text string) domain.LessonStep {
	return domain.LessonStep{Title: title, Text: text}
}

// cutAfterSteps is a Storage whose transactions cancel the import's context
// after `left` Steps, as a client that went away would.
type cutAfterSteps struct {
	storage.Storage
	left   int
	cancel context.CancelFunc
}

func (c *cutAfterSteps) BeginTx(ctx context.Context) (storage.Tx, error) {
	tx, err := c.Storage.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	return &cutTx{Tx: tx, cut: c}, nil
}

type cutTx struct {
	storage.Tx
	cut *cutAfterSteps
}

func (t *cutTx) Lessons() storage.LessonStore {
	return &cutLessons{LessonStore: t.Tx.Lessons(), cut: t.cut}
}

type cutLessons struct {
	storage.LessonStore
	cut *cutAfterSteps
}

func (l *cutLessons) AddStep(ctx context.Context, scope string, lessonID int64, step domain.LessonStep) (int64, error) {
	if l.cut.left--; l.cut.left < 0 {
		l.cut.cancel()
		return 0, ctx.Err()
	}
	return l.LessonStore.AddStep(ctx, scope, lessonID, step)
}
