package postgres

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type filterStore struct{ db execer }

var _ storage.FilterStore = (*filterStore)(nil)

func (*filterStore) Save(context.Context, string, string, string) (int64, error) {
	return 0, notImpl("Filter", "Save")
}
func (*filterStore) Update(context.Context, string, int64, string, string) error {
	return notImpl("Filter", "Update")
}
func (*filterStore) Delete(context.Context, string, int64) error {
	return notImpl("Filter", "Delete")
}
func (*filterStore) List(context.Context, string) iter.Seq2[*storage.Filter, error] {
	return errSeq2[storage.Filter](notImpl("Filter", "List"))
}
func (*filterStore) SaveEditPosition(context.Context, string, string, string) error {
	return notImpl("Filter", "SaveEditPosition")
}
func (*filterStore) LoadEditPosition(context.Context, string, string) (string, error) {
	return "", notImpl("Filter", "LoadEditPosition")
}
