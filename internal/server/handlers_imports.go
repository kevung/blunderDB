package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// importRegistry tracks in-flight imports so imports.cancel can abort them.
// Keys are opaque import ids scoped to a tenant.
type importRegistry struct {
	mu      sync.Mutex
	cancels map[string]context.CancelFunc
	seq     atomic.Uint64
}

func newImportRegistry() *importRegistry {
	return &importRegistry{cancels: make(map[string]context.CancelFunc)}
}

func (reg *importRegistry) start(scope string, cancel context.CancelFunc) string {
	id := scope + "-" + strconv.FormatUint(reg.seq.Add(1), 10)
	reg.mu.Lock()
	reg.cancels[id] = cancel
	reg.mu.Unlock()
	return id
}

func (reg *importRegistry) finish(id string) {
	reg.mu.Lock()
	delete(reg.cancels, id)
	reg.mu.Unlock()
}

// cancel aborts the import with the given id if it belongs to scope. Returns
// false if no such in-flight import exists for this tenant.
func (reg *importRegistry) cancel(scope, id string) bool {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	c, ok := reg.cancels[id]
	if !ok || !belongsTo(id, scope) {
		return false
	}
	c()
	return true
}

func belongsTo(id, scope string) bool {
	return len(id) > len(scope)+1 && id[:len(scope)] == scope && id[len(scope)] == '-'
}

// importerFor returns the Importer for a format, or nil if unsupported on this
// server build. PR3a wires JSON; the parser formats land in PR3b/PR3c.
func (s *Server) importerFor(f ingest.Format) ingest.Importer {
	switch f {
	case ingest.FormatJSON:
		return ingest.JSONImporter{S: s.opts.Storage}
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

// ingestRoutes registers the import/export endpoints supported by this build.
func (s *Server) ingestRoutes() []route {
	return []route{
		{http.MethodPost, "/v1/imports.json", s.handleImport(ingest.FormatJSON)},
		{http.MethodPost, "/v1/imports.cancel", s.handleImportCancel},
		{http.MethodPost, "/v1/exports.json", s.handleExport(ingest.FormatJSON)},
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

		r.Body = http.MaxBytesReader(w, r.Body, s.opts.ImportMaxBodyBytes)
		file, _, err := r.FormFile("file")
		if err != nil {
			writeErrorCode(w, CodeInvalid, "missing multipart 'file' field: "+err.Error())
			return
		}
		defer file.Close()

		tmpPath, cleanup, err := spoolToTemp(file)
		if err != nil {
			writeStorageError(w, err)
			return
		}
		defer cleanup()

		// A cancellable context, registered so imports.cancel can abort it.
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
			emit(map[string]any{
				"event": "error",
				"error": errorBody{Code: codeForErr(err), Message: err.Error()},
			})
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
		writeErrorCode(w, CodeInvalid, "invalid JSON body: "+err.Error())
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
			_ = json.NewEncoder(w).Encode(errorEnvelope{Error: errorBody{
				Code: codeForErr(err), Message: err.Error(),
			}})
		}
	}
}

// spoolToTemp copies r to a temporary file and returns its path plus a cleanup
// func that removes it.
func spoolToTemp(r io.Reader) (string, func(), error) {
	f, err := os.CreateTemp("", "blunderdb-import-*")
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
