package sqlshared

import (
	"context"
	"errors"
	"math"
	"slices"
	"sort"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// MetadataStore implements storage.MetadataStore. The metadata table is
// database-global (not tenant-scoped) in both schemas; only Counts, which reads
// the domain tables, is confined to the scope's tenant.
type MetadataStore struct{ DB Execer }

var _ storage.MetadataStore = (*MetadataStore)(nil)

// metadataVersionKey is the metadata row that records the schema version.
const metadataVersionKey = "database_version"

// upsertMetadataSQL writes one key/value pair, replacing an existing key. The
// ON CONFLICT form is common to PostgreSQL and SQLite (≥ 3.24).
const upsertMetadataSQL = `INSERT INTO metadata (key, value) VALUES (?,?)
	ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`

// Version returns the recorded schema version, or ErrNotFound when the row is
// absent.
func (s *MetadataStore) Version(ctx context.Context, scope string) (string, error) {
	var v string
	err := s.DB.QueryRow(ctx, `SELECT value FROM metadata WHERE key = ?`, metadataVersionKey).Scan(&v)
	if errors.Is(err, ErrNoRows) {
		return "", errf(s.DB, "database version", storage.ErrNotFound)
	}
	if err != nil {
		return "", errf(s.DB, "database version", err)
	}
	return v, nil
}

// SetVersion records the schema version.
func (s *MetadataStore) SetVersion(ctx context.Context, scope string, version string) error {
	if _, err := s.DB.Exec(ctx, upsertMetadataSQL, metadataVersionKey, version); err != nil {
		return errf(s.DB, "set database version", err)
	}
	return nil
}

// Load returns every metadata key/value pair.
func (s *MetadataStore) Load(ctx context.Context, scope string) (map[string]string, error) {
	rows, err := s.DB.Query(ctx, `SELECT key, COALESCE(value,'') FROM metadata`)
	if err != nil {
		return nil, errf(s.DB, "load metadata", err)
	}
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, errf(s.DB, "load metadata", err)
		}
		out[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "load metadata", err)
	}
	return out, nil
}

