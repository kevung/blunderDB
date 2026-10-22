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
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// importRegistry tracks in-flight imports so imports.cancel can abort them.
// An id is opaque — 16 random bytes, hex — and says nothing about who
// started the import: the owning tenant is recorded beside the cancel func
// and checked by equality, never encoded into the id. Encoding the scope
// into the id ("<scope>-<counter>") and matching by prefix let tenant "a"
// cancel tenant "a-1"'s imports, and let anyone enumerate the counter.
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
	reg.register(id, scope, cancel)
	return id
}

// register records a job under an id the caller already chose.
func (reg *importRegistry) register(id, scope string, cancel context.CancelFunc) {
	reg.mu.Lock()
	reg.jobs[id] = importJob{scope: scope, cancel: cancel}
	reg.mu.Unlock()
}

// startExclusive is start for a job a tenant may only have one of at a time.
// It returns ErrConflict rather than a second id when one is already running
// for that scope: the gammonNet sweep is the case that matters, since it
// takes every core the daemon has for as long as it runs, and two runs for
// the same tenant do not go twice as fast — they halve each other while both
// write analyses into the rows the other is reading as missing.
func (reg *importRegistry) startExclusive(scope string, cancel context.CancelFunc) (string, error) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	for _, j := range reg.jobs {
		if j.scope == scope {
			return "", fmt.Errorf("%w: an analysis, a rollout or a maintenance operation is already running for this tenant", storage.ErrConflict)
		}
	}
	id := newImportID()
	reg.jobs[id] = importJob{scope: scope, cancel: cancel}
	return id, nil
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
// scope. Server.Run calls this on both registries just before Shutdown: each
// job's handler watches its context and, once cancelled, emits a trailing
// {"event":"cancelled"} and returns on its own, so shutdown need not wait out
// the fixed ShutdownTimeout only to cut every remaining stream with no
// explanation.
func (reg *importRegistry) cancelAll() {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	for _, j := range reg.jobs {
		j.cancel()
	}
}

// count returns the number of in-flight jobs, across every tenant — backs the
// blunderdb_imports_inflight / blunderdb_gammonnet_sweep_inflight gauges (see
// gammonnetJobs on Server).
func (reg *importRegistry) count() int {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	return len(reg.jobs)
}

// spoolQuota bounds the total bytes any in-flight import may hold spooled to
// $TMPDIR at once, across every tenant: with no ceiling, N concurrent imports
// each up to ImportMaxBodyBytes would have no upper bound on disk usage. A
// reservation claims the worst case up front rather than metering bytes as
// they arrive: the quota is never under-counted while an import is still
// spooling, only released once it (successfully or not) is done.
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

// usage returns the bytes currently reserved — backs blunderdb_import_spool_bytes.
func (q *spoolQuota) usage() int64 { return q.inFlight.Load() }

// allowedUploadExtensions is the allow-list for the spool file's suffix,
// taken from the upload's filename. An extension outside it is dropped
// rather than the upload refused: the suffix is load-bearing only for
// formats that dispatch on it (GnuBG's .sgf vs .mat/.txt, single-position's
// .xgp vs .txt — see ingest.MapGnuBG / ingest.PositionImporter) and purely
// cosmetic for the rest. Either way, an attacker-controlled filename —
// arbitrary bytes, a path separator — must never reach os.CreateTemp's
// pattern unfiltered.
var allowedUploadExtensions = map[string]bool{
	".xg": true, ".xgp": true, ".sgf": true, ".mat": true,
	".bgf": true, ".ogxm": true, ".txt": true, ".db": true, ".dbx": true,
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
	case ingest.FormatOGXM:
		return ingest.OGXMImporter{S: s.opts.Storage}
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
	default:
		return nil
	}
}

// uploadRoutes are the import endpoints whose body is a multipart file
// upload. One table drives both their registration (ingestRoutes) and their
// exemption from limitBody's default cap (uploadPaths): a new format can
// never be registered without its exemption, and no other route can inherit
// the exemption by sharing the "/v1/imports." prefix — imports.cancel carries
// a small JSON body and stays under the default cap like everything else.
var uploadRoutes = []struct {
	pattern string
	format  ingest.Format
}{
	{"/v1/imports.json", ingest.FormatJSON},
	{"/v1/imports.xg", ingest.FormatXG},
	{"/v1/imports.gnubg", ingest.FormatGnuBG},
	{"/v1/imports.bgf", ingest.FormatBGF},
	{"/v1/imports.ogxm", ingest.FormatOGXM},
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
	m["/v1/imports.batch"] = true
	m["/v1/matches.repair"] = true
	return m
}

