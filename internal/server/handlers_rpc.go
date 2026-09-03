package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"mime"
	"net/http"

	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// This file holds the generic RPC plumbing shared by every /v1 domain handler.
// A handler is one of three shapes:
//
//   - rpc      decode JSON request → call → encode JSON response
//   - rpcStream decode JSON request → call → stream NDJSON items
//   - rpcVoid  decode JSON request → call → {"ok":true}
//
// Each concrete route is a tiny closure that binds the storage method, so the
// surface stays type-safe and mechanical (see handlers_<family>.go).

// okResp is the body returned by rpcVoid handlers.
type okResp struct {
	OK bool `json:"ok"`
}

// idResp wraps a freshly-created row id.
type idResp struct {
	ID int64 `json:"id"`
}

// idReq is the common "operate on this id" request.
type idReq struct {
	ID int64 `json:"id"`
}

// errMissing is the ErrInvalid a handler returns when a request omits the
// object it exists to store — {"position": null}, or just {}. The storage
// layer takes the pointer without looking at it (its contract is a value),
// so the check belongs at the edge, before the nil reaches SQL.
func errMissing(field string) error {
	return fmt.Errorf("%w: missing %s", storage.ErrInvalid, field)
}

// scopeOf returns the tenant scope for the request (empty if none, e.g. in a
// test that bypasses the tenant middleware).
func scopeOf(r *http.Request) string {
	scope, _ := middleware.TenantFromContext(r.Context())
	return scope
}

// acceptableContentType reports whether ct — a request's Content-Type header,
// possibly absent — is acceptable for a JSON-bodied /v1 call: empty (many
// simple clients never set it on a POST body, and a body-less call sends
// none at all) or "application/json" (parameters such as charset are
// ignored). Anything else — a stray "text/plain", a browser's
// "multipart/form-data" from a form that posted here by mistake — is
// rejected before a single byte is parsed as JSON, rather than surfacing as
// a confusing "invalid character" message for bytes that were never JSON to
// begin with (#232).
func acceptableContentType(ct string) bool {
	if ct == "" {
		return true
	}
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return false
	}
	return mediaType == "application/json"
}

// decodeJSON decodes the request body into dst. An empty body is accepted and
// leaves dst at its zero value, so methods with all-optional fields can be
// called with no body.
func decodeJSON(r *http.Request, dst any) error {
	if r.Body == nil {
		return nil
	}
	if ct := r.Header.Get("Content-Type"); !acceptableContentType(ct) {
		return fmt.Errorf("Content-Type must be application/json, got %q", ct)
	}
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return nil
}

// maxPageSize bounds a client-supplied page size (the family of "limit"
// request fields backed by storage.ListOpts and its siblings): a request for
// more than this many rows in one page is refused rather than honoured
// unbounded (#232). 1000 comfortably covers every real UI page while
// keeping a single query, and a single response, to a predictable size.
const maxPageSize = 1000

// pagedReq is implemented by a request type that carries a client-supplied
// page size — positions.list/listIds' listReq, matches.list's matchListReq,
// anki.reviewLog's reviewLogReq. rpc and rpcStream check every decoded
// request against it once, generically, so a new listing route inherits the
// cap by construction instead of a per-handler check someone has to
// remember to add.
type pagedReq interface {
	pageLimit() int
}

// checkPageLimit enforces maxPageSize on req when it implements pagedReq;
// every other request type is untouched.
func checkPageLimit(req any) error {
	pr, ok := req.(pagedReq)
	if !ok {
		return nil
	}
	if limit := pr.pageLimit(); limit > maxPageSize {
		return fmt.Errorf("%w: limit %d exceeds the maximum page size %d", storage.ErrInvalid, limit, maxPageSize)
	}
	return nil
}

// writeJSONResp writes v as a 200 JSON response.
func writeJSONResp(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(v)
}

// rpc builds a handler that decodes Req, invokes fn with the tenant scope, and
// encodes the Resp as JSON.
func rpc[Req any, Resp any](fn func(ctx context.Context, scope string, req Req) (Resp, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req Req
		if err := decodeJSON(r, &req); err != nil {
			writeDecodeError(w, "invalid JSON body", err)
			return
		}
		if err := checkPageLimit(req); err != nil {
			writeStorageError(w, err)
			return
		}
		resp, err := fn(r.Context(), scopeOf(r), req)
		if err != nil {
			writeStorageError(w, err)
			return
		}
		writeJSONResp(w, resp)
	}
}

// rpcVoid builds a handler for a storage method that returns only an error.
func rpcVoid[Req any](fn func(ctx context.Context, scope string, req Req) error) http.HandlerFunc {
	return rpc(func(ctx context.Context, scope string, req Req) (okResp, error) {
		if err := fn(ctx, scope, req); err != nil {
			return okResp{}, err
		}
		return okResp{OK: true}, nil
	})
}

// rpcStream builds a handler that decodes Req and streams the resulting
// iter.Seq2 as NDJSON.
func rpcStream[Req any, T any](fn func(ctx context.Context, scope string, req Req) iter.Seq2[T, error]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req Req
		if err := decodeJSON(r, &req); err != nil {
			writeDecodeError(w, "invalid JSON body", err)
			return
		}
		if err := checkPageLimit(req); err != nil {
			writeStorageError(w, err)
			return
		}
		streamSeq2(w, fn(r.Context(), scopeOf(r), req))
	}
}
