package sqlite

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type searchHistoryStore struct{ db execer }

var _ storage.SearchHistoryStore = (*searchHistoryStore)(nil)

func (*searchHistoryStore) Save(context.Context, string, string, string) error {
	return notImpl("SearchHistory", "Save")
}
func (*searchHistoryStore) List(context.Context, string) iter.Seq2[*storage.SearchHistory, error] {
	return errSeq2[storage.SearchHistory](notImpl("SearchHistory", "List"))
}
func (*searchHistoryStore) DeleteEntry(context.Context, string, int64) error {
	return notImpl("SearchHistory", "DeleteEntry")
}
