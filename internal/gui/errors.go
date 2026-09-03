package gui

import "fmt"

// Error codes a bound App method can attach to a failure, mirroring the
// stable set internal/server.errors.go already exposes over HTTP
// (CodeNotFound/CodeConflict/CodeInvalid/CodeInternal) so the two front
// ends — the desktop GUI and the headless daemon — describe the same kinds
// of failure the same way instead of each inventing its own vocabulary
// (#241). There is no GUI equivalent of the server's CodeRateLimited: this
// process has no rate limiter to trigger it.
const (
	CodeNotFound = "not_found"
	CodeConflict = "conflict"
	CodeInvalid  = "invalid"
	CodeInternal = "internal"
)

// GUIError is a bound method's error carrying a stable Code alongside its
// human-readable Message, so the frontend can eventually branch on Code
// (translated, stable) rather than pattern-matching an English sentence
// (today only Error() is available to a Wails caller — Wails serializes a
// returned error as the string a rejected promise carries — so Code is
// folded into that string as a "code: message" prefix a caller can already
// split on; a native structured payload would need a different bound-method
// return shape, i.e. (T, error) with the code carried some other way, which
// is left to whoever wires the frontend side of this up).
type GUIError struct {
	Code    string
	Message string
}

func (e *GUIError) Error() string {
	return e.Code + ": " + e.Message
}

// newGUIError builds a *GUIError, formatting message the way fmt.Sprintf
// would if extra args are given (mirrors fmt.Errorf's ergonomics without
// pulling in %w — a GUIError is never meant to wrap another error, it IS
// the terminal one shown to the user).
func newGUIError(code, message string, args ...any) error {
	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}
	return &GUIError{Code: code, Message: message}
}
