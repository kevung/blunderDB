package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type aliasSetReq struct {
	Alias     string `json:"alias"`
	Canonical string `json:"canonical"`
}

type aliasRemoveReq struct {
	Alias string `json:"alias"`
}

type aliasListResp struct {
	Aliases []storage.Alias `json:"aliases"`
}

type aliasSuggestResp struct {
	Suggestions []storage.AliasSuggestion `json:"suggestions"`
}

type aliasRemoveResp struct {
	Removed bool `json:"removed"`
}

// aliasRoutes serves the other spellings of players (/v1/players.alias.*)
// and events (/v1/events.alias.*), the same four gestures for each.
func (s *Server) aliasRoutes() []route {
	as := func() storage.AliasStore { return s.opts.Storage.Aliases() }
	return []route{
		{http.MethodPost, "/v1/players.alias.list", rpc(func(ctx context.Context, scope string, _ struct{}) (aliasListResp, error) {
			return listAliases(ctx, as(), scope, storage.AliasPlayer)
		})},
		{http.MethodPost, "/v1/players.alias.set", rpcVoid(func(ctx context.Context, scope string, req aliasSetReq) error {
			return as().Set(ctx, scope, storage.AliasPlayer, req.Alias, req.Canonical)
		})},
		{http.MethodPost, "/v1/players.alias.remove", rpc(func(ctx context.Context, scope string, req aliasRemoveReq) (aliasRemoveResp, error) {
			removed, err := as().Remove(ctx, scope, storage.AliasPlayer, req.Alias)
			return aliasRemoveResp{Removed: removed}, err
		})},
		{http.MethodPost, "/v1/players.alias.suggest", rpc(func(ctx context.Context, scope string, _ struct{}) (aliasSuggestResp, error) {
			return suggestAliases(ctx, as(), scope, storage.AliasPlayer)
		})},
		{http.MethodPost, "/v1/events.alias.list", rpc(func(ctx context.Context, scope string, _ struct{}) (aliasListResp, error) {
			return listAliases(ctx, as(), scope, storage.AliasEvent)
		})},
		{http.MethodPost, "/v1/events.alias.set", rpcVoid(func(ctx context.Context, scope string, req aliasSetReq) error {
			return as().Set(ctx, scope, storage.AliasEvent, req.Alias, req.Canonical)
		})},
		{http.MethodPost, "/v1/events.alias.remove", rpc(func(ctx context.Context, scope string, req aliasRemoveReq) (aliasRemoveResp, error) {
			removed, err := as().Remove(ctx, scope, storage.AliasEvent, req.Alias)
			return aliasRemoveResp{Removed: removed}, err
		})},
		{http.MethodPost, "/v1/events.alias.suggest", rpc(func(ctx context.Context, scope string, _ struct{}) (aliasSuggestResp, error) {
			return suggestAliases(ctx, as(), scope, storage.AliasEvent)
		})},
	}
}

func listAliases(ctx context.Context, as storage.AliasStore, scope string, kind storage.AliasKind) (aliasListResp, error) {
	out, err := as.List(ctx, scope, kind)
	if out == nil {
		out = []storage.Alias{}
	}
	return aliasListResp{Aliases: out}, err
}

func suggestAliases(ctx context.Context, as storage.AliasStore, scope string, kind storage.AliasKind) (aliasSuggestResp, error) {
	out, err := as.Suggest(ctx, scope, kind)
	if out == nil {
		out = []storage.AliasSuggestion{}
	}
	return aliasSuggestResp{Suggestions: out}, err
}
