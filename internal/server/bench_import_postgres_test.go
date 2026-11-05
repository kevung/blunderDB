//go:build postgres

// Reproducible import bench over the real path: HTTP upload, spool, parser,
// write transaction, end-of-import report, on PostgreSQL under RLS with a
// non-superuser owner, as `blunderdb serve --backend postgres --rls` runs.
//
//	BLUNDERDB_BENCH_CORPUS=/dir/of/xg \
//	BLUNDERDB_BENCH_DSN=postgres://super:pw@host:5432/postgres?sslmode=disable \
//	go test -tags postgres -count=1 -timeout 60m ./internal/server/ -run TestBenchImportHTTP -v
//
// The corpus stays outside the repository. Files are taken in digest order, so
// the selection is stable; the output holds sizes and counts, never a name.
//
// BLUNDERDB_BENCH_FILES   how many files to measure (default 50)
// BLUNDERDB_BENCH_SEED    synthetic matches stored beforehand (default 0)
// BLUNDERDB_BENCH_QUERY   query string of each upload (e.g. report=deferred)
// BLUNDERDB_BENCH_BATCH   send the files as one .tar to imports.batch
// BLUNDERDB_BENCH_DB      send the files as one native .db to imports.db
// BLUNDERDB_BENCH_OUT     where to write the JSON result
// BLUNDERDB_BENCH_NEW_TENANT  measure on a tenant other than the seeded one
// BLUNDERDB_BENCH_TEMPLATE  database to clone instead of seeding; when it does
// not exist yet, it is created by seeding and kept for the next run.
package server

import (
	"archive/tar"
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/internal/server/metrics"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

const (
	benchRole   = "bench_owner"
	benchTenant = int64(1000)
)

// benchMeasuredTenant is the tenant the measured uploads go to: the seeded one,
// or with BLUNDERDB_BENCH_NEW_TENANT a new account in a database others filled.
func benchMeasuredTenant() int64 {
	if os.Getenv("BLUNDERDB_BENCH_NEW_TENANT") != "" {
		return benchTenant + 1
	}
	return benchTenant
}

type benchFile struct {
	path   string
	digest string
	size   int64
}

// benchCorpus lists the distinct .xg files of dir in digest order.
func benchCorpus(t *testing.T, dir string) []benchFile {
	t.Helper()
	seen := map[string]bool{}
	var out []benchFile
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.EqualFold(filepath.Ext(p), ".xg") {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		h := hex.EncodeToString(sum[:])
		if len(data) == 0 || seen[h] {
			return nil
		}
		seen[h] = true
		out = append(out, benchFile{p, h, int64(len(data))})
		return nil
	})
	if err != nil {
		t.Fatalf("corpus: %v", err)
	}
	slices.SortFunc(out, func(a, b benchFile) int { return cmp.Compare(a.digest, b.digest) })
	return out
}

func benchExec(t *testing.T, ctx context.Context, dsn, sql string, args ...any) {
	t.Helper()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, sql, args...); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("%s: %v", sql, err)
	}
}

func benchDBExists(t *testing.T, ctx context.Context, dsn, name string) bool {
	t.Helper()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)
	var ok bool
	if err := conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, name).Scan(&ok); err != nil {
		t.Fatal(err)
	}
	return ok
}

func benchDSN(t *testing.T, super, user, db string) string {
	t.Helper()
	u, err := url.Parse(super)
	if err != nil {
		t.Fatal(err)
	}
	if user != "" {
		u.User = url.UserPassword(user, user)
	}
	u.Path = "/" + db
	return u.String()
}

// benchOpen migrates db as its owner and installs RLS, as serve --rls does.
func benchOpen(t *testing.T, ctx context.Context, dsn string) storage.Storage {
	t.Helper()
	st, err := OpenStorage(ctx, "postgres", dsn, true)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := st.(*pg.Storage).ApplyRLS(ctx); err != nil {
		t.Fatalf("rls: %v", err)
	}
	return st
}

