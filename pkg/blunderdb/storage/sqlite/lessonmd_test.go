package sqlite_test

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

func TestLessonMarkdown_SQLite(t *testing.T) {
	s, _ := openLib(t)
	storagetest.CheckLessonMarkdown(t, s, context.Background(), "")
	storagetest.CheckLessonImportAtomic(t, s, "", func(c context.Context) context.Context { return c })
}
