package sqlite_test

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

func TestChangeSetLinks_SQLite(t *testing.T) {
	ctx := context.Background()
	storagetest.CheckChangeSetLinks(t, func(t *testing.T) *storagetest.ChangeSetPair {
		a, _ := openLib(t)
		b, _ := openLib(t)
		return storagetest.NewChangeSetPair(t, a, ctx, "", b, ctx, "")
	})
}
