package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type sessionSaveReq struct {
	State storage.SessionState `json:"state"`
}

type searchHistorySaveReq struct {
	Command  string `json:"command"`
	Position string `json:"position"`
}

type searchHistoryDeleteReq struct {
	Timestamp int64 `json:"timestamp"`
}

type commandSaveReq struct {
	Command string `json:"command"`
}

type stringsResp struct {
	Items []string `json:"items"`
}

func (s *Server) sessionRoutes() []route {
	ss := func() storage.SessionStore { return s.opts.Storage.Session() }
	sh := func() storage.SearchHistoryStore { return s.opts.Storage.SearchHistory() }
	ch := func() storage.CommandHistoryStore { return s.opts.Storage.History() }
	return []route{
		{http.MethodPost, "/v1/session.save", rpcVoid(func(ctx context.Context, scope string, req sessionSaveReq) error {
			return ss().Save(ctx, scope, req.State)
		})},
		{http.MethodPost, "/v1/session.load", rpc(func(ctx context.Context, scope string, _ struct{}) (*storage.SessionState, error) {
			return ss().Load(ctx, scope)
		})},
		{http.MethodPost, "/v1/session.clear", rpcVoid(func(ctx context.Context, scope string, _ struct{}) error {
			return ss().Clear(ctx, scope)
		})},

		{http.MethodPost, "/v1/searchHistory.save", rpcVoid(func(ctx context.Context, scope string, req searchHistorySaveReq) error {
			return sh().Save(ctx, scope, req.Command, req.Position)
		})},
		{http.MethodPost, "/v1/searchHistory.list", rpcStream(func(ctx context.Context, scope string, _ struct{}) iterSearchHis {
			return sh().List(ctx, scope)
		})},
		{http.MethodPost, "/v1/searchHistory.deleteEntry", rpcVoid(func(ctx context.Context, scope string, req searchHistoryDeleteReq) error {
			return sh().DeleteEntry(ctx, scope, req.Timestamp)
		})},

		{http.MethodPost, "/v1/history.save", rpcVoid(func(ctx context.Context, scope string, req commandSaveReq) error {
			return ch().Save(ctx, scope, req.Command)
		})},
		{http.MethodPost, "/v1/history.load", rpc(func(ctx context.Context, scope string, _ struct{}) (stringsResp, error) {
			items, err := ch().Load(ctx, scope)
			return stringsResp{Items: items}, err
		})},
		{http.MethodPost, "/v1/history.clear", rpcVoid(func(ctx context.Context, scope string, _ struct{}) error {
			return ch().Clear(ctx, scope)
		})},
	}
}