// ingestRoutes registers the import/export endpoints supported by this build.
func (s *Server) ingestRoutes() []route {
	rs := make([]route, 0, len(uploadRoutes)+3)
	for _, u := range uploadRoutes {
		rs = append(rs, route{http.MethodPost, u.pattern, s.handleImport(u.format)})
	}
	rs = append(rs, s.batchRoutes()...)
	return append(rs,
		route{http.MethodPost, "/v1/imports.cancel", s.handleImportCancel},
		// Recomputed on every call, so a client that re-analyses the positions
		// an import brought in and asks again gets the new figures.
		route{http.MethodPost, "/v1/imports.report", rpc(func(ctx context.Context, scope string, req importReportReq) (*domain.ImportBatch, error) {
			return s.opts.Storage.ImportBatches().Report(ctx, scope, req.BatchID, req.Players)
		})},
		// What to look at now, in the order to look at it, following the
		// report above. Measured on every call and never stored — running the
		// same queue again is a legitimate thing to want.
		route{http.MethodPost, "/v1/imports.studyQueue", rpc(func(ctx context.Context, scope string, req importStudyQueueReq) ([]domain.StudyQueueEntry, error) {
			return s.opts.Storage.ImportBatches().StudyQueue(ctx, scope, req.BatchID, req.Players, req.Limit)
		})},
		// The library-wide backlog of unhandled blunders, and the user's
		// "studied" mark that takes a position out of it. The mark is written
		// only by this call (ADR-0007) and is never exported.
		route{http.MethodPost, "/v1/study.backlog", rpc(func(ctx context.Context, scope string, req importStudyQueueReq) ([]domain.StudyQueueEntry, error) {
			return s.opts.Storage.ImportBatches().StudyBacklog(ctx, scope, req.Players, req.Limit)
		})},
		route{http.MethodPost, "/v1/study.setStudied", rpcVoid(func(ctx context.Context, scope string, req studySetStudiedReq) error {
			return s.opts.Storage.ImportBatches().SetStudied(ctx, scope, req.PositionID, req.Studied)
		})},
		route{http.MethodPost, "/v1/study.isStudied", rpc(func(ctx context.Context, scope string, req studyIsStudiedReq) (bool, error) {
			at, err := s.opts.Storage.ImportBatches().StudyMark(ctx, scope, req.PositionID)
			return at != 0, err
		})},
		// The batch's per-file journal: which file gave which match.
		route{http.MethodPost, "/v1/imports.files", rpc(func(ctx context.Context, scope string, req importReportReq) ([]domain.ImportFileEntry, error) {
			return s.opts.Storage.ImportBatches().Files(ctx, scope, req.BatchID)
		})},
		route{http.MethodPost, "/v1/imports.list", rpc(func(ctx context.Context, scope string, req listReq) ([]*domain.ImportBatch, error) {
			return s.opts.Storage.ImportBatches().List(ctx, scope, storage.ListOpts{Limit: req.Limit, Offset: req.Offset})
		})},
		route{http.MethodPost, "/v1/exports.json", s.handleExport(ingest.FormatJSON)},
		route{http.MethodPost, "/v1/exports.sqlite", s.handleExportSQLite()},
		// What exports.sqlite would write for the same body, counted family by family
		// without writing anything; the watermark fields are ignored.
		route{http.MethodPost, "/v1/exports.plan", rpc(func(ctx context.Context, scope string, req exportSQLiteReq) (ingest.ExportPlan, error) {
			opts, err := exportSQLiteOptions(ctx, s.opts.Storage, scope, req)
			if err != nil {
				return ingest.ExportPlan{}, err
			}
			return ingest.PlanExport(ctx, s.opts.Storage, scope, opts)
		})},
		// One page of the ids behind a count of exports.plan, for a family named in
		// ingest.Plan*; a page holds at most ingest.MaxPlannedIDs.
		route{http.MethodPost, "/v1/exports.planIds", rpc(func(ctx context.Context, scope string, req exportPlanIDsReq) (ingest.PlannedIDs, error) {
			opts, err := exportSQLiteOptions(ctx, s.opts.Storage, scope, req.Export)
			if err != nil {
				return ingest.PlannedIDs{}, err
			}
			return ingest.PlanExportIDs(ctx, s.opts.Storage, scope, opts, req.Family, req.Offset, req.Limit)
		})},
	)
}

