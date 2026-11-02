//go:build postgres

package postgres_test

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

// TestLessonMarkdown_Postgres: the Markdown round trip under FORCE RLS.
func TestLessonMarkdown_Postgres(t *testing.T) {
	s, _ := openRLS(t)
	storagetest.CheckLessonMarkdown(t, s, storage.WithTenant(context.Background(), 1), "1")
	storagetest.CheckLessonImportAtomic(t, s, "1", func(c context.Context) context.Context { return storage.WithTenant(c, 1) })
}