// benchSeed stores n matches derived from the corpus under positions of their
// own: a score no match reaches changes the Zobrist identity (it clamps at 63),
// so the library grows as n different matches would make it.
func benchSeed(t *testing.T, ctx context.Context, st storage.Storage, files []benchFile, n int) {
	t.Helper()
	scope := strconv.FormatInt(benchTenant, 10)
	ctx = storage.WithTenant(ctx, benchTenant)
	for i := 0; i < n; i++ {
		f := files[i%len(files)]
		g, err := ingest.MapXG(f.path)
		if err != nil {
			continue
		}
		replica := i / len(files)
		g.Match.MatchHash = fmt.Sprintf("seed-%d", i)
		g.Match.CanonicalHash = fmt.Sprintf("seed-canon-%d", i)
		g.Match.DiceHash = fmt.Sprintf("seed-dice-%d", i)
		g.Match.Event = ""
		shift := func(p *domain.Position) {
			if p != nil {
				p.Score[0], p.Score[1] = 40+replica, 40+replica
			}
		}
		for gi := range g.Games {
			for mi := range g.Games[gi].Moves {
				shift(g.Games[gi].Moves[mi].Position)
			}
		}
		tx, err := st.BeginTx(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ingest.WriteMatch(ctx, tx, scope, g, nil); err != nil {
			_ = tx.Rollback()
			t.Fatalf("seed: %v", err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}
}

func benchLoad() float64 {
	b, _ := os.ReadFile("/proc/loadavg")
	v, _ := strconv.ParseFloat(strings.Fields(string(b))[0], 64)
	return v
}

func benchMedian(d []time.Duration) time.Duration {
	s := slices.Sorted(slices.Values(d))
	if len(s) == 0 {
		return 0
	}
	return s[len(s)/2]
}

func newBenchHTTPServer(t *testing.T, srv *Server) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func benchUpload(t *testing.T, ts *httptest.Server, f benchFile, query string) {
	t.Helper()
	benchPost(t, ts, f.path, "imports.xg", query)
}

// benchUploadAny uploads a match file of any format by its extension.
func benchUploadAny(t *testing.T, ts *httptest.Server, path string) {
	t.Helper()
	method := "imports.gnubg"
	if strings.EqualFold(filepath.Ext(path), ".xg") {
		method = "imports.xg"
	}
	benchPost(t, ts, path, method, "")
}

func benchPost(t *testing.T, ts *httptest.Server, path, method, query string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "bench"+strings.ToLower(filepath.Ext(path)))
	_, _ = fw.Write(data)
	_ = mw.Close()
	target := ts.URL + "/v1/" + method
	if query != "" {
		target += "?" + query
	}
	req, _ := http.NewRequest(http.MethodPost, target, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-Tenant-ID", strconv.FormatInt(benchMeasuredTenant(), 10))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || bytes.Contains(out, []byte(`"event":"error"`)) {
		t.Errorf("upload refused: %d (%d bytes back): %.200s", resp.StatusCode, len(out), lastLine(out))
	}
}

// benchNativeDB imports files into a new SQLite library and returns its path.
func benchNativeDB(t *testing.T, ctx context.Context, files []benchFile) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "source.db")
	lite, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer lite.Close()
	if err := lite.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.path
	}
	if _, err := ingest.ImportFiles(ctx, lite, paths, ingest.PipelineOptions{}); err != nil {
		t.Fatal(err)
	}
	return path
}

