package sqlshared

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// LessonStore implements storage.LessonStore over the lesson and lesson_step
// tables. Every statement is confined to the scope's tenant, and a Step may
// only point at a Collection or a Position of that same tenant.
type LessonStore struct{ DB Execer }

var _ storage.LessonStore = (*LessonStore)(nil)

func (s *LessonStore) Create(ctx context.Context, scope string, name, description string) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("%s: create lesson: empty name: %w", s.DB.Name(), storage.ErrInvalid)
	}
	cols, args := s.DB.TenantColumns(scope)
	cols = append(cols, "name", "description")
	args = append(args, name, description)
	id, err := s.DB.Insert(ctx,
		`INSERT INTO lesson (`+strings.Join(cols, ", ")+`) VALUES (`+Placeholders(len(cols))+`)`, args...)
	if err != nil {
		return 0, errf(s.DB, "create lesson", err)
	}
	return id, nil
}

func (s *LessonStore) lessonCols() string {
	return `l.id, l.name, COALESCE(l.description,''), ` + s.DB.TimestampText("l.created_at") + `, ` +
		s.DB.TimestampText("l.updated_at") + `, (SELECT COUNT(*) FROM lesson_step st WHERE st.lesson_id = l.id)`
}

func scanLesson(sc interface{ Scan(...any) error }) (*domain.Lesson, error) {
	var l domain.Lesson
	if err := sc.Scan(&l.ID, &l.Name, &l.Description, &l.CreatedAt, &l.UpdatedAt, &l.StepCount); err != nil {
		return nil, err
	}
	l.Steps = []domain.LessonStep{}
	return &l, nil
}

func (s *LessonStore) Get(ctx context.Context, scope string, id int64) (*domain.Lesson, error) {
	tenant, targs := s.DB.TenantFilter("l", scope)
	l, err := scanLesson(s.DB.QueryRow(ctx,
		`SELECT `+s.lessonCols()+` FROM lesson l WHERE l.id = ? AND `+tenant,
		append([]any{id}, targs...)...))
	if errors.Is(err, ErrNoRows) {
		return nil, fmt.Errorf("%s: get lesson %d: %w", s.DB.Name(), id, storage.ErrNotFound)
	}
	if err != nil {
		return nil, errf(s.DB, "get lesson", err)
	}
	stenant, sargs := s.DB.TenantFilter("", scope)
	rows, err := s.DB.Query(ctx,
		`SELECT id, lesson_id, title, text, COALESCE(collection_id, 0), COALESCE(position_id, 0)
		 FROM lesson_step WHERE lesson_id = ? AND `+stenant+` ORDER BY sort_order, id`,
		append([]any{id}, sargs...)...)
	if err != nil {
		return nil, errf(s.DB, "list lesson steps", err)
	}
	defer rows.Close()
	for rows.Next() {
		var st domain.LessonStep
		if err := rows.Scan(&st.ID, &st.LessonID, &st.Title, &st.Text, &st.CollectionID, &st.PositionID); err != nil {
			return nil, errf(s.DB, "scan lesson step", err)
		}
		l.Steps = append(l.Steps, st)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "list lesson steps", err)
	}
	l.StepCount = len(l.Steps)
	return l, nil
}

func (s *LessonStore) List(ctx context.Context, scope string) ([]*domain.Lesson, error) {
	tenant, targs := s.DB.TenantFilter("l", scope)
	rows, err := s.DB.Query(ctx,
		`SELECT `+s.lessonCols()+` FROM lesson l WHERE `+tenant+` ORDER BY l.name, l.id`, targs...)
	if err != nil {
		return nil, errf(s.DB, "list lessons", err)
	}
	defer rows.Close()
	out := []*domain.Lesson{}
	for rows.Next() {
		l, err := scanLesson(rows)
		if err != nil {
			return nil, errf(s.DB, "list lessons", err)
		}
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "list lessons", err)
	}
	return out, nil
}

