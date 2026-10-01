package direction

import (
	"errors"
	"fmt"
)

// ErrRefused marks a gesture the rules refuse — an empty name, a table already taken, an event
// the engine rejects: the request's fault, never the database's. A caller that answers for the
// gesture (the serve daemon) tells a refusal, which it shows, from a failure, which it hides.
var ErrRefused = errors.New("refused")

// refusal keeps its own message and is ErrRefused too.
type refusal struct{ err error }

func (r refusal) Error() string   { return r.err.Error() }
func (r refusal) Unwrap() []error { return []error{r.err, ErrRefused} }

// Refusef is a refusal with its message.
func Refusef(format string, args ...any) error {
	return refusal{fmt.Errorf(format, args...)}
}

// Refused marks err — an engine's verdict, a malformed request — as a refusal.
func Refused(err error) error {
	if err == nil || errors.Is(err, ErrRefused) {
		return err
	}
	return refusal{err}
}
