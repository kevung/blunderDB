package database

import (
	"strings"
	"testing"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
)

// Latecomers and lengths per round (issue #392).
//
// The rule these tests hold is the one the engine states and blunderDB must not soften: a draw
// already made is NEVER redrawn. The latecomer takes an empty place or enters later, and the
// interface says which BEFORE the entry is validated.

// drawnBracket prepares a bracket of n players and draws it, which is when free byes appear.
func drawnBracket(t *testing.T, d *Database, n int, phase string) int64 {
	t.Helper()
	tID, err := d.CreateTournament("Open de Lyon", "2026-09-12", "Lyon")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Open de Lyon","tables":{"count":16},"phases":[` + phase + `]}`
	if err := d.CreateDirection(tID, cfg, 7); err != nil {
		t.Fatal(err)
	}
	var players []string
	for i := 0; i < n; i++ {
		id := string(rune('a'+i%26)) + string(rune('a'+i/26))
		players = append(players, `{"id":"`+id+`","name":"Joueur `+id+`"}`)
	}
	if err := d.EnterParticipants(tID, "["+strings.Join(players, ",")+"]"); err != nil {
		t.Fatal(err)
	}
	// Confirm the draw, and only the draw: the byes must still be free.
	v, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range v.Proposals {
		if a.Kind == tournoi.ActDraw {
			if _, err := d.ConfirmProposal(tID, proposalJSON(t, a)); err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	return tID
}

// Six players in a bracket of eight: two byes, and a latecomer takes one without a word of
// warning — they simply appear in the bracket.
func TestLatecomer_TakesAFreeBye(t *testing.T) {
	d := newTestDB(t)
	tID := drawnBracket(t, d, 6, `{"kind":"bracket","length":5}`)

	slots, err := d.DirectionFreeSlots(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) == 0 {
		t.Fatal("a bracket of eight with six players should leave free byes")
	}
	before, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}

	v, err := d.AddParticipantAtSlot(tID, "Yanis Ferrand", "Lyon", 4, slots[0].Section, slots[0].Key)
	if err != nil {
		t.Fatalf("entering on a free bye: %v", err)
	}
	if len(v.Players) != len(before.Players)+1 {
		t.Errorf("%d players, expected %d", len(v.Players), len(before.Players)+1)
	}
	if len(v.Warnings) != len(before.Warnings) {
		t.Errorf("taking a free bye raised a warning: %+v", v.Warnings)
	}
	// And they are engaged: nothing left to say about where they enter.
	for _, i := range v.Infos {
		if i.Player == "yanis_ferrand" {
			t.Errorf("the latecomer is still waiting for a place: %+v", i)
		}
	}
	// One bye fewer.
	after, err := d.DirectionFreeSlots(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(slots)-1 {
		t.Errorf("%d free byes left, expected %d", len(after), len(slots)-1)
	}
}

// With no place left, the entry still happens and the view SAYS where they will enter — before
// the director validates, since the free byes are what they are shown.
func TestLatecomer_WithNoPlaceLeftTheViewSaysWhereTheyEnter(t *testing.T) {
	d := newTestDB(t)
	// Eight players in a bracket of eight: no bye at all. A consolation follows, open to all.
	tID := drawnBracket(t, d, 8, `{"kind":"bracket","length":5},{"kind":"bracket","length":5,"entry":"all"}`)

	slots, err := d.DirectionFreeSlots(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) != 0 {
		t.Fatalf("a full bracket should leave no free bye, got %+v", slots)
	}

	v, err := d.AddParticipant(tID, "Yanis Ferrand", "Lyon", 4)
	if err != nil {
		t.Fatal(err)
	}
	var info *tournoi.Info
	for i := range v.Infos {
		if strings.HasPrefix(string(v.Infos[i].Player), "yanis") {
			info = &v.Infos[i]
		}
	}
	if info == nil {
		t.Fatalf("nothing said about where the latecomer enters: %+v", v.Infos)
	}
	if info.Code != tournoi.InfoEntersAt {
		t.Errorf("the latecomer's info reads %+v", info)
	}
	if info.Phase != 1 {
		t.Errorf("they should enter in the second phase, got %d", info.Phase)
	}
}

// Taking a place already played is refused: the engine does not guess what the director meant.
func TestLatecomer_APlaceAlreadyPlayedIsRefused(t *testing.T) {
	d := newTestDB(t)
	tID := drawnBracket(t, d, 6, `{"kind":"bracket","length":5}`)
	slots, err := d.DirectionFreeSlots(tID)
	if err != nil || len(slots) == 0 {
		t.Fatalf("slots: %+v %v", slots, err)
	}
	if _, err := d.AddParticipantAtSlot(tID, "Yanis Ferrand", "", 0, slots[0].Section, slots[0].Key); err != nil {
		t.Fatal(err)
	}
	// The same place, twice.
	if _, err := d.AddParticipantAtSlot(tID, "Zoé Garnier", "", 0, slots[0].Section, slots[0].Key); err == nil {
		t.Error("a place already taken was accepted")
	}
	if _, err := d.AddParticipantAtSlot(tID, "Zoé Garnier", "", 0, slots[0].Section, "inexistante"); err == nil {
		t.Error("an unknown place was accepted")
	}
}

// Nothing already drawn is redrawn: the pairings of the drawn round are the same before and
// after a latecomer enters.
func TestLatecomer_NothingIsRedrawn(t *testing.T) {
	d := newTestDB(t)
	tID := drawnBracket(t, d, 6, `{"kind":"bracket","length":5}`)
	before, err := d.Brackets(tID)
	if err != nil {
		t.Fatal(err)
	}
	// Ce que « rien n'est retiré » veut dire : les appariements du PREMIER TOUR entre deux
	// joueurs réels — ceux que les joueurs ont lus sur le mur — ne bougent pas. Une place
	// d'exemption, elle, change forcément : c'est celle que le retardataire prend, et le
	// joueur qui allait passer sans jouer joue désormais.
	pairings := func(phases []BracketPhase) []string {
		var out []string
		for _, ph := range phases {
			for _, sec := range ph.Sections {
				for _, m := range sec.Matches {
					if m.Round != 0 || m.A == "" || m.B == "" {
						continue
					}
					if m.A == string(tournoi.BYE) || m.B == string(tournoi.BYE) {
						continue
					}
					out = append(out, m.A+"|"+m.B)
				}
			}
		}
		return out
	}
	was := pairings(before)
	if len(was) == 0 {
		t.Fatal("no real first-round pairing: the test would prove nothing")
	}

	slots, err := d.DirectionFreeSlots(tID)
	if err != nil || len(slots) == 0 {
		t.Fatalf("slots: %+v %v", slots, err)
	}
	if _, err := d.AddParticipantAtSlot(tID, "Yanis Ferrand", "", 0, slots[0].Section, slots[0].Key); err != nil {
		t.Fatal(err)
	}
	after, err := d.Brackets(tID)
	if err != nil {
		t.Fatal(err)
	}
	now := pairings(after)
	for _, p := range was {
		found := false
		for _, q := range now {
			if p == q {
				found = true
			}
		}
		if !found {
			t.Errorf("the pairing %s disappeared: a draw already made was redrawn", p)
		}
	}
}

// Lengths per round: "15, 13, 11" reads from the LAST round backwards, which is the order an
// organiser announces a tournament in. Each proposed match carries the length of its round.
func TestLengths_PerRoundReachTheProposals(t *testing.T) {
	d := newTestDB(t)
	tID := drawnBracket(t, d, 8, `{"kind":"bracket","length":9,"lengths":[15,13,11]}`)

	v, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	// Round 1 of a bracket of eight is the quarter-finals: third from the end, so 11.
	seen := map[int]int{}
	for _, a := range v.Proposals {
		if a.Kind == tournoi.ActStartMatch {
			seen[a.Length]++
		}
	}
	if seen[11] == 0 {
		t.Fatalf("no quarter-final proposed at 11 points: %v", seen)
	}
	if len(seen) != 1 {
		t.Errorf("the first round mixes lengths: %v", seen)
	}
}
