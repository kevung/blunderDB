package database

import (
	"context"
	"strings"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// The history of a Direction (ADR-0047 §5.8, issue #375).
//
// The Direction is the record of everything the director decided, so it has to be READABLE.
// It is what a director prints after a dispute, and what a director-player re-reads on coming
// back to their own table. Nothing here is a translated sentence: an entry is a kind and its
// facts, and the frontend renders it.

// HistoryEntry is one decision, with enough to situate it.
type HistoryEntry struct {
	Seq int `json:"seq"`
	// Kind is the engine's event kind, a code.
	Kind       string        `json:"kind"`
	Time       string        `json:"time"`
	MatchID    string        `json:"matchId,omitempty"`
	Label      tournoi.Label `json:"label,omitempty"`
	Section    string        `json:"section,omitempty"`
	A          string        `json:"a,omitempty"`
	B          string        `json:"b,omitempty"`
	AName      string        `json:"aName,omitempty"`
	BName      string        `json:"bName,omitempty"`
	Winner     string        `json:"winner,omitempty"`
	WinnerName string        `json:"winnerName,omitempty"`
	ScoreA     int           `json:"scoreA,omitempty"`
	ScoreB     int           `json:"scoreB,omitempty"`
	Length     int           `json:"length,omitempty"`
	Table      int           `json:"table,omitempty"`
	Forfeit    bool          `json:"forfeit,omitempty"`
	// Player names the entry's subject when it is about one person: an entry, a withdrawal,
	// a bye.
	Player     string `json:"player,omitempty"`
	PlayerName string `json:"playerName,omitempty"`
	// Text is the director's own words: a remark on a result, a free note. It is the only
	// free text in the log.
	Text string `json:"text,omitempty"`
	// Correctable and Cancellable say which gestures this entry still admits, so the view
	// offers them here rather than sending the director hunting for the match.
	Correctable bool `json:"correctable,omitempty"`
	Cancellable bool `json:"cancellable,omitempty"`
}

// History returns the decisions of a Direction, oldest first.
//
// `player` and `match`, when given, filter it — which is how a director answers "what happened
// to Hugo?" without reading the whole log.
func (d *Database) History(tournamentID int64, player, match string) ([]HistoryEntry, error) {
	dir, err := direction.Open(context.Background(), d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	st := dir.State()
	if st == nil {
		return nil, nil
	}
	out := make([]HistoryEntry, 0, len(dir.Journal()))
	for _, ev := range dir.Journal() {
		e := HistoryEntry{
			Seq: ev.Seq, Kind: string(ev.Kind), Time: ev.Time.Format(time.RFC3339),
			MatchID: string(ev.MatchID), Label: ev.Label, Section: ev.Section,
			Length: ev.Length, Table: ev.Table, Text: ev.Text, Forfeit: ev.Forfeit,
			ScoreA: ev.ScoreA, ScoreB: ev.ScoreB,
		}
		if ev.Player != nil {
			e.Player, e.PlayerName = string(ev.Player.ID), ev.Player.Name
		} else if ev.ID != "" {
			e.Player, e.PlayerName = string(ev.ID), playerNameIn(st, ev.ID)
		}
		if m := st.Matches[ev.MatchID]; m != nil {
			e.A, e.B = string(m.A), string(m.B)
			e.AName, e.BName = playerNameIn(st, m.A), playerNameIn(st, m.B)
			if e.Length == 0 {
				e.Length = m.Length
			}
		} else if ev.A != "" {
			e.A, e.B = string(ev.A), string(ev.B)
			e.AName, e.BName = playerNameIn(st, ev.A), playerNameIn(st, ev.B)
		}
		if ev.Winner != "" {
			e.Winner, e.WinnerName = string(ev.Winner), playerNameIn(st, ev.Winner)
		}
		switch ev.Kind {
		case tournoi.EvResult, tournoi.EvResultCorrected:
			e.Correctable = true
		case tournoi.EvMatchStarted:
			if m := st.Matches[ev.MatchID]; m != nil && m.Status == tournoi.Running {
				e.Cancellable = true
			}
		}
		if !historyMatches(e, player, match) {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

// historyMatches applies the filters. A filter on a player keeps every entry that names them,
// whether as an entrant, a competitor or a winner.
func historyMatches(e HistoryEntry, player, match string) bool {
	if match != "" && e.MatchID != match {
		return false
	}
	if player == "" {
		return true
	}
	q := strings.ToLower(player)
	for _, s := range []string{e.Player, e.PlayerName, e.A, e.B, e.AName, e.BName, e.Winner, e.WinnerName} {
		if s != "" && strings.Contains(strings.ToLower(s), q) {
			return true
		}
	}
	return false
}

// AddDirectionNote records a free note of the director's, timestamped. It is the one place in
// the log where their own words go, and it is refused by nothing — including after the close,
// since a note about a closed tournament is exactly when one is written.
func (d *Database) AddDirectionNote(tournamentID int64, text string) (*DirectionView, error) {
	ctx := context.Background()
	dir, err := direction.Open(ctx, d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	if err := dir.Apply(ctx, tournoi.NoteEvent(strings.TrimSpace(text), time.Now())); err != nil {
		return nil, err
	}
	return d.GetDirection(tournamentID)
}

// SinceLastGesture returns the decisions recorded after a given sequence number: what happened
// while the director was somewhere else.
//
// A director who also plays comes back to their laptop between two of their own matches, and
// what they need first is "what changed", not the whole log.
func (d *Database) SinceLastGesture(tournamentID int64, seq int) ([]HistoryEntry, error) {
	all, err := d.History(tournamentID, "", "")
	if err != nil {
		return nil, err
	}
	var out []HistoryEntry
	for _, e := range all {
		if e.Seq > seq {
			out = append(out, e)
		}
	}
	return out, nil
}
