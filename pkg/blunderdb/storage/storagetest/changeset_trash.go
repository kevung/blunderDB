package storagetest

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/trash"
)

func trashCollection(ctx context.Context, s storage.Storage, scope string, id int64) (int64, error) {
	return trash.Collection(ctx, s, scope, id)
}

func restoreCollection(ctx context.Context, s storage.Storage, scope string, trashID int64) error {
	_, err := trash.Restore(ctx, s, scope, trashID)
	return err
}
