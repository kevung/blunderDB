package ingest

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ReadSourceLessons reads every Lesson of a native .db being imported, with
// its steps in order.
func ReadSourceLessons(ctx context.Context, source storage.Stores, scope string) ([]*domain.Lesson, error) {
	list, err := source.Lessons().List(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("ingest: list source lessons: %w", err)
	}
	out := make([]*domain.Lesson, 0, len(list))
	for _, l := range list {
		full, err := source.Lessons().Get(ctx, scope, l.ID)
		if err != nil {
			return nil, fmt.Errorf("ingest: read source lesson %q: %w", l.Name, err)
		}
		out = append(out, full)
	}
	return out, nil
}

// MergeLessons is the one rule both native .db imports apply to Lessons
// (ADR-0066), after MergeCollections: a source Lesson whose name the target
// already holds is left alone, so importing the same file twice changes
// nothing and a Lesson the recipient edited is never overwritten; any other
// is created with its steps. A step's position is remapped through targetOf,
// its collection through the source collection's name, which MergeCollections
// has just made exist in the target. It returns the number of Lessons created.
func MergeLessons(ctx context.Context, tx storage.Stores, scope string, src []*domain.Lesson, srcCollections []SourceCollection, targetOf map[int64]int64) (int, error) {
	if len(src) == 0 {
		return 0, nil
	}
	held, err := tx.Lessons().List(ctx, scope)
	if err != nil {
		return 0, err
	}
	names := make(map[string]bool, len(held))
	for _, l := range held {
		names[l.Name] = true
	}
	collName := make(map[int64]string, len(srcCollections))
	for _, sc := range srcCollections {
		collName[sc.Coll.ID] = sc.Coll.Name
	}
	targetColl := map[string]int64{}
	for c, err := range tx.Collections().List(ctx, scope) {
		if err != nil {
			return 0, err
		}
		targetColl[c.Name] = c.ID
	}
	created := 0
	for _, l := range src {
		if err := ctx.Err(); err != nil {
			return created, err
		}
		if names[l.Name] {
			continue
		}
		id, err := tx.Lessons().Create(ctx, scope, l.Name, l.Description)
		if err != nil {
			return created, err
		}
		names[l.Name] = true
		for _, st := range l.Steps {
			st.PositionID = targetOf[st.PositionID]
			name, ok := collName[st.CollectionID]
			st.CollectionID = 0
			if ok {
				st.CollectionID = targetColl[name]
			}
			if _, err := tx.Lessons().AddStep(ctx, scope, id, st); err != nil {
				return created, fmt.Errorf("ingest: import a step of lesson %q: %w", l.Name, err)
			}
		}
		created++
	}
	return created, nil
}
