package postgres

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type searchStore struct{ db execer }

var _ storage.SearchStore = (*searchStore)(nil)

func (*searchStore) Find(context.Context, string, domain.SearchFilters) iter.Seq2[*domain.Position, error] {
	return errSeq2[domain.Position](notImpl("Search", "Find"))
}

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
