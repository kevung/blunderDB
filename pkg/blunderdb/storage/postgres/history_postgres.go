package postgres

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type commandHistoryStore struct{ db execer }

var _ storage.CommandHistoryStore = (*commandHistoryStore)(nil)

func (*commandHistoryStore) Save(context.Context, string, string) error {
	return notImpl("History", "Save")
}
func (*commandHistoryStore) Load(context.Context, string) ([]string, error) {
	return nil, notImpl("History", "Load")
}
func (*commandHistoryStore) Clear(context.Context, string) error {
	return notImpl("History", "Clear")
}
