package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// exportArchiveReq is the body of exports.archive: the matches to archive,
// none meaning every match of the tenant, and the producer's line printed at
// the foot of each sheet.
type exportArchiveReq struct {
	MatchIDs []int64 `json:"matchIds,omitempty"`
	Origin   string  `json:"origin,omitempty"`
}

// handleExportArchive serves POST /v1/exports.archive: the printable paper
// archive page (ADR-0119), text/html, as an attachment. The report — matches,
// codes, the matches skipped and why — travels in ExportReportHeader, as for
// exports.sqlite. The page is built whole before a byte is written, so a
// failure is an ordinary error response.
func (s *Server) handleExportArchive(w http.ResponseWriter, r *http.Request) {
	var req exportArchiveReq
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &req); err != nil {
			writeErrorCode(w, CodeInvalid, "invalid request body")
			return
		}
	}
	page, res, err := ingest.ExportArchive(r.Context(), s.opts.Storage, scopeOf(r), req.MatchIDs, req.Origin)
	if err != nil {
		writeStorageError(w, err)
		return
	}
	if rep, err := json.Marshal(res); err == nil {
		w.Header().Set(ExportReportHeader, string(rep))
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="blunderdb-archive.html"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(page)))
	_, _ = w.Write(page)
}
