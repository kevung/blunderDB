// Package syncremote keeps the open library and a remote one level through
// the change journal (package changeset), over the two HTTP calls any server
// speaking the /v1 changeset routes answers: POST {base}/v1/changeset.export
// (a cursor in, a page of records out) and POST {base}/v1/changeset.import
// (records in, a merge report out), authenticated by a bearer token.
//
// A run sends first, then receives, a page at a time. Each side has its own
// cursor, kept by the caller per server (and per library): the local one marks
// what was already sent, the remote one what was already received. A cursor is
// advanced only once its page is settled, so a run cut short resumes where it
// stopped; a cursor the journal refuses (storage.ErrCursorStale) restarts from
// the beginning, the import absorbing the repetition. Records a run received
// come back in the next run's send, which the remote absorbs the same way.
package syncremote
