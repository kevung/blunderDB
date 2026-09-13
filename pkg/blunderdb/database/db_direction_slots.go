package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// Slots: where a directed Tournament meets the library (ADR-0047, issues #381-#384).
//
// This is the whole point of the decision. A tournament directed here creates its Matches
// before they are played: every launched match is a Slot that a Transcription or an imported
// file then fills. A club tournament where nothing is recorded leaves its Slots empty; a
// BMAB-style tournament fills them all, and the director gets what nobody has today — the
// matches of one tournament, ranged by round and by player, analysed and searchable.
//
// One rule shapes every function here: NOTHING IS ATTACHED BY INFERENCE. A coincidence of names
// is a suggestion the director accepts; it is never a decision the software makes.

// SlotRow is one match of the Direction seen as a place a Match can fill.
type SlotRow struct {
	SlotID string        `json:"slotId"`
	Label  tournoi.Label `json:"label"`
	Phase  int           `json:"phase"`
	A      string        `json:"a"`
	B      string        `json:"b"`
	AName  string        `json:"aName"`
	BName  string        `json:"bName"`
	Length int           `json:"length"`
	Table  int           `json:"table,omitempty"`
	// Winner and the scores are what the DIRECTOR said, which during a tournament is what
	// stands (ADR-0047 §5.3).
	Winner     string `json:"winner,omitempty"`
	WinnerName string `json:"winnerName,omitempty"`
	ScoreA     int    `json:"scoreA,omitempty"`
	ScoreB     int    `json:"scoreB,omitempty"`
	Done       bool   `json:"done"`
	// MatchID is the library Match filling this Slot, 0 when it is empty.
	MatchID int64 `json:"matchId,omitempty"`
	// DraftID is a Transcription started from this Slot and not yet saved: the Slot is
	// reserved from the draft, not only from the save.
	DraftID int64 `json:"draftId,omitempty"`
	// Disagreement says the attached Match's own result contradicts what the director
	// recorded. It is SHOWN, never resolved: the software does not overrule the director.
	Disagreement string `json:"disagreement,omitempty"`
}

