package sqlshared_test

import (
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/statsequal"
)

// TestReadOnlyStatsProbeAndReadShareASnapshot: a writer invalidating a
// match between a read-only reader's completeness probe and its read of the
// cells changes nothing the reader returns — the read is the snapshot the
// probe vouched for, not a table missing a match.
func TestReadOnlyStatsProbeAndReadShareASnapshot(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "demo.db")
	src, err := os.Open(filepath.Join("..", "..", "..", "..", "internal", "gui", "demo.db.gz"))
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	zr, err := gzip.NewReader(src)
	if err != nil {
		t.Fatal(err)
	}
	dst, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(dst, zr); err != nil {
		t.Fatal(err)
	}
	if err := dst.Close(); err != nil {
		t.Fatal(err)
	}

	wdb, err := sql.Open("sqlite", sqlite.DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	defer wdb.Close()
	w := sqlite.New(wdb)
	rdb, err := sql.Open("sqlite", sqlite.DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	defer rdb.Close()
	rdb.SetMaxOpenConns(1)
	if _, err := rdb.Exec(`PRAGMA query_only = ON`); err != nil {
		t.Fatal(err)
	}
	ro := sqlite.New(rdb)

	f := storage.StatsFilter{DecisionType: -1}
	type reader struct {
		name string
		run  func(storage.Storage) (any, error)
	}
	for _, r := range []reader{
		{"Compute", func(st storage.Storage) (any, error) { return st.Stats().Compute(ctx, "", f) }},
		{"MatchSeries", func(st storage.Storage) (any, error) { return st.Stats().MatchSeries(ctx, "", f) }},
		{"PlayerTable", func(st storage.Storage) (any, error) { return st.Stats().PlayerTable(ctx, "", f) }},
	} {
		t.Run(r.name, func(t *testing.T) {
			// The writer fills the table, then the reader's answer is the reference.
			if _, err := w.Stats().FillMatchStats(ctx, "", nil); err != nil {
				t.Fatal(err)
			}
			want, err := r.run(ro)
			if err != nil {
				t.Fatal(err)
			}
			fired := false
			restore := sqlshared.SetAfterMatchStatsProbe(func() {
				if fired {
					return
				}
				fired = true
				if _, err := wdb.ExecContext(ctx, `DELETE FROM match_stats WHERE match_id = (SELECT MIN(match_id) FROM match_stats WHERE decisions > 0)`); err != nil {
					t.Error(err)
				}
			})
			got, err := r.run(ro)
			restore()
			if err != nil {
				t.Fatal(err)
			}
			if !fired {
				t.Fatal("the reader never probed the table")
			}
			a, _ := json.Marshal(want)
			b, _ := json.Marshal(got)
			if same, err := statsequal.JSON(string(a), string(b)); err != nil || !same {
				t.Errorf("a match invalidated after the probe changed the read-only figures (err %v)", err)
			}
		})
	}
}
