//go:build postgres

package bulkload_test

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"

	internalserver "github.com/kevung/blunderdb/internal/server"
	"github.com/kevung/blunderdb/pkg/blunderdb/bulkload"
	_ "github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
)

const loaderPassword = "loader-secret"

// cluster is a throwaway server; each database in it has the engine's schema,
// RLS forced, and the loader role's policy, as a started server leaves them.
type cluster struct {
	t     *testing.T
	admin string
}

func startCluster(t *testing.T) *cluster {
	t.Helper()
	ctx := context.Background()
	c, err := tcpg.Run(ctx, "postgres:16-alpine", tcpg.WithDatabase("postgres"),
		tcpg.WithUsername("test"), tcpg.WithPassword("test"), tcpg.BasicWaitStrategies())
	if err != nil {
		if os.Getenv("BLUNDERDB_REQUIRE_PG") == "1" {
			t.Fatalf("postgres container: %v", err)
		}
		t.Skipf("postgres container unavailable: %v", err)
	}
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(c) })
	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	cl := &cluster{t: t, admin: dsn}
	cl.exec("postgres", `CREATE ROLE `+pg.LoaderRole+` LOGIN PASSWORD '`+loaderPassword+`' NOSUPERUSER NOBYPASSRLS`)
	return cl
}

func (c *cluster) dsn(db, user, password string) string {
	u, _ := url.Parse(c.admin)
	u.Path = "/" + db
	if user != "" {
		u.User = url.UserPassword(user, password)
	}
	return u.String()
}

func (c *cluster) exec(db, sql string, args ...any) {
	c.t.Helper()
	conn, err := pgx.Connect(context.Background(), c.dsn(db, "", ""))
	if err != nil {
		c.t.Fatal(err)
	}
	defer conn.Close(context.Background())
	if _, err := conn.Exec(context.Background(), sql, args...); err != nil {
		c.t.Fatalf("%s: %v", sql, err)
	}
}

// database creates db and gives it the schema, RLS and the loader's grants.
func (c *cluster) database(db string) {
	c.t.Helper()
	c.exec("postgres", `CREATE DATABASE `+db)
	ctx := context.Background()
	owner, err := pg.Open(ctx, c.dsn(db, "", ""), nil)
	if err != nil {
		c.t.Fatal(err)
	}
	defer owner.Close()
	if err := owner.ApplyRLS(ctx); err != nil {
		c.t.Fatal(err)
	}
}

func (c *cluster) loaderDSN(db string) string {
	u, _ := url.Parse(c.dsn(db, "", ""))
	u.User = url.User(pg.LoaderRole)
	return u.String()
}

func fixtures(t *testing.T) []string {
	t.Helper()
	dir := os.Getenv("BULKLOAD_SAMPLE")
	if dir == "" {
		dir = "../../../testdata"
	}
	var files []string
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.EqualFold(filepath.Ext(p), ".xg") {
			files = append(files, p)
		}
		return nil
	})
	if len(files) == 0 {
		t.Fatalf("no .xg under %s", dir)
	}
	return files
}