func (s *LessonStore) Update(ctx context.Context, scope string, id int64, name, description string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("%s: update lesson %d: empty name: %w", s.DB.Name(), id, storage.ErrInvalid)
	}
	tenant, targs := s.DB.TenantFilter("", scope)
	n, err := s.DB.Exec(ctx,
		`UPDATE lesson SET name = ?, description = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND `+tenant,
		append([]any{name, description, id}, targs...)...)
	if err != nil {
		return errf(s.DB, "update lesson", err)
	}
	if n == 0 {
		return fmt.Errorf("%s: update lesson %d: %w", s.DB.Name(), id, storage.ErrNotFound)
	}
	return nil
}

func (s *LessonStore) Delete(ctx context.Context, scope string, id int64) error {
	return s.DB.Transact(ctx, func(tx Execer) error {
		tenant, targs := tx.TenantFilter("", scope)
		if _, err := tx.Exec(ctx, `DELETE FROM lesson_step WHERE lesson_id = ? AND `+tenant,
			append([]any{id}, targs...)...); err != nil {
			return errf(tx, "delete lesson steps", err)
		}
		n, err := tx.Exec(ctx, `DELETE FROM lesson WHERE id = ? AND `+tenant, append([]any{id}, targs...)...)
		if err != nil {
			return errf(tx, "delete lesson", err)
		}
		if n == 0 {
			return fmt.Errorf("%s: delete lesson %d: %w", tx.Name(), id, storage.ErrNotFound)
		}
		return nil
	})
}

// rowExists reports whether table holds row id in the scope's tenant.
func rowExists(ctx context.Context, db Execer, scope, table string, id int64) (bool, error) {
	tenant, targs := db.TenantFilter("", scope)
	var one int
	err := db.QueryRow(ctx, `SELECT 1 FROM `+table+` WHERE id = ? AND `+tenant,
		append([]any{id}, targs...)...).Scan(&one)
	if errors.Is(err, ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, errf(db, "probe "+table, err)
	}
	return true, nil
}

// checkTargets refuses a Step that points at a Collection or a Position the
// tenant does not hold: a dangling reference would read as an empty Step.
func checkTargets(ctx context.Context, db Execer, scope string, st domain.LessonStep) error {
	for _, t := range []struct {
		table string
		id    int64
	}{{"collection", st.CollectionID}, {"position", st.PositionID}} {
		if t.id == 0 {
			continue
		}
		ok, err := rowExists(ctx, db, scope, t.table, t.id)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%s: lesson step: no %s %d: %w", db.Name(), t.table, t.id, storage.ErrInvalid)
		}
	}
	return nil
}

func nullID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

func (s *LessonStore) touch(ctx context.Context, db Execer, scope string, lessonID int64) error {
	tenant, targs := db.TenantFilter("", scope)
	if _, err := db.Exec(ctx, `UPDATE lesson SET updated_at = CURRENT_TIMESTAMP WHERE id = ? AND `+tenant,
		append([]any{lessonID}, targs...)...); err != nil {
		return errf(db, "touch lesson", err)
	}
	return nil
}

func (s *LessonStore) AddStep(ctx context.Context, scope string, lessonID int64, step domain.LessonStep) (int64, error) {
	var id int64
	err := s.DB.Transact(ctx, func(tx Execer) error {
		ok, err := rowExists(ctx, tx, scope, "lesson", lessonID)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%s: add step to lesson %d: %w", tx.Name(), lessonID, storage.ErrNotFound)
		}
		if err := checkTargets(ctx, tx, scope, step); err != nil {
			return err
		}
		tenant, targs := tx.TenantFilter("", scope)
		var next int
		if err := tx.QueryRow(ctx,
			`SELECT COALESCE(MAX(sort_order), 0) + 1 FROM lesson_step WHERE lesson_id = ? AND `+tenant,
			append([]any{lessonID}, targs...)...).Scan(&next); err != nil {
			return errf(tx, "next step order", err)
		}
		cols, args := tx.TenantColumns(scope)
		cols = append(cols, "lesson_id", "sort_order", "title", "text", "collection_id", "position_id")
		args = append(args, lessonID, next, step.Title, step.Text, nullID(step.CollectionID), nullID(step.PositionID))
		id, err = tx.Insert(ctx,
			`INSERT INTO lesson_step (`+strings.Join(cols, ", ")+`) VALUES (`+Placeholders(len(cols))+`)`, args...)
		if err != nil {
			return errf(tx, "add lesson step", err)
		}
		return s.touch(ctx, tx, scope, lessonID)
	})
	return id, err
}