// exportSQLiteReq is the body of exports.sqlite, exports.plan and the export of
// exports.planIds: the export options of the desktop and the CLI, under the same
// JSON keys, read by the same ingest.OptionsFor. A field left out keeps its
// value in ingest.WholeLibraryOptions, so an absent or empty body exports the
// whole tenant; an unknown key, or metadata, is refused. positions hands
// over positions whole, as the desktop does for unsaved ones. watermark / watermarkNote ask for a watermark signed with the
// daemon's own identity; exportPath is the daemon's affair and is ignored; a
// password is refused, the daemon streaming a plain SQLite file.
type exportSQLiteReq struct {
	domain.ExportOptions

	// decoded tells a body read over the preset from no body at all.
	decoded bool
}

// UnmarshalJSON reads the body over ingest.WholeLibraryOptions. A key the
// options do not hold is refused rather than ignored: an ignored one would
// widen the export to the whole tenant behind the caller's back.
func (r *exportSQLiteReq) UnmarshalJSON(b []byte) error {
	// encoding/json matches keys without regard to case, so the earlier
	// collectionIds would land on collectionIDs under the whole-library preset:
	// every key is checked against the options' exact names first.
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(b, &keys); err != nil {
		return err
	}
	for k := range keys {
		if !exportOptionKeys[k] {
			return fmt.Errorf("unknown export option %q: the export body takes the export options of the desktop and the CLI (wholeLibrary, include…, collectionIDs, deckIDs, lessonIDs, matchIDs, tournamentIDs, positionIDs, watermark…)", k)
		}
	}
	opts := ingest.WholeLibraryOptions()
	if err := json.Unmarshal(b, &opts); err != nil {
		return err
	}
	if len(opts.Metadata) > 0 {
		return errors.New(`"metadata" is not an export option of the daemon: it writes only the file's creation date and the watermark`)
	}
	r.ExportOptions, r.decoded = opts, true
	return nil
}

// exportOptionKeys holds the exact JSON names of domain.ExportOptions.
var exportOptionKeys = func() map[string]bool {
	keys := map[string]bool{}
	t := reflect.TypeFor[domain.ExportOptions]()
	for i := range t.NumField() {
		if name, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ","); name != "" && name != "-" {
			keys[name] = true
		}
	}
	return keys
}()

// options is what the request asks for: the preset when no body came.
func (r exportSQLiteReq) options() domain.ExportOptions {
	if !r.decoded {
		return ingest.WholeLibraryOptions()
	}
	return r.ExportOptions
}

// exportPlanIDsReq asks exports.planIds for a page of one family's ids, under
// the export Export describes.
type exportPlanIDsReq struct {
	Export exportSQLiteReq `json:"export"`
	Family string          `json:"family"`
	Offset int             `json:"offset,omitempty"`
	Limit  int             `json:"limit,omitempty"`
}

// ExportReportHeader carries, on an exports.sqlite answer, the export's report as
// JSON: what the file holds, family by family, the counts exports.plan announced.
// The body is the file itself, so the report rides in a header.
const ExportReportHeader = "X-Blunderdb-Export-Report"

// exportSQLiteOptions is what an exports.sqlite or exports.plan body asks for,
// short of the watermark.
func exportSQLiteOptions(ctx context.Context, st storage.Storage, scope string, req exportSQLiteReq) (ingest.ExportOptions, error) {
	opts := req.options()
	if opts.Password != "" {
		return ingest.ExportOptions{}, fmt.Errorf("%w: the daemon exports no password-protected file", storage.ErrInvalid)
	}
	return ingest.OptionsFor(ctx, st, scope, opts)
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
		return "", fmt.Errorf("%w: watermark was given but this server has no signing identity (start it with --identity-dir)", storage.ErrInvalid)
	}
	return ingest.SealWatermark(s.opts.Identity, origin, note)
}

