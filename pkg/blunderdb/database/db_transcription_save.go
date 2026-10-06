package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcription"
)

// Finishing a draft (ADR-0045 §2, §3, §8): transcription.Service writes the
// Match; the analysis batch is the caller's to start.

// TranscriptionSaveResult is what finishing a draft reports. ToAnalyze is the
// derived, never stored, count of the match's positions without analysis
// (ADR-0045 §8).
type TranscriptionSaveResult struct {
	MatchID int64 `json:"match_id"`
	// Replaced is false when the draft created its Match and true when it was
	// opened from one and rewrote it in place.
	Replaced  bool `json:"replaced"`
	Games     int  `json:"games"`
	Moves     int  `json:"moves"`
	Positions int  `json:"positions"`
	ToAnalyze int  `json:"to_analyze"`
	// Inconsistent: the draft carries an Inconsistency; finished all the same
	// (ADR-0044), flagged for callers that did not warn.
	Inconsistent bool `json:"inconsistent"`
	// Conflict: another writer changed the draft, nothing was written, and
	// State is the draft as it now stands, its undo stack reset.
	Conflict bool                `json:"conflict"`
	State    *TranscriptionState `json:"state,omitempty"`
}

// FinishTranscription materialises the draft as a Match and releases it: a
// draft that names no Match creates one, a draft opened from a Match
// (EditMatchTranscription) REPLACES it, id and all (ADR-0045 §2), in one
// transaction with the deletion of the draft.
func (d *Database) FinishTranscription(id int64) (*TranscriptionSaveResult, error) {
	var res *transcription.SaveResult
	err := d.writeTranscripts(func(svc *transcription.Service) (err error) {
		res, err = svc.Finish(context.Background(), "", id, transcription.Expect{})
		return err
	})
	if fresh := conflictState(err); fresh != nil {
		return &TranscriptionSaveResult{Conflict: true, State: fresh}, nil
	}
	if err != nil {
		return nil, err
	}
	return &TranscriptionSaveResult{
		MatchID: res.MatchID, Replaced: res.Replaced, Games: res.Games, Moves: res.Moves,
		Positions: res.Positions, ToAnalyze: res.ToAnalyze, Inconsistent: res.Inconsistent,
	}, nil
}

// MaterializeTranscription writes, in one call, the Match a document of
// Actions plays, or nothing: the first Action the rule machine refuses stops
// everything and comes back as a *transcription.RefusedAction. No draft is
// made.
func (d *Database) MaterializeTranscription(header transcript.Header, actions []transcript.Action) (*TranscriptionSaveResult, error) {
	var res *transcription.SaveResult
	err := d.writeTranscripts(func(svc *transcription.Service) (err error) {
		res, err = svc.Materialize(context.Background(), "", header, actions)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &TranscriptionSaveResult{
		MatchID: res.MatchID, Games: res.Games, Moves: res.Moves,
		Positions: res.Positions, ToAnalyze: res.ToAnalyze,
	}, nil
}

// TranscriptionAnalysisResume is the fact ADR-0045 §8 refuses to store,
// recounted instead: the most recent transcribed Match, and how many of its
// positions still lack analysis.
type TranscriptionAnalysisResume struct {
	MatchID int64 `json:"match_id"`
	// Label names the match the offer is about, as the drafts list would.
	Label string `json:"label"`
	// ToAnalyze is CountMatchPositionsToAnalyze at the moment of asking, and
	// never anything else: the figure the targeted batch is about to work
	// through.
	ToAnalyze int `json:"to_analyze"`
}

// PendingTranscriptionAnalysis reports whether the most recent transcribed
// Match has positions left to analyse, and returns nil when it has none —
// which is also what a library with no transcribed match answers.
//
// Asked on open; ignoring the offer stores nothing. A finished draft is gone
// (ADR-0045 §2), so "transcribed" is read off the Match itself: no source file
// and no import batch, which only an import sets. Scoped to that one match on
// purpose, never the library's imported positions.
func (d *Database) PendingTranscriptionAnalysis() (*TranscriptionAnalysisResume, error) {
	d.mu.RLock()
	if d.db == nil {
		d.mu.RUnlock()
		return nil, fmt.Errorf("no database is currently open")
	}
	ids, err := queryInt64s(d.db, `
		SELECT id FROM match
		 WHERE COALESCE(file_path, '') = '' AND import_batch_id IS NULL
		 ORDER BY id DESC LIMIT 1`)
	d.mu.RUnlock()
	if err != nil || len(ids) == 0 {
		return nil, err
	}

	// CountMatchPositionsToAnalyze takes the read lock itself; holding it
	// here as well would deadlock the moment a writer queued up between the
	// two (sync.RWMutex does not admit nested readers).
	n, err := d.CountMatchPositionsToAnalyze(ids[0])
	if err != nil || n == 0 {
		return nil, err
	}
	m, err := d.GetMatchByID(ids[0])
	if err != nil {
		return nil, err
	}
	return &TranscriptionAnalysisResume{
		MatchID:   ids[0],
		Label:     transcription.Label(transcript.Header{Player1: m.Player1Name, Player2: m.Player2Name, Event: m.Event}),
		ToAnalyze: n,
	}, nil
}
