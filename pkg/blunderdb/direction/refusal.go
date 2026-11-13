package direction

import (
	"errors"
	"fmt"
)

// ErrRefused marks a gesture the rules refuse — an empty name, a table already taken, an event
// the engine rejects: the request's fault, never the database's. A caller that answers for the
// gesture (the serve daemon) tells a refusal, which it shows, from a failure, which it hides.
var ErrRefused = errors.New("refused")

// Reason is the stable code of a refusal a client may want to tell from the others without
// reading its message, which is written for a person and may be reworded.
type Reason string

const (
	// ReasonTableTaken: the table is played on, in this event or a sister of its room.
	ReasonTableTaken Reason = "table_taken"
	// ReasonPlayerBusy: a player of the match sits at a match of a sister event.
	ReasonPlayerBusy Reason = "player_busy"
	// ReasonNoSuchTable: the table is beyond the last one of the room.
	ReasonNoSuchTable Reason = "no_such_table"
	// ReasonTableOutside: the table is in no room the event plays in.
	ReasonTableOutside Reason = "table_outside"
	// ReasonTableClosed: the table is reserved, or kept for someone else.
	ReasonTableClosed Reason = "table_closed"
	// ReasonProposalChanged: a proposal of a confirmed list is no longer proposed as listed.
	ReasonProposalChanged Reason = "proposal_changed"
	// ReasonProposalHeld: a proposal of a confirmed list waits for a table or a player, or is
	// a wait: there is nothing to launch.
	ReasonProposalHeld Reason = "proposal_held"
)

// refusal keeps its own message and is ErrRefused too.
type refusal struct {
	err    error
	reason Reason
}

func (r refusal) Error() string   { return r.err.Error() }
func (r refusal) Unwrap() []error { return []error{r.err, ErrRefused} }

// ErrorDetails hands the reason to a caller that answers with a structured error.
func (r refusal) ErrorDetails() map[string]any {
	if r.reason == "" {
		return nil
	}
	return map[string]any{"reason": string(r.reason)}
}

// Refusef is a refusal with its message.
func Refusef(format string, args ...any) error {
	return refusal{err: fmt.Errorf(format, args...)}
}

// RefuseAs is Refusef with the reason a client can branch on.
func RefuseAs(reason Reason, format string, args ...any) error {
	return refusal{err: fmt.Errorf(format, args...), reason: reason}
}

// ReasonOf is the reason of a refusal, empty when it states none or err is not one.
func ReasonOf(err error) Reason {
	var r refusal
	if errors.As(err, &r) {
		return r.reason
	}
	return ""
}

// Refused marks err — an engine's verdict, a malformed request — as a refusal.
func Refused(err error) error {
	if err == nil || errors.Is(err, ErrRefused) {
		return err
	}
	return refusal{err: err}
}
