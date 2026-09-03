package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// importRegistry tracks in-flight imports so imports.cancel can abort them.
// An id is opaque — 16 random bytes, hex — and says nothing about who
// started the import: the owning tenant is recorded beside the cancel func
// and checked by equality. The first design encoded the scope into the id
// ("<scope>-<counter>") and checked it by prefix, which let tenant "a"
// cancel the imports of tenant "a-1", and let anyone enumerate the counter
// (#160).
type importRegistry struct {
	mu   sync.Mutex
	jobs map[string]importJob
}

type importJob struct {
	scope  string
	cancel context.CancelFunc
}

func newImportRegistry() *importRegistry {
	return &importRegistry{jobs: make(map[string]importJob)}
}

// newImportID returns a fresh opaque id: 128 bits from crypto/rand, hex.
// crypto/rand.Read never fails on Go 1.24+ (it aborts the process rather
// than return a short read).
func newImportID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func (reg *importRegistry) start(scope string, cancel context.CancelFunc) string {
	id := newImportID()
	reg.mu.Lock()
	reg.jobs[id] = importJob{scope: scope, cancel: cancel}
	reg.mu.Unlock()
	return id
}

func (reg *importRegistry) finish(id string) {
	reg.mu.Lock()
	delete(reg.jobs, id)
	reg.mu.Unlock()
}

// cancel aborts the import with the given id if it belongs to scope. Returns
// false if no such in-flight import exists for this tenant — the same
// answer whether the id is unknown or another tenant's, so a caller learns
// nothing about other tenants' imports.
func (reg *importRegistry) cancel(scope, id string) bool {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	j, ok := reg.jobs[id]
	if !ok || j.scope != scope {
		return false
	}
	j.cancel()
	return true
}

// cancelAll aborts every in-flight job across every tenant, regardless of
// scope. Server.Run calls this on both registries just before Shutdown
// (#234): each job's own handler is watching its context and, once
// cancelled, emits a trailing {"event":"cancelled"} and returns on its own,
// so shutdown does not have to wait out the fixed ShutdownTimeout only to
// cut every remaining stream's connection with no explanation.
func (reg *importRegistry) cancelAll() {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	for _, j := range reg.jobs {
		j.cancel()
	}
}

// count returns the number of in-flight jobs, across every tenant — the
// blunderdb_imports_inflight / blunderdb_gammonnet_sweep_inflight gauges
// (#238), both backed by an importRegistry (see gammonnetJobs on Server).
func (reg *importRegistry) count() int {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	return len(reg.jobs)
}

// spoolQuota bounds the total bytes any in-flight import may hold spooled to
// $TMPDIR at once, across every tenant: with no ceiling, N concurrent
// imports each up to ImportMaxBodyBytes have no upper bound on disk usage
// (#234). A reservation claims the worst case (ImportMaxBodyBytes) up front
// rather than metering bytes as they arrive — simpler, and conservative in
// the direction that matters: the quota is never under-counted while an
// import is still spooling, only ever released once it (successfully or
// not) is done.
type spoolQuota struct {
	max      int64
	inFlight atomic.Int64
}

func newSpoolQuota(max int64) *spoolQuota { return &spoolQuota{max: max} }

// reserve claims n bytes of the quota, reverting and refusing if that would
// exceed max.
func (q *spoolQuota) reserve(n int64) bool {
	if q.inFlight.Add(n) > q.max {
		q.inFlight.Add(-n)
		return false
	}
	return true
}

func (q *spoolQuota) release(n int64) { q.inFlight.Add(-n) }

// usage returns the bytes currently reserved — blunderdb_import_spool_bytes
// (#238).
func (q *spoolQuota) usage() int64 { return q.inFlight.Load() }

// allowedUploadExtensions is the allow-list for the spool file's suffix,
// taken from the upload's filename. An extension outside it is dropped
// rather than the upload refused: the suffix is load-bearing only for the
// formats that dispatch on it (GnuBG's .sgf vs .mat/.txt, single-position's
// .xgp vs .txt — see ingest.MapGnuBG / ingest.PositionImporter) and purely
// cosmetic for the rest (the endpoint's fixed ingest.Format already says
// what to parse); either way, an attacker-controlled filename — arbitrary
// bytes, a path separator — must never reach os.CreateTemp's pattern
// unfiltered (#234).
var allowedUploadExtensions = map[string]bool{
	".xg": true, ".xgp": true, ".sgf": true, ".mat": true,
	".bgf": true, ".txt": true, ".db": true, ".dbx": true,
}

