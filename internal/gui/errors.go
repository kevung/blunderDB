package gui

import "fmt"

// Error codes a bound App method can attach to a failure, mirroring
// internal/server's errors.go (no rate limiting in the GUI).
const (
	CodeNotFound = "not_found"
	CodeConflict = "conflict"
	CodeInvalid  = "invalid"
	CodeInternal = "internal"
)

// GUIError carries a stable Code with its Message. Wails passes only the
// Error() string, so the code travels as a "code: message" prefix.
type GUIError struct {
	Code    string
	Message string
}

func (e *GUIError) Error() string {
	return e.Code + ": " + e.Message
}

// newGUIError builds a *GUIError, formatting message with args if any. It
// wraps nothing: it is the terminal error.
func newGUIError(code, message string, args ...any) error {
	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}
	return &GUIError{Code: code, Message: message}
}
