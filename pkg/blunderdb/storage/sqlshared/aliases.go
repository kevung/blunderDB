package sqlshared

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// AliasStore implements storage.AliasStore over player_alias and event_alias.
type AliasStore struct{ DB Execer }

var _ storage.AliasStore = (*AliasStore)(nil)

func aliasTable(kind storage.AliasKind) (string, error) {
	switch kind {
	case storage.AliasPlayer:
		return "player_alias", nil
	case storage.AliasEvent:
		return "event_alias", nil
	}
	return "", fmt.Errorf("alias kind %q: %w", kind, storage.ErrInvalid)
}

// Set records alias → canonical, keeping the table flat (see the contract).
func (s *AliasStore) Set(ctx context.Context, scope string, kind storage.AliasKind, alias, canonical string) error {
	table, err := aliasTable(kind)
	if err != nil {
		return errf(s.DB, "set alias", err)
	}
	alias, canonical = strings.TrimSpace(alias), strings.TrimSpace(canonical)
	if alias == "" || canonical == "" || alias == canonical {
		return errf(s.DB, "set alias", fmt.Errorf("alias %q → %q: %w", alias, canonical, storage.ErrInvalid))
	}
	err = s.DB.Transact(ctx, func(tx Execer) error {
		tenant, targs := tx.TenantFilter("", scope)
		var target string
		err := tx.QueryRow(ctx, `SELECT canonical FROM `+table+` WHERE `+tenant+` AND alias = ?`,
			append(append([]any{}, targs...), canonical)...).Scan(&target)
		switch {
		case err == nil && target == alias:
			// canonical was an alias of alias: the pair turns round.
			if _, err := tx.Exec(ctx, `DELETE FROM `+table+` WHERE `+tenant+` AND alias = ?`,
				append(append([]any{}, targs...), canonical)...); err != nil {
				return err
			}
		case err == nil:
			canonical = target
		case !errors.Is(err, ErrNoRows):
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE `+table+` SET canonical = ? WHERE `+tenant+` AND canonical = ?`,
			append(append([]any{canonical}, targs...), alias)...); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM `+table+` WHERE `+tenant+` AND alias = ?`,
			append(append([]any{}, targs...), alias)...); err != nil {
			return err
		}
		cols, args := tx.TenantColumns(scope)
		cols = append(cols, "alias", "canonical")
		args = append(args, alias, canonical)
		_, err = tx.Exec(ctx, `INSERT INTO `+table+` (`+strings.Join(cols, ", ")+`) VALUES (`+Placeholders(len(cols))+`)`, args...)
		return err
	})
	if err != nil {
		return errf(s.DB, "set alias", err)
	}
	return nil
}

// Remove forgets alias.
func (s *AliasStore) Remove(ctx context.Context, scope string, kind storage.AliasKind, alias string) (bool, error) {
	table, err := aliasTable(kind)
	if err != nil {
		return false, errf(s.DB, "remove alias", err)
	}
	tenant, targs := s.DB.TenantFilter("", scope)
	n, err := s.DB.Exec(ctx, `DELETE FROM `+table+` WHERE `+tenant+` AND alias = ?`,
		append(targs, strings.TrimSpace(alias))...)
	if err != nil {
		return false, errf(s.DB, "remove alias", err)
	}
	return n > 0, nil
}

// List returns the aliases of kind, ordered by canonical then alias.
func (s *AliasStore) List(ctx context.Context, scope string, kind storage.AliasKind) ([]storage.Alias, error) {
	table, err := aliasTable(kind)
	if err != nil {
		return nil, errf(s.DB, "list aliases", err)
	}
	tenant, targs := s.DB.TenantFilter("", scope)
	rows, err := s.DB.Query(ctx, `SELECT alias, canonical FROM `+table+` WHERE `+tenant+` ORDER BY canonical, alias`, targs...)
	if err != nil {
		return nil, errf(s.DB, "list aliases", err)
	}
	defer rows.Close()
	var out []storage.Alias
	for rows.Next() {
		var a storage.Alias
		if err := rows.Scan(&a.Alias, &a.Canonical); err != nil {
			return nil, errf(s.DB, "list aliases", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "list aliases", err)
	}
	return out, nil
}

// aliasMap loads the aliases of kind as a resolver.
func aliasMap(ctx context.Context, db Execer, scope string, kind storage.AliasKind) (storage.AliasMap, error) {
	list, err := (&AliasStore{DB: db}).List(ctx, scope, kind)
	if err != nil {
		return nil, err
	}
	return storage.NewAliasMap(list), nil
}

// MergeAliases makes every name of names other than canonical one of its
// aliases, atomically. An empty canonical or an empty list is ErrInvalid.
func MergeAliases(ctx context.Context, db Execer, scope string, kind storage.AliasKind, names []string, canonical string) error {
	canonical = strings.TrimSpace(canonical)
	if canonical == "" {
		return errf(db, "merge names", fmt.Errorf("canonical name must not be empty: %w", storage.ErrInvalid))
	}
	if len(names) == 0 {
		return errf(db, "merge names", fmt.Errorf("no names to merge: %w", storage.ErrInvalid))
	}
	return db.Transact(ctx, func(tx Execer) error {
		as := &AliasStore{DB: tx}
		for _, n := range names {
			if n = strings.TrimSpace(n); n == "" || n == canonical {
				continue
			}
			if err := as.Set(ctx, scope, kind, n, canonical); err != nil {
				return err
			}
		}
		return nil
	})
}

// Suggest proposes the names of kind that fold to one (storage.SuggestAliases).
func (s *AliasStore) Suggest(ctx context.Context, scope string, kind storage.AliasKind) ([]storage.AliasSuggestion, error) {
	if _, err := aliasTable(kind); err != nil {
		return nil, errf(s.DB, "suggest aliases", err)
	}
	tenant, targs := s.DB.TenantFilter("", scope)
	var q string
	var args []any
	if kind == storage.AliasPlayer {
		q = `SELECT name, COUNT(*) FROM (
			SELECT player1_name AS name FROM match WHERE ` + tenant + ` AND player1_name != ''
			UNION ALL
			SELECT player2_name AS name FROM match WHERE ` + tenant + ` AND player2_name != ''
		) AS names GROUP BY name`
		args = append(append(args, targs...), targs...)
	} else {
		q = `SELECT event, COUNT(*) FROM match WHERE ` + tenant + ` AND event IS NOT NULL AND event != '' GROUP BY event`
		args = targs
	}
	rows, err := s.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, errf(s.DB, "suggest aliases", err)
	}
	counts := map[string]int{}
	for rows.Next() {
		var name string
		var n int
		if err := rows.Scan(&name, &n); err != nil {
			rows.Close()
			return nil, errf(s.DB, "suggest aliases", err)
		}
		counts[name] = n
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, errf(s.DB, "suggest aliases", err)
	}
	existing, err := aliasMap(ctx, s.DB, scope, kind)
	if err != nil {
		return nil, err
	}
	return storage.SuggestAliases(counts, existing), nil
}
