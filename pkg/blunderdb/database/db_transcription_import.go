package database

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/transcription"
)

// ImportTranscription opens a new draft on a file's text: a transcription
// document in JSON or a .mat, told apart by their first character. Nothing is
// written when the text is neither; the draft owns no Match until finished.
func (d *Database) ImportTranscription(text string) (*TranscriptionState, error) {
	return d.transcriptState(func(svc *transcription.Service) (*transcription.State, error) {
		return svc.Import(context.Background(), "", text)
	})
}
