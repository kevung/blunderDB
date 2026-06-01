package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type commentAddReq struct {
	PositionID int64  `json:"positionId"`
	Text       string `json:"text"`
}

type commentUpdateReq struct {
	CommentID int64  `json:"commentId"`
	Text      string `json:"text"`
}

type commentIDReq struct {
	CommentID int64 `json:"commentId"`
}

type commentSearchReq struct {
	Query string `json:"query"`
}

type textResp struct {
	Text string `json:"text"`
}

func (s *Server) commentRoutes() []route {
	cs := func() storage.CommentStore { return s.opts.Storage.Comments() }
	return []route{
		{http.MethodPost, "/v1/comments.add", rpc(func(ctx context.Context, scope string, req commentAddReq) (idResp, error) {
			id, err := cs().Add(ctx, scope, req.PositionID, req.Text)
			return idResp{ID: id}, err
		})},
		{http.MethodPost, "/v1/comments.update", rpcVoid(func(ctx context.Context, scope string, req commentUpdateReq) error {
			return cs().Update(ctx, scope, req.CommentID, req.Text)
		})},
		{http.MethodPost, "/v1/comments.delete", rpcVoid(func(ctx context.Context, scope string, req commentIDReq) error {
			return cs().Delete(ctx, scope, req.CommentID)
		})},
		{http.MethodPost, "/v1/comments.deleteForPosition", rpcVoid(func(ctx context.Context, scope string, req positionIDReq) error {
			return cs().DeleteForPosition(ctx, scope, req.PositionID)
		})},
		{http.MethodPost, "/v1/comments.text", rpc(func(ctx context.Context, scope string, req positionIDReq) (textResp, error) {
			t, err := cs().Text(ctx, scope, req.PositionID)
			return textResp{Text: t}, err
		})},
		{http.MethodPost, "/v1/comments.byPosition", rpcStream(func(ctx context.Context, scope string, req positionIDReq) iterComments {
			return cs().ByPosition(ctx, scope, req.PositionID)
		})},
		{http.MethodPost, "/v1/comments.listAll", rpcStream(func(ctx context.Context, scope string, _ struct{}) iterComments {
			return cs().ListAll(ctx, scope)
		})},
		{http.MethodPost, "/v1/comments.search", rpcStream(func(ctx context.Context, scope string, req commentSearchReq) iterComments {
			return cs().Search(ctx, scope, req.Query)
		})},
	}
}
