package server

import (
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// handleMatchRepair rewrites a match from its source file
// (ingest.RepairMatch). The daemon never opens a path a tenant stored: the
// client uploads the file (multipart field "file", its name giving the
// format), with the match "id" and "dryRun" as form fields. The upload is
// capped and spooled as an import's is.
func (s *Server) handleMatchRepair(w http.ResponseWriter, r *http.Request) {
	if !s.spool.reserve(s.opts.ImportMaxBodyBytes) {
		writeErrorCode(w, CodeRateLimited, "too many uploads in flight, try again shortly")
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
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeErrorCode(w, CodeInvalid, "missing or invalid 'id' field")
		return
	}
	dryRun, _ := strconv.ParseBool(r.FormValue("dryRun"))

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

	graph, err := ingest.MapMatchFile(tmpPath)
	if err != nil {
		writeErrorCode(w, CodeInvalid, "the uploaded file holds no readable match")
		return
	}
	rep, err := ingest.RepairMatch(r.Context(), s.opts.Storage, scopeOf(r), id, graph, dryRun)
	if err != nil {
		writeStorageError(w, err)
		return
	}
	writeJSONResp(w, rep)
}
