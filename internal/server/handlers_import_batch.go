package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// imports.batch brings in a whole corpus: an uploaded .zip or .tar archive, or
// a directory of the daemon's own host that the operator opened with
// --import-dir. The files go through the same pipeline as the CLI's folder
// import (ingest.ImportFiles), so the same folder gives the same matches.
//
// The call answers with an import id as soon as the files are known and the
// import runs on in the background: a corpus takes hours and no HTTP request
// should be held that long. imports.batch.status reads the progress the
// pipeline's meter measures, imports.batch.cancel stops it. The groups already
// committed stay, as for an interrupted CLI import.

const (
	batchStateReceiving = "receiving"
	batchStateRunning   = "running"
	batchStateDone      = "done"
	batchStateCancelled = "cancelled"
	batchStateFailed    = "failed"
)

// batchErrorWindow bounds the file errors a status keeps: a corpus of a
// hundred thousand files may refuse thousands, and the answer to a poll must
// not grow with them. The count is exact, the list is the first ones.
const batchErrorWindow = 100

// batchKeepFinished is how long a finished job stays readable.
const batchKeepFinished = time.Hour

// batchExtractFactor sets the unpacked-size budget of an archive, in multiples
// of ImportMaxBodyBytes: match files compress by a few times, and the budget
// is what keeps a small archive from filling the disk.
const batchExtractFactor = 2

type batchFileError struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

// batchJob is one batch import. All its mutable fields sit behind mu.
type batchJob struct {
	id     string
	scope  string
	key    string // the caller's Idempotency-Key, "" when none
	cancel context.CancelFunc
	// done is closed when the job has ended, whatever the way.
	done chan struct{}
	// skipDuplicates is the request's ?skip_duplicates: an exact duplicate
	// is skipped outright instead of offering its deeper analyses.
	skipDuplicates bool

	mu       sync.Mutex
	state    string
	progress ingest.BatchProgress
	batchID  int64
	errors   []batchFileError
	failed   int
	err      string
	finished time.Time
}

// batchStatus is what imports.batch.status answers.
type batchStatus struct {
	ImportID string               `json:"importId"`
	State    string               `json:"state"`
	BatchID  int64                `json:"batchId,omitempty"`
	Progress ingest.BatchProgress `json:"progress"`
	// Errors holds the first files the batch refused, ErrorsTotal how many it
	// refused in all.
	Errors      []batchFileError `json:"errors,omitempty"`
	ErrorsTotal int              `json:"errorsTotal"`
	Error       string           `json:"error,omitempty"`
}

func (j *batchJob) status() *batchStatus {
	j.mu.Lock()
	defer j.mu.Unlock()
	return &batchStatus{
		ImportID:    j.id,
		State:       j.state,
		BatchID:     j.batchID,
		Progress:    j.progress,
		Errors:      append([]batchFileError(nil), j.errors...),
		ErrorsTotal: j.failed,
		Error:       j.err,
	}
}

func (j *batchJob) setState(state, errMsg string) {
	j.mu.Lock()
	j.state, j.err = state, errMsg
	if state != batchStateRunning && state != batchStateReceiving {
		j.finished = time.Now()
	}
	j.mu.Unlock()
}

// batchRegistry holds the batch jobs by id, and by (tenant, Idempotency-Key)
// so a retried call finds the job its first attempt started.
type batchRegistry struct {
	mu    sync.Mutex
	jobs  map[string]*batchJob
	byKey map[string]string
}

func newBatchRegistry() *batchRegistry {
	return &batchRegistry{jobs: make(map[string]*batchJob), byKey: make(map[string]string)}
}

func batchKey(scope, key string) string { return scope + "\x00" + key }

// open registers a job for scope. When key is already the key of a live job of
// that scope it returns that job and false: the call is a replay.
func (reg *batchRegistry) open(scope, key string) (*batchJob, bool) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	reg.sweepLocked()
	if key != "" {
		if id, ok := reg.byKey[batchKey(scope, key)]; ok {
			if j, ok := reg.jobs[id]; ok {
				return j, false
			}
		}
	}
	j := &batchJob{id: newImportID(), scope: scope, key: key, state: batchStateReceiving, done: make(chan struct{})}
	reg.jobs[j.id] = j
	if key != "" {
		reg.byKey[batchKey(scope, key)] = j.id
	}
	return j, true
}

