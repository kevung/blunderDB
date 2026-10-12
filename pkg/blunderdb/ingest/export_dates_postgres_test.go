//go:build postgres

package ingest

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
)

func TestExportKeepsOriginDates_Postgres(t *testing.T) {
	ctx := context.Background()
	container, err := tcpg.Run(ctx, "postgres:16-alpine",
		tcpg.WithDatabase("blunderdb"),
		tcpg.WithUsername("test"),
		tcpg.WithPassword("test"),
		tcpg.BasicWaitStrategies(),
	)
	if err != nil {
		if os.Getenv("BLUNDERDB_REQUIRE_PG") == "1" {
			t.Fatalf("postgres container unavailable (BLUNDERDB_REQUIRE_PG=1 requires Docker): %v", err)
		}
		t.Skipf("postgres container unavailable (Docker required): %v", err)
	}
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(container) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	src, err := postgres.Open(ctx, dsn, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	checkExportKeepsOriginDates(t, src, "2", func(table, column string) {
		if _, err := conn.Exec(ctx, `UPDATE `+table+` SET `+column+` = $1::timestamptz`, originDate+"+00"); err != nil {
			t.Fatal(err)
		}
	})
}
