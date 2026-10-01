//go:build postgres

package postgres_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"

	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

// TestMigrate_028_GameWinner replays 028 on a library holding one match per
// source of storagetest.WinnerMigrationCases, the fixture SQLite's
// TestMigrate_2_25_0_to_2_26_0_GameWinner runs too, with RLS applied: the
// migration must convert every tenant's games and leave RLS FORCEd.
func TestMigrate_028_GameWinner(t *testing.T) {
	ctx := context.Background()
	dsn := startPostgres(t)

	s, err := pg.Open(ctx, dsn, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	if err := s.ApplyRLS(ctx); err != nil {
		t.Fatalf("ApplyRLS: %v", err)
	}
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	cases := storagetest.WinnerMigrationCases()
	gameIDs := make([][]int64, len(cases))
	for i, c := range cases {
		tenant := int64(i % 2) // two tenants: the migration is not scoped
		var batchID *int64
		if c.BatchFormat != "" {
			var id int64
			if err := conn.QueryRow(ctx, `INSERT INTO import_batch (tenant_id, source, format) VALUES ($1, $2, $3) RETURNING id`,
				tenant, c.FilePath, c.BatchFormat).Scan(&id); err != nil {
				t.Fatalf("%s: insert batch: %v", c.Name, err)
			}
			batchID = &id
		}
		var matchID int64
		if err := conn.QueryRow(ctx, `INSERT INTO match (tenant_id, player1_name, player2_name, match_length, file_path, import_batch_id)
			VALUES ($1, 'A', 'B', $2, $3, $4) RETURNING id`, tenant, c.Length, c.FilePath, batchID).Scan(&matchID); err != nil {
			t.Fatalf("%s: insert match: %v", c.Name, err)
		}
		for n, g := range c.Games {
			var id int64
			if err := conn.QueryRow(ctx, `INSERT INTO game (tenant_id, match_id, game_number, initial_score_1, initial_score_2, winner, points_won)
				VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`, tenant, matchID, n+1, g.S1, g.S2, g.Winner, g.PointsWon).Scan(&id); err != nil {
				t.Fatalf("%s: insert game: %v", c.Name, err)
			}
			gameIDs[i] = append(gameIDs[i], id)
		}
	}

	if _, err := conn.Exec(ctx, `DELETE FROM schema_migrations WHERE version = '028_game_winner_encoding'`); err != nil {
		t.Fatalf("forget 028: %v", err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	for i, c := range cases {
		for n, g := range c.Games {
			var got int32
			if err := conn.QueryRow(ctx, `SELECT winner FROM game WHERE id = $1`, gameIDs[i][n]).Scan(&got); err != nil {
				t.Fatalf("%s: read game %d: %v", c.Name, n+1, err)
			}
			if got != g.Want {
				t.Errorf("%s, game %d: winner %d, want %d", c.Name, n+1, got, g.Want)
			}
		}
	}
	for _, table := range []string{"game", "match", "import_batch"} {
		var forced bool
		if err := conn.QueryRow(ctx, `SELECT relforcerowsecurity FROM pg_class WHERE oid = to_regclass($1)`, table).Scan(&forced); err != nil {
			t.Fatalf("inspect %s: %v", table, err)
		}
		if !forced {
			t.Errorf("%s: RLS no longer FORCEd after 028", table)
		}
	}
}
