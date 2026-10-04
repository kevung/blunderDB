//go:build postgres

package postgres_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

// TestReencodeAnalysesUpgradesLegacy_Postgres runs the shared legacy check,
// planting blobs through a raw connection.
func TestReencodeAnalysesUpgradesLegacy_Postgres(t *testing.T) {
	ctx := context.Background()
	s, dsn := openMatchStore(t)
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("pgx.Connect: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })
	storagetest.CheckReencodeUpgradesLegacy(t, s, func(id int64, blob []byte) {
		if _, err := conn.Exec(ctx, `UPDATE analysis SET data = $1 WHERE position_id = $2`, blob, id); err != nil {
			t.Fatal(err)
		}
	})
}