// sampleDir links the sample's files into one directory, in the order the
// loader reads them, so both paths import the same files in the same order.
func sampleDir(t *testing.T, files []string, limit int) (string, []string) {
	dir := t.TempDir()
	var paths []string
	for i, f := range files {
		if limit > 0 && i == limit {
			break
		}
		abs, _ := filepath.Abs(f)
		p := filepath.Join(dir, strings.Repeat("0", 6-len(itoa(i)))+itoa(i)+".xg")
		if err := os.Symlink(abs, p); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	return dir, paths
}

func itoa(i int) string {
	b := []byte{}
	for {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
		if i == 0 {
			return string(b)
		}
	}
}

// digest is one checksum per engine table, over every column but the ones
// that differ by nature between two runs (times, random uuids) and the
// import's own bookkeeping, which names the files. A float is rounded: a sum
// in another row order differs in its last bits.
func digest(t *testing.T, dsn string) map[string]string {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	rows, err := conn.Query(ctx, `SELECT table_name, string_agg(CASE WHEN data_type = 'double precision'
			THEN 'round(' || quote_ident(column_name) || '::numeric, 9)' ELSE quote_ident(column_name) END, ',' ORDER BY ordinal_position)
		FROM information_schema.columns
		WHERE table_schema = 'public' AND data_type NOT IN ('timestamp with time zone', 'timestamp without time zone', 'uuid', 'date')
		  AND table_name NOT IN ('import_batch', 'import_batch_file', 'change_log', 'library_settings', 'schema_migrations', 'metadata')
		  AND column_name NOT IN ('import_batch_id', 'source_path')
		  AND NOT (table_name = 'analysis' AND column_name = 'data')
		GROUP BY table_name`)
	if err != nil {
		t.Fatal(err)
	}
	cols := map[string]string{}
	for rows.Next() {
		var tb, cs string
		if err := rows.Scan(&tb, &cs); err != nil {
			t.Fatal(err)
		}
		cols[tb] = cs
	}
	rows.Close()
	out := map[string]string{}
	for tb, cs := range cols {
		var n int64
		var sum *string
		q := `SELECT count(*), md5(string_agg(r::text, '|' ORDER BY r::text)) FROM (SELECT ` + cs + ` FROM ` + tb + `) r`
		if err := conn.QueryRow(ctx, q).Scan(&n, &sum); err != nil {
			t.Fatalf("%s: %v", tb, err)
		}
		if n > 0 {
			out[tb] = itoa(int(n)) + ":" + *sum
		}
	}
	out["analysis.data"] = analysisDigest(t, dsn)
	return out
}

// analysisDigest reads every analysis back through the engine, its dates (the
// time of the import) cleared, since they ride inside the encoded data.
func analysisDigest(t *testing.T, dsn string) string {
	t.Helper()
	ctx := storage.WithTenant(context.Background(), 0)
	st, err := pg.Open(ctx, dsn, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ids, err := st.Positions().ListIDs(ctx, "", storage.ListOpts{})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(ids)
	got, err := st.Analyses().LoadMany(ctx, "", ids)
	if err != nil {
		t.Fatal(err)
	}
	h := md5.New()
	for _, id := range ids {
		a := got[id]
		if a == nil {
			continue
		}
		a.CreationDate, a.LastModifiedDate = time.Time{}, time.Time{}
		b, err := json.Marshal(a)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(h, "%d:%s|", id, b)
	}
	return itoa(len(got)) + ":" + hex.EncodeToString(h.Sum(nil))
}

func load(t *testing.T, ctx context.Context, c *cluster, db, src string, batch int64) (bulkload.Report, error) {
	return bulkload.Load(ctx, bulkload.Options{
		DSN: c.loaderDSN(db), Password: loaderPassword, Source: src, Batch: batch, ChunkFiles: 7,
	})
}

func TestLoadMatchesTheOrdinaryImport(t *testing.T) {
	c := startCluster(t)
	ctx := context.Background()
	dir, paths := sampleDir(t, fixtures(t), 300)

	c.database("ordinary")
	st, err := pg.Open(ctx, c.dsn("ordinary", "", ""), &storage.Options{EnableRLS: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.ChangeJournal().SetJournalOff(storage.WithTenant(ctx, 0), "", true); err != nil {
		t.Fatal(err)
	}
	for _, p := range paths {
		if _, err := ingest.ImportFiles(storage.WithTenant(ctx, 0), st, []string{p}, ingest.PipelineOptions{Workers: 1, FilesPerTx: 1}); err != nil {
			t.Fatal(err)
		}
	}
	st.Close()
	want := digest(t, c.dsn("ordinary", "", ""))
	if want["position"] == "" || want["analysis"] == "" || want["match_stats"] == "" {
		t.Fatalf("the ordinary import wrote no positions, analyses or statistics: %v", want)
	}

	c.database("bulk")
	rep, err := load(t, ctx, c, "bulk", dir, 0)
	if err != nil {
		t.Fatal(err)
	}
	if rep.FilesDone != len(paths) || rep.Failed != 0 {
		t.Fatalf("load: %+v for %d files", rep, len(paths))
	}
	got := digest(t, c.dsn("bulk", "", ""))
	for tb := range union(want, got) {
		if want[tb] != got[tb] {
			t.Errorf("%s: ordinary %s, bulk load %s", tb, want[tb], got[tb])
		}
	}
	t.Logf("%d files, %d tables compared, positions %s", len(paths), len(want), want["position"])
}

func union(a, b map[string]string) map[string]bool {
	u := map[string]bool{}
	for k := range a {
		u[k] = true
	}
	for k := range b {
		u[k] = true
	}
	return u
}

func TestLoadResumesWithoutDuplicates(t *testing.T) {
	c := startCluster(t)
	ctx := context.Background()
	dir, paths := sampleDir(t, fixtures(t), 40)

	c.database("whole")
	if _, err := load(t, ctx, c, "whole", dir, 0); err != nil {
		t.Fatal(err)
	}
	want := digest(t, c.dsn("whole", "", ""))

	c.database("cut")
	cut, cancel := context.WithCancel(ctx)
	stop := len(paths) / 2
	rep, err := bulkload.Load(cut, bulkload.Options{
		DSN: c.loaderDSN("cut"), Password: loaderPassword, Source: dir, ChunkFiles: 3, FilesPerTx: 2,
		Progress: func(r bulkload.Report) {
			if r.FilesDone >= stop {
				cancel()
			}
		},
	})
	cancel()
	if err == nil || rep.FilesDone >= len(paths) {
		t.Fatalf("the run was not cut: %+v, %v", rep, err)
	}
	again, err := load(t, ctx, c, "cut", dir, rep.Batch)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Imported == 0 || again.Imported == 0 || again.Failed+rep.Failed != 0 {
		t.Fatalf("cut %+v, resumed %+v: both runs must import", rep, again)
	}
	if again.Imported+rep.Imported > len(paths) {
		t.Fatalf("resumed run imported %d after %d, for %d files", again.Imported, rep.Imported, len(paths))
	}
	got := digest(t, c.dsn("cut", "", ""))
	for tb := range union(want, got) {
		if want[tb] != got[tb] {
			t.Errorf("%s: whole %s, cut then resumed %s", tb, want[tb], got[tb])
		}
	}
}

func TestLoaderRoleReachesOnlyThePublicLibrary(t *testing.T) {
	c := startCluster(t)
	ctx := context.Background()
	c.database("lib")
	c.exec("lib", `INSERT INTO library_settings (tenant_id, key, value) VALUES (7, 'k', 'private')`)

	conn, err := pgx.Connect(ctx, c.dsn("lib", pg.LoaderRole, loaderPassword))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, `SELECT set_config('app.tenant_id', '7', false)`); err != nil {
		t.Fatal(err)
	}
	var seen int
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM library_settings`).Scan(&seen); err != nil || seen != 0 {
		t.Errorf("the loader reads tenant 7: %d rows, %v", seen, err)
	}
	if _, err := conn.Exec(ctx, `INSERT INTO library_settings (tenant_id, key, value) VALUES (7, 'k2', 'x')`); err == nil {
		t.Error("the loader wrote into tenant 7")
	}
	if tag, err := conn.Exec(ctx, `DELETE FROM library_settings WHERE tenant_id = 7`); err != nil || tag.RowsAffected() != 0 {
		t.Errorf("the loader deleted in tenant 7: %v, %v", tag, err)
	}
	if _, err := conn.Exec(ctx, `TRUNCATE library_settings`); err == nil {
		t.Error("the loader truncated a tenant table")
	}
	if _, err := conn.Exec(ctx, `CREATE TABLE escape (x int)`); err == nil {
		t.Error("the loader created a table")
	}
	if _, err := conn.Exec(ctx, `ALTER TABLE position DISABLE ROW LEVEL SECURITY`); err == nil {
		t.Error("the loader lifted row-level security")
	}
	if _, err := conn.Exec(ctx, `SELECT set_config('app.tenant_id', '0', false)`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, `INSERT INTO library_settings (tenant_id, key, value) VALUES (0, 'k3', 'x')`); err != nil {
		t.Errorf("the loader cannot write the public library: %v", err)
	}

	if _, err := internalserver.OpenStorage(ctx, "postgres", c.dsn("lib", pg.LoaderRole, loaderPassword), true); err == nil ||
		!strings.Contains(err.Error(), "never serves") {
		t.Errorf("the server started as the loader: %v", err)
	}
	if _, err := bulkload.Load(ctx, bulkload.Options{DSN: c.dsn("lib", pg.LoaderRole, loaderPassword), Password: loaderPassword, Source: "."}); err == nil ||
		strings.Contains(err.Error(), loaderPassword) {
		t.Errorf("a DSN with its password: %v", err)
	}
	u, _ := url.Parse(c.dsn("lib", "test", ""))
	u.User = url.User("test")
	if _, err := bulkload.Load(ctx, bulkload.Options{DSN: u.String(), Password: "test", Source: "."}); err == nil {
		t.Error("the loader ran as another role")
	}
}
