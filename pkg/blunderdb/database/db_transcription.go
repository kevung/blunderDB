package database

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/mets"
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
	// Conflict: another writer changed the draft, the gesture was NOT
	// recorded, and this is the draft as it now stands, its undo stack reset.
	// The panel redraws it and says so.
	Conflict bool `json:"conflict"`
}

// transcriptService returns the service over the open library, made on first
// use. Caller holds d.mu; transcriptMu guards the pointer, lock ORDER
// mu -> transcriptMu, and forgetTranscriptSessions takes transcriptMu alone.
//
// forgetTranscriptSessions runs before d.mu is taken, so a call slipping in
// between can make a service on the library about to be replaced; it is
// recognised here by its store and dropped with its sessions, which name rows
// of that library.
func (d *Database) transcriptService() *transcription.Service {
	d.transcriptMu.Lock()
	defer d.transcriptMu.Unlock()
	if d.transcriptSvc != nil && d.transcriptOn != d.store {
		d.transcriptSvc.Forget()
		d.transcriptSvc = nil
	}
	if d.transcriptSvc == nil {
		d.transcriptSvc = transcription.New(d.store, transcription.Options{})
		d.transcriptOn = d.store
	}
	return d.transcriptSvc
}

// withTranscripts runs a read on the service under the read lock, so the
// library cannot be closed or replaced under it.
func (d *Database) withTranscripts(fn func(*transcription.Service) error) error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return fn(d.transcriptService())
}

// writeTranscripts runs a gesture that may write under the write lock, as
// every other write of the legacy wrapper does.
func (d *Database) writeTranscripts(fn func(*transcription.Service) error) error {
	d.mu.Lock()
	defer d.mu.Unlock()
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

// ApplyTranscriptionGesture records one gesture and returns the draft. When
// another writer changed the draft first, the gesture is not recorded and
// the draft comes back as it now stands, flagged Conflict.
func (d *Database) ApplyTranscriptionGesture(id int64, g transcript.Gesture) (*TranscriptionState, error) {
	return d.transcriptState(func(svc *transcription.Service) (*transcription.State, error) {
		return svc.Apply(context.Background(), "", id, transcription.Expect{}, g)
	})
}

// TranscriptionCandidates lists the plays of the roll being entered in the
// draft's session, both dice typed: every legal play, ranked at 0-ply without
// pruning, as the daemon's transcriptions.candidates ranks them. It never waits
// for a core a batch holds (rankTranscribedPlays). Writes nothing.
func (d *Database) TranscriptionCandidates(id int64) (*transcription.Candidates, error) {
	var out *transcription.Candidates
	err := d.withTranscripts(func(svc *transcription.Service) (err error) {
		out, err = svc.Candidates(context.Background(), "", id, "", d.rankTranscribedPlays)
		return err
	})
	return out, err
}

// rankTranscribedPlays is the desktop's Ranker: the library's own match
// equity table, a pooled serial searcher, pruning off. Caller holds d.mu.
//
// It takes no token of gammonnet.ProcessCores, unlike a search of a batch: a
// 0-ply ranking forks nothing and lasts a few milliseconds, up to ~9 ms for a
// double on a heavy contact board (152 plays), so it can only overrun the
// budget by one goroutine for that long, whereas waiting for a
// token behind a saturating batch would stall a keystroke for a whole deep
// position, with the library held under d.mu.RLock all the while. The daemon
// does queue it, on its own analysis pool, for tenant fairness.
func (d *Database) rankTranscribedPlays(ctx context.Context, pos domain.Position) ([]domain.CheckerMove, error) {
	_, met, err := mets.Current(ctx, d.store, "")
	if err != nil {
		return nil, err
	}
	return gammonnet.RankAllPlaysWithMET(nil, pos, met)
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
		_ = svc.Close("", id, "")
	}
	return nil
}

// AbandonTranscription deletes the draft without a Match (ADR-0045 §3); the
// caller confirms beforehand.
func (d *Database) AbandonTranscription(id int64) error {
	return d.writeTranscripts(func(svc *transcription.Service) error {
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

// transcriptState runs a gesture that opens or changes a draft under the
// write lock. A conflict carrying the fresh draft is handed back as that
// draft, flagged, rather than as an error the panel could only print.
func (d *Database) transcriptState(fn func(*transcription.Service) (*transcription.State, error)) (*TranscriptionState, error) {
	var st *transcription.State
	err := d.writeTranscripts(func(svc *transcription.Service) (err error) {
		st, err = fn(svc)
		return err
	})
	if fresh := conflictState(err); fresh != nil {
		return fresh, nil
	}
	if err != nil {
		return nil, err
	}
	return &TranscriptionState{ID: st.ID, Annotated: st.Annotated, CanUndo: st.CanUndo, CanRedo: st.CanRedo}, nil
}

// conflictState is the fresh draft a conflict carries, flagged Conflict, or
// nil when err is not such a conflict.
func conflictState(err error) *TranscriptionState {
	var stale *transcription.StaleError
	if !errors.As(err, &stale) || stale.State == nil {
		return nil
	}
	st := stale.State
	return &TranscriptionState{ID: st.ID, Annotated: st.Annotated, CanUndo: st.CanUndo, CanRedo: st.CanRedo, Conflict: true}
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
	d.transcriptOn = nil
}
