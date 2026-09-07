package database

import (
	"strings"
	"testing"
)

// TestSlotsAreEmptyUntilSomeoneFillsThem: a club tournament where nothing is recorded leaves
// every Slot empty, and that is the ordinary case — not a gap to fill by inference.
func TestSlotsAreEmptyUntilSomeoneFillsThem(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)
	if _, err := d.ConfirmAllProposals(tID); err != nil {
		t.Fatal(err)
	}

	slots, err := d.Slots(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) == 0 {
		t.Fatal("matches were launched, so slots exist")
	}
	for _, s := range slots {
		if s.MatchID != 0 || s.DraftID != 0 {
			t.Errorf("slot %s is filled by nothing yet: %+v", s.SlotID, s)
		}
		if s.AName == "" || s.AName == s.A {
			t.Errorf("a slot shows names: %+v", s)
		}
	}
}

// TestAttachIsAlwaysAGesture: a coincidence of names is a SUGGESTION. Importing files attaches
// nothing on its own.
func TestAttachIsAlwaysAGesture(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)
	v, err := d.ConfirmAllProposals(tID)
	if err != nil {
		t.Fatal(err)
	}
	m := v.Running[0]
	slots, err := d.Slots(tID)
	if err != nil {
		t.Fatal(err)
	}
	var slot SlotRow
	for _, s := range slots {
		if s.SlotID == string(m.ID) {
			slot = s
		}
	}
	if slot.SlotID == "" {
		t.Fatal("slot not found")
	}

	// A Match of this tournament, named exactly like the slot's two players.
	res, err := RawConn(d).Exec(
		`INSERT INTO match (player1_name, player2_name, match_length, tournament_id) VALUES (?, ?, ?, ?)`,
		slot.AName, slot.BName, slot.Length, tID)
	if err != nil {
		t.Fatal(err)
	}
	matchID, _ := res.LastInsertId()

	// It is suggested, and it is NOT attached.
	sugg, err := d.UnattachedMatches(tID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range sugg {
		if s.MatchID == matchID {
			found = true
			if s.SuggestSlot != slot.SlotID {
				t.Errorf("the slot with both names should be suggested: %+v", s)
			}
		}
	}
	if !found {
		t.Fatal("the match is unattached and must be listed")
	}
	again, err := d.Slots(tID)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range again {
		if s.MatchID != 0 {
			t.Errorf("nothing is attached without a gesture: %+v", s)
		}
	}

	// The gesture.
	if err := d.AttachMatchToSlot(tID, slot.SlotID, matchID); err != nil {
		t.Fatalf("attaching: %v", err)
	}
	again, err = d.Slots(tID)
	if err != nil {
		t.Fatal(err)
	}
	attached := false
	for _, s := range again {
		if s.SlotID == slot.SlotID && s.MatchID == matchID {
			attached = true
		}
	}
	if !attached {
		t.Error("the match should now fill its slot")
	}
	// And it no longer shows as unattached.
	sugg, err = d.UnattachedMatches(tID)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range sugg {
		if s.MatchID == matchID {
			t.Error("an attached match is no longer unattached")
		}
	}
}

// TestPartialNameMatchIsNotSuggested: half a coincidence is not evidence. The director would
// have to check it anyway, so suggesting it only wastes their attention.
func TestPartialNameMatchIsNotSuggested(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)
	if _, err := d.ConfirmAllProposals(tID); err != nil {
		t.Fatal(err)
	}
	slots, err := d.Slots(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) == 0 {
		t.Fatal("no slot")
	}
	// One name right, one wrong.
	res, err := RawConn(d).Exec(
		`INSERT INTO match (player1_name, player2_name, match_length, tournament_id) VALUES (?, 'Quelqu''un d''autre', 7, ?)`,
		slots[0].AName, tID)
	if err != nil {
		t.Fatal(err)
	}
	matchID, _ := res.LastInsertId()

	sugg, err := d.UnattachedMatches(tID)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range sugg {
		if s.MatchID == matchID && s.SuggestSlot != "" {
			t.Errorf("a partial name match must not be suggested: %+v", s)
		}
	}
}

