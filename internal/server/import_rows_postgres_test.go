//go:build postgres

package server

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/internal/server/metrics"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
)

const importRowsGolden = "testdata/import_rows_golden.json"

// importRowsTables are the tables an import writes, with what a run cannot
// reproduce: random uuids, clocks and the journal's sequence. analysis.id goes
// too: nothing refers to it, the row is known by its position.
var importRowsTables = map[string][]string{
	"position":             {"uuid", "public_id"},
	"analysis":             {"id", "data", "creation_date"},
	"match":                {"uuid", "import_date", "file_path"},
	"game":                 {"uuid"},
	"move":                 {"uuid"},
	"comment":              {"id", "uuid", "created_at", "modified_at"},
	"tournament":           {"uuid", "created_at", "updated_at"},
	"import_batch":         {"started_at", "finished_at"},
	"match_stats":          {"computed_at"},
	"match_stats_cell":     {"mwc_loss"},
	"match_stats_position": {"id"},
	"action_label":         {"id"},
	"change_log":           {"seq", "uuid", "changed_at", "xid"},
}

// importRowsExtra adds what is compared otherwise than as stored. A cell's
// mwc_loss is a float summed in the order the rows come back: its last bits
// are not a fact about the import.
var importRowsExtra = map[string]string{
	"match_stats_cell": `|| jsonb_build_object('mwc', round(t.mwc_loss::numeric, 6))`,
}