func (s *LessonStore) stepLesson(ctx context.Context, db Execer, scope string, stepID int64) (int64, error) {
	tenant, targs := db.TenantFilter("", scope)
	var lessonID int64
	err := db.QueryRow(ctx, `SELECT lesson_id FROM lesson_step WHERE id = ? AND `+tenant,
		append([]any{stepID}, targs...)...).Scan(&lessonID)
	if errors.Is(err, ErrNoRows) {
		return 0, fmt.Errorf("%s: lesson step %d: %w", db.Name(), stepID, storage.ErrNotFound)
	}
	if err != nil {
		return 0, errf(db, "probe lesson step", err)
	}
	return lessonID, nil
}

func (s *LessonStore) UpdateStep(ctx context.Context, scope string, step domain.LessonStep) error {
	return s.DB.Transact(ctx, func(tx Execer) error {
		lessonID, err := s.stepLesson(ctx, tx, scope, step.ID)
		if err != nil {
			return err
		}
		if err := checkTargets(ctx, tx, scope, step); err != nil {
			return err
		}
		tenant, targs := tx.TenantFilter("", scope)
		if _, err := tx.Exec(ctx,
			`UPDATE lesson_step SET title = ?, text = ?, collection_id = ?, position_id = ? WHERE id = ? AND `+tenant,
			append([]any{step.Title, step.Text, nullID(step.CollectionID), nullID(step.PositionID), step.ID}, targs...)...); err != nil {
			return errf(tx, "update lesson step", err)
		}
		return s.touch(ctx, tx, scope, lessonID)
	})
}

func (s *LessonStore) RemoveStep(ctx context.Context, scope string, stepID int64) error {
	return s.DB.Transact(ctx, func(tx Execer) error {
		lessonID, err := s.stepLesson(ctx, tx, scope, stepID)
		if err != nil {
			return err
		}
		tenant, targs := tx.TenantFilter("", scope)
		if _, err := tx.Exec(ctx, `DELETE FROM lesson_step WHERE id = ? AND `+tenant,
			append([]any{stepID}, targs...)...); err != nil {
			return errf(tx, "remove lesson step", err)
		}
		return s.touch(ctx, tx, scope, lessonID)
	})
}

func (s *LessonStore) ReorderSteps(ctx context.Context, scope string, lessonID int64, stepIDs []int64) error {
	return s.DB.Transact(ctx, func(tx Execer) error {
		ok, err := rowExists(ctx, tx, scope, "lesson", lessonID)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%s: reorder lesson %d: %w", tx.Name(), lessonID, storage.ErrNotFound)
		}
		tenant, targs := tx.TenantFilter("", scope)
		rows, err := tx.Query(ctx, `SELECT id FROM lesson_step WHERE lesson_id = ? AND `+tenant,
			append([]any{lessonID}, targs...)...)
		if err != nil {
			return errf(tx, "list lesson steps", err)
		}
		held := map[int64]bool{}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return errf(tx, "scan lesson step", err)
			}
			held[id] = true
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return errf(tx, "list lesson steps", err)
		}
		seen := map[int64]bool{}
		for _, id := range stepIDs {
			if !held[id] || seen[id] {
				return fmt.Errorf("%s: reorder lesson %d: step %d: %w", tx.Name(), lessonID, id, storage.ErrInvalid)
			}
			seen[id] = true
		}
		if len(seen) != len(held) {
			return fmt.Errorf("%s: reorder lesson %d: %d steps named, %d held: %w",
				tx.Name(), lessonID, len(seen), len(held), storage.ErrInvalid)
		}
		for i, id := range stepIDs {
			if _, err := tx.Exec(ctx, `UPDATE lesson_step SET sort_order = ? WHERE id = ? AND `+tenant,
				append([]any{i + 1, id}, targs...)...); err != nil {
				return errf(tx, "reorder lesson step", err)
			}
		}
		return s.touch(ctx, tx, scope, lessonID)
	})
}
