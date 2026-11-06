// Package bulkload loads a large collection of match files into one tenant's
// library of a PostgreSQL database, offline, as the loader role: the parse and
// the writes of the ordinary import, the rows sent by COPY, the library's
// change journal cut, and a run that resumes where it stopped.
package bulkload

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
)

// DefaultPasswordFile is where a container holds the loader's secret.
const DefaultPasswordFile = "/run/secrets/blunderdb_loader_password"

// DefaultMaxWorkers caps the default parsing goroutines: each holds a parsed
// match, and the writer behind them is the bottleneck anyway.
const DefaultMaxWorkers = 8

// DefaultMemoryLimitMiB is the Go runtime's soft memory limit of the CLI.
const DefaultMemoryLimitMiB = 4096

// MaxMemoryLimitMiB (1 PiB) keeps the limit in bytes, MiB << 20, far from int64 overflow.
const MaxMemoryLimitMiB = 1 << 30

// Options drive Load.
type Options struct {
	// DSN names the database and the loader role, without a password.
	DSN string
	// Password is the loader role's secret, read from a file by the CLI.
	Password string
	// Tenant is the library the load writes, above 0; the server's policy for
	// the loader role must be installed for the same tenant.
	Tenant int64
	// Source is a directory of .xg files or a .zip of them.
	Source string
	// Limit, when positive, loads only the first files in name order.
	Limit int
	// Batch resumes the import batch of that id; 0 opens a new one.
	Batch int64
	// ChunkFiles is how many archive members are unpacked at a time.
	ChunkFiles int
	// Workers and FilesPerTx tune the pipeline (0: DefaultMaxWorkers or one
	// per CPU if fewer, and the pipeline's default).
	Workers, FilesPerTx int
	// Unnest writes rows by the ordinary unnest statements instead of COPY,
	// to compare the two.
	Unnest bool
	// Progress receives the run's progress, once as soon as the batch is
	// known and at each committed group.
	Progress func(Report)
}

// Report is what a run did so far.
type Report struct {
	Batch      int64
	FilesTotal int
	FilesDone  int
	Imported   int
	Duplicates int
	Failed     int
	Positions  int
	// Errors are the first failures' reasons.
	Errors  []string
	Elapsed time.Duration
	ETA     time.Duration
}

// ConnString puts the password into the DSN for the pool only: it is never
// printed, and a DSN that already carries one is refused, since a command line
// shows in the process list and in shell histories.
func ConnString(dsn, password string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		return "", errors.New("bulkload: --dsn must be a postgres:// URL")
	}
	if _, has := u.User.Password(); has {
		return "", errors.New("bulkload: --dsn carries a password; give it by --password-file")
	}
	if u.User == nil || u.User.Username() != postgres.LoaderRole {
		return "", fmt.Errorf("bulkload: --dsn must name the role %s", postgres.LoaderRole)
	}
	if password == "" {
		return "", errors.New("bulkload: the loader password is empty")
	}
	u.User = url.UserPassword(postgres.LoaderRole, password)
	q := u.Query()
	q.Set("synchronous_commit", "off")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// Load runs the load to its end or to ctx's cancellation; what committed
