package remote

import (
	"context"
	"iter"
	"slices"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type idResp struct {
	ID int64 `json:"id"`
}

type textResp struct {
	Text string `json:"text"`
}

// Add names the author the context carries; the daemon also reads it from
// the X-User-Name header, which post sets from the same context.
func (s commentStore) Add(ctx context.Context, scope string, positionID int64, text string) (int64, error) {
	var r idResp
	err := s.c.call(ctx, scope, "comments.add", struct {
		PositionID int64  `json:"positionId"`
		Text       string `json:"text"`
		Author     string `json:"author,omitempty"`
	}{positionID, text, storage.CommentAuthorFromContext(ctx)}, &r)
	return r.ID, err
}

func (s commentStore) Update(ctx context.Context, scope string, commentID int64, text string) error {
	return s.c.call(ctx, scope, "comments.update", struct {
		CommentID int64  `json:"commentId"`
		Text      string `json:"text"`
	}{commentID, text}, nil)
}

type commentIDReq struct {
	CommentID int64 `json:"commentId"`
}

func (s commentStore) Delete(ctx context.Context, scope string, commentID int64) error {
	return s.c.call(ctx, scope, "comments.delete", commentIDReq{commentID}, nil)
}

func (s commentStore) DeleteForPosition(ctx context.Context, scope string, positionID int64) error {
	return s.c.call(ctx, scope, "comments.deleteForPosition", positionIDReq{positionID}, nil)
}

func (s commentStore) Text(ctx context.Context, scope string, positionID int64) (string, error) {
	var r textResp
	err := s.c.call(ctx, scope, "comments.text", positionIDReq{positionID}, &r)
	return r.Text, err
}

// ByPositions has no batch route: it reads comments.byPosition once per id
// and turns its newest-first answer into the oldest-first order of the
// contract.
func (s commentStore) ByPositions(ctx context.Context, scope string, positionIDs []int64) (map[int64][]*domain.CommentEntry, error) {
	out := make(map[int64][]*domain.CommentEntry)
	for _, id := range positionIDs {
		cs, err := collect(s.ByPosition(ctx, scope, id))
		if err != nil {
			return nil, err
		}
		if len(cs) > 0 {
			slices.Reverse(cs)
			out[id] = cs
		}
	}
	return out, nil
}

func (s commentStore) ByPosition(ctx context.Context, scope string, positionID int64) iter.Seq2[*domain.CommentEntry, error] {
	return stream[*domain.CommentEntry](ctx, s.c, scope, "comments.byPosition", positionIDReq{positionID})
}

func (s commentStore) ListAll(ctx context.Context, scope string, opts storage.ListOpts) iter.Seq2[*domain.CommentEntry, error] {
	return stream[*domain.CommentEntry](ctx, s.c, scope, "comments.listAll", listReq{opts.Limit, opts.Offset})
}

func (s commentStore) Search(ctx context.Context, scope string, query string) iter.Seq2[*domain.CommentEntry, error] {
	return stream[*domain.CommentEntry](ctx, s.c, scope, "comments.search", struct {
		Query string `json:"query"`
	}{query})
}

func (s commentStore) Tags(ctx context.Context, scope string) ([]domain.TagCount, error) {
	var r struct {
		Tags []domain.TagCount `json:"tags"`
	}
	err := s.c.call(ctx, scope, "comments.tags", nil, &r)
	return r.Tags, err
}