// drop forgets a job that never started, key included: only an attempt that
// went through is remembered, so a retry after a refusal is a new attempt.
func (reg *batchRegistry) drop(j *batchJob) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	delete(reg.jobs, j.id)
	if j.key != "" {
		delete(reg.byKey, batchKey(j.scope, j.key))
	}
}

// get returns a job of scope; another tenant's id answers like an unknown one.
func (reg *batchRegistry) get(scope, id string) (*batchJob, bool) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	reg.sweepLocked()
	j, ok := reg.jobs[id]
	if !ok || j.scope != scope {
		return nil, false
	}
	return j, true
}

func (reg *batchRegistry) sweepLocked() {
	for id, j := range reg.jobs {
		j.mu.Lock()
		old := !j.finished.IsZero() && time.Since(j.finished) > batchKeepFinished
		j.mu.Unlock()
		if old {
			delete(reg.jobs, id)
			if j.key != "" {
				delete(reg.byKey, batchKey(j.scope, j.key))
			}
		}
	}
}

type importBatchPathReq struct {
	// Path names a directory of the daemon's host: absolute inside ImportDir,
	// or relative to it.
	Path      string `json:"path"`
	Recursive *bool  `json:"recursive,omitempty"`
	// Resume names an earlier batch (its batchId) to continue: the files its
	// journal already decided are skipped.
	Resume int64 `json:"resume,omitempty"`
}

type importBatchResp struct {
	ImportID string `json:"importId"`
	Files    int    `json:"files"`
}

type importBatchIDReq struct {
	ImportID string `json:"importId"`
}

// resolveBatchDir turns a requested path into a directory inside ImportDir,
// symbolic links resolved on both sides so a link cannot lead out.
func (s *Server) resolveBatchDir(p string) (string, error) {
	root := s.opts.ImportDir
	if root == "" {
		return "", fmt.Errorf("%w: importing from a path is off on this server (start it with --import-dir)", storage.ErrInvalid)
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("server: import directory: %w", err)
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(realRoot, p)
	}
	real, err := filepath.EvalSymlinks(p)
	if err != nil {
		return "", fmt.Errorf("%w: no such directory", storage.ErrInvalid)
	}
	rel, err := filepath.Rel(realRoot, real)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: the path is outside the import directory", storage.ErrInvalid)
	}
	if fi, err := os.Stat(real); err != nil || !fi.IsDir() {
		return "", fmt.Errorf("%w: the path is not a directory", storage.ErrInvalid)
	}
	return real, nil
}

