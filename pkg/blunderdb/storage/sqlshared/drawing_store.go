package sqlshared

import (
	"context"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// DrawingStore implements storage.DrawingStore over position_drawing.
type DrawingStore struct{ DB Execer }

var _ storage.DrawingStore = (*DrawingStore)(nil)

func (s *DrawingStore) Drawing(ctx context.Context, scope string, positionID int64) ([]domain.DrawingMark, error) {
	if err := requirePositionIn(ctx, s.DB, scope, positionID, "drawing"); err != nil {
		return nil, err
	}
	m, err := s.Drawings(ctx, scope, []int64{positionID})
	return m[positionID], err
}

func (s *DrawingStore) SetDrawing(ctx context.Context, scope string, positionID int64, marks []domain.DrawingMark) error {
	if err := domain.ValidateDrawing(marks); err != nil {
		return fmt.Errorf("%w: %w", storage.ErrInvalid, err)
	}
	if err := requirePositionIn(ctx, s.DB, scope, positionID, "drawing"); err != nil {
		return err
	}
	return s.DB.Transact(ctx, func(tx Execer) error {
		tenant, targs := tx.TenantFilter("", scope)
		if _, err := tx.Exec(ctx, `DELETE FROM position_drawing WHERE position_id = ? AND `+tenant,
			append([]any{positionID}, targs...)...); err != nil {
			return errf(tx, "clear drawing", err)
		}
		for i, m := range marks {
			if m.Kind == domain.DrawingSquare {
				m.To = m.From
			}
			cols, args := tx.TenantColumns(scope)
			cols = append(cols, "position_id", "rank", "kind", "from_point", "to_point", "color")
			args = append(args, positionID, i, m.Kind, m.From, m.To, m.Color)
			if _, err := tx.Exec(ctx, `INSERT INTO position_drawing (`+strings.Join(cols, ", ")+`) VALUES (`+
				Placeholders(len(cols))+`)`, args...); err != nil {
				return errf(tx, "draw mark", err)
			}
		}
		return nil
	})
}

func (s *DrawingStore) Drawings(ctx context.Context, scope string, positionIDs []int64) (map[int64][]domain.DrawingMark, error) {
	out := make(map[int64][]domain.DrawingMark)
	tenant, targs := s.DB.TenantFilter("", scope)
	for start := 0; start < len(positionIDs); start += byPositionsChunk {
		batch := positionIDs[start:min(start+byPositionsChunk, len(positionIDs))]
		args := make([]any, 0, len(batch)+len(targs))
		for _, id := range batch {
			args = append(args, id)
		}
		args = append(args, targs...)
		rows, err := s.DB.Query(ctx, `SELECT position_id, kind, from_point, to_point, color FROM position_drawing
			WHERE position_id IN (`+Placeholders(len(batch))+`) AND `+tenant+` ORDER BY position_id, rank`, args...)
		if err != nil {
			return nil, errf(s.DB, "drawings", err)
		}
		for rows.Next() {
			var id int64
			var m domain.DrawingMark
			if err := rows.Scan(&id, &m.Kind, &m.From, &m.To, &m.Color); err != nil {
				rows.Close()
				return nil, errf(s.DB, "drawings", err)
			}
			out[id] = append(out[id], m)
		}
		rerr := rows.Err()
		rows.Close()
		if rerr != nil {
			return nil, errf(s.DB, "drawings", rerr)
		}
	}
	return out, nil
}
