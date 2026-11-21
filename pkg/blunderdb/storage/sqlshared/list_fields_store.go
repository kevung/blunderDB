package sqlshared

import (
	"context"
	"encoding/hex"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ListFieldsStore implements storage.ListFieldsStore: three queries per chunk
// of ids, one over position and its analysis, one over the collections, one
// over the study decks.
type ListFieldsStore struct{ DB Execer }

var _ storage.ListFieldsStore = (*ListFieldsStore)(nil)

func (s *ListFieldsStore) ListFields(ctx context.Context, scope string, ids []int64) ([]storage.PositionListFields, error) {
	byID := make(map[int64]*storage.PositionListFields, len(ids))
	for start := 0; start < len(ids); start += byPositionsChunk {
		batch := ids[start:min(start+byPositionsChunk, len(ids))]
		if err := s.readPositions(ctx, scope, batch, byID); err != nil {
			return nil, err
		}
		if err := s.readCollections(ctx, scope, batch, byID); err != nil {
			return nil, err
		}
		if err := s.readDecks(ctx, scope, batch, byID); err != nil {
			return nil, err
		}
	}
	out := make([]storage.PositionListFields, 0, len(ids))
	for _, id := range ids {
		if f, ok := byID[id]; ok {
			out = append(out, *f)
		}
	}
	return out, nil
}

func (s *ListFieldsStore) idArgs(scope string, batch []int64) (string, []any) {
	tenant, targs := s.DB.TenantFilter("p", scope)
	args := make([]any, 0, len(batch)+len(targs))
	for _, id := range batch {
		args = append(args, id)
	}
	return tenant, append(args, targs...)
}

func (s *ListFieldsStore) readPositions(ctx context.Context, scope string, batch []int64, byID map[int64]*storage.PositionListFields) error {
	tenant, args := s.idArgs(scope, batch)
	rows, err := s.DB.Query(ctx, `SELECT p.id, COALESCE(p.decision_type, 0), p.match_date, CAST(p.uuid AS TEXT),
		       a.position_id IS NOT NULL, a.cube_error, a.best_move_equity_error
		  FROM position p LEFT JOIN analysis a ON a.position_id = p.id`+andSameTenant(s.DB, "a", "p")+`
		 WHERE p.id IN (`+Placeholders(len(batch))+`) AND `+tenant, args...)
	if err != nil {
		return errf(s.DB, "list fields", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id               int64
			decisionType     int
			matchDate        any
			uuid             *string
			analysed         bool
			cubeErr, moveErr *int64
		)
		if err := rows.Scan(&id, &decisionType, &matchDate, &uuid, &analysed, &cubeErr, &moveErr); err != nil {
			return errf(s.DB, "list fields", err)
		}
		f := &storage.PositionListFields{PositionID: id, Analysed: analysed, Collections: []storage.CollectionRef{}, Decks: []storage.DeckRef{}}
		if analysed {
			f.PlayedErrorMP = moveErr
			if decisionType == 1 {
				f.PlayedErrorMP = cubeErr
			}
		}
		if day, ok := positionMatchDay(matchDate); ok {
			f.Date, f.MatchDated = day, true
		} else if uuid != nil {
			f.Date = uuidV7Day(*uuid)
		}
		byID[id] = f
	}
	if err := rows.Err(); err != nil {
		return errf(s.DB, "list fields", err)
	}
	return nil
}

func (s *ListFieldsStore) readCollections(ctx context.Context, scope string, batch []int64, byID map[int64]*storage.PositionListFields) error {
	tenant, targs := s.DB.TenantFilter("cp", scope)
	args := make([]any, 0, len(batch)+len(targs))
	for _, id := range batch {
		args = append(args, id)
	}
	args = append(args, targs...)
	rows, err := s.DB.Query(ctx, `SELECT cp.position_id, c.id, COALESCE(c.name, '')
		  FROM collection_position cp JOIN collection c ON c.id = cp.collection_id`+andSameTenant(s.DB, "c", "cp")+`
		 WHERE cp.position_id IN (`+Placeholders(len(batch))+`) AND `+tenant+`
		 ORDER BY cp.position_id, c.sort_order, c.id`, args...)
	if err != nil {
		return errf(s.DB, "list fields collections", err)
	}
	defer rows.Close()
	for rows.Next() {
		var posID int64
		var c storage.CollectionRef
		if err := rows.Scan(&posID, &c.ID, &c.Name); err != nil {
			return errf(s.DB, "list fields collections", err)
		}
		if f, ok := byID[posID]; ok {
			f.Collections = append(f.Collections, c)
		}
	}
	if err := rows.Err(); err != nil {
		return errf(s.DB, "list fields collections", err)
	}
	return nil
}

func (s *ListFieldsStore) readDecks(ctx context.Context, scope string, batch []int64, byID map[int64]*storage.PositionListFields) error {
	tenant, targs := s.DB.TenantFilter("ac", scope)
	args := make([]any, 0, len(batch)+len(targs))
	for _, id := range batch {
		args = append(args, id)
	}
	args = append(args, targs...)
	rows, err := s.DB.Query(ctx, `SELECT DISTINCT ac.position_id, d.id, COALESCE(d.name, '')
		  FROM anki_card ac JOIN anki_deck d ON d.id = ac.deck_id`+andSameTenant(s.DB, "d", "ac")+`
		 WHERE ac.position_id IN (`+Placeholders(len(batch))+`) AND `+tenant+`
		 ORDER BY ac.position_id, d.id`, args...)
	if err != nil {
		return errf(s.DB, "list fields decks", err)
	}
	defer rows.Close()
	for rows.Next() {
		var posID int64
		var d storage.DeckRef
		if err := rows.Scan(&posID, &d.ID, &d.Name); err != nil {
			return errf(s.DB, "list fields decks", err)
		}
		if f, ok := byID[posID]; ok {
			f.Decks = append(f.Decks, d)
		}
	}
	if err := rows.Err(); err != nil {
		return errf(s.DB, "list fields decks", err)
	}
	return nil
}

// positionMatchDay reads position.match_date as a UTC day: a TIMESTAMPTZ on
// PostgreSQL, Unix seconds on SQLite.
func positionMatchDay(v any) (string, bool) {
	switch t := v.(type) {
	case time.Time:
		return t.UTC().Format(time.DateOnly), true
	case int64:
		return time.Unix(t, 0).UTC().Format(time.DateOnly), true
	}
	return "", false
}

// uuidV7Day is the UTC day a row was written, read from the millisecond
// timestamp its version-7 uuid opens with. A uuid derived for a row older than
// the uuid column carries a zero timestamp: the day is unknown, "".
func uuidV7Day(u string) string {
	h := strings.ReplaceAll(u, "-", "")
	if len(h) != 32 || h[12] != '7' {
		return ""
	}
	b, err := hex.DecodeString(h[:12])
	if err != nil {
		return ""
	}
	var ms int64
	for _, x := range b {
		ms = ms<<8 | int64(x)
	}
	if ms == 0 {
		return ""
	}
	return time.UnixMilli(ms).UTC().Format(time.DateOnly)
}