func (s *Server) handleImportBatch(w http.ResponseWriter, r *http.Request) {
	scope := scopeOf(r)
	job, fresh := s.batches.open(scope, r.Header.Get(IdempotencyKeyHeader))
	if !fresh {
		w.Header().Set(idempotencyReplayedHeader, "true")
		writeBatchAccepted(w, job)
		return
	}
	started := false
	defer func() {
		if !started {
			s.batches.drop(job)
		}
	}()

	if s.refuseImport(r.Context(), w, scope) {
		return
	}
	endQuota := true
	defer func() {
		if endQuota {
			s.quota.endImport(scope)
		}
	}()

	var (
		root, workDir string
		paths         []string
		label         string
		reserved      int64
		resume        int64
	)
	release := func() {
		if reserved > 0 {
			s.spool.release(reserved)
		}
		if workDir != "" {
			os.RemoveAll(workDir)
		}
	}
	defer func() {
		if !started {
			release()
		}
	}()

	mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if mt == "multipart/form-data" {
		reserved = s.opts.ImportMaxBodyBytes * (1 + batchExtractFactor)
		if !s.spool.reserve(reserved) {
			reserved = 0
			writeErrorCode(w, CodeRateLimited, "too many imports in flight, try again shortly")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, s.opts.ImportMaxBodyBytes)
		file, header, err := r.FormFile("file")
		if err != nil {
			writeDecodeError(w, "missing multipart 'file' field", err)
			return
		}
		defer file.Close()
		ext := strings.ToLower(filepath.Ext(header.Filename))
		if ext != ".zip" && ext != ".tar" {
			writeErrorCode(w, CodeInvalid, "the archive must be a .zip or a .tar file")
			return
		}
		if workDir, err = os.MkdirTemp("", "blunderdb-batch-*"); err != nil {
			writeStorageError(w, fmt.Errorf("server: temp dir: %w", err))
			return
		}
		archive := filepath.Join(workDir, "archive"+ext)
		if err := copyToFile(archive, file); err != nil {
			writeStorageError(w, err)
			return
		}
		root = filepath.Join(workDir, "files")
		n, err := ingest.ExtractArchive(archive, root, s.opts.ImportMaxBodyBytes*batchExtractFactor)
		os.Remove(archive) //nolint:gosec // the spool file this handler wrote under its own temp dir
		if err != nil {
			if errors.Is(err, ingest.ErrArchiveTooLarge) || errors.Is(err, ingest.ErrArchiveTooManyEntries) {
				writeErrorCode(w, CodeInvalid, err.Error())
				return
			}
			writeErrorCode(w, CodeInvalid, "unreadable archive: "+err.Error())
			return
		}
		if n == 0 {
			writeErrorCode(w, CodeInvalid, "no match file in the archive ("+strings.Join(ingest.ImportableExtensions(), ", ")+")")
			return
		}
		label = filepath.Base(header.Filename)
		if v := r.FormValue("resume"); v != "" {
			if resume, err = strconv.ParseInt(v, 10, 64); err != nil || resume <= 0 {
				writeErrorCode(w, CodeInvalid, "resume must be a batch id")
				return
			}
		}
	} else {
		var req importBatchPathReq
		if err := decodeJSON(r, &req); err != nil {
			writeDecodeError(w, "invalid JSON body", err)
			return
		}
		dir, err := s.resolveBatchDir(req.Path)
		if err != nil {
			writeStorageError(w, err)
			return
		}
		root, resume = dir, req.Resume
		label = filepath.Base(dir)
		recursive := req.Recursive == nil || *req.Recursive
		files, err := ingest.CollectFiles(root, recursive)
		if err != nil {
			writeStorageError(w, fmt.Errorf("server: scan directory: %w", err))
			return
		}
		paths = files
	}
	if paths == nil {
		var err error
		if paths, err = ingest.CollectFiles(root, true); err != nil {
			writeStorageError(w, fmt.Errorf("server: scan archive: %w", err))
			return
		}
	}
	if len(paths) == 0 {
		writeErrorCode(w, CodeInvalid, "no match file found ("+strings.Join(ingest.ImportableExtensions(), ", ")+")")
		return
	}

	var journal *ingest.Journal
	if resume != 0 {
		var err error
		if journal, err = ingest.LoadJournal(r.Context(), s.opts.Storage.ImportBatches(), scope, resume); err != nil {
			writeStorageError(w, err)
			return
		}
		paths, _ = journal.Pending(paths)
	}

	var size int64
	for _, p := range paths {
		if fi, err := os.Stat(p); err == nil {
			size += fi.Size()
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.imports.register(job.id, scope, cancel)
	job.mu.Lock()
	job.skipDuplicates = skipDuplicatesParam(r)
	job.cancel = cancel
	job.state = batchStateRunning
	job.progress = ingest.BatchProgress{FilesTotal: len(paths), BytesTotal: size, ETASeconds: -1}
	job.mu.Unlock()
	started, endQuota = true, false
	go func() {
		defer close(job.done)
		defer cancel()
		defer s.imports.finish(job.id)
		defer s.quota.endImport(scope)
		defer release()
		s.runBatch(ctx, job, root, paths, label, resume, journal)
	}()

	writeBatchAccepted(w, job)
}

func writeBatchAccepted(w http.ResponseWriter, job *batchJob) {
	job.mu.Lock()
	total := job.progress.FilesTotal
	job.mu.Unlock()
	writeJSONResp(w, importBatchResp{ImportID: job.id, Files: total})
}

func copyToFile(path string, r io.Reader) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) //nolint:gosec // a path under the handler's own temp dir
	if err != nil {
		return fmt.Errorf("server: spool archive: %w", err)
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return fmt.Errorf("server: spool archive: %w", err)
	}
	return f.Close()
}

