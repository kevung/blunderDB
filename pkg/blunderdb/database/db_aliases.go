package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Alias is one other spelling of a player's or an event's name.
type Alias = storage.Alias

// AliasSuggestion is a set of player names that are probably one person.
type AliasSuggestion = storage.AliasSuggestion

func aliasKind(kind string) (storage.AliasKind, error) {
	k := storage.AliasKind(kind)
	if !k.Valid() {
		return "", fmt.Errorf("alias kind %q: want %q or %q", kind, storage.AliasPlayer, storage.AliasEvent)
	}
	return k, nil
}

// ListAliases returns the aliases of kind ("player" or "event").
func (d *Database) ListAliases(kind string) ([]Alias, error) {
	k, err := aliasKind(kind)
	if err != nil {
		return nil, err
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}
	out, err := d.store.Aliases().List(context.Background(), "", k)
	if out == nil {
		out = []Alias{}
	}
	return out, err
}

// SetAlias makes alias a spelling of canonical for kind ("player" or
// "event"). The matches keep their names; the import, the stats and the
// search read canonical.
func (d *Database) SetAlias(kind, alias, canonical string) error {
	k, err := aliasKind(kind)
	if err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Aliases().Set(context.Background(), "", k, alias, canonical)
}

// RemoveAlias forgets alias for kind; false when it was not one.
func (d *Database) RemoveAlias(kind, alias string) (bool, error) {
	k, err := aliasKind(kind)
	if err != nil {
		return false, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return false, fmt.Errorf("no database is currently open")
	}
	return d.store.Aliases().Remove(context.Background(), "", k, alias)
}

// SuggestAliases proposes, for kind, the names that differ only by case,
// accents, punctuation or word order. Nothing is applied.
func (d *Database) SuggestAliases(kind string) ([]AliasSuggestion, error) {
	k, err := aliasKind(kind)
	if err != nil {
		return nil, err
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}
	out, err := d.store.Aliases().Suggest(context.Background(), "", k)
	if out == nil {
		out = []AliasSuggestion{}
	}
	return out, err
}
