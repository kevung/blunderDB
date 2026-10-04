package sqlshared

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// MatchEquityTableStore implements storage.MatchEquityTableStore over the
// match_equity_table table, confined to the scope's tenant.
type MatchEquityTableStore struct{ DB Execer }

var _ storage.MatchEquityTableStore = (*MatchEquityTableStore)(nil)

func (s *MatchEquityTableStore) Save(ctx context.Context, scope string, met domain.MatchEquityTable) (int64, error) {
	name := strings.TrimSpace(met.Name)
	if name == "" || met.Digest == "" || met.Source == "" {
		return 0, fmt.Errorf("%s: save match equity table: empty name, digest or source: %w", s.DB.Name(), storage.ErrInvalid)
	}
	var id int64
	err := s.DB.Transact(ctx, func(tx Execer) error {
		tenant, targs := tx.TenantFilter("", scope)
		err := tx.QueryRow(ctx, `SELECT id FROM match_equity_table WHERE digest = ? AND `+tenant,
			append([]any{met.Digest}, targs...)...).Scan(&id)
		if err == nil {
			return nil
		}
		if !errors.Is(err, ErrNoRows) {
			return errf(tx, "probe match equity table", err)
		}
		cols, args := tx.TenantColumns(scope)
		cols = append(cols, "name", "digest", "source")
		args = append(args, name, met.Digest, met.Source)
		id, err = tx.Insert(ctx, `INSERT INTO match_equity_table (`+strings.Join(cols, ", ")+
			`) VALUES (`+Placeholders(len(cols))+`)`, args...)
		if err != nil {
			return errf(tx, "save match equity table", err)
		}
		return nil
	})
	return id, err
}

func (s *MatchEquityTableStore) cols(withSource bool) string {
	source := `''`
	if withSource {
		source = `source`
	}
	return `id, name, digest, ` + source + `, is_current, ` + s.DB.TimestampText("created_at")
}

func scanMET(sc interface{ Scan(...any) error }) (*domain.MatchEquityTable, error) {
	var m domain.MatchEquityTable
	var current int
	if err := sc.Scan(&m.ID, &m.Name, &m.Digest, &m.Source, &current, &m.CreatedAt); err != nil {
		return nil, err
	}
	m.Current = current != 0
	return &m, nil
}

func (s *MatchEquityTableStore) List(ctx context.Context, scope string) ([]*domain.MatchEquityTable, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	rows, err := s.DB.Query(ctx, `SELECT `+s.cols(false)+` FROM match_equity_table WHERE `+tenant+
		` ORDER BY name, id`, targs...)
	if err != nil {
		return nil, errf(s.DB, "list match equity tables", err)
	}
	defer rows.Close()
	out := []*domain.MatchEquityTable{}
	for rows.Next() {
		m, err := scanMET(rows)
		if err != nil {
			return nil, errf(s.DB, "scan match equity table", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "list match equity tables", err)
	}
	return out, nil
}

func (s *MatchEquityTableStore) Current(ctx context.Context, scope string) (*domain.MatchEquityTable, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	m, err := scanMET(s.DB.QueryRow(ctx, `SELECT `+s.cols(true)+` FROM match_equity_table WHERE `+tenant+
		` AND is_current = 1 ORDER BY id LIMIT 1`, targs...))
	if errors.Is(err, ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, errf(s.DB, "current match equity table", err)
	}
	return m, nil
}

func (s *MatchEquityTableStore) SetCurrent(ctx context.Context, scope string, id int64) error {
	return s.DB.Transact(ctx, func(tx Execer) error {
		if id != 0 {
			ok, err := rowExists(ctx, tx, scope, "match_equity_table", id)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("%s: match equity table %d: %w", tx.Name(), id, storage.ErrNotFound)
			}
		}
		tenant, targs := tx.TenantFilter("", scope)
		if _, err := tx.Exec(ctx, `UPDATE match_equity_table SET is_current = CASE WHEN id = ? THEN 1 ELSE 0 END WHERE `+tenant,
			append([]any{id}, targs...)...); err != nil {
			return errf(tx, "set current match equity table", err)
		}
		return nil
	})
}

func (s *MatchEquityTableStore) TagAnalyses(ctx context.Context, scope string, metID int64, positionIDs []int64) error {
	if len(positionIDs) == 0 {
		return nil
	}
	var value any
	if metID != 0 {
		value = metID
	}
	return s.DB.Transact(ctx, func(tx Execer) error {
		tenant, targs := tx.TenantFilter("", scope)
		for start := 0; start < len(positionIDs); start += 500 {
			chunk := positionIDs[start:min(start+500, len(positionIDs))]
			args := append([]any{value}, targs...)
			for _, id := range chunk {
				args = append(args, id)
			}
			if _, err := tx.Exec(ctx, `UPDATE analysis SET met_id = ? WHERE `+tenant+
				` AND position_id IN (`+Placeholders(len(chunk))+`)`, args...); err != nil {
				return errf(tx, "tag analyses with their match equity table", err)
			}
		}
		return nil
	})
}

func (s *MatchEquityTableStore) OfAnalysis(ctx context.Context, scope string, positionID int64) (int64, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	var id *int64
	err := s.DB.QueryRow(ctx, `SELECT met_id FROM analysis WHERE `+tenant+` AND position_id = ?`,
		append(targs, positionID)...).Scan(&id)
	if errors.Is(err, ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, errf(s.DB, "match equity table of an analysis", err)
	}
	if id == nil {
		return 0, nil
	}
	return *id, nil
}