// handleExportSQLite serializes the caller's whole tenant — every position,
// collection, match and tournament, with analyses, comments, played moves,
// the filter library and Anki decks — into a blunderDB SQLite file and
// returns it as a binary download. An optional JSON body asks for a
// watermark and narrows it to collections, lessons or decks (see
// exportSQLiteReq); with none given the export is WholeTenant, matching the
// GUI/CLI's "export everything" preset.
//
// The export is written to a private temp file, never straight to the
// ResponseWriter: the moment a body byte is written, Go commits whatever
// status/headers are set at that instant, so a mid-export failure would go out
// as HTTP 200 with an octet-stream Content-Type and a JSON error body wearing a
// ".sqlite" Content-Disposition. The binary headers — and the export's report,
// in ExportReportHeader — are set, and the file streamed to the client, only
// once the export has returned successfully. On failure, writeStorageError
// sets a proper status/content-type — nothing has reached the client yet.
func (s *Server) handleExportSQLite() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req exportSQLiteReq
		if err := decodeJSON(r, &req); err != nil {
			writeDecodeError(w, "invalid JSON body", err)
			return
		}
		opts, err := exportSQLiteOptions(r.Context(), s.opts.Storage, scopeOf(r), req)
		if err != nil {
			writeStorageError(w, err)
			return
		}
		watermark, err := s.sealExportWatermark(req.Watermark, req.WatermarkNote)
		if err != nil {
			writeStorageError(w, err)
			return
		}

		tmp, tmpErr := os.CreateTemp("", "blunderdb-export-resp-*.sqlite")
		if tmpErr != nil {
			writeStorageError(w, fmt.Errorf("server: create temp export file: %w", tmpErr))
			return
		}
		tmpPath := tmp.Name()
		defer os.Remove(tmpPath)
		if err := tmp.Close(); err != nil {
			writeStorageError(w, fmt.Errorf("server: close temp export file: %w", err))
			return
		}

		opts.Watermark = watermark
		if watermark != "" {
			opts.Signer = s.opts.Identity
		}
		report, err := ingest.ExportSQLite(r.Context(), s.opts.Storage, scopeOf(r), tmpPath, opts)
		if err != nil {
			writeStorageError(w, err)
			return
		}
		report.Path = "" // the daemon's temp file, nothing the client can open
		reportJSON, err := json.Marshal(report)
		if err != nil {
			writeStorageError(w, fmt.Errorf("server: encode export report: %w", err))
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
		w.Header().Set(ExportReportHeader, string(reportJSON))
		if _, err := io.Copy(w, f); err != nil {
			// Headers/status are already committed (the copy is the first
			// write): nothing more to do than let the client see a truncated
			// download with no error field to explain it. The log line is the
			// only place this failure is recorded — the Error case per
			// logging.go's scale, since nobody else will see it.
			slog.Error("server: stream sqlite export", "err", err)
		}
	}
}