// sanitizeUploadExt returns ext lower-cased when it is on
// allowedUploadExtensions, or "" otherwise.
func sanitizeUploadExt(ext string) string {
	ext = strings.ToLower(ext)
	if allowedUploadExtensions[ext] {
		return ext
	}
	return ""
}

// importerFor returns the Importer for a format, or nil if unsupported on this
// server build. PR3a wires JSON; the parser formats land in PR3b/PR3c.
func (s *Server) importerFor(f ingest.Format) ingest.Importer {
	switch f {
	case ingest.FormatJSON:
		return ingest.JSONImporter{S: s.opts.Storage}
	case ingest.FormatXG:
		return ingest.XGImporter{S: s.opts.Storage}
	case ingest.FormatGnuBG:
		return ingest.GnuBGImporter{S: s.opts.Storage}
	case ingest.FormatBGF:
		return ingest.BGFImporter{S: s.opts.Storage}
	case ingest.FormatNativeDB:
		return ingest.DBImporter{S: s.opts.Storage}
	case ingest.FormatPosition:
		return ingest.PositionImporter{S: s.opts.Storage}
	default:
		return nil
	}
}

func (s *Server) exporterFor(f ingest.Format) ingest.Exporter {
	switch f {
	case ingest.FormatJSON, "":
		return ingest.JSONExporter{S: s.opts.Storage}
	case ingest.FormatSQLite:
		return ingest.SQLiteExporter{S: s.opts.Storage}
	default:
		return nil
	}
}

// uploadRoutes are the import endpoints whose body is a multipart file
// upload. One table drives both their registration (ingestRoutes) and their
// exemption from limitBody's default cap (uploadPaths): a new format can
// never be registered without its exemption, and no other route can inherit
// the exemption by sharing the "/v1/imports." prefix — imports.cancel carries
// a small JSON body and stays under the default cap like everything else
// (#160).
var uploadRoutes = []struct {
	pattern string
	format  ingest.Format
}{
	{"/v1/imports.json", ingest.FormatJSON},
	{"/v1/imports.xg", ingest.FormatXG},
	{"/v1/imports.gnubg", ingest.FormatGnuBG},
	{"/v1/imports.bgf", ingest.FormatBGF},
	{"/v1/imports.db", ingest.FormatNativeDB},
	{"/v1/imports.position", ingest.FormatPosition},
}

// uploadPaths is the exact set of routes limitBody leaves to handleImport's
// own (larger) cap.
func uploadPaths() map[string]bool {
	m := make(map[string]bool, len(uploadRoutes))
	for _, u := range uploadRoutes {
		m[u.pattern] = true
	}
	return m
}

// ingestRoutes registers the import/export endpoints supported by this build.
func (s *Server) ingestRoutes() []route {
	rs := make([]route, 0, len(uploadRoutes)+3)
	for _, u := range uploadRoutes {
		rs = append(rs, route{http.MethodPost, u.pattern, s.handleImport(u.format)})
	}
	return append(rs,
		route{http.MethodPost, "/v1/imports.cancel", s.handleImportCancel},
		route{http.MethodPost, "/v1/exports.json", s.handleExport(ingest.FormatJSON)},
		route{http.MethodPost, "/v1/exports.sqlite", s.handleExportSQLite()},
	)
}

// exportSQLiteReq optionally asks for a watermark on the export — origin is
// free text (who this copy is for, or why it left), mirroring the desktop's
// export dialog and the CLI's --watermark/--watermark-note. An empty (or
// absent) origin means the export carries none.
type exportSQLiteReq struct {
	WatermarkOrigin string `json:"watermarkOrigin"`
	WatermarkNote   string `json:"watermarkNote"`
}

// sealExportWatermark seals a watermark for origin/note with this daemon's own
// signing identity (Options.Identity — see its doc comment), or returns ""
// when origin is empty (no watermark asked for). A watermark asked for
// without a configured identity is a CodeInvalid error, not a silently
// unmarked export.
func (s *Server) sealExportWatermark(origin, note string) (string, error) {
	if strings.TrimSpace(origin) == "" {
		return "", nil
	}
	if s.opts.Identity == nil {
		return "", fmt.Errorf("%w: watermarkOrigin was given but this server has no signing identity (start it with --identity-dir)", storage.ErrInvalid)
	}
	return ingest.SealWatermark(s.opts.Identity, origin, note)
}

