package database

import (
	"context"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// Undoing a mistyped result (ADR-0047 §5.4, issue #372).
//
// A director is interrupted every two minutes: they will click the wrong name, and they will
// notice within the second. What makes that survivable is that the mistake undoes itself IN
// PLACE — and that undoing is never erasing. The log is append-only, so a correction is one
// more event, and the first result stays in the history where it happened.

// LastDecision is what the panel shows under the queue: the last thing the director did, with
// enough to take it back in two clicks.
type LastDecision struct {
	// Kind is the engine's event kind — a code, not a sentence.
	Kind       string `json:"kind"`
	Seq        int    `json:"seq"`
	MatchID    string `json:"matchId,omitempty"`
	A          string `json:"a,omitempty"`
	B          string `json:"b,omitempty"`
	AName      string `json:"aName,omitempty"`
	BName      string `json:"bName,omitempty"`
	Winner     string `json:"winner,omitempty"`
	WinnerName string `json:"winnerName,omitempty"`
	ScoreA     int    `json:"scoreA,omitempty"`
	ScoreB     int    `json:"scoreB,omitempty"`
	Length     int    `json:"length,omitempty"`
	Table      int    `json:"table,omitempty"`
	Forfeit    bool   `json:"forfeit,omitempty"`
	// Correctable says whether this decision can be taken back from here: a result can, a
	// draw cannot (undoing a draw would be a re-draw, which ADR-0047 forbids).
	Correctable bool `json:"correctable"`
	// Cancellable says whether the match it launched can be cancelled.
	Cancellable bool `json:"cancellable"`
}

// LastDecision returns the director's last decision, or nil if they have made none.
//
// It reads the log backwards past the events nobody takes back — entries, notes — so that
// "undo" lands on the gesture the director is actually thinking of, which is the last RESULT
// or the last match launched, not the note they typed in between.
func (d *Database) LastDecision(tournamentID int64) (*LastDecision, error) {
	ctx := context.Background()
	dir, err := direction.Open(ctx, d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	st := dir.State()
	if st == nil {
		return nil, nil
	}
	j := dir.Journal()
	for i := len(j) - 1; i >= 0; i-- {
		ev := j[i]
		switch ev.Kind {
		case tournoi.EvResult, tournoi.EvResultCorrected:
			m := st.Matches[ev.MatchID]
			if m == nil {
				continue
			}
			return &LastDecision{
				Kind: string(ev.Kind), Seq: ev.Seq, MatchID: string(m.ID),
				A: string(m.A), B: string(m.B),
				AName: playerNameIn(st, m.A), BName: playerNameIn(st, m.B),
				Winner: string(m.Winner), WinnerName: playerNameIn(st, m.Winner),
				ScoreA: m.ScoreA, ScoreB: m.ScoreB, Length: m.Length,
				Forfeit: m.Forfeit, Correctable: true,
			}, nil
		case tournoi.EvMatchStarted:
			m := st.Matches[ev.MatchID]
			if m == nil || m.Status != tournoi.Running {
				continue
			}
			return &LastDecision{
				Kind: string(ev.Kind), Seq: ev.Seq, MatchID: string(m.ID),
				A: string(m.A), B: string(m.B),
				AName: playerNameIn(st, m.A), BName: playerNameIn(st, m.B),
				Length: m.Length, Table: m.Table, Cancellable: true,
			}, nil
		}
	}
	return nil, nil
}

// CorrectResult replaces the result of a match that is already finished. The first result stays
// in the log where it happened; this is one more event, and the state is recomputed from both.
//
// It is what the director reaches for when they clicked the wrong name — and what they reach
// for three rounds later when a score sheet turns up. The engine raises whatever the correction
// breaks (a bracket match now played by the wrong players); nothing is repaired behind their
// back, which is issue #389's job.
func (d *Database) CorrectResult(tournamentID int64, matchID, winner string, scoreA, scoreB int, note string) (*DirectionView, error) {
	ctx := context.Background()
	dir, err := direction.Open(ctx, d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	ev := tournoi.CorrectionEvent(tournoi.MatchID(matchID), tournoi.PlayerID(winner), scoreA, scoreB, time.Now())
	if note != "" {
		ev = ev.WithNote(note)
	}
	if err := dir.Apply(ctx, ev); err != nil {
		return nil, err
	}
	return d.GetDirection(tournamentID)
}

// FinishedMatches lists the matches already played, most recent first, so a result can be
// corrected long after the table it was played on has been taken by someone else.
func (d *Database) FinishedMatches(tournamentID int64, limit int) ([]TableCell, error) {
	dir, err := direction.Open(context.Background(), d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	st := dir.State()
	if st == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	var out []TableCell
	for i := len(st.MatchOrder) - 1; i >= 0 && len(out) < limit; i-- {
		m := st.Matches[st.MatchOrder[i]]
		if m == nil || m.Status != tournoi.Finished {
			continue
		}
		out = append(out, TableCell{
			Table: m.Table, MatchID: string(m.ID),
			A: string(m.A), B: string(m.B),
			AName: playerNameIn(st, m.A), BName: playerNameIn(st, m.B),
			Length: m.Length,
		})
	}
	return out, nil
}