// TestDisagreementIsShownNeverResolved: during a tournament it is the director's word that
// stands. A file that says otherwise is flagged, and neither side is changed.
func TestDisagreementIsShownNeverResolved(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)
	v, err := d.ConfirmAllProposals(tID)
	if err != nil {
		t.Fatal(err)
	}
	m := v.Running[0]
	if _, err := d.EnterResult(tID, string(m.ID), string(m.A), 7, 4, ""); err != nil {
		t.Fatal(err)
	}
	// A Match whose own length disagrees with what the director recorded.
	res, err := RawConn(d).Exec(
		`INSERT INTO match (player1_name, player2_name, match_length, tournament_id) VALUES ('a', 'b', 9, ?)`, tID)
	if err != nil {
		t.Fatal(err)
	}
	matchID, _ := res.LastInsertId()
	if err := d.AttachMatchToSlot(tID, string(m.ID), matchID); err != nil {
		t.Fatal(err)
	}

	slots, err := d.Slots(tID)
	if err != nil {
		t.Fatal(err)
	}
	flagged := false
	for _, s := range slots {
		if s.SlotID != string(m.ID) {
			continue
		}
		if s.Disagreement == "" {
			t.Errorf("a length of 9 against a match of 7 must be flagged: %+v", s)
		} else {
			flagged = true
		}
		// And the director's own result is untouched.
		if s.ScoreA != 7 || s.ScoreB != 4 {
			t.Errorf("the recorded result must not be overruled: %+v", s)
		}
	}
	if !flagged {
		t.Error("the disagreement should be visible on the slot")
	}
}

// TestDetachKeepsEverything: emptying a Slot touches neither the Match nor the result.
func TestDetachKeepsEverything(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)
	v, err := d.ConfirmAllProposals(tID)
	if err != nil {
		t.Fatal(err)
	}
	m := v.Running[0]
	res, err := RawConn(d).Exec(
		`INSERT INTO match (player1_name, player2_name, match_length, tournament_id) VALUES ('a','b',7,?)`, tID)
	if err != nil {
		t.Fatal(err)
	}
	matchID, _ := res.LastInsertId()
	if err := d.AttachMatchToSlot(tID, string(m.ID), matchID); err != nil {
		t.Fatal(err)
	}
	if err := d.DetachMatchFromSlot(tID, string(m.ID)); err != nil {
		t.Fatal(err)
	}
	var tid *int64
	var slot string
	if err := RawConn(d).QueryRow(
		`SELECT tournament_id, direction_match_id FROM match WHERE id = ?`, matchID).Scan(&tid, &slot); err != nil {
		t.Fatalf("the match must still be there: %v", err)
	}
	if slot != "" {
		t.Errorf("the slot should be empty, it holds %q", slot)
	}
	if tid == nil || *tid != tID {
		t.Error("the match keeps its tournament")
	}
}

// TestTranscribeFromSlotInheritsTheHeaderAndReservesTheSlot: a director who starts typing a
// match must see the Slot taken, or two people will type the same match.
func TestTranscribeFromSlotInheritsTheHeaderAndReservesTheSlot(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)
	v, err := d.ConfirmAllProposals(tID)
	if err != nil {
		t.Fatal(err)
	}
	m := v.Running[0]

	st, err := d.StartTranscriptionFromSlot(tID, string(m.ID))
	if err != nil {
		t.Fatalf("starting a transcription from a slot: %v", err)
	}
	if st == nil {
		t.Fatal("no draft")
	}

	slots, err := d.Slots(tID)
	if err != nil {
		t.Fatal(err)
	}
	reserved := false
	for _, s := range slots {
		if s.SlotID == string(m.ID) && s.DraftID != 0 {
			reserved = true
		}
	}
	if !reserved {
		t.Error("the slot must be reserved from the draft, not only from the save")
	}

	// The header is filled: nothing to retype.
	doc, err := d.OpenTranscription(st.ID)
	if err != nil {
		t.Fatal(err)
	}
	blob, err := RawConn(d).Query(`SELECT document FROM transcription WHERE id = ?`, st.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer blob.Close()
	var document string
	if blob.Next() {
		if err := blob.Scan(&document); err != nil {
			t.Fatal(err)
		}
	}
	for _, want := range []string{"Open de Lyon", string(m.ID)} {
		if !strings.Contains(document, want) {
			t.Errorf("the header should carry %q: %s", want, document[:min(400, len(document))])
		}
	}
	_ = doc
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestSlotOfMatch: the Match panel shows where a Match comes from; a Match filling no Slot
// shows nothing extra, which is the ordinary case.
func TestSlotOfMatch(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)
	v, err := d.ConfirmAllProposals(tID)
	if err != nil {
		t.Fatal(err)
	}
	m := v.Running[0]
	res, err := RawConn(d).Exec(
		`INSERT INTO match (player1_name, player2_name, match_length, tournament_id) VALUES ('a','b',7,?)`, tID)
	if err != nil {
		t.Fatal(err)
	}
	matchID, _ := res.LastInsertId()

	got, err := d.SlotOfMatch(matchID)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("a match filling no slot shows nothing: %+v", got)
	}

	if err := d.AttachMatchToSlot(tID, string(m.ID), matchID); err != nil {
		t.Fatal(err)
	}
	got, err = d.SlotOfMatch(matchID)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.SlotID != string(m.ID) {
		t.Fatalf("the match should name its slot: %+v", got)
	}
	if got.TournamentName != "Open de Lyon" {
		t.Errorf("and its tournament: %+v", got)
	}
	if got.Label.Kind == "" {
		t.Error("the label is the engine's code, for the frontend to render")
	}
}
