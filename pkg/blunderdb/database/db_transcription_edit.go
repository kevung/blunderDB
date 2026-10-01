package database

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/transcription"
)

// Editing the transcription of an existing Match (ADR-0045 §2): a draft is
// opened from the Match's own .mat rendering, and finishing it replaces the
// Match in place.

// TranscriptionLosses is what opening a draft from a Match cannot carry and
// finishing it may drop, counted before anything is written.
type TranscriptionLosses struct {
	MatchID int64 `json:"match_id"`
	// Imported: the Match came from a file (XG, GnuBG, BGF…), not from a
	// transcription. A transcribed Match loses nothing and counts zero.
	Imported bool `json:"imported"`
	// Analyses and Comments are held by the Match's positions and moves; a
	// .mat carries neither.
	Analyses int `json:"analyses"`
	Comments int `json:"comments"`
	// DraftID is the draft already open on this Match, 0 when there is none:
	// reopening it loses nothing more, so no warning is due.
	DraftID int64 `json:"draft_id"`
}

// Lossy reports whether opening a draft would warrant a warning.
func (l TranscriptionLosses) Lossy() bool {
	return l.DraftID == 0 && l.Imported && l.Analyses+l.Comments > 0
}

// MatchTranscriptionLosses counts what EditMatchTranscription would leave
// behind for this Match. Writes nothing.
func (d *Database) MatchTranscriptionLosses(matchID int64) (*TranscriptionLosses, error) {
	var l *transcription.Losses
	err := d.withTranscripts(func(svc *transcription.Service) (err error) {
		l, err = svc.MatchLosses(context.Background(), "", matchID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &TranscriptionLosses{MatchID: l.MatchID, Imported: l.Imported, Analyses: l.Analyses,
		Comments: l.Comments, DraftID: l.DraftID}, nil
}

// EditMatchTranscription opens a draft on an existing Match: the draft
// already open on it when there is one (one draft per Match), otherwise a new
// one replayed from the Match's .mat rendering. The caller shows
// MatchTranscriptionLosses first; this method does not refuse a lossy Match.
func (d *Database) EditMatchTranscription(matchID int64) (*TranscriptionState, error) {
	return d.transcriptState(func(svc *transcription.Service) (*transcription.State, error) {
		st, _, err := svc.EditMatch(context.Background(), "", matchID)
		return st, err
	})
}
