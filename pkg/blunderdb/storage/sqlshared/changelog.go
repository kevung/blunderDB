package sqlshared

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ChangeJournalStore implements storage.ChangeJournalStore over change_log.
type ChangeJournalStore struct{ DB Execer }

var _ storage.ChangeJournalStore = (*ChangeJournalStore)(nil)

// LocalColumns are, per table, the columns that belong to the library, not to
// the object: the ones its changeset payload leaves out, or that a local
// gesture rewrites without the object having changed. Changing them alone
// dates nothing. 055_change_log.sql and 060_deck_tournament_uuid.sql spell the
// same lists.
//
// A match's tournament is not among them: it travels with the match. A
// synchronisation of a deck and the attaching of a match to a tournament touch
// updated_at and nothing the payload carries.
var LocalColumns = map[string][]string{
	"match":      {"import_date", "import_batch_id", "last_visited_position", "last_visited_at"},
	"anki_deck":  {"updated_at"},
	"tournament": {"updated_at", "sort_order", "rencontre_id", "rencontre_rooms"},
}

// changedAtText renders changed_at as UTC RFC 3339 with milliseconds on both
// backends (SQLite stores that text; PostgreSQL a timestamptz).
func (s *ChangeJournalStore) changedAtText() string {
	if s.DB.Name() == "sqlite" {
		return "changed_at"
	}
	return `to_char(changed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')`
}

func (s *ChangeJournalStore) Changes(ctx context.Context, scope string, after int64, limit int) ([]storage.Change, error) {
	return s.ChangesOf(ctx, scope, nil, storage.Cursor{Seq: after}, limit)
}

func (s *ChangeJournalStore) ChangesOf(ctx context.Context, scope string, families []string, after storage.Cursor, limit int) ([]storage.Change, error) {
	if limit <= 0 {
		limit = 1000
	}
	if _, err := s.checkCursor(ctx, scope, after); err != nil {
		return nil, err
	}
	tenant, targs := s.DB.TenantFilter("", scope)
	var args []any
	args = append(args, targs...)
	where, order, xid := tenant+` AND seq > ?`, "seq", "0"
	args = append(args, after.Seq)
	if s.DB.Name() == "postgres" {
		// Only entries whose writer, and every older transaction, has ended,
		// read in the order of those transactions: a change cannot still be
		// committed behind the cursor.
		// A cursor without Xid is a bare seq (Changes, an export from 0): the
		// row comparison would pass every row, since any xid is above 0.
		if after.Xid > 0 {
			where = tenant + ` AND (xid, seq) > (CAST(CAST(? AS TEXT) AS xid8), ?)`
			args = append(args[:len(targs)], strconv.FormatInt(after.Xid, 10), after.Seq)
		}
		where += ` AND xid < pg_snapshot_xmin(pg_current_snapshot())`
		order, xid = "xid, seq", "CAST(xid AS TEXT) AS xid_text"
	}
	if len(families) > 0 {
		where += ` AND family IN (` + Placeholders(len(families)) + `)`
		for _, f := range families {
			args = append(args, f)
		}
	}
	rows, err := s.DB.Query(ctx, `SELECT seq, CAST(uuid AS TEXT), family, op, `+s.changedAtText()+`, `+xid+` FROM change_log
		WHERE `+where+` ORDER BY `+order+` LIMIT ?`, append(args, limit)...)
	if err != nil {
		return nil, errf(s.DB, "changes", err)
	}
	defer rows.Close()
	var out []storage.Change
	for rows.Next() {
		var c storage.Change
		var x string
		if err := rows.Scan(&c.Seq, &c.UUID, &c.Family, &c.Op, &c.ChangedAt, &x); err != nil {
			return nil, errf(s.DB, "changes", err)
		}
		c.Xid, _ = strconv.ParseInt(x, 10, 64)
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "changes", err)
	}
	return out, nil
}

