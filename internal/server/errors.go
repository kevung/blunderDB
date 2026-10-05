package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/duel"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcription"
)

// Error codes. This is a near-closed set — external clients depend on it.
// Adding a code is an additive API change (bump the API minor version).
const (
	CodeNotFound = "not_found"
	// CodeUnknownRoute answers a path no route serves. Same 404 status as
	// not_found (no resource lives at that URI, and status-only clients keep
	// their behaviour), but a distinct code: a client whose method was renamed
	// or removed by a version skew must see a fault, not an absent row.
	CodeUnknownRoute = "unknown_route"
	CodeConflict     = "conflict"
	CodeInvalid      = "invalid"
	CodeInternal     = "internal"
	CodeRateLimited  = "rate_limited"
	// CodeGone: the transcription session the request names expired or was
	// never this instance's; reopen the draft (410).
	CodeGone = "gone"
	// CodePreconditionRequired: a gesture that writes named no revision in
	// If-Match (428).
	CodePreconditionRequired = "precondition_required"
	// CodeIdempotencyMismatch: an Idempotency-Key already names another
	// request (method, path or body differ); mint a new key (422).
	CodeIdempotencyMismatch = "idempotency_key_reused"
	// CodeUnavailable: the daemon is stopping and takes nothing new; retry
	// against another instance or later (503).
	CodeUnavailable = "unavailable"
	// CodeQuotaExceeded: the tenant spent its engine time for the day, or
	// runs as many imports as it may (429). details.quota names which.
	CodeQuotaExceeded = "quota_exceeded"
	// CodeStorageQuotaExceeded: the tenant stores as many positions as it
	// may (413).
	CodeStorageQuotaExceeded = "storage_quota_exceeded"
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
	case CodeNotFound, CodeUnknownRoute:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeInvalid:
		return http.StatusBadRequest
	case CodeRateLimited:
		return http.StatusTooManyRequests
	case CodeGone:
		return http.StatusGone
	case CodePreconditionRequired:
		return http.StatusPreconditionRequired
	case CodeIdempotencyMismatch:
		return http.StatusUnprocessableEntity
	case CodeUnavailable:
		return http.StatusServiceUnavailable
	case CodeQuotaExceeded:
		return http.StatusTooManyRequests
	case CodeStorageQuotaExceeded:
		return http.StatusRequestEntityTooLarge
	default:
		return http.StatusInternalServerError
	}
}

// codeForErr maps a storage sentinel error to an API error code.
func codeForErr(err error) string {
	switch {
	case errors.Is(err, transcription.ErrSessionGone):
		return CodeGone
	case isDuelRefusal(err):
		return CodeInvalid
	case errors.Is(err, duel.ErrNotOpen):
		return CodeConflict
	case errors.Is(err, storage.ErrNotFound), errors.Is(err, direction.ErrNoDirection):
		// A Tournament that was never directed has no Direction to read.
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

// errSetter is implemented by the logging middleware's response wrapper
// (internal/server/middleware.responseRecorder). Matched structurally so this
// package does not need to import the concrete type.
type errSetter interface {
	SetErr(error)
}

// internalErrorMessage is the only text a client ever sees for an internal
// error. The real cause (a DSN, a file path, a SQL statement) belongs to the
// server-side log, never to the wire — see errorBodyFor.
const internalErrorMessage = "internal error"

// notFoundMessage replaces a not_found message that carries a driver error.
const notFoundMessage = "not found"

// carriesDriverError reports whether err wraps a database driver error (one
// that states an SQLSTATE, as pgconn.PgError does), matched structurally so
// this package imports no driver.
func carriesDriverError(err error) bool {
	var d interface{ SQLState() string }
	return errors.As(err, &d)
}

// errorBodyFor maps err onto the error body every error response carries:
// the code from codeForErr, and the error's own message — unless the code
// is internal, in which case the raw message is hidden behind a generic
// string to avoid leaking backend internals to clients. The real error must
// not vanish entirely: it is stashed on the ResponseWriter (when it supports
// it, which it always does once the request has passed through the Logging
// middleware — see server.go's chain()) so the request's server-side log
// line still carries the actual cause instead of just "internal error".
//
// It is the ONE place an error becomes a message for the client. Handlers
// that have already committed a 200 and can only append a trailing NDJSON
// line or event (streamSeq2, handleImport, handleExport,
// handleGammonNetAnalyzeMissing) call it directly; everything else goes
// through writeStorageError, which writes the envelope around it.
func errorBodyFor(w http.ResponseWriter, err error) errorBody {
	code := codeForErr(err)
	msg := err.Error()
	if code == CodeNotFound && carriesDriverError(err) {
		// A refused foreign key reaches here as not_found with the driver's
		// text inside: constraint and column names, which would let a caller
		// tell one refusal from another. Same message as any absent row; the
		// cause goes to the log.
		if es, ok := w.(errSetter); ok {
			es.SetErr(err)
		}
		msg = notFoundMessage
	}
	if code == CodeInternal {
		if es, ok := w.(errSetter); ok {
			es.SetErr(err)
		}
		msg = internalErrorMessage
	}
	body := errorBody{Code: code, Message: msg}
	// An error that knows more than its message (a conflict's current
	// revision) hands it over as details.
	var d interface{ ErrorDetails() map[string]any }
	if code != CodeInternal && errors.As(err, &d) {
		body.Details = d.ErrorDetails()
	}
	return body
}

// writeStorageError maps a storage error onto the envelope — errorBodyFor's
// masking, then the matching HTTP status.
func writeStorageError(w http.ResponseWriter, err error) {
	body := errorBodyFor(w, err)
	writeErrorDetails(w, body.Code, body.Message, body.Details)
}

// writeDecodeError reports a request body that could not be read as `what`
// (the caller's phrase, e.g. "invalid JSON body"). A body cut off by a
// MaxBytesReader — limitBody's default cap, or handleImport's own upload
// cap — is answered 413 rather than 400: the JSON was not malformed, the
// request was too big, and the client's fix is a smaller request. The
// envelope keeps CodeInvalid — the error-code set is near-closed (see the
// constants above) and the HTTP status already carries the distinction.
// Every other decode failure is the client's malformed body: 400, with the
// decoder's message, which describes their bytes and nothing of ours.
func writeDecodeError(w http.ResponseWriter, what string, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeBodyTooLarge(w, tooLarge.Limit)
		return
	}
	writeErrorCode(w, CodeInvalid, what+": "+err.Error())
}

// writeBodyTooLarge answers 413 for a request body over limit bytes. It is
// the one envelope whose status is not statusForCode's: CodeInvalid at 413,
// see writeDecodeError.
func writeBodyTooLarge(w http.ResponseWriter, limit int64) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusRequestEntityTooLarge)
	_ = json.NewEncoder(w).Encode(errorEnvelope{Error: errorBody{
		Code:    CodeInvalid,
		Message: "request body too large",
		Details: map[string]any{"limit_bytes": limit},
	}})
}

// writeMethodNotAllowed answers 405 for a known path called with the wrong
// method, Allow naming the one it accepts. Another envelope whose status is
// not statusForCode's, like writeBodyTooLarge's 413 — the error-code set
// stays near-closed (CodeInvalid) while the HTTP status carries the
// distinction.
func writeMethodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	_ = json.NewEncoder(w).Encode(errorEnvelope{Error: errorBody{
		Code:    CodeInvalid,
		Message: "method not allowed, use " + allow,
	}})
}
