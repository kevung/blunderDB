package sqlite

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type collectionStore struct{ db execer }

var _ storage.CollectionStore = (*collectionStore)(nil)

func (*collectionStore) Create(context.Context, string, string, string) (int64, error) {
	return 0, notImpl("Collection", "Create")
}
func (*collectionStore) Get(context.Context, string, int64) (*storage.Collection, error) {
	return nil, notImpl("Collection", "Get")
}
func (*collectionStore) List(context.Context, string) iter.Seq2[*storage.Collection, error] {
	return errSeq2[storage.Collection](notImpl("Collection", "List"))
}
func (*collectionStore) Update(context.Context, string, int64, string, string) error {
	return notImpl("Collection", "Update")
}
func (*collectionStore) Delete(context.Context, string, int64) error {
	return notImpl("Collection", "Delete")
}
func (*collectionStore) Reorder(context.Context, string, []int64) error {
	return notImpl("Collection", "Reorder")
}
func (*collectionStore) AddPosition(context.Context, string, int64, int64) error {
	return notImpl("Collection", "AddPosition")
}
func (*collectionStore) AddPositions(context.Context, string, int64, []int64) error {
	return notImpl("Collection", "AddPositions")
}
func (*collectionStore) RemovePosition(context.Context, string, int64, int64) error {
	return notImpl("Collection", "RemovePosition")
}
func (*collectionStore) RemovePositions(context.Context, string, int64, []int64) error {
	return notImpl("Collection", "RemovePositions")
}
func (*collectionStore) ReorderPositions(context.Context, string, int64, []int64) error {
	return notImpl("Collection", "ReorderPositions")
}
func (*collectionStore) MovePosition(context.Context, string, int64, int64, int64) error {
	return notImpl("Collection", "MovePosition")
}
func (*collectionStore) CopyPosition(context.Context, string, int64, int64) error {
	return notImpl("Collection", "CopyPosition")
}
func (*collectionStore) Positions(context.Context, string, int64) iter.Seq2[*domain.Position, error] {
	return errSeq2[domain.Position](notImpl("Collection", "Positions"))
}
func (*collectionStore) CollectionsOf(context.Context, string, int64) iter.Seq2[*storage.Collection, error] {
	return errSeq2[storage.Collection](notImpl("Collection", "CollectionsOf"))
}
func (*collectionStore) PositionIndexMap(context.Context, string) (map[int64]int, error) {
	return nil, notImpl("Collection", "PositionIndexMap")
}