// handleImport streams an uploaded file through the importer, emitting NDJSON
// progress events. The upload is spooled to a temp file so parser-backed
// formats can seek; JSON reads it back sequentially.
func (s *Server) handleImport(format ingest.Format) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		imp := s.importerFor(format)
		if imp == nil {
			writeErrorCode(w, CodeInvalid, "import format not supported on this server: "+string(format))
			return
		}

		// Claim the worst case against the global spool quota (see
		// spoolQuota) before touching the body at all.
		if !s.spool.reserve(s.opts.ImportMaxBodyBytes) {
			writeErrorCode(w, CodeRateLimited, "too many imports in flight, try again shortly")
			return
		}
		defer s.spool.release(s.opts.ImportMaxBodyBytes)
		if s.refuseImport(r.Context(), w, scopeOf(r)) {
			return
		}
		defer s.quota.endImport(scopeOf(r))

		r.Body = http.MaxBytesReader(w, r.Body, s.opts.ImportMaxBodyBytes)
		file, header, err := r.FormFile("file")
		if err != nil {
			writeDecodeError(w, "missing multipart 'file' field", err)
			return
		}
		defer file.Close()

		// Preserve the upload's extension when it is on allowedUploadExtensions
		// (see its doc comment); anything else is dropped rather than letting
		// an attacker-controlled filename reach the temp name unfiltered.
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
		// Shutdown.
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

		// One batch per upload: the report the client can ask for afterwards
		// is about THIS file, not about the tenant. A batch that cannot be
		// opened is not fatal — the import is what the caller asked for, and
		// losing its summary must not cost them the import.
		batches := s.opts.Storage.ImportBatches()
		batchID, batchErr := batches.Begin(ctx, scope, sourceLabel(header), string(format))
		if batchErr != nil {
			batchID = 0
		}

		emit(map[string]any{"event": "started", "import_id": importID, "batch_id": batchID})

		prog := func(p ingest.Progress) {
			emit(map[string]any{
				"event":     "progress",
				"matches":   p.Matches,
				"games":     p.Games,
				"positions": p.Positions,
			})
		}

		sum, err := imp.Import(ctx, scope, ingest.Source{Format: format, Path: tmpPath, BatchID: batchID, SkipDuplicates: skipDuplicatesParam(r)}, prog)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				// imports.cancel, or a graceful shutdown cancelling every
				// in-flight job (Server.Run) — either way the client asked for
				// or was told about this, not a failure.
				emit(map[string]any{"event": "cancelled"})
				return
			}
			emit(map[string]any{"event": "error", "error": errorBodyFor(w, err)})
			return
		}

		done := map[string]any{
			"event":              "done",
			"saved_positions":    sum.SavedPositions,
			"skipped_duplicates": sum.SkippedDuplicates,
			"matches":            sum.Matches,
			"match_id":           sum.MatchID,
			"flags_applied":      sum.FlagsApplied,
			"deepened":           sum.Deepened,
			"dropped_analyses":   sum.DroppedAnalyses,
		}
		// The same end-of-import report the desktop panel shows, in the
		// terminal event: a client that streams the import gets its summary
		// without a second call. Best-effort — the import has already
		// succeeded and committed by here.
		if batchID != 0 {
			done["batch_id"] = batchID
			counts := domain.ImportReport{
				MatchesImported:  sum.Matches - sum.SkippedDuplicates - sum.Enriched,
				MatchesSkipped:   sum.SkippedDuplicates,
				MatchesEnriched:  sum.Enriched,
				PositionsSaved:   sum.SavedPositions,
				AnalysesDeepened: sum.Deepened,
				AnalysesDropped:  sum.DroppedAnalyses,
			}
			if sum.Deepened > 0 {
				counts.MatchesDeepened = 1
			}
			if err := batches.Finish(ctx, scope, batchID, counts); err == nil {
				if rep, err := batches.Report(ctx, scope, batchID, nil); err == nil {
					done["report"] = rep.Report
				}
			}
		}
		emit(done)
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
		os.Remove(f.Name())
		return "", func() {}, fmt.Errorf("server: spool upload: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", func() {}, fmt.Errorf("server: spool close: %w", err)
	}
	path := f.Name()
	return path, func() { os.Remove(path) }, nil
}

// sourceLabel is what an import batch shows as its source: the name of the
// file the client uploaded. The daemon never sees the client's path, only the
// multipart filename, and that name is attacker-controlled — it is stored and
// displayed as data, never used to open anything (spoolToTemp already refuses
// everything but a known extension).
func sourceLabel(header *multipart.FileHeader) string {
	if header == nil || header.Filename == "" {
		return "upload"
	}
	return filepath.Base(header.Filename)
}

// importReportReq asks for the report of one past import. Players names the
// decisions the PR is about — the reference player and their other spellings;
// empty scores both seats, and the report says which.
type importReportReq struct {
	BatchID int64    `json:"batchId"`
	Players []string `json:"players,omitempty"`
}

// importStudyQueueReq asks for the queue that follows a report. Limit 0 means
// the queue's own bound (domain.MaxStudyQueue) — a queue nobody finishes is a
// queue nobody starts.
type importStudyQueueReq struct {
	BatchID int64    `json:"batchId"`
	Players []string `json:"players,omitempty"`
	Limit   int      `json:"limit,omitempty"`
}

// studyIsStudiedReq asks whether one position carries the "studied" mark.
type studyIsStudiedReq struct {
	PositionID int64 `json:"positionId"`
}

// studySetStudiedReq writes (Studied true) or withdraws the "studied" mark of
// one position.
type studySetStudiedReq struct {
	PositionID int64 `json:"positionId"`
	Studied    bool  `json:"studied"`
}
