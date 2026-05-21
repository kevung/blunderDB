package postgres

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type positionStore struct{ db execer }

var _ storage.PositionStore = (*positionStore)(nil)

func (*positionStore) Save(context.Context, string, *domain.Position) (int64, error) {
	return 0, notImpl("Position", "Save")
}
func (*positionStore) Update(context.Context, string, *domain.Position) error {
	return notImpl("Position", "Update")
}
func (*positionStore) Load(context.Context, string, int64) (*domain.Position, error) {
	return nil, notImpl("Position", "Load")
}
func (*positionStore) Exists(context.Context, string, uint64) (int64, bool, error) {
	return 0, false, notImpl("Position", "Exists")
}
func (*positionStore) Delete(context.Context, string, int64) error {
	return notImpl("Position", "Delete")
}
func (*positionStore) List(context.Context, string, storage.ListOpts) iter.Seq2[*domain.Position, error] {
	return errSeq2[domain.Position](notImpl("Position", "List"))
}
