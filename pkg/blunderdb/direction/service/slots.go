package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// Slots: where a directed Tournament meets the library (ADR-0047). Every launched match is a
// Slot that a Transcription or an imported file then fills.
//
// NOTHING IS ATTACHED BY INFERENCE: a coincidence of names is a suggestion the director
// accepts, never a decision the software makes.

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
	// stands (tasks/nicomaque/fonctionnel.md §5.3).
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
func (d *Service) Slots(ctx context.Context, tournamentID int64) ([]SlotRow, error) {
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
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
			r.Disagreement = disagreement(m, r.AName, mt)
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

func (d *Service) slotMatches(ctx context.Context, tournamentID int64) (map[string]filledMatch, error) {
	rows, err := d.st.Directions().FilledSlots(ctx, d.scope, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("reading slots: %w", err)
	}
	out := make(map[string]filledMatch, len(rows))
	for _, r := range rows {
		out[r.SlotID] = filledMatch{
			id: r.MatchID, player1: r.Player1, player2: r.Player2, length: r.Length,
			score1: r.Score1, score2: r.Score2, hasScore: r.HasScore,
		}
	}
	return out, nil
}

// slotDrafts finds the Transcriptions started from a Slot and not yet saved. When a Slot has
// several, it leads to the one typed last: List gives the most recently updated first.
func (d *Service) slotDrafts(ctx context.Context, tournamentID int64) (map[string]int64, error) {
	out := map[string]int64{}
	for t, err := range d.st.Transcriptions().List(ctx, d.scope) {
		if err != nil {
			return nil, fmt.Errorf("reading drafts: %w", err)
		}
		if t.MatchID != 0 {
			continue
		}
		slot, tid := draftSlot(t.Document)
		if _, seen := out[slot]; slot != "" && tid == tournamentID && !seen {
			out[slot] = t.ID
		}
	}
	return out, nil
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
// aName is the name of the Slot's A, which is what a Match's file knows its players by.
func disagreement(m *tournoi.Match, aName string, f filledMatch) string {
	if f.length > 0 && m.Length > 0 && f.length != m.Length {
		return fmt.Sprintf("length %d vs %d", f.length, m.Length)
	}
	if !f.hasScore || (m.ScoreA == 0 && m.ScoreB == 0) {
		// The director entered no score, or the file has none: there is nothing to disagree
		// about. A result with no score is an ordinary result (tasks/nicomaque/fonctionnel.md §5.3).
		return ""
	}
	// The Match's player1 is not necessarily the Slot's A.
	a, b := f.score1, f.score2
	if !strings.EqualFold(f.player1, aName) && strings.EqualFold(f.player2, aName) {
		a, b = f.score2, f.score1
	}
	if a != m.ScoreA || b != m.ScoreB {
		return fmt.Sprintf("score %d-%d vs %d-%d", a, b, m.ScoreA, m.ScoreB)
	}
	return ""
}

// AttachMatchToSlot fills a Slot with a Match of the library. It is always an explicit gesture:
// a coincidence of names is a suggestion, never a decision (tasks/nicomaque/fonctionnel.md §7.1).
//
// Attaching also puts the Match in the Tournament if it was not there, since a Match filling a
// Slot of that tournament is a match OF that tournament.
func (d *Service) AttachMatchToSlot(ctx context.Context, tournamentID int64, slotID string, matchID int64) (err error) {
	d, release, err := d.lockDirection(ctx, tournamentID)
	if err != nil {
		return err
	}
	defer release(&err)
	if slotID == "" {
		return direction.Refusef("direction: no slot given")
	}
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return err
	}
	st := dir.State()
	if st == nil || st.Matches[tournoi.MatchID(slotID)] == nil {
		return direction.Refusef("direction: no match %q in this tournament", slotID)
	}
	// A Match fills at most one Slot: leaving the one it held is part of attaching it here.
	if err := d.st.Directions().AttachSlot(ctx, d.scope, tournamentID, slotID, matchID); err != nil {
		return fmt.Errorf("attaching match %d to slot %s: %w", matchID, slotID, err)
	}
	return nil
}

// DetachMatchFromSlot empties a Slot. It touches neither the Match nor the recorded result:
// the Match keeps its Tournament, and the Slot keeps what the director said happened.
func (d *Service) DetachMatchFromSlot(ctx context.Context, tournamentID int64, slotID string) (err error) {
	d, release, err := d.lockDirection(ctx, tournamentID)
	if err != nil {
		return err
	}
	defer release(&err)
	return d.st.Directions().DetachSlot(ctx, d.scope, tournamentID, slotID)
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
// The suggestion requires BOTH names and the same Tournament: half a coincidence is not evidence.
func (d *Service) UnattachedMatches(ctx context.Context, tournamentID int64) ([]SlotSuggestion, error) {
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
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

// unattachedMatchRows reads the Matches of a Tournament that fill no Slot.
func (d *Service) unattachedMatchRows(ctx context.Context, tournamentID int64) ([]SlotSuggestion, error) {
	rows, err := d.st.Directions().UnattachedMatches(ctx, d.scope, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("listing unattached matches: %w", err)
	}
	var out []SlotSuggestion
	for _, r := range rows {
		out = append(out, SlotSuggestion{MatchID: r.MatchID, Player1: r.Player1, Player2: r.Player2, Length: r.Length, Date: r.Date})
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

// SlotHeader is the header a draft started from a Slot opens with: both names, the length, the
// tournament, the round or the bracket label, and the date.
//
// The Slot is reserved FROM THE DRAFT, not only from the save (tasks/nicomaque/fonctionnel.md §7.1): a director who
// starts typing a match must see the Slot taken, or two people will type the same match.
func (d *Service) SlotHeader(ctx context.Context, tournamentID int64, slotID string) (transcript.Header, error) {
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return transcript.Header{}, err
	}
	st := dir.State()
	if st == nil {
		return transcript.Header{}, direction.ErrNoDirection
	}
	m := st.Matches[tournoi.MatchID(slotID)]
	if m == nil {
		return transcript.Header{}, direction.Refusef("direction: no match %q in this tournament", slotID)
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
	return h, nil
}

// roundLabelFor writes the round a match belongs to, ending with the Slot marker so a draft can
// be traced back to its place — and so the .mat a director opens says which match it was.
func roundLabelFor(slotID string) string {
	// The engine's label is a CODE and must not be rendered here; what travels is the Slot,
	// which the frontend renders alongside the label it already knows how to translate.
	return slotMarker + slotID
}

func (d *Service) tournamentHeader(ctx context.Context, tournamentID int64) (string, time.Time, string) {
	var name, date, location string
	if t, err := d.st.Tournaments().Get(ctx, d.scope, tournamentID); err == nil && t != nil {
		name, date, location = t.Name, t.Date, t.Location
	}
	var when time.Time
	if date != "" {
		if t, err := time.Parse("2006-01-02", date); err == nil {
			when = t
		}
	}
	if when.IsZero() {
		when = time.Now()
	}
	return name, when, location
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
func (d *Service) SlotOfMatch(ctx context.Context, matchID int64) (*MatchSlot, error) {
	tid, slot, err := d.st.Directions().SlotOf(ctx, d.scope, matchID)
	if err != nil || slot == "" {
		return nil, nil
	}
	var tname string
	if t, err := d.st.Tournaments().Get(ctx, d.scope, tid); err == nil && t != nil {
		tname = t.Name
	}
	dir, err := direction.Open(ctx, d.dirStore(), tid)
	if err != nil {
		return nil, nil
	}
	st := dir.State()
	if st == nil {
		return nil, nil
	}
	m := st.Matches[tournoi.MatchID(slot)]
	if m == nil {
		return nil, nil
	}
	return &MatchSlot{
		TournamentID: tid, TournamentName: tname,
		SlotID: slot, Label: m.Label, Phase: m.Phase, Table: m.Table,
	}, nil
}