// handleExportSQLite serializes the caller's whole tenant — every position,
// collection, match and tournament, with analyses, comments, played moves,
// the filter library and Anki decks — into a blunderDB SQLite file and
// returns it as a binary download. An optional JSON body asks for a
// watermark (see exportSQLiteReq); everything else about the export is
// WholeTenant, matching the GUI/CLI's "export everything" preset — a
// selective server-side export is not offered yet.
//
// ingest.SQLiteExporter already materializes the whole file into its own temp
// path before copying it to the writer it is given, but this handler must not
// hand it the live ResponseWriter directly: the moment Export writes its first
// byte, Go commits whatever status/headers are set at that instant, and this
// handler used to set the binary headers *before* calling Export — so a
// mid-export failure still went out as HTTP 200 with
// Content-Type: application/octet-stream and a JSON error body wearing a
// ".sqlite" Content-Disposition. Instead, Export targets a private temp file
// here; the binary headers are set, and the file streamed to the client, only
// once Export has returned successfully. On failure, writeStorageError sets a
// proper status/content-type — nothing has reached the client yet.
func (s *Server) handleExportSQLite() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req exportSQLiteReq
		if err := decodeJSON(r, &req); err != nil {
			writeDecodeError(w, "invalid JSON body", err)
			return
		}
		watermark, err := s.sealExportWatermark(req.WatermarkOrigin, req.WatermarkNote)
		if err != nil {
			writeStorageError(w, err)
			return
		}

		exp := s.exporterFor(ingest.FormatSQLite)

		tmp, tmpErr := os.CreateTemp("", "blunderdb-export-resp-*.sqlite")
		if tmpErr != nil {
			writeStorageError(w, fmt.Errorf("server: create temp export file: %w", tmpErr))
			return
		}
		tmpPath := tmp.Name()
		defer os.Remove(tmpPath)

		opts := ingest.WholeTenant(ingest.FormatSQLite)
		opts.Watermark = watermark
		exportErr := exp.Export(r.Context(), scopeOf(r), tmp, opts)
		closeErr := tmp.Close()
		if exportErr != nil {
			writeStorageError(w, exportErr)
			return
		}
		if closeErr != nil {
			writeStorageError(w, fmt.Errorf("server: close temp export file: %w", closeErr))
			return
		}

		f, err := os.Open(tmpPath)
		if err != nil {
			writeStorageError(w, fmt.Errorf("server: reopen temp export file: %w", err))
			return
		}
		defer f.Close()

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="blunderdb-export.sqlite"`)
		if _, err := io.Copy(w, f); err != nil {
			// Headers/status are already committed at this point (the copy is
			// the first write); nothing more to do than let the client see a
			// truncated download with no error field to explain it. The log
			// line is the only place this failure is ever recorded, which is
			// exactly the Error case (see logging.go's scale): nobody else
			// is going to see it.
			slog.Error("server: stream sqlite export", "err", err)
		}
	}
}

// handleImport streams an uploaded file through the importer, emitting NDJSON
// progress events. The upload is spooled to a temp file so parser-backed
// formats (PR3b/c) can seek; JSON reads it back sequentially.
func (s *Server) handleImport(format ingest.Format) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		imp := s.importerFor(format)
		if imp == nil {
			writeErrorCode(w, CodeInvalid, "import format not supported on this server: "+string(format))
			return
		}

		// Claim the worst case (ImportMaxBodyBytes) against the global spool
		// quota before touching the body at all: N concurrent imports each up
		// to ImportMaxBodyBytes would otherwise have no ceiling on
		// $TMPDIR usage (#234).
		if !s.spool.reserve(s.opts.ImportMaxBodyBytes) {
			writeErrorCode(w, CodeRateLimited, "too many imports in flight, try again shortly")
			return
		}
		defer s.spool.release(s.opts.ImportMaxBodyBytes)

		r.Body = http.MaxBytesReader(w, r.Body, s.opts.ImportMaxBodyBytes)
		file, header, err := r.FormFile("file")
		if err != nil {
			writeDecodeError(w, "missing multipart 'file' field", err)
			return
		}
		defer file.Close()

		// Preserve the upload's extension on the spool file when it is on
		// allowedUploadExtensions: parser-backed formats dispatch on it
		// (e.g. GnuBG .sgf vs .mat) — anything else is dropped rather than
		// letting an attacker-controlled filename reach the temp name
		// unfiltered (#234).
		ext := ""
		if header != nil {
			ext = sanitizeUploadExt(filepath.Ext(header.Filename))
		}
		tmpPath, cleanup, err := spoolToTemp(file, ext)
		if err != nil {
			writeStorageError(w, err)
			return
		}
		defer cleanup()

		// A cancellable context, registered so imports.cancel can abort it —
		// and so can Server.Run, on every in-flight import, just before
		// Shutdown (#234).
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()
		scope := scopeOf(r)
		importID := s.imports.start(scope, cancel)
		defer s.imports.finish(importID)

		w.Header().Set("Content-Type", ndjsonContentType)
		w.WriteHeader(http.StatusOK)
		enc := json.NewEncoder(w)
		fl, _ := w.(http.Flusher)
		emit := func(v any) {
			_ = enc.Encode(v)
			if fl != nil {
				fl.Flush()
			}
		}

		emit(map[string]any{"event": "started", "import_id": importID})

		prog := func(p ingest.Progress) {
			emit(map[string]any{
				"event":     "progress",
				"matches":   p.Matches,
				"games":     p.Games,
				"positions": p.Positions,
			})
		}

		sum, err := imp.Import(ctx, scope, ingest.Source{Format: format, Path: tmpPath}, prog)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				// imports.cancel, or a graceful shutdown cancelling every
				// in-flight job (Server.Run) — either way the client asked
				// for or was told about this, not a failure (#234).
				emit(map[string]any{"event": "cancelled"})
				return
			}
			emit(map[string]any{"event": "error", "error": errorBodyFor(w, err)})
			return
		}
		emit(map[string]any{
			"event":              "done",
			"saved_positions":    sum.SavedPositions,
			"skipped_duplicates": sum.SkippedDuplicates,
			"matches":            sum.Matches,
			"match_id":           sum.MatchID,
		})
	}
}

type importCancelReq struct {
	ImportID string `json:"importId"`
}

func (s *Server) handleImportCancel(w http.ResponseWriter, r *http.Request) {
	var req importCancelReq
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, "invalid JSON body", err)
		return
	}
	if req.ImportID == "" {
		writeErrorCode(w, CodeInvalid, "importId is required")
		return
	}
	if !s.imports.cancel(scopeOf(r), req.ImportID) {
		writeErrorCode(w, CodeNotFound, "no in-flight import with that id")
		return
	}
	writeJSONResp(w, okResp{OK: true})
}

// handleExport streams stored data out in the given format.
func (s *Server) handleExport(format ingest.Format) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exp := s.exporterFor(format)
		if exp == nil {
			writeErrorCode(w, CodeInvalid, "export format not supported on this server: "+string(format))
			return
		}
		w.Header().Set("Content-Type", ndjsonContentType)
		// Status defaults to 200 on first write; a pre-stream failure cannot be
		// re-statused once bytes are sent, so this mirrors streamSeq2.
		if err := exp.Export(r.Context(), scopeOf(r), w, ingest.ExportOptions{Format: format}); err != nil {
			// Best-effort trailing error line; header may already be 200.
			_ = json.NewEncoder(w).Encode(errorEnvelope{Error: errorBodyFor(w, err)})
		}
	}
}

// spoolToTemp copies r to a temporary file and returns its path plus a cleanup
// func that removes it.
func spoolToTemp(r io.Reader, ext string) (string, func(), error) {
	f, err := os.CreateTemp("", "blunderdb-import-*"+ext)
	if err != nil {
		return "", func() {}, fmt.Errorf("server: temp file: %w", err)
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		os.Remove(f.Name()) //nolint:gosec // G703: ext (part of the CreateTemp pattern above) is allowlisted by sanitizeUploadExt before it ever reaches here (#234) — never a path separator
		return "", func() {}, fmt.Errorf("server: spool upload: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name()) //nolint:gosec // G703: same allowlisted ext as above
		return "", func() {}, fmt.Errorf("server: spool close: %w", err)
	}
	path := f.Name()
	return path, func() { os.Remove(path) }, nil //nolint:gosec // G703: same allowlisted ext as above
}