// checkCursor gives the journal's epoch and refuses a cursor from another one.
//
// A row whose xid this cluster's counter has not reached was restored from a
// dump of a cluster further along: no reader's snapshot would ever pass it. It
// is older than anything written here, so it goes to the front (xid 1 precedes
// every assigned xid, and seq keeps its order among its peers).
//
// The epoch is the cluster and its timeline: a point-in-time recovery keeps the
// cluster's identifier and rewinds the xid counter, and once the counter is back
// past a cursor nothing else tells the entries written since from the ones lost.
// The timeline is read from the WAL being written, which has it from the
// promotion on; pg_control_checkpoint has it only after the next checkpoint.
func (s *ChangeJournalStore) checkCursor(ctx context.Context, scope string, after storage.Cursor) (string, error) {
	if s.DB.Name() != "postgres" {
		return "", nil
	}
	tenant, targs := s.DB.TenantFilter("", scope)
	if _, err := s.DB.Exec(ctx, `UPDATE change_log SET xid = CAST('1' AS xid8) WHERE `+tenant+`
		AND xid >= pg_snapshot_xmax(pg_current_snapshot()) AND xid IS DISTINCT FROM pg_current_xact_id_if_assigned()`, targs...); err != nil {
		return "", errf(s.DB, "journal cursor", err)
	}
	var xmax, epoch string
	if err := s.DB.QueryRow(ctx, `SELECT CAST(pg_snapshot_xmax(pg_current_snapshot()) AS TEXT),
		CAST(system_identifier AS TEXT) || '/' || substr(pg_walfile_name(pg_current_wal_lsn()), 1, 8)
		FROM pg_control_system()`).Scan(&xmax, &epoch); err != nil {
		return "", errf(s.DB, "journal cursor", err)
	}
	horizon, err := strconv.ParseInt(xmax, 10, 64)
	if err != nil {
		return "", errf(s.DB, "journal cursor", err)
	}
	if after.Xid >= horizon || (after.Epoch != "" && after.Epoch != epoch) {
		return "", errf(s.DB, "journal cursor", storage.ErrCursorStale)
	}
	return epoch, nil
}

// Horizon — see storage.ChangeJournalStore.
func (s *ChangeJournalStore) Horizon(ctx context.Context, scope string, families []string, after storage.Cursor) (storage.Cursor, error) {
	epoch, err := s.checkCursor(ctx, scope, after)
	if err != nil {
		return storage.Cursor{}, err
	}
	tenant, args := s.DB.TenantFilter("", scope)
	where, order, xid := tenant, "seq DESC", "'0'"
	if s.DB.Name() == "postgres" {
		where += ` AND xid < pg_snapshot_xmin(pg_current_snapshot())`
		order, xid = "xid DESC, seq DESC", "CAST(xid AS TEXT) AS xid_text"
	}
	if len(families) > 0 {
		where += ` AND family IN (` + Placeholders(len(families)) + `)`
		for _, f := range families {
			args = append(args, f)
		}
	}
	out := storage.Cursor{Epoch: epoch}
	var x string
	err = s.DB.QueryRow(ctx, `SELECT `+xid+`, seq FROM change_log WHERE `+where+` ORDER BY `+order+` LIMIT 1`, args...).Scan(&x, &out.Seq)
	if errors.Is(err, ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return storage.Cursor{}, errf(s.DB, "journal horizon", err)
	}
	out.Xid, _ = strconv.ParseInt(x, 10, 64)
	return out, nil
}

