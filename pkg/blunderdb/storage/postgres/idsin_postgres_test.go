//go:build postgres

package postgres_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5"

	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
)

// The IDsIn condition must be accepted by a real Postgres and select exactly
// the listed ids, for a short list and a long one, both with a parameter.
func TestIDsInExecutes_Postgres(t *testing.T) {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, startPostgres(t))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, `CREATE TABLE ids_t AS SELECT g::bigint AS id FROM generate_series(1, 5000) g`); err != nil {
		t.Fatal(err)
	}
	for _, n := range []int{3, 4000} {
		ids := make([]string, n)
		for i := range ids {
			ids[i] = strconv.Itoa(i + 1)
		}
		cond, args := pg.IDsIn("id + 0", ids)
		var got int
		if err := conn.QueryRow(ctx, "SELECT count(*) FROM ids_t WHERE "+replaceQ(cond), args...).Scan(&got); err != nil {
			t.Fatalf("%d ids: %v", n, err)
		}
		if got != n {
			t.Errorf("%d ids: matched %d", n, got)
		}
	}
}

func replaceQ(s string) string {
	out := ""
	k := 0
	for _, c := range s {
		if c == '?' {
			k++
			out += "$" + strconv.Itoa(k)
		} else {
			out += string(c)
		}
	}
	return out
}