// stays, and Batch resumes it.
func Load(ctx context.Context, opts Options) (Report, error) {
	conn, err := ConnString(opts.DSN, opts.Password)
	if err != nil {
		return Report{}, err
	}
	if opts.Tenant <= 0 {
		return Report{}, errors.New("bulkload: the tenant must be above 0")
	}
	// A library's scope is its tenant in decimal; the empty scope is tenant 0.
	scope := strconv.FormatInt(opts.Tenant, 10)
	ctx = storage.WithTenant(ctx, opts.Tenant)
	st, err := postgres.OpenLoader(ctx, conn, opts.Tenant)
	if err != nil {
		return Report{}, scrub(err, opts.Password)
	}
	defer st.Close()
	if opts.Unnest {
		st.WriteByUnnest()
	}
	if err := st.ChangeJournal().SetJournalOff(ctx, scope, true); err != nil {
		return Report{}, fmt.Errorf("bulkload: cut the public journal: %w", err)
	}

	src, err := openSource(opts.Source, opts.Limit)
	if err != nil {
		return Report{}, err
	}
	defer src.Close()

	batches := st.ImportBatches()
	rep := Report{FilesTotal: src.Len()}
	var counts domain.ImportReport
	var journal *ingest.Journal
	if opts.Batch != 0 {
		b, err := batches.Load(ctx, scope, opts.Batch)
		if err != nil {
			return rep, fmt.Errorf("bulkload: resume batch %d: %w", opts.Batch, err)
		}
		counts = b.Report
		counts.FilesFailed, counts.Failures = 0, nil
		if journal, err = ingest.LoadJournal(ctx, batches, scope, opts.Batch); err != nil {
			return rep, fmt.Errorf("bulkload: resume batch %d: %w", opts.Batch, err)
		}
		rep.Batch = opts.Batch
	} else if rep.Batch, err = batches.Begin(ctx, scope, "bulkload", "batch"); err != nil {
		return rep, fmt.Errorf("bulkload: open the batch: %w", err)
	}

	start := time.Now()
	emit := func() {
		rep.Elapsed = time.Since(start)
		rep.ETA = -1
		if rep.FilesDone > 0 && rep.FilesDone < rep.FilesTotal {
			rep.ETA = time.Duration(float64(rep.Elapsed) / float64(rep.FilesDone) * float64(rep.FilesTotal-rep.FilesDone))
		}
		if opts.Progress != nil {
			opts.Progress(rep)
		}
	}
	emit()
	workers := opts.Workers
	if workers <= 0 {
		workers = min(runtime.NumCPU(), DefaultMaxWorkers)
	}
	chunk := opts.ChunkFiles
	if chunk <= 0 {
		chunk = 2000
	}
	var runErr error
	for off := 0; off < src.Len() && runErr == nil; off += chunk {
		paths, cleanup, err := src.Stage(off, min(off+chunk, src.Len()))
		if err != nil {
			runErr = err
			break
		}
		popts := ingest.PipelineOptions{
			Scope:         scope,
			ImportBatchID: rep.Batch,
			Workers:       workers,
			FilesPerTx:    opts.FilesPerTx,
			OnCommit: func(group []ingest.FileOutcome) {
				for _, o := range group {
					rep.FilesDone++
					switch o.Status {
					case ingest.FileImported, ingest.FileEnriched:
						rep.Imported++
						rep.Positions += o.Positions
						counts.MatchesImported++
						counts.PositionsSaved += o.Positions
					case ingest.FileDuplicate:
						rep.Duplicates++
						if o.MatchID != 0 {
							counts.MatchesSkipped++
						}
					case ingest.FileFailed:
						rep.Failed++
						counts.FilesFailed++
						if len(rep.Errors) < 5 {
							rep.Errors = append(rep.Errors, o.Error)
						}
					}
				}
				emit()
			},
		}
		if journal != nil {
			popts.Known = journal.Known
		}
		_, runErr = ingest.ImportFiles(ctx, st, paths, popts)
		cleanup()
	}
	if ferr := batches.Finish(context.WithoutCancel(ctx), scope, rep.Batch, counts); ferr != nil && runErr == nil {
		runErr = fmt.Errorf("bulkload: close the batch: %w", ferr)
	}
	emit()
	return rep, scrub(runErr, opts.Password)
}

// scrub keeps the secret out of an error a driver may have built from the DSN.
func scrub(err error, password string) error {
	if err == nil || password == "" || !strings.Contains(err.Error(), password) {
		return err
	}
	return errors.New(strings.ReplaceAll(err.Error(), password, "***"))
}

// source is the ordered list of files to load, staged a slice at a time.
type source interface {
	Len() int
	Stage(from, to int) (paths []string, cleanup func(), err error)
	Close() error
}

func isMatchFile(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".xg")
}

func openSource(path string, limit int) (source, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("bulkload: source: %w", err)
	}
	if fi.IsDir() {
		var files []string
		err := filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && isMatchFile(p) {
				files = append(files, p)
			}
			return err
		})
		if err != nil {
			return nil, fmt.Errorf("bulkload: source: %w", err)
		}
		sort.Strings(files)
		return dirSource(cut(files, limit)), nil
	}
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("bulkload: source: %w", err)
	}
	var members []*zip.File
	for _, f := range zr.File {
		if !f.FileInfo().IsDir() && isMatchFile(f.Name) {
			members = append(members, f)
		}
	}
	sort.Slice(members, func(i, j int) bool { return members[i].Name < members[j].Name })
	return &zipSource{zr: zr, members: cut(members, limit)}, nil
}

func cut[T any](s []T, limit int) []T {
	if limit > 0 && limit < len(s) {
		return s[:limit]
	}
	return s
}

type dirSource []string

func (d dirSource) Len() int     { return len(d) }
func (d dirSource) Close() error { return nil }
func (d dirSource) Stage(from, to int) ([]string, func(), error) {
	return d[from:to], func() {}, nil
}

// zipSource unpacks a slice of members into a private directory, so the
// whole archive never sits on disk at once.
type zipSource struct {
	zr      *zip.ReadCloser
	members []*zip.File
}

func (z *zipSource) Len() int     { return len(z.members) }
func (z *zipSource) Close() error { return z.zr.Close() }
func (z *zipSource) Stage(from, to int) ([]string, func(), error) {
	dir, err := os.MkdirTemp("", "bulkload-")
	if err != nil {
		return nil, nil, fmt.Errorf("bulkload: stage: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	paths := make([]string, 0, to-from)
	for i, m := range z.members[from:to] {
		p := filepath.Join(dir, fmt.Sprintf("%07d.xg", from+i))
		if err := unpack(m, p); err != nil {
			cleanup()
			return nil, nil, err
		}
		paths = append(paths, p)
	}
	return paths, cleanup, nil
}

func unpack(m *zip.File, dst string) error {
	r, err := m.Open()
	if err != nil {
		return fmt.Errorf("bulkload: unpack: %w", err)
	}
	defer r.Close()
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("bulkload: unpack: %w", err)
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return fmt.Errorf("bulkload: unpack: %w", err)
	}
	return f.Close()
}
