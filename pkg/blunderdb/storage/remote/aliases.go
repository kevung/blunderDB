package remote

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// aliasFamily is the route prefix of an alias kind: players.alias.* or
// events.alias.*.
func aliasFamily(kind storage.AliasKind) (string, error) {
	switch kind {
	case storage.AliasPlayer:
		return "players.alias.", nil
	case storage.AliasEvent:
		return "events.alias.", nil
	default:
		return "", fmt.Errorf("%w: alias kind %q", storage.ErrInvalid, kind)
	}
}

func (s aliasStore) Set(ctx context.Context, scope string, kind storage.AliasKind, alias, canonical string) error {
	fam, err := aliasFamily(kind)
	if err != nil {
		return err
	}
	return s.c.call(ctx, scope, fam+"set", struct {
		Alias     string `json:"alias"`
		Canonical string `json:"canonical"`
	}{alias, canonical}, nil)
}

func (s aliasStore) Remove(ctx context.Context, scope string, kind storage.AliasKind, alias string) (bool, error) {
	fam, err := aliasFamily(kind)
	if err != nil {
		return false, err
	}
	var r struct {
		Removed bool `json:"removed"`
	}
	err = s.c.call(ctx, scope, fam+"remove", struct {
		Alias string `json:"alias"`
	}{alias}, &r)
	return r.Removed, err
}

func (s aliasStore) List(ctx context.Context, scope string, kind storage.AliasKind) ([]storage.Alias, error) {
	fam, err := aliasFamily(kind)
	if err != nil {
		return nil, err
	}
	var r struct {
		Aliases []storage.Alias `json:"aliases"`
	}
	err = s.c.call(ctx, scope, fam+"list", nil, &r)
	return r.Aliases, err
}

func (s aliasStore) Suggest(ctx context.Context, scope string, kind storage.AliasKind) ([]storage.AliasSuggestion, error) {
	fam, err := aliasFamily(kind)
	if err != nil {
		return nil, err
	}
	var r struct {
		Suggestions []storage.AliasSuggestion `json:"suggestions"`
	}
	err = s.c.call(ctx, scope, fam+"suggest", nil, &r)
	return r.Suggestions, err
}

func (s trashStore) List(ctx context.Context, scope string, kind domain.TrashKind, opts storage.ListOpts) ([]*domain.TrashEntry, error) {
	var out []*domain.TrashEntry
	err := s.c.call(ctx, scope, "trash.list", struct {
		Kind   string `json:"kind,omitempty"`
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
	}{string(kind), opts.Limit, opts.Offset}, &out)
	return out, err
}

func (s trashStore) Discard(ctx context.Context, scope string, id int64) error {
	return s.c.call(ctx, scope, "trash.discard", idReq{id}, nil)
}

func (s trashStore) Purge(ctx context.Context, scope string, olderThanDays int) (int, error) {
	var r struct {
		Purged int `json:"purged"`
	}
	err := s.c.call(ctx, scope, "trash.empty", struct {
		OlderThanDays int `json:"olderThanDays"`
	}{olderThanDays}, &r)
	return r.Purged, err
}

func (s trashStore) Count(ctx context.Context, scope string) (int, error) {
	var r struct {
		Count int `json:"count"`
	}
	err := s.c.call(ctx, scope, "trash.count", nil, &r)
	return r.Count, err
}

func (s matchEquityTableStore) SetCurrent(ctx context.Context, scope string, id int64) error {
	return s.c.call(ctx, scope, "met.setCurrent", idReq{id}, nil)
}
