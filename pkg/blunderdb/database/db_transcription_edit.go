package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
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
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}
	ctx := context.Background()
	m, err := d.store.Matches().Get(ctx, "", matchID)
	if err != nil {
		return nil, fmt.Errorf("match %d: %w", matchID, err)
	}
	out := &TranscriptionLosses{MatchID: matchID, Imported: isImported(m)}
	if out.DraftID, err = d.draftOfMatch(ctx, matchID); err != nil {
		return nil, err
	}
	if !out.Imported {
		return out, nil
	}

	const positions = `SELECT mv.position_id FROM move mv JOIN game g ON g.id = mv.game_id WHERE g.match_id = ?`
	err = d.db.QueryRowContext(ctx, `
		SELECT (SELECT COUNT(*) FROM analysis WHERE position_id IN (`+positions+`))
		     + (SELECT COUNT(*) FROM move_analysis ma JOIN move mv ON mv.id = ma.move_id
		                                              JOIN game g ON g.id = mv.game_id
		         WHERE g.match_id = ?),
		       (SELECT COUNT(*) FROM comment WHERE position_id IN (`+positions+`))`,
		matchID, matchID, matchID).Scan(&out.Analyses, &out.Comments)
	if err != nil {
		return nil, fmt.Errorf("match %d: counting what a transcription drops: %w", matchID, err)
	}
	return out, nil
}

// isImported: only an importer sets a source file or an import batch.
func isImported(m *domain.Match) bool {
	return m.FilePath != "" || m.ImportBatchID != 0
}

// draftOfMatch is the draft opened on matchID, 0 when none. Caller holds d.mu.
func (d *Database) draftOfMatch(ctx context.Context, matchID int64) (int64, error) {
	for row, err := range d.store.Transcriptions().List(ctx, "") {
		if err != nil {
			return 0, err
		}
		if row.MatchID == matchID {
			return row.ID, nil
		}
	}
	return 0, nil
}

// EditMatchTranscription opens a draft on an existing Match: the draft
// already open on it when there is one (one draft per Match), otherwise a new
// one replayed from the Match's .mat rendering, naming the Match so that
// FinishTranscription replaces it. The caller shows MatchTranscriptionLosses
// first; this method does not refuse a lossy Match.
func (d *Database) EditMatchTranscription(matchID int64) (*TranscriptionState, error) {
	d.transcriptMu.Lock()
	defer d.transcriptMu.Unlock()

	d.mu.RLock()
	if d.db == nil {
		d.mu.RUnlock()
		return nil, fmt.Errorf("no database is currently open")
	}
	ctx := context.Background()
	existing, err := d.draftOfMatch(ctx, matchID)
	var m *domain.Match
	if err == nil && existing == 0 {
		m, err = d.store.Matches().Get(ctx, "", matchID)
	}
	d.mu.RUnlock()
	if err != nil {
		return nil, fmt.Errorf("match %d: %w", matchID, err)
	}

	if existing != 0 {
		ed, err := d.session(existing)
		if err != nil {
			return nil, err
		}
		return opened(existing, ed), nil
	}

	mat, err := d.MatchMAT(matchID)
	if err != nil {
		return nil, fmt.Errorf("match %d: %w", matchID, err)
	}
	doc, err := transcript.FromMAT(mat)
	if err != nil {
		return nil, fmt.Errorf("match %d: %w", matchID, err)
	}
	id := matchID
	doc.Header.MatchID = &id
	// The .mat does not carry the tournament, and finishing with none would
	// detach the Match from it.
	doc.Header.TournamentID = m.TournamentID

	draftID, err := d.saveTranscription(0, doc)
	if err != nil {
		return nil, err
	}
	return opened(draftID, d.openTranscript(draftID, doc)), nil
}