// benchBatch sends files as one .tar to imports.batch, waits for its end and
// reads its report once, as a client of the batch route does.
func benchBatch(t *testing.T, ts *httptest.Server, srv *Server, files []benchFile) {
	t.Helper()
	var archive bytes.Buffer
	tw := tar.NewWriter(&archive)
	for i, f := range files {
		data, err := os.ReadFile(f.path)
		if err != nil {
			t.Fatal(err)
		}
		if err := tw.WriteHeader(&tar.Header{Name: fmt.Sprintf("%d.xg", i), Mode: 0o600, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		_, _ = tw.Write(data)
	}
	_ = tw.Close()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "lot.tar")
	_, _ = fw.Write(archive.Bytes())
	_ = mw.Close()
	scope := strconv.FormatInt(benchMeasuredTenant(), 10)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/imports.batch", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-Tenant-ID", scope)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out importBatchResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || resp.StatusCode/100 != 2 {
		t.Fatalf("batch refused: %d %v", resp.StatusCode, err)
	}
	job, ok := srv.batches.get(scope, out.ImportID)
	if !ok {
		t.Fatal("no batch job")
	}
	<-job.done
	st := job.status()
	if st.State != batchStateDone {
		t.Fatalf("batch ended %s", st.State)
	}
	reportReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/imports.report", strings.NewReader(fmt.Sprintf(`{"batchId":%d}`, st.BatchID)))
	reportReq.Header.Set("X-Tenant-ID", scope)
	reportReq.Header.Set("Content-Type", "application/json")
	reportResp, err := http.DefaultClient.Do(reportReq)
	if err != nil {
		t.Fatal(err)
	}
	defer reportResp.Body.Close()
	_, _ = io.Copy(io.Discard, reportResp.Body)
	if reportResp.StatusCode != http.StatusOK {
		t.Fatalf("report refused: %d", reportResp.StatusCode)
	}
}

func lastLine(b []byte) []byte {
	b = bytes.TrimSpace(b)
	if i := bytes.LastIndexByte(b, '\n'); i >= 0 {
		return b[i+1:]
	}
	return b
}

