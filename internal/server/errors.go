package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Error codes. This is a near-closed set — external clients depend on it.
// Adding a code is an additive API change (bump the API minor version). See
// tasks/headless/06-serve-http.md ("Error envelope (frozen)") and
// tasks/headless/11-tenant-rate-limit.md (which added rate_limited).
const (
	CodeNotFound    = "not_found"
	CodeConflict    = "conflict"
	CodeInvalid     = "invalid"
	CodeInternal    = "internal"
	CodeRateLimited = "rate_limited"
)

// errorEnvelope is the wire shape of every error response:
//
//	{"error":{"code":"...","message":"...","details":{...}}}
type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// statusForCode maps an error code to its HTTP status.
func statusForCode(code string) int {
	switch code {
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeInvalid:
		return http.StatusBadRequest
	case CodeRateLimited:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// codeForErr maps a storage sentinel error to an API error code.
func codeForErr(err error) string {
	switch {
	case errors.Is(err, storage.ErrNotFound):
		return CodeNotFound
	case errors.Is(err, storage.ErrConflict):
		return CodeConflict
	case errors.Is(err, storage.ErrInvalid):
		return CodeInvalid
	default:
		return CodeInternal
	}
}

// writeErrorCode writes a JSON error envelope with the given code and message.
func writeErrorCode(w http.ResponseWriter, code, message string) {
	writeErrorDetails(w, code, message, nil)
}

// writeErrorDetails writes a JSON error envelope including optional details.
func writeErrorDetails(w http.ResponseWriter, code, message string, details map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusForCode(code))
	_ = json.NewEncoder(w).Encode(errorEnvelope{Error: errorBody{
		Code:    code,
		Message: message,
		Details: details,
	}})
}

// writeStorageError maps a storage error onto the envelope. For internal
// errors the raw message is hidden behind a generic string to avoid leaking
// backend internals to clients.
func writeStorageError(w http.ResponseWriter, err error) {
	code := codeForErr(err)
	msg := err.Error()
	if code == CodeInternal {
		msg = "internal error"
	}
	writeErrorCode(w, code, msg)
}
