package sqlite

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type commentStore struct{ db execer }

var _ storage.CommentStore = (*commentStore)(nil)

func (*commentStore) Add(context.Context, string, int64, string) (int64, error) {
	return 0, notImpl("Comment", "Add")
}
func (*commentStore) Update(context.Context, string, int64, string) error {
	return notImpl("Comment", "Update")
}
func (*commentStore) Delete(context.Context, string, int64) error {
	return notImpl("Comment", "Delete")
}
func (*commentStore) DeleteForPosition(context.Context, string, int64) error {
	return notImpl("Comment", "DeleteForPosition")
}
func (*commentStore) Text(context.Context, string, int64) (string, error) {
	return "", notImpl("Comment", "Text")
}
func (*commentStore) ByPosition(context.Context, string, int64) iter.Seq2[*domain.CommentEntry, error] {
	return errSeq2[domain.CommentEntry](notImpl("Comment", "ByPosition"))
}
func (*commentStore) ListAll(context.Context, string) iter.Seq2[*domain.CommentEntry, error] {
	return errSeq2[domain.CommentEntry](notImpl("Comment", "ListAll"))
}
func (*commentStore) Search(context.Context, string, string) iter.Seq2[*domain.CommentEntry, error] {
	return errSeq2[domain.CommentEntry](notImpl("Comment", "Search"))
}
