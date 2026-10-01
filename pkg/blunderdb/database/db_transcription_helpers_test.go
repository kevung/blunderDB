package database

import (
	"context"
	"strconv"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcription"
)

// Test access to the transcription service's internals, through its own
// methods: the tests below reach into a live session or write a row as the
// service would.

var (
	durableJSON         = transcription.DurableJSON
	decodeTranscription = transcription.Decode
)

// dropTranscriptSession forgets a draft's session, as a restart would.
func (d *Database) dropTranscriptSession(id int64) { _ = d.CloseTranscription(id) }

// withSession runs fn on the draft's live Editor; it fails the caller's
// expectation silently when there is none, which the assertions then catch.
func (d *Database) withSession(id int64, fn func(*transcript.Editor)) {
	d.mu.RLock()
	svc := d.transcriptService()
	d.mu.RUnlock()
	svc.WithEditor("", id, fn)
}

// sessionDoc is a copy of the live session's document.
func (d *Database) sessionDoc(id int64) transcript.Document {
	var doc transcript.Document
	d.withSession(id, func(ed *transcript.Editor) { doc = ed.Doc })
	return doc
}

// loadTranscription reads a row as a resumed draft: Cursor at the end.
func (d *Database) loadTranscription(id int64) (transcript.Document, error) {
	row, err := d.store.Transcriptions().Get(context.Background(), "", id)
	if err != nil {
		return transcript.Document{}, err
	}
	doc, err := transcription.Decode(row)
	if err != nil {
		return doc, err
	}
	doc.Cursor = len(doc.Actions)
	return doc, nil
}

// saveTranscription writes doc into row id (0 to insert), unconditionally.
func (d *Database) saveTranscription(id int64, doc transcript.Document) (int64, error) {
	blob, err := transcription.DurableJSON(doc)
	if err != nil {
		return 0, err
	}
	row := &storage.Transcription{ID: id, FormatVersion: strconv.Itoa(doc.FormatVersion),
		Label: transcription.Label(doc.Header), Document: string(blob)}
	if doc.Header.MatchID != nil {
		row.MatchID = *doc.Header.MatchID
	}
	return d.store.Transcriptions().Save(context.Background(), "", row)
}
