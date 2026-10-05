//go:build postgres

package postgres_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

// TestMigrate_028_GameWinner replays 028 on a library holding one match per
// source of storagetest.WinnerMigrationCases, the fixture SQLite's
// TestMigrate_2_25_0_to_2_26_0_GameWinner runs too, with RLS applied and
// FORCEd on an owner that is neither superuser nor BYPASSRLS: a superuser
// would see every row even if 028 forgot to lift FORCE. The migration must
// convert every tenant's games and leave RLS FORCEd.
func TestMigrate_028_GameWinner(t *testing.T) {
	ctx := context.Background()
	s, conn, _ := openAsRLSOwner(t)

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

	if err := s.ApplyRLS(ctx); err != nil {
		t.Fatalf("ApplyRLS: %v", err)
	}

	// 028 is not idempotent, so it commits with its schema_migrations row or
	// not at all: a failure while recording it must leave every game as it
	// was, and the next Migrate converts them once.
	for _, stmt := range []string{
		`DELETE FROM schema_migrations WHERE version = '028_game_winner_encoding'`,
		`CREATE FUNCTION record_fails() RETURNS trigger LANGUAGE plpgsql AS $$
		 BEGIN RAISE EXCEPTION 'interrupted'; END $$`,
		`CREATE TRIGGER record_fails BEFORE INSERT ON schema_migrations
		 FOR EACH ROW WHEN (NEW.version = '028_game_winner_encoding') EXECUTE FUNCTION record_fails()`,
	} {
		if _, err := conn.Exec(ctx, stmt); err != nil {
			t.Fatalf("prepare the interruption (%s): %v", stmt, err)
		}
	}
	if err := s.Migrate(ctx); err == nil {
		t.Fatal("Migrate succeeded although recording 028 failed")
	}
	checkWinners := func(converted bool) {
		t.Helper()
		for i, c := range cases {
			for n, g := range c.Games {
				var got *int32
				asTenant(t, conn, strconv.Itoa(i%2), func() {
					if err := conn.QueryRow(ctx, `SELECT winner FROM game WHERE id = $1`, gameIDs[i][n]).Scan(&got); err != nil {
						t.Fatalf("%s: read game %d: %v", c.Name, n+1, err)
					}
				})
				switch {
				case converted && (got == nil || *got != g.Want):
					t.Errorf("%s, game %d: winner %v, want %d", c.Name, n+1, got, g.Want)
				case !converted && g.Winner != nil && (got == nil || *got != *g.Winner):
					t.Errorf("%s, game %d: winner moved before 028 was recorded (seeded %d)", c.Name, n+1, *g.Winner)
				}
			}
		}
	}
	checkWinners(false)
	for _, stmt := range []string{`DROP TRIGGER record_fails ON schema_migrations`, `DROP FUNCTION record_fails()`} {
		if _, err := conn.Exec(ctx, stmt); err != nil {
			t.Fatalf("lift the interruption: %v", err)
		}
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	checkWinners(true)
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	checkWinners(true)
	assertForced(t, conn, "game", "match", "import_batch")
}