// runBatch is the job's goroutine: it feeds the paths to the pipeline, keeps
// the job's progress and error list current, and closes the import batch with
// the counts of what was written.
func (s *Server) runBatch(ctx context.Context, job *batchJob, root string, paths []string, label string, resume int64, journal *ingest.Journal) {
	job.mu.Lock()
	total := job.progress.BytesTotal
	job.mu.Unlock()
	meter := ingest.NewProgressMeter(len(paths), total, ingest.ProgressInterval, func(p ingest.BatchProgress) {
		job.mu.Lock()
		job.progress = p
		job.mu.Unlock()
	})
	scope := job.scope
	batches := s.opts.Storage.ImportBatches()
	var (
		batchID int64
		counts  domain.ImportReport
		err     error
	)
	if journal != nil {
		// A resumed batch goes on from its stored counts; the files that
		// failed before are tried again and counted again if they fail again.
		batchID = resume
		if b, lerr := batches.Load(ctx, scope, resume); lerr == nil {
			counts = b.Report
			counts.FilesFailed, counts.Failures = 0, nil
		}
	} else if batchID, err = batches.Begin(ctx, scope, label, "batch"); err != nil {
		slog.Warn("import batch: opening the batch failed", "err", err)
		batchID = 0
	}
	job.mu.Lock()
	job.batchID = batchID
	job.mu.Unlock()

	popts := ingest.PipelineOptions{
		Scope:          scope,
		ImportBatchID:  batchID,
		SkipDuplicates: job.skipDuplicates,
		OnRead:         meter.Read,
		OnCommit: func(group []ingest.FileOutcome) {
			ingest.RecordOutcomes(context.Background(), batches, scope, batchID, group)
			for _, o := range group {
				meter.File(o)
				switch o.Status {
				case ingest.FileImported:
					counts.MatchesImported++
				case ingest.FileEnriched:
					counts.MatchesEnriched++
				case ingest.FileDuplicate:
					if o.MatchID != 0 {
						counts.MatchesSkipped++
					}
					if o.Deepened > 0 {
						counts.MatchesDeepened++
						counts.AnalysesDeepened += o.Deepened
					}
				}
				if o.Status == ingest.FileImported || o.Status == ingest.FileEnriched {
					counts.PositionsSaved += o.Positions
				}
				if o.ProbableDuplicate != nil {
					counts.ProbableDuplicates = append(counts.ProbableDuplicates, *o.ProbableDuplicate)
				}
				if o.Status == ingest.FileFailed {
					rel, rerr := filepath.Rel(root, o.Path)
					if rerr != nil {
						rel = filepath.Base(o.Path)
					}
					counts.FilesFailed++
					if len(counts.Failures) < batchErrorWindow {
						counts.Failures = append(counts.Failures, domain.ImportFailure{Source: rel, Reason: o.Error})
					}
					job.mu.Lock()
					job.failed++
					if len(job.errors) < batchErrorWindow {
						job.errors = append(job.errors, batchFileError{Path: rel, Error: o.Error})
					}
					job.mu.Unlock()
				}
			}
		},
	}
	if journal != nil {
		popts.Known = journal.Known
	}
	_, err = ingest.ImportFiles(ctx, s.opts.Storage, paths, popts)
	final := meter.Finish()
	job.mu.Lock()
	job.progress = final
	job.mu.Unlock()

	if batchID != 0 {
		// Whatever the end, the groups committed are in the database: the
		// batch's report must say so. A fresh context, the job's is over.
		if ferr := batches.Finish(context.Background(), scope, batchID, counts); ferr != nil {
			slog.Warn("import batch: closing the batch failed", "err", ferr)
		}
	}
	switch {
	case err == nil:
		job.setState(batchStateDone, "")
	case errors.Is(err, context.Canceled):
		job.setState(batchStateCancelled, "")
	default:
		slog.Error("import batch failed", "err", err)
		job.setState(batchStateFailed, err.Error())
	}
}

func (s *Server) batchRoutes() []route {
	return []route{
		{http.MethodPost, "/v1/imports.batch", s.handleImportBatch},
		{http.MethodPost, "/v1/imports.batch.status", rpc(func(ctx context.Context, scope string, req importBatchIDReq) (*batchStatus, error) {
			job, ok := s.batches.get(scope, req.ImportID)
			if !ok {
				return nil, fmt.Errorf("%w: no batch import with that id", storage.ErrNotFound)
			}
			return job.status(), nil
		})},
		{http.MethodPost, "/v1/imports.batch.cancel", rpc(func(ctx context.Context, scope string, req importBatchIDReq) (okResp, error) {
			job, ok := s.batches.get(scope, req.ImportID)
			if !ok {
				return okResp{}, fmt.Errorf("%w: no batch import with that id", storage.ErrNotFound)
			}
			job.mu.Lock()
			cancel, running := job.cancel, job.state == batchStateRunning
			job.mu.Unlock()
			if running && cancel != nil {
				cancel()
			}
			return okResp{OK: true}, nil
		})},
	}
}

// skipDuplicatesParam reads ?skip_duplicates, the same switch on every import
// route: true makes an exact duplicate a plain skip, without offering its
// deeper analyses to the stored positions.
func skipDuplicatesParam(r *http.Request) bool {
	v, _ := strconv.ParseBool(r.URL.Query().Get("skip_duplicates"))
	return v
}
