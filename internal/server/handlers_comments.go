package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
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

// tagsResp carries the vocabulary a tenant has actually built AND the one
// blunderDB suggests. Both in one answer because a client showing the first
// wants the second beside it: a fresh library has no tags of its own, and a
// panel that stays empty until somebody guesses the convention teaches
// nothing.
type tagsResp struct {
	Tags        []domain.TagCount `json:"tags"`
	Recommended []string          `json:"recommended"`
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
		{http.MethodPost, "/v1/comments.listAll", rpcStream(func(ctx context.Context, scope string, req listReq) iterComments {
			return cs().ListAll(ctx, scope, storage.ListOpts{Limit: req.Limit, Offset: req.Offset})
		})},
		{http.MethodPost, "/v1/comments.search", rpcStream(func(ctx context.Context, scope string, req commentSearchReq) iterComments {
			return cs().Search(ctx, scope, req.Query)
		})},
		// The tag vocabulary (#265). It lives under comments because that is
		// where a tag lives: nothing declares one, no column holds one, and
		// the count is the number of POSITIONS a tag would yield.
		{http.MethodPost, "/v1/comments.tags", rpc(func(ctx context.Context, scope string, _ struct{}) (tagsResp, error) {
			tags, err := cs().Tags(ctx, scope)
			return tagsResp{Tags: tags, Recommended: domain.RecommendedTags}, err
		})},
	}
}