func (s *ChangeJournalStore) Unjournaled(ctx context.Context, scope, family string, afterID int64, limit int) ([]string, int64, error) {
	table, err := objectUUIDTable(family)
	if err != nil {
		return nil, 0, errf(s.DB, "unjournaled", err)
	}
	tenant, targs := s.DB.TenantFilter("t", scope)
	same := ""
	if s.DB.Name() == "postgres" {
		same = ` AND c.tenant_id = t.tenant_id`
	}
	rows, err := s.DB.Query(ctx, `SELECT t.id, CAST(t.uuid AS TEXT) FROM `+table+` t WHERE `+tenant+` AND t.uuid IS NOT NULL AND t.id > ?
		AND NOT EXISTS (SELECT 1 FROM change_log c WHERE c.uuid = t.uuid`+same+`) ORDER BY t.id LIMIT ?`, append(append(targs, afterID), limit)...)
	if err != nil {
		return nil, 0, errf(s.DB, "unjournaled", err)
	}
	defer rows.Close()
	var out []string
	last := afterID
	for rows.Next() {
		var u string
		if err := rows.Scan(&last, &u); err != nil {
			return nil, 0, errf(s.DB, "unjournaled", err)
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, errf(s.DB, "unjournaled", err)
	}
	return out, last, nil
}

func (s *ChangeJournalStore) Entry(ctx context.Context, scope, uuid string) (storage.Change, error) {
	if !isUUID(uuid) {
		return storage.Change{}, errf(s.DB, "change entry", storage.ErrNotFound)
	}
	tenant, targs := s.DB.TenantFilter("", scope)
	var c storage.Change
	err := s.DB.QueryRow(ctx, `SELECT seq, CAST(uuid AS TEXT), family, op, `+s.changedAtText()+` FROM change_log
		WHERE `+tenant+` AND uuid = ?`, append(targs, uuid)...).Scan(&c.Seq, &c.UUID, &c.Family, &c.Op, &c.ChangedAt)
	if errors.Is(err, ErrNoRows) {
		return storage.Change{}, errf(s.DB, "change entry", storage.ErrNotFound)
	}
	if err != nil {
		return storage.Change{}, errf(s.DB, "change entry", err)
	}
	return c, nil
}

func (s *ChangeJournalStore) Mark(ctx context.Context, scope, family, uuid, op, at string) error {
	if uuid == "" {
		return nil
	}
	if _, err := objectUUIDTable(family); err != nil {
		return errf(s.DB, "mark change", err)
	}
	switch op {
	case storage.ChangeCreate, storage.ChangeUpdate, storage.ChangeDelete, storage.ChangeTrash:
	default:
		return errf(s.DB, "mark change", fmt.Errorf("operation %q: %w", op, storage.ErrInvalid))
	}
	var err error
	if s.DB.Name() == "sqlite" {
		if at == "" {
			// Selected before the replace: the date this write has to pass.
			_, err = s.DB.Exec(ctx, `INSERT OR REPLACE INTO change_log (uuid, family, op, changed_at) VALUES (?, ?, ?,
				MAX(strftime('%Y-%m-%dT%H:%M:%fZ', 'now'),
					COALESCE((SELECT strftime('%Y-%m-%dT%H:%M:%fZ', changed_at, '+0.001 seconds') FROM change_log WHERE uuid = ?), '')))`,
				uuid, family, op, uuid)
		} else {
			_, err = s.DB.Exec(ctx, `INSERT OR REPLACE INTO change_log (uuid, family, op, changed_at) VALUES (?, ?, ?, ?)`, uuid, family, op, at)
		}
	} else {
		// at is bound as text and cast; "" falls to now(), past what the entry holds.
		_, err = s.DB.Exec(ctx, `INSERT INTO change_log (tenant_id, uuid, family, op, changed_at)
			VALUES (?, CAST(? AS uuid), ?, ?, COALESCE(CAST(NULLIF(?, '') AS timestamptz), now()))
			ON CONFLICT (tenant_id, uuid) DO UPDATE SET seq = nextval('change_log_seq'), family = EXCLUDED.family,
			op = EXCLUDED.op, changed_at = CASE WHEN ? = '' THEN blunderdb_stamp_after(change_log.changed_at) ELSE EXCLUDED.changed_at END,
			xid = pg_current_xact_id()`, s.DB.ScopeArg(scope), uuid, family, op, at, at)
	}
	if err != nil {
		return errf(s.DB, "mark change", err)
	}
	return nil
}

// Applying — see storage.ChangeJournalStore. The triggers read it: a setting
// local to the transaction on PostgreSQL, a row removed before the commit on
// SQLite, whose triggers can read neither a setting nor a temporary table.
func (s *ChangeJournalStore) Applying(ctx context.Context, _ string, at string) error {
	if s.DB.Name() != "sqlite" {
		if _, err := s.DB.Exec(ctx, `SELECT set_config('blunderdb.applying_at', ?, true)`, at); err != nil {
			return errf(s.DB, "applying", err)
		}
		return nil
	}
	if _, err := s.DB.Exec(ctx, `DELETE FROM change_log_applying`); err != nil {
		return errf(s.DB, "applying", err)
	}
	if at == "" {
		return nil
	}
	if _, err := s.DB.Exec(ctx, `INSERT INTO change_log_applying (at) VALUES (?)`, at); err != nil {
		return errf(s.DB, "applying", err)
	}
	return nil
}

// PartFact — see storage.ChangeJournalStore.
func (s *ChangeJournalStore) PartFact(ctx context.Context, scope string, positionID int64, part string) (storage.PartFact, bool, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	f := storage.PartFact{PositionID: positionID, Part: part}
	err := s.DB.QueryRow(ctx, `SELECT changed_at, COALESCE(pending_origin, ''), COALESCE(pending_move, '') FROM position_part_fact
		WHERE position_id = ? AND part = ? AND `+tenant, append([]any{positionID, part}, targs...)...).Scan(&f.At, &f.PendingOrigin, &f.PendingMove)
	if errors.Is(err, ErrNoRows) {
		return f, false, nil
	}
	if err != nil {
		return f, false, errf(s.DB, "part fact", err)
	}
	return f, true, nil
}

// SetPartFact — see storage.ChangeJournalStore.
func (s *ChangeJournalStore) SetPartFact(ctx context.Context, scope string, f storage.PartFact) error {
	tenant, targs := s.DB.TenantFilter("", scope)
	if _, err := s.DB.Exec(ctx, `DELETE FROM position_part_fact WHERE position_id = ? AND part = ? AND `+tenant,
		append([]any{f.PositionID, f.Part}, targs...)...); err != nil {
		return errf(s.DB, "part fact", err)
	}
	cols, args := s.DB.TenantColumns(scope)
	cols = append(cols, "position_id", "part", "changed_at", "pending_origin", "pending_move")
	args = append(args, f.PositionID, f.Part, f.At, nullIfEmpty(f.PendingOrigin), nullIfEmpty(f.PendingMove))
	if _, err := s.DB.Exec(ctx, `INSERT INTO position_part_fact (`+strings.Join(cols, ", ")+`) VALUES (`+Placeholders(len(cols))+`)`, args...); err != nil {
		return errf(s.DB, "part fact", err)
	}
	return nil
}

func nullIfEmpty(v string) any {
	if v == "" {
		return nil
	}
	return v
}

// PendingLinks — see storage.ChangeJournalStore.
func (s *ChangeJournalStore) PendingLinks(ctx context.Context, scope string) ([]storage.PartFact, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	rows, err := s.DB.Query(ctx, `SELECT position_id, changed_at, pending_origin, COALESCE(pending_move, '') FROM position_part_fact
		WHERE pending_origin IS NOT NULL AND `+tenant+` ORDER BY position_id`, targs...)
	if err != nil {
		return nil, errf(s.DB, "pending links", err)
	}
	defer rows.Close()
	var out []storage.PartFact
	for rows.Next() {
		f := storage.PartFact{Part: storage.PartDerivation}
		if err := rows.Scan(&f.PositionID, &f.At, &f.PendingOrigin, &f.PendingMove); err != nil {
			return nil, errf(s.DB, "pending links", err)
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "pending links", err)
	}
	return out, nil
}

// Adopt — see storage.ChangeJournalStore.
func (s *ChangeJournalStore) Adopt(ctx context.Context, scope, family string, uuids []string, at string) error {
	if _, err := objectUUIDTable(family); err != nil {
		return errf(s.DB, "adopt", err)
	}
	for _, u := range uuids {
		var err error
		if s.DB.Name() == "sqlite" {
			_, err = s.DB.Exec(ctx, `INSERT OR IGNORE INTO change_log (uuid, family, op, changed_at) VALUES (?, ?, 'create', ?)`, u, family, at)
		} else {
			_, err = s.DB.Exec(ctx, `INSERT INTO change_log (tenant_id, uuid, family, op, changed_at)
				VALUES (?, CAST(? AS uuid), ?, 'create', CAST(? AS timestamptz)) ON CONFLICT (tenant_id, uuid) DO NOTHING`,
				s.DB.ScopeArg(scope), u, family, at)
		}
		if err != nil {
			return errf(s.DB, "adopt", err)
		}
	}
	return nil
}
