package sqlshared

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// DuelStore implements storage.DuelStore over the duel and match_origin
// tables, both confined to the scope's tenant. The document is one TEXT
// column the store never looks inside; the dice seed is another, written at
// insert only.
type DuelStore struct{ DB Execer }

var _ storage.DuelStore = (*DuelStore)(nil)

func (s *DuelStore) selectCols() string {
	return `id, ` + s.DB.TimestampText("created_at") + `, ` + s.DB.TimestampText("updated_at") +
		`, format_version, label, document, dice_seed, revision, is_open`
}

func scanDuel(sc interface{ Scan(...any) error }) (*storage.Duel, error) {
	var d storage.Duel
	var open int
	if err := sc.Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt, &d.FormatVersion, &d.Label, &d.Document, &d.DiceSeed, &d.Revision, &open); err != nil {
		return nil, err
	}
	d.Open = open != 0
	return &d, nil
}

// List streams the scope's Duels, most recently updated first; the id breaks
// ties so the order is total. The seed column is not selected: an entry has
// nowhere to hold it.
func (s *DuelStore) List(ctx context.Context, scope string) iter.Seq2[*storage.DuelEntry, error] {
	return func(yield func(*storage.DuelEntry, error) bool) {
		tenant, targs := s.DB.TenantFilter("", scope)
		rows, err := s.DB.Query(ctx,
			`SELECT id, `+s.DB.TimestampText("created_at")+`, `+s.DB.TimestampText("updated_at")+
				`, format_version, label, revision, is_open FROM duel WHERE `+tenant+` ORDER BY updated_at DESC, id DESC`, targs...)
		if err != nil {
			yield(nil, errf(s.DB, "list duels", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			var d storage.DuelEntry
			var open int
			if err := rows.Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt, &d.FormatVersion, &d.Label, &d.Revision, &open); err != nil {
				yield(nil, errf(s.DB, "list duels", err))
				return
			}
			d.Open = open != 0
			if !yield(&d, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, errf(s.DB, "list duels", err))
		}
	}
}

func (s *DuelStore) Get(ctx context.Context, scope string, id int64) (*storage.Duel, error) {
	what := fmt.Sprintf("get duel %d", id)
	tenant, targs := s.DB.TenantFilter("", scope)
	d, err := scanDuel(s.DB.QueryRow(ctx,
		`SELECT `+s.selectCols()+` FROM duel WHERE id = ? AND `+tenant, append([]any{id}, targs...)...))
	if errors.Is(err, ErrNoRows) {
		return nil, errf(s.DB, what, storage.ErrNotFound)
	}
	if err != nil {
		return nil, errf(s.DB, what, err)
	}
	return d, nil
}

func (s *DuelStore) Save(ctx context.Context, scope string, d *storage.Duel) (int64, error) {
	if d == nil {
		return 0, errf(s.DB, "save duel", storage.ErrInvalid)
	}
	if d.ID != 0 {
		return d.ID, s.update(ctx, scope, d)
	}
	if d.DiceSeed == "" {
		return 0, errf(s.DB, "save duel: no dice seed", storage.ErrInvalid)
	}
	cols, args := s.DB.TenantColumns(scope)
	cols = append(cols, "format_version", "label", "document", "dice_seed", "revision", "is_open")
	args = append(args, d.FormatVersion, d.Label, d.Document, d.DiceSeed, 1, boolInt(d.Open))
	id, err := s.DB.Insert(ctx,
		`INSERT INTO duel (`+strings.Join(cols, ", ")+`) VALUES (`+Placeholders(len(cols))+`)`, args...)
	if err != nil {
		return 0, errf(s.DB, "save duel", err)
	}
	d.Revision = 1
	return id, nil
}

// update rewrites the document, the label and whether the Duel is open, never
// the seed, and reads the revision this write produced from the statement
// itself.
func (s *DuelStore) update(ctx context.Context, scope string, d *storage.Duel) error {
	what := fmt.Sprintf("save duel %d", d.ID)
	tenant, targs := s.DB.TenantFilter("", scope)
	args := append([]any{d.FormatVersion, d.Label, d.Document, boolInt(d.Open), d.ID, d.Revision, d.Revision}, targs...)
	var rev int64
	err := s.DB.QueryRow(ctx,
		`UPDATE duel
		 SET format_version = ?, label = ?, document = ?, is_open = ?,
		     revision = revision + 1, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND (CAST(? AS BIGINT) = 0 OR revision = ?) AND `+tenant+`
		 RETURNING revision`, args...).Scan(&rev)
	if errors.Is(err, ErrNoRows) {
		if ok, perr := rowExists(ctx, s.DB, scope, "duel", d.ID); perr != nil {
			return perr
		} else if !ok {
			return errf(s.DB, what, storage.ErrNotFound)
		}
		return errf(s.DB, what, storage.ErrConflict)
	}
	if err != nil {
		return errf(s.DB, what, err)
	}
	d.Revision = rev
	return nil
}

func (s *DuelStore) Delete(ctx context.Context, scope string, id int64) error {
	what := fmt.Sprintf("delete duel %d", id)
	tenant, targs := s.DB.TenantFilter("", scope)
	n, err := s.DB.Exec(ctx, `DELETE FROM duel WHERE id = ? AND `+tenant, append([]any{id}, targs...)...)
	if err != nil {
		return errf(s.DB, what, err)
	}
	if n == 0 {
		return errf(s.DB, what, storage.ErrNotFound)
	}
	return nil
}

func (s *DuelStore) SetOrigin(ctx context.Context, scope string, o *storage.MatchOrigin) error {
	if o == nil || o.MatchID == 0 {
		return errf(s.DB, "set match origin", storage.ErrInvalid)
	}
	what := fmt.Sprintf("set origin of match %d", o.MatchID)
	return s.DB.Transact(ctx, func(tx Execer) error {
		ok, err := rowExists(ctx, tx, scope, "match", o.MatchID)
		if err != nil {
			return err
		}
		if !ok {
			return errf(tx, what, storage.ErrNotFound)
		}
		cols, args := tx.TenantColumns(scope)
		declared, err := encodeJSONColumn(o.DeclaredBots)
		if err != nil {
			return errf(tx, what, err)
		}
		contributions, err := encodeJSONColumn(o.Contributions)
		if err != nil {
			return errf(tx, what, err)
		}
		cols = append(cols, "match_id", "start", "dice_seed", "stopped_early", "over_time", "bot_level", "cadence", "bot_engine",
			"declared_bots", "contributions")
		args = append(args, o.MatchID, o.Start, o.DiceSeed, boolInt(o.StoppedEarly), o.OverTime, o.BotLevel, o.Cadence, o.BotEngine,
			declared, contributions)
		if _, err := tx.Exec(ctx, `INSERT INTO match_origin (`+strings.Join(cols, ", ")+`) VALUES (`+
			Placeholders(len(cols))+`) ON CONFLICT (match_id) DO UPDATE SET
			 start = excluded.start, dice_seed = excluded.dice_seed,
			 stopped_early = excluded.stopped_early, over_time = excluded.over_time,
			 bot_level = excluded.bot_level, cadence = excluded.cadence,
			 bot_engine = excluded.bot_engine, declared_bots = excluded.declared_bots,
			 contributions = excluded.contributions`, args...); err != nil {
			return errf(tx, what, err)
		}
		return nil
	})
}

func (s *DuelStore) Origin(ctx context.Context, scope string, matchID int64) (*storage.MatchOrigin, error) {
	what := fmt.Sprintf("origin of match %d", matchID)
	tenant, targs := s.DB.TenantFilter("", scope)
	o := storage.MatchOrigin{MatchID: matchID}
	var stopped int
	var declared, contributions string
	err := s.DB.QueryRow(ctx,
		`SELECT start, dice_seed, stopped_early, over_time, bot_level, cadence, bot_engine, declared_bots, contributions
		 FROM match_origin WHERE match_id = ? AND `+tenant, append([]any{matchID}, targs...)...).
		Scan(&o.Start, &o.DiceSeed, &stopped, &o.OverTime, &o.BotLevel, &o.Cadence, &o.BotEngine, &declared, &contributions)
	if errors.Is(err, ErrNoRows) {
		return nil, errf(s.DB, what, storage.ErrNotFound)
	}
	if err != nil {
		return nil, errf(s.DB, what, err)
	}
	o.StoppedEarly = stopped != 0
	if declared != "" {
		if err := json.Unmarshal([]byte(declared), &o.DeclaredBots); err != nil {
			return nil, errf(s.DB, what, err)
		}
	}
	if contributions != "" {
		if err := json.Unmarshal([]byte(contributions), &o.Contributions); err != nil {
			return nil, errf(s.DB, what, err)
		}
	}
	return &o, nil
}

// encodeJSONColumn is a list column of match_origin (declared_bots,
// contributions): JSON, the empty string when the list is, so that an origin
// without one is stored as before.
func encodeJSONColumn[T any](v []T) (string, error) {
	if len(v) == 0 {
		return "", nil
	}
	b, err := json.Marshal(v)
	return string(b), err
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