// importRowsDigests reads each table as sorted JSON lines, cut of the volatile
// columns, and returns "rows:md5" by table.
func importRowsDigests(t *testing.T, ctx context.Context, dsn string) map[string]string {
	t.Helper()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	out := map[string]string{}
	for table, volatile := range importRowsTables {
		var n int64
		var sum string
		query := fmt.Sprintf(`SELECT COUNT(*), COALESCE(md5(string_agg(x, E'\n' ORDER BY x)), '')
			FROM (SELECT ((to_jsonb(t) - $1::text[]) %s)::text AS x FROM %s t) s`, importRowsExtra[table], table)
		if err := conn.QueryRow(ctx, query, volatile).Scan(&n, &sum); err != nil {
			t.Fatalf("%s: %v", table, err)
		}
		out[table] = fmt.Sprintf("%d:%s", n, sum)
	}
	// An analysis blob holds clocks: its content key says what it means.
	rows, err := conn.Query(ctx, `SELECT position_id, data FROM analysis ORDER BY position_id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	h := sha256.New()
	var n int
	for rows.Next() {
		var id int64
		var data []byte
		if err := rows.Scan(&id, &data); err != nil {
			t.Fatal(err)
		}
		a, err := engine.DecodeAnalysisFromStorage(data)
		if err != nil {
			t.Fatal(err)
		}
		a.CreationDate = time.Time{}
		key, err := engine.AnalysisContentKey(&a)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(h, "%d:%x\n", id, key)
		n++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	out["analysis_data"] = fmt.Sprintf("%d:%x", n, h.Sum(nil))
	return out
}

// importRowsFiles is the corpus in a stable order: every match file of the
// repository, the .xg again (the duplicate path), the others after them (the
// cross-format enrichment of a match already stored).
func importRowsFiles(t *testing.T, dir string) (uploads []string) {
	t.Helper()
	var xg, others []string
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		switch strings.ToLower(filepath.Ext(p)) {
		case ".xg":
			xg = append(xg, p)
		case ".sgf", ".mat":
			others = append(others, p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(xg)
	slices.Sort(others)
	return slices.Concat(xg, others, xg)
}

// TestImportStoredRowsAreStable holds what an import stores, row by row, to
// the digests recorded when each position still took its own round trips: a
// write path may change how it reaches the database, never what it leaves
// there. After a deliberate change of a written column, regenerate with
// BLUNDERDB_IMPORT_ROWS_UPDATE=1 and say why in the commit.
//
// BLUNDERDB_IMPORT_ROWS_CORPUS points it at another corpus: the digests then go
// to BLUNDERDB_BENCH_OUT (they name nothing) for two builds to be compared.
func TestImportStoredRowsAreStable(t *testing.T) {
	ctx := context.Background()
	super := os.Getenv("BLUNDERDB_BENCH_DSN")
	if super == "" {
		super = postgresDSN(t)
	}
	name := fmt.Sprintf("rows_%d", os.Getpid())
	benchExec(t, ctx, super, fmt.Sprintf(`CREATE ROLE %s LOGIN PASSWORD '%s'`, benchRole, benchRole))
	benchExec(t, ctx, super, fmt.Sprintf(`CREATE DATABASE %s OWNER %s`, name, benchRole))
	t.Cleanup(func() { benchExec(t, ctx, super, fmt.Sprintf(`DROP DATABASE %s WITH (FORCE)`, name)) })

	st := benchOpen(t, ctx, benchDSN(t, super, benchRole, name))
	defer st.Close()
	if _, ok := st.(*pg.Storage); !ok {
		t.Fatal("postgres storage expected")
	}
	srv, err := New(Options{Storage: st, Metrics: metrics.New(), EnableMetrics: true})
	if err != nil {
		t.Fatal(err)
	}
	ts := newBenchHTTPServer(t, srv)

	corpus := cmp.Or(os.Getenv("BLUNDERDB_IMPORT_ROWS_CORPUS"), filepath.Join("..", "..", "testdata"))
	for _, p := range importRowsFiles(t, corpus) {
		benchUploadAny(t, ts, p)
	}
	got := importRowsDigests(t, ctx, benchDSN(t, super, "", name))

	if os.Getenv("BLUNDERDB_IMPORT_ROWS_CORPUS") != "" {
		blob, _ := json.MarshalIndent(got, "", "  ")
		if err := os.WriteFile(os.Getenv("BLUNDERDB_BENCH_OUT"), blob, 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	if os.Getenv("BLUNDERDB_IMPORT_ROWS_UPDATE") != "" {
		blob, _ := json.MarshalIndent(got, "", "  ")
		if err := os.WriteFile(importRowsGolden, append(blob, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	blob, err := os.ReadFile(importRowsGolden)
	if err != nil {
		t.Fatal(err)
	}
	var want map[string]string
	if err := json.Unmarshal(blob, &want); err != nil {
		t.Fatal(err)
	}
	for table, w := range want {
		if got[table] != w {
			t.Errorf("%s: stored rows %s, recorded %s", table, got[table], w)
		}
	}
}

// TestImportBatchUnderRLS: a batch outlives its request, and its writes must
// still carry the tenant the policies check. Without it every file fails and
// the batch still ends "done".
func TestImportBatchUnderRLS(t *testing.T) {
	ctx := context.Background()
	super := os.Getenv("BLUNDERDB_BENCH_DSN")
	if super == "" {
		super = postgresDSN(t)
	}
	name := fmt.Sprintf("rlsbatch_%d", os.Getpid())
	benchExec(t, ctx, super, fmt.Sprintf(`CREATE ROLE %s LOGIN PASSWORD '%s'`, benchRole, benchRole))
	benchExec(t, ctx, super, fmt.Sprintf(`CREATE DATABASE %s OWNER %s`, name, benchRole))
	t.Cleanup(func() { benchExec(t, ctx, super, fmt.Sprintf(`DROP DATABASE %s WITH (FORCE)`, name)) })
	st := benchOpen(t, ctx, benchDSN(t, super, benchRole, name))
	defer st.Close()
	srv, err := New(Options{Storage: st, Metrics: metrics.New()})
	if err != nil {
		t.Fatal(err)
	}
	ts := newBenchHTTPServer(t, srv)

	files := []benchFile{{path: filepath.Join("..", "..", "testdata", "charlot1-charlot2_7p_2025-11-08-2305.xg")}}
	benchBatch(t, ts, srv, files)

	conn, err := pgx.Connect(ctx, benchDSN(t, super, "", name))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	var matches, batches int
	if err := conn.QueryRow(ctx, `SELECT (SELECT COUNT(*) FROM match), (SELECT COUNT(*) FROM import_batch WHERE finished_at IS NOT NULL)`).Scan(&matches, &batches); err != nil {
		t.Fatal(err)
	}
	if matches != 1 || batches != 1 {
		t.Errorf("after a batch under RLS: %d match(es), %d closed batch(es), want 1 and 1", matches, batches)
	}
}
