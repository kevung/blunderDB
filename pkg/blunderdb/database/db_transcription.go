package database

import (
	"context"
	"fmt"
	"os"

	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcription"
)

// Transcriptions (ADR-0045, ADR-0057): a façade over transcription.Service,
// the one implementation the desktop, the CLI and the daemon share. The
// desktop is one user in one process: its sessions never expire, and its
// gestures name no revision, so the session's own is the expectation and a
// write still fails if anything else moved the row.

// TranscriptionSummary is one line of the drafts list. Its facts are decoded
// from the document rather than duplicated in columns that could drift.
type TranscriptionSummary struct {
	ID            int64  `json:"id"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	FormatVersion string `json:"format_version"`
	// MatchID is the Match this draft has already produced, 0 while it has
	// produced none.
	MatchID int64  `json:"match_id"`
	Label   string `json:"label"`

	Player1 string `json:"player1"`
	Player2 string `json:"player2"`
	// MatchLength is 0 for a money session, as in transcript.Header.
	MatchLength int `json:"match_length"`
	ActionCount int `json:"action_count"`
}

// TranscriptionState is what every gesture binding hands back: the row's id
// and the annotated document. The panel derives nothing itself (ADR-0045
// rule 9).
type TranscriptionState struct {
	ID        int64                `json:"id"`
	Annotated transcript.Annotated `json:"annotated"`
	// CanUndo and CanRedo are the two sides of the session's stack, which is
	// in memory and nowhere else (ADR-0045 rule 1): they are false again the
	// moment the application is restarted, and that is the promise, not a gap.
	CanUndo bool `json:"can_undo"`
	CanRedo bool `json:"can_redo"`
}

// transcriptService returns the service over the open library, made on first
// use. Caller holds d.mu (read); transcriptMu guards the pointer, lock ORDER
// mu -> transcriptMu, and forgetTranscriptSessions takes transcriptMu alone.
func (d *Database) transcriptService() *transcription.Service {
	d.transcriptMu.Lock()
	defer d.transcriptMu.Unlock()
	if d.transcriptSvc == nil {
		d.transcriptSvc = transcription.New(d.store, transcription.Options{})
	}
	return d.transcriptSvc
}

// withTranscripts runs fn on the service under the read lock, so the library
// cannot be closed or replaced under a gesture.
func (d *Database) withTranscripts(fn func(*transcription.Service) error) error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return fn(d.transcriptService())
}

// ListTranscriptions returns the drafts, most recently updated first.
func (d *Database) ListTranscriptions() ([]TranscriptionSummary, error) {
	var out []TranscriptionSummary
	err := d.withTranscripts(func(svc *transcription.Service) error {
		rows, err := svc.List(context.Background(), "")
		for _, r := range rows {
			out = append(out, TranscriptionSummary{
				ID: r.ID, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, FormatVersion: r.FormatVersion,
				MatchID: r.MatchID, Label: r.Label, Player1: r.Player1, Player2: r.Player2,
				MatchLength: r.MatchLength, ActionCount: r.ActionCount,
			})
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = []TranscriptionSummary{}
	}
	return out, nil
}

// CreateTranscription opens a new draft and returns it, ready to type into.
func (d *Database) CreateTranscription(header transcript.Header) (*TranscriptionState, error) {
	return d.transcriptState(func(svc *transcription.Service) (*transcription.State, error) {
		return svc.Create(context.Background(), "", header)
	})
}

// OpenTranscription loads a draft and returns it replayed in full, Cursor at
// the end of the document; a draft already open keeps its session.
func (d *Database) OpenTranscription(id int64) (*TranscriptionState, error) {
	return d.transcriptState(func(svc *transcription.Service) (*transcription.State, error) {
		return svc.Open(context.Background(), "", id)
	})
}

// ApplyTranscriptionGesture records one gesture and returns the draft.
func (d *Database) ApplyTranscriptionGesture(id int64, g transcript.Gesture) (*TranscriptionState, error) {
	return d.transcriptState(func(svc *transcription.Service) (*transcription.State, error) {
		return svc.Apply(context.Background(), "", id, transcription.Expect{}, g)
	})
}

// TranscriptionMAT renders the draft as .mat text through the library's own
// renderer. Writes nothing.
func (d *Database) TranscriptionMAT(id int64) (string, error) {
	var text string
	err := d.withTranscripts(func(svc *transcription.Service) (err error) {
		text, err = svc.MAT(context.Background(), "", id)
		return err
	})
	return text, err
}

// CloseTranscription releases the in-memory session only; the draft resumes
// from the drafts list and leaves only by Finish or Abandon (ADR-0045 §2).
func (d *Database) CloseTranscription(id int64) error {
	d.transcriptMu.Lock()
	svc := d.transcriptSvc
	d.transcriptMu.Unlock()
	if svc != nil {
		svc.Close("", id, "")
	}
	return nil
}

// AbandonTranscription deletes the draft without a Match (ADR-0045 §3); the
// caller confirms beforehand.
func (d *Database) AbandonTranscription(id int64) error {
	return d.withTranscripts(func(svc *transcription.Service) error {
		return svc.Abandon(context.Background(), "", id, transcription.Expect{})
	})
}

// SuggestTranscriptionMatFilename is the default file name of a draft's .mat.
func (d *Database) SuggestTranscriptionMatFilename(id int64) (string, error) {
	var name string
	err := d.withTranscripts(func(svc *transcription.Service) (err error) {
		name, err = svc.SuggestMATFilename(context.Background(), "", id)
		return err
	})
	return name, err
}

// ExportTranscriptionMAT writes the draft's .mat to outputPath.
func (d *Database) ExportTranscriptionMAT(id int64, outputPath string) error {
	text, err := d.TranscriptionMAT(id)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, []byte(text), 0o644)
}

func (d *Database) transcriptState(fn func(*transcription.Service) (*transcription.State, error)) (*TranscriptionState, error) {
	var st *transcription.State
	err := d.withTranscripts(func(svc *transcription.Service) (err error) {
		st, err = fn(svc)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &TranscriptionState{ID: st.ID, Annotated: st.Annotated, CanUndo: st.CanUndo, CanRedo: st.CanRedo}, nil
}

// forgetTranscriptSessions drops every open draft when the handle is replaced
// or closed, BEFORE d.mu is taken; otherwise a row id of the previous library
// would answer for the next one.
func (d *Database) forgetTranscriptSessions() {
	d.transcriptMu.Lock()
	defer d.transcriptMu.Unlock()
	if d.transcriptSvc != nil {
		d.transcriptSvc.Forget()
	}
	d.transcriptSvc = nil
}