// Save writes the given metadata key/value pairs, replacing existing keys.
func (s *MetadataStore) Save(ctx context.Context, scope string, metadata map[string]string) error {
	err := s.DB.Transact(ctx, func(tx Execer) error {
		for key, value := range metadata {
			if _, err := tx.Exec(ctx, upsertMetadataSQL, key, value); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return errf(s.DB, "save metadata", err)
	}
	return nil
}

// Counts returns the headline row counts of the scope's tenant.
func (s *MetadataStore) Counts(ctx context.Context, scope string) (storage.Counts, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	tables := []string{
		"position", "analysis", "match", "game", "move",
		"position WHERE " + s.DB.Bool("individually_imported", true) + " AND",
		"anki_card",
	}
	selects := make([]string, len(tables))
	var args []any
	for i, t := range tables {
		if strings.Contains(t, " WHERE ") {
			selects[i] = "(SELECT COUNT(*) FROM " + t + " " + tenant + ")"
		} else {
			selects[i] = "(SELECT COUNT(*) FROM " + t + " WHERE " + tenant + ")"
		}
		args = append(args, targs...)
	}
	var c storage.Counts
	err := s.DB.QueryRow(ctx, "SELECT "+strings.Join(selects, ", "), args...).
		Scan(&c.Positions, &c.Analyses, &c.Matches, &c.Games, &c.Moves,
			&c.IndividualPositions, &c.AnkiCards)
	if err != nil {
		return storage.Counts{}, errf(s.DB, "database counts", err)
	}
	if c.Blunders, err = s.blunderCount(ctx, scope); err != nil {
		return storage.Counts{}, err
	}
	return c, nil
}

// EstimatedCounts counts a table exactly when its highest id is under
// exactBelow and otherwise reports that highest id, flagged approximate. Where
// the database holds one library (no tenant column) MAX(id) is one probe of the
// primary-key index, so its cost does not grow with the table. A multi-tenant
// database has no such shortcut: ids are drawn from one sequence shared by every
// tenant, so a tenant's highest id says nothing about how many rows it owns, and
// no (tenant_id, id) index makes the probe cheap. There every table is counted.
func (s *MetadataStore) EstimatedCounts(ctx context.Context, scope string, exactBelow int) (storage.CountsEstimate, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	tenantCols, _ := s.DB.TenantColumns(scope)
	shared := len(tenantCols) > 0
	est := storage.CountsEstimate{Approximate: []string{}}
	fields := []struct {
		table, json string
		dst         *int
	}{
		{"position", "positions", &est.Positions},
		{"analysis", "analyses", &est.Analyses},
		{"match", "matches", &est.Matches},
		{"game", "games", &est.Games},
		{"move", "moves", &est.Moves},
	}
	for _, f := range fields {
		var top int64
		if !shared {
			if err := s.DB.QueryRow(ctx, "SELECT COALESCE(MAX(id), 0) FROM "+f.table+" WHERE "+tenant, targs...).Scan(&top); err != nil {
				return storage.CountsEstimate{}, errf(s.DB, "database counts", err)
			}
		}
		if top > int64(exactBelow) {
			*f.dst = int(top)
			est.Approximate = append(est.Approximate, f.json)
			continue
		}
		var n int64
		if err := s.DB.QueryRow(ctx, "SELECT COUNT(*) FROM "+f.table+" WHERE "+tenant, targs...).Scan(&n); err != nil {
			return storage.CountsEstimate{}, errf(s.DB, "database counts", err)
		}
		*f.dst = int(n)
	}
	// The blunders scan every analysis, so they are counted only while the
	// positions are: a library whose positions are exact is small enough that
	// the scan is acceptable, one whose positions are estimated is not.
	if !slices.Contains(est.Approximate, "positions") {
		b, err := s.blunderCount(ctx, scope)
		if err != nil {
			return storage.CountsEstimate{}, err
		}
		est.Blunders, est.BlundersKnown = b, true
	}
	return est, nil
}

// blunderCount counts the Positions whose largest recorded cost reaches the
// library's blunder threshold — the set `E>x` returns at that threshold, and
// therefore the set the status bar's counter may promise.
//
// The denormalised error column scores ONE play, so it does the counting and
// the few multi-played Positions (multiPlayedPlayer1Positions, as the search)
// are re-scored one by one the way the search scores them.
func (s *MetadataStore) blunderCount(ctx context.Context, scope string) (int, error) {
	settings, err := librarySettings(ctx, s.DB, scope)
	if err != nil {
		return 0, err
	}
	threshold := settings.BlunderThresholdMP

	tenant, targs := s.DB.TenantFilter("p", scope)
	const join = ` FROM position p INNER JOIN analysis a ON a.position_id = p.id WHERE `
	byColumn := ` AND COALESCE(` + statsErrExpr + `, 0) >= ?`

	var total int
	if err := s.DB.QueryRow(ctx, `SELECT COUNT(*)`+join+tenant+byColumn,
		append(append([]any{}, targs...), threshold)...).Scan(&total); err != nil {
		return 0, errf(s.DB, "blunder count", err)
	}

	multi, err := multiPlayedPlayer1Positions(ctx, s.DB, scope)
	if err != nil {
		return 0, err
	}
	if len(multi) == 0 {
		return total, nil
	}
	ids := make([]int64, 0, len(multi))
	for id := range multi {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	// What the column already counted among them, so it can be taken back
	// out. Chunked like every other id list here, though this one is small:
	// the bound-variable limit is a property of the query, not of the data.
	const chunk = 900
	var counted int
	for start := 0; start < len(ids); start += chunk {
		end := min(start+chunk, len(ids))
		batch := ids[start:end]
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(batch)), ",")
		args := append(append([]any{}, targs...), threshold)
		for _, id := range batch {
			args = append(args, id)
		}
		var n int
		if err := s.DB.QueryRow(ctx,
			`SELECT COUNT(*)`+join+tenant+byColumn+` AND p.id IN (`+placeholders+`)`,
			args...).Scan(&n); err != nil {
			return 0, errf(s.DB, "blunder count", err)
		}
		counted += n
	}

	moves, err := loadPlayer1Moves(ctx, s.DB, ids)
	if err != nil {
		return 0, err
	}
	var byLargest int
	for _, id := range ids {
		analysis := loadAnalysis(ctx, s.DB, id)
		if analysis == nil {
			continue
		}
		e, found := player1MaxMoveError(analysis, moves[id].checkerMoves, moves[id].cubeActions)
		if found && math.Round(e*1000) >= float64(threshold) {
			byLargest++
		}
	}
	return total - counted + byLargest, nil
}