// TestBenchImportHTTP measures the throughput and the time per file, and
// reports the library's size around the measure.
func TestBenchImportHTTP(t *testing.T) {
	dir, super := os.Getenv("BLUNDERDB_BENCH_CORPUS"), os.Getenv("BLUNDERDB_BENCH_DSN")
	if dir == "" || super == "" {
		t.Skip("BLUNDERDB_BENCH_CORPUS and BLUNDERDB_BENCH_DSN not set")
	}
	ctx := context.Background()
	nFiles, _ := strconv.Atoi(cmp.Or(os.Getenv("BLUNDERDB_BENCH_FILES"), "50"))
	seed, _ := strconv.Atoi(os.Getenv("BLUNDERDB_BENCH_SEED"))
	template := os.Getenv("BLUNDERDB_BENCH_TEMPLATE")
	files := benchCorpus(t, dir)
	if len(files) < nFiles {
		t.Fatalf("corpus holds %d files, need %d", len(files), nFiles)
	}
	measured, rest := files[:nFiles], files[nFiles:]

	benchExec(t, ctx, super, fmt.Sprintf(`CREATE ROLE %s LOGIN PASSWORD '%s'`, benchRole, benchRole))

	// A template is seeded once and cloned afterwards: the clone holds the
	// same library whatever the writer under test.
	if template != "" && !benchDBExists(t, ctx, super, template) {
		benchExec(t, ctx, super, fmt.Sprintf(`CREATE DATABASE %s OWNER %s`, template, benchRole))
		st := benchOpen(t, ctx, benchDSN(t, super, benchRole, template))
		t0 := time.Now()
		benchSeed(t, ctx, st, rest, seed)
		st.Close()
		// A library at rest has been analysed by autovacuum long ago.
		benchExec(t, ctx, benchDSN(t, super, "", template), `ANALYZE`)
		t.Logf("seeded %d matches in %s", seed, time.Since(t0).Round(time.Second))
	}
	name := fmt.Sprintf("bench_%d", time.Now().UnixNano())
	if template != "" {
		benchExec(t, ctx, super, fmt.Sprintf(`CREATE DATABASE %s TEMPLATE %s OWNER %s`, name, template, benchRole))
	} else {
		benchExec(t, ctx, super, fmt.Sprintf(`CREATE DATABASE %s OWNER %s`, name, benchRole))
	}
	if os.Getenv("BLUNDERDB_BENCH_KEEP") == "" {
		t.Cleanup(func() { benchExec(t, ctx, super, fmt.Sprintf(`DROP DATABASE %s WITH (FORCE)`, name)) })
	}
	st := benchOpen(t, ctx, benchDSN(t, super, benchRole, name))
	defer st.Close()
	if template == "" && seed > 0 {
		benchSeed(t, ctx, st, rest, seed)
	}

	srv, err := New(Options{Storage: st, Metrics: metrics.New(), EnableMetrics: true})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	count := func(sql string) int64 {
		conn, err := pgx.Connect(ctx, benchDSN(t, super, "", name))
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close(ctx)
		var n int64
		if err := conn.QueryRow(ctx, sql).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	resetStats := func() {
		conn, err := pgx.Connect(ctx, benchDSN(t, super, "", name))
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close(ctx)
		_, _ = conn.Exec(ctx, `SELECT pg_stat_statements_reset()`)
	}
	matchesBefore := count(`SELECT COUNT(*) FROM match`)
	positionsBefore := count(`SELECT COUNT(*) FROM position`)
	resetStats()

	var bytesIn int64
	for _, f := range measured {
		bytesIn += f.size
	}
	// A native base holding the measured files, built before the clock starts.
	dbSource := ""
	if os.Getenv("BLUNDERDB_BENCH_DB") != "" {
		dbSource = benchNativeDB(t, ctx, measured)
	}
	loadBefore := benchLoad()
	durations := make([]time.Duration, 0, len(measured))
	t0 := time.Now()
	switch {
	case os.Getenv("BLUNDERDB_BENCH_BATCH") != "":
		benchBatch(t, ts, srv, measured)
	case dbSource != "":
		benchPost(t, ts, dbSource, "imports.db", "")
	default:
		for _, f := range measured {
			t1 := time.Now()
			benchUpload(t, ts, f, os.Getenv("BLUNDERDB_BENCH_QUERY"))
			durations = append(durations, time.Since(t1))
		}
	}
	total := time.Since(t0)
	loadAfter := benchLoad()
	if len(durations) == 0 {
		durations = append(durations, total/time.Duration(len(measured)))
	}

	third := max(len(durations)/3, 1)
	result := map[string]any{
		"fichiers":           len(measured),
		"octets":             bytesIn,
		"matchs_avant":       matchesBefore,
		"positions_avant":    positionsBefore,
		"matchs_apres":       count(`SELECT COUNT(*) FROM match`),
		"positions_apres":    count(`SELECT COUNT(*) FROM position`),
		"coups_ecrits":       count(`SELECT COUNT(*) FROM move`),
		"total_s":            total.Seconds(),
		"matchs_par_s":       float64(len(measured)) / total.Seconds(),
		"positions_par_s":    float64(count(`SELECT COUNT(*) FROM position`)-positionsBefore) / total.Seconds(),
		"median_fichier_ms":  benchMedian(durations).Milliseconds(),
		"median_premier_ms":  benchMedian(durations[:third]).Milliseconds(),
		"median_dernier_ms":  benchMedian(durations[len(durations)-third:]).Milliseconds(),
		"charge_avant":       loadBefore,
		"charge_apres":       loadAfter,
		"allers_retours_sql": count(`SELECT COALESCE(SUM(calls),0)::bigint FROM pg_stat_statements WHERE dbid = (SELECT oid FROM pg_database WHERE datname = current_database())`),
		"valide":             loadBefore <= 16 && loadAfter <= 16,
		"base":               name,
	}
	blob, _ := json.MarshalIndent(result, "", "  ")
	t.Logf("%s", blob)
	if out := os.Getenv("BLUNDERDB_BENCH_OUT"); out != "" {
		if err := os.WriteFile(out, blob, 0o600); err != nil {
			t.Error(err)
		}
	}
}
