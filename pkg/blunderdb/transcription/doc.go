// Package transcription is the service a Transcription is typed through
// (ADR-0045, ADR-0057): drafts over storage.TranscriptionStore, the
// in-memory sessions that hold what a draft does not persist (the Entry being
// typed and the undo stack), the revision every gesture names, and the exits
// — Finish into a Match through ingest.WriteMatch, Abandon, and EditMatch to
// reopen a Match as a draft.
//
// The rules of the game live in pkg/blunderdb/transcript; this package is the
// plumbing between them and the storage contract, written once for the
// desktop (database.Database is a façade over it), the CLI and the daemon.
//
// A session is a cache, never the truth: the row is written after every
// gesture, so losing a session (Options.TTL, a restart, another instance)
// loses the undo stack and nothing typed. A gesture naming a session that is
// gone gets ErrSessionGone; one naming a revision the row no longer has gets
// a StaleError.
package transcription