// Slots lists the matches of a Direction with what fills them.
func (d *Database) Slots(tournamentID int64) ([]SlotRow, error) {
	ctx := context.Background()
	dir, err := direction.Open(ctx, d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	st := dir.State()
	if st == nil {
		return nil, nil
	}
	filled, err := d.slotMatches(ctx, tournamentID)
	if err != nil {
		return nil, err
	}
	drafts, err := d.slotDrafts(ctx, tournamentID)
	if err != nil {
		return nil, err
	}
	out := make([]SlotRow, 0, len(st.MatchOrder))
	for _, id := range st.MatchOrder {
		m := st.Matches[id]
		if m == nil || m.Status == tournoi.Cancelled {
			continue
		}
		r := SlotRow{
			SlotID: string(m.ID), Label: m.Label, Phase: m.Phase,
			A: string(m.A), B: string(m.B),
			AName: playerNameIn(st, m.A), BName: playerNameIn(st, m.B),
			Length: m.Length, Table: m.Table,
			Winner: string(m.Winner), WinnerName: playerNameIn(st, m.Winner),
			ScoreA: m.ScoreA, ScoreB: m.ScoreB,
			Done: m.Status == tournoi.Finished,
		}
		if mt, ok := filled[string(m.ID)]; ok {
			r.MatchID = mt.id
			r.Disagreement = disagreement(m, mt)
		}
		r.DraftID = drafts[string(m.ID)]
		out = append(out, r)
	}
	return out, nil
}

// filledMatch is the library Match sitting in a Slot, with what its own file says happened.
type filledMatch struct {
	id       int64
	player1  string
	player2  string
	length   int
	score1   int
	score2   int
	hasScore bool
}

func (d *Database) slotMatches(ctx context.Context, tournamentID int64) (map[string]filledMatch, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	rows, err := d.db.QueryContext(ctx, `
		SELECT direction_match_id, id, COALESCE(player1_name,''), COALESCE(player2_name,''),
		       COALESCE(match_length,0)
		  FROM match
		 WHERE tournament_id = ? AND direction_match_id <> ''`, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("reading slots: %w", err)
	}
	defer rows.Close()
	out := map[string]filledMatch{}
	for rows.Next() {
		var slot string
		var m filledMatch
		if err := rows.Scan(&slot, &m.id, &m.player1, &m.player2, &m.length); err != nil {
			return nil, err
		}
		out[slot] = m
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// The final score of a Match is the sum of its games, which the games table holds.
	for slot, m := range out {
		var s1, s2 sql.NullInt64
		err := d.db.QueryRowContext(ctx, `
			SELECT MAX(initial_score_1 + CASE WHEN winner = 1 THEN points ELSE 0 END),
			       MAX(initial_score_2 + CASE WHEN winner = 2 THEN points ELSE 0 END)
			  FROM game WHERE match_id = ?`, m.id).Scan(&s1, &s2)
		if err != nil || !s1.Valid || !s2.Valid {
			continue
		}
		m.score1, m.score2, m.hasScore = int(s1.Int64), int(s2.Int64), true
		out[slot] = m
	}
	return out, nil
}

// slotDrafts finds the Transcriptions started from a Slot and not yet saved.
func (d *Database) slotDrafts(ctx context.Context, tournamentID int64) (map[string]int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	rows, err := d.db.QueryContext(ctx,
		`SELECT id, document FROM transcription WHERE match_id IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("reading drafts: %w", err)
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var id int64
		var doc string
		if err := rows.Scan(&id, &doc); err != nil {
			return nil, err
		}
		slot, tid := draftSlot(doc)
		if slot != "" && tid == tournamentID {
			out[slot] = id
		}
	}
	return out, rows.Err()
}

// draftSlot reads the Slot a draft was started from out of its document. The draft carries it
// in its header's round field, prefixed, because a Transcription's header is the one place a
// draft has for what it came from — and it must survive a save into the Match.
func draftSlot(document string) (string, int64) {
	var doc struct {
		Header struct {
			Round        string `json:"round"`
			TournamentID *int64 `json:"tournament_id"`
		} `json:"header"`
	}
	if err := json.Unmarshal([]byte(document), &doc); err != nil {
		return "", 0
	}
	slot := slotFromRound(doc.Header.Round)
	if slot == "" || doc.Header.TournamentID == nil {
		return "", 0
	}
	return slot, *doc.Header.TournamentID
}

// slotMarker is what a Slot looks like inside a round label. It is deliberately visible: a
// director who opens the .mat sees which match of the tournament it was.
const slotMarker = "#"

func slotFromRound(round string) string {
	i := strings.LastIndex(round, slotMarker)
	if i < 0 {
		return ""
	}
	return strings.TrimSpace(round[i+len(slotMarker):])
}

// disagreement compares what the director recorded with what the attached Match's own file
// says. It returns a description, or "" when they agree — and it never changes either.
func disagreement(m *tournoi.Match, f filledMatch) string {
	if f.length > 0 && m.Length > 0 && f.length != m.Length {
		return fmt.Sprintf("length %d vs %d", f.length, m.Length)
	}
	if !f.hasScore || (m.ScoreA == 0 && m.ScoreB == 0) {
		// The director entered no score, or the file has none: there is nothing to disagree
		// about. A result with no score is an ordinary result (ADR-0047 §5.3).
		return ""
	}
	// The Match's player1 is not necessarily the Slot's A.
	a, b := f.score1, f.score2
	if !strings.EqualFold(f.player1, string(m.A)) && strings.EqualFold(f.player2, string(m.A)) {
		a, b = f.score2, f.score1
	}
	if a != m.ScoreA || b != m.ScoreB {
		return fmt.Sprintf("score %d-%d vs %d-%d", a, b, m.ScoreA, m.ScoreB)
	}
	return ""
}

// AttachMatchToSlot fills a Slot with a Match of the library. It is always an explicit gesture:
// a coincidence of names is a suggestion, never a decision (ADR-0047 §7.1).
//
// Attaching also puts the Match in the Tournament if it was not there, since a Match filling a
// Slot of that tournament is a match OF that tournament.
func (d *Database) AttachMatchToSlot(tournamentID int64, slotID string, matchID int64) error {
	if slotID == "" {
		return fmt.Errorf("direction: no slot given")
	}
	ctx := context.Background()
	dir, err := direction.Open(ctx, d.DirectionStore(), tournamentID)
	if err != nil {
		return err
	}
	st := dir.State()
	if st == nil || st.Matches[tournoi.MatchID(slotID)] == nil {
		return fmt.Errorf("direction: no match %q in this tournament", slotID)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	// A Match fills at most one Slot: leaving the one it held is part of attaching it here.
	if _, err := tx.ExecContext(ctx,
		`UPDATE match SET tournament_id = ?, direction_match_id = ? WHERE id = ?`,
		tournamentID, slotID, matchID); err != nil {
		return fmt.Errorf("attaching match %d to slot %s: %w", matchID, slotID, err)
	}
	return tx.Commit()
}

// DetachMatchFromSlot empties a Slot. It touches neither the Match nor the recorded result:
// the Match keeps its Tournament, and the Slot keeps what the director said happened.
func (d *Database) DetachMatchFromSlot(tournamentID int64, slotID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.ExecContext(context.Background(),
		`UPDATE match SET direction_match_id = '' WHERE tournament_id = ? AND direction_match_id = ?`,
		tournamentID, slotID)
	return err
}

// SlotSuggestion is an unattached Match of a Tournament, with the Slot whose two names match.
type SlotSuggestion struct {
	MatchID     int64  `json:"matchId"`
	Player1     string `json:"player1"`
	Player2     string `json:"player2"`
	Length      int    `json:"length"`
	Date        string `json:"date,omitempty"`
	SuggestSlot string `json:"suggestSlot,omitempty"`
	SlotLabel   string `json:"slotLabel,omitempty"`
}

// UnattachedMatches lists the Matches of a Tournament that fill no Slot, each with the Slot
// whose two names coincide — when one does.
//
// The suggestion requires BOTH names and the same Tournament. A partial match is not suggested:
// half a coincidence is not evidence, and the director would have to check it anyway.
func (d *Database) UnattachedMatches(tournamentID int64) ([]SlotSuggestion, error) {
	ctx := context.Background()
	dir, err := direction.Open(ctx, d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	st := dir.State()

	out, err := d.unattachedMatchRows(ctx, tournamentID)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return out, nil
	}

	taken, err := d.slotMatches(ctx, tournamentID)
	if err != nil {
		return nil, err
	}
	for i := range out {
		for _, id := range st.MatchOrder {
			m := st.Matches[id]
			if m == nil || m.Status == tournoi.Cancelled {
				continue
			}
			if _, filled := taken[string(m.ID)]; filled {
				continue
			}
			a, b := playerNameIn(st, m.A), playerNameIn(st, m.B)
			if namesMatch(out[i].Player1, out[i].Player2, a, b) {
				out[i].SuggestSlot = string(m.ID)
				out[i].SlotLabel = a + " – " + b
				break
			}
		}
	}
	return out, nil
}

// unattachedMatchRows reads the Matches of a Tournament that fill no Slot. The read lock and
// the cursor are released together, by defer, before the caller goes on to slotMatches: the
// lock is not reentrant, and a cursor left open would outlive the read it belongs to.
func (d *Database) unattachedMatchRows(ctx context.Context, tournamentID int64) ([]SlotSuggestion, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	rows, err := d.db.QueryContext(ctx, `
		SELECT id, COALESCE(player1_name,''), COALESCE(player2_name,''),
		       COALESCE(match_length,0), COALESCE(match_date,'')
		  FROM match
		 WHERE tournament_id = ? AND (direction_match_id IS NULL OR direction_match_id = '')
		 ORDER BY id`, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("listing unattached matches: %w", err)
	}
	defer rows.Close()
	var out []SlotSuggestion
	for rows.Next() {
		var s SlotSuggestion
		if err := rows.Scan(&s.MatchID, &s.Player1, &s.Player2, &s.Length, &s.Date); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// namesMatch: both names, in either order, ignoring case and surrounding space.
func namesMatch(p1, p2, a, b string) bool {
	n := func(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
	p1, p2, a, b = n(p1), n(p2), n(a), n(b)
	if p1 == "" || p2 == "" || a == "" || b == "" {
		return false
	}
	return (p1 == a && p2 == b) || (p1 == b && p2 == a)
}

// StartTranscriptionFromSlot opens a draft with its header already filled from the Slot: both
// names, the length, the tournament, the round or the bracket label, and the date.
//
// The Slot is reserved FROM THE DRAFT, not only from the save (ADR-0047 §7.1): a director who
// starts typing a match must see the Slot taken, or two people will type the same match.
func (d *Database) StartTranscriptionFromSlot(tournamentID int64, slotID string) (*TranscriptionState, error) {
	ctx := context.Background()
	dir, err := direction.Open(ctx, d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	st := dir.State()
	if st == nil {
		return nil, direction.ErrNoDirection
	}
	m := st.Matches[tournoi.MatchID(slotID)]
	if m == nil {
		return nil, fmt.Errorf("direction: no match %q in this tournament", slotID)
	}
	name, date, location := d.tournamentHeader(ctx, tournamentID)
	tid := tournamentID
	h := transcript.Header{
		MatchLength:  m.Length,
		Player1:      playerNameIn(st, m.A),
		Player2:      playerNameIn(st, m.B),
		Event:        name,
		Location:     location,
		Round:        roundLabelFor(slotID),
		Date:         date,
		TournamentID: &tid,
	}
	return d.CreateTranscription(h)
}

// roundLabelFor writes the round a match belongs to, ending with the Slot marker so a draft can
// be traced back to its place — and so the .mat a director opens says which match it was.
func roundLabelFor(slotID string) string {
	// The engine's label is a CODE and must not be rendered here; what travels is the Slot,
	// which the frontend renders alongside the label it already knows how to translate.
	return slotMarker + slotID
}

func (d *Database) tournamentHeader(ctx context.Context, tournamentID int64) (string, time.Time, string) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var name, date, location sql.NullString
	_ = d.db.QueryRowContext(ctx,
		`SELECT name, date, location FROM tournament WHERE id = ?`, tournamentID).
		Scan(&name, &date, &location)
	var when time.Time
	if date.Valid && date.String != "" {
		if t, err := time.Parse("2006-01-02", date.String); err == nil {
			when = t
		}
	}
	if when.IsZero() {
		when = time.Now()
	}
	return name.String, when, location.String
}

// MatchSlot says which Slot a Match fills, for the Match panel to show where it comes from.
type MatchSlot struct {
	TournamentID   int64         `json:"tournamentId"`
	TournamentName string        `json:"tournamentName"`
	SlotID         string        `json:"slotId"`
	Label          tournoi.Label `json:"label"`
	Phase          int           `json:"phase"`
	Table          int           `json:"table,omitempty"`
	Opponent       string        `json:"opponent,omitempty"`
}

// SlotOfMatch returns the Slot a Match fills, or nil when it fills none — which is the ordinary
// case and not an error.
func (d *Database) SlotOfMatch(matchID int64) (*MatchSlot, error) {
	ctx := context.Background()
	d.mu.RLock()
	var tid sql.NullInt64
	var slot, tname sql.NullString
	err := d.db.QueryRowContext(ctx, `
		SELECT m.tournament_id, m.direction_match_id, t.name
		  FROM match m LEFT JOIN tournament t ON t.id = m.tournament_id
		 WHERE m.id = ?`, matchID).Scan(&tid, &slot, &tname)
	d.mu.RUnlock()
	if err != nil || !tid.Valid || !slot.Valid || slot.String == "" {
		return nil, nil
	}
	dir, err := direction.Open(ctx, d.DirectionStore(), tid.Int64)
	if err != nil {
		return nil, nil
	}
	st := dir.State()
	if st == nil {
		return nil, nil
	}
	m := st.Matches[tournoi.MatchID(slot.String)]
	if m == nil {
		return nil, nil
	}
	return &MatchSlot{
		TournamentID: tid.Int64, TournamentName: tname.String,
		SlotID: slot.String, Label: m.Label, Phase: m.Phase, Table: m.Table,
	}, nil
}
