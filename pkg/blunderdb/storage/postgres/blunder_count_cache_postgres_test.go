//go:build postgres

package postgres_test

import (
	"context"
	"strings"
	"testing"

	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

func TestBlunderCountCache_Postgres(t *testing.T) {
	s, _ := openMatchStore(t)
	storagetest.RunBlunderCountCache(t, s, pg.SharedExecer(s))
}

// TRUNCATE empties a table for every tenant and fires no row trigger: it
// would leave every cached count stale, so it is refused.
func TestBlunderCountCacheRefusesTruncate_Postgres(t *testing.T) {
	s, _ := openMatchStore(t)
	_, err := pg.SharedExecer(s).Exec(context.Background(), `TRUNCATE action_label`)
	if err == nil || !strings.Contains(err.Error(), "blunder count") {
		t.Fatalf("TRUNCATE action_label: %v, want refused for the blunder count", err)
	}
}
