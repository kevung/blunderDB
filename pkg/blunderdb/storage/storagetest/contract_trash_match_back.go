// Contract cases for a match restored into a library that moved on while it
// waited in the trash: a purged position stored again under another id, a
// Slot whose Direction is gone, a Tournament deleted. The table that runs
// them lives in contract.go.
package storagetest

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction/service"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/trash"
)

// testTrashMatchPositionBackElsewhere: the delete of match A purges P; an
// import of B stores P again under another id; restoring A ties P's study
// mark and training answers to that row, adds the notes it lacks, and leaves
// the analysis B brought. The match's last visited position 0 comes back.
func testTrashMatchPositionBackElsewhere(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ms := s.Matches()
	newMatchWith := func(p1, p2, hash string, posID int64) int64 {
		m := domain.Match{Player1Name: p1, Player2Name: p2, MatchLength: 5, MatchHash: hash}
		id, err := ms.Save(ctx, "", &m)
		if err != nil {
			t.Fatalf("Save match: %v", err)
		}
		gid, err := ms.CreateGame(ctx, "", &domain.Game{MatchID: id, GameNumber: 1})
		if err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
		if _, err := ms.CreateMove(ctx, "", &domain.Move{GameID: gid, MoveNumber: 1, MoveType: "checker",
			PositionID: posID, Player: 1, Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5"}); err != nil {
			t.Fatalf("CreateMove: %v", err)
		}
		return id
	}
	p := provenancePos(51)
	pid, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	a := newMatchWith("Ann", "Ben", "h-back-a", pid)
	if err := ms.SetLastVisitedPosition(ctx, "", a, 0); err != nil {
		t.Fatalf("SetLastVisitedPosition: %v", err)
	}
	if err := s.Analyses().Save(ctx, "", pid, verdictBy("XG", time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC))); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}
	if _, err := s.Comments().AddFrom(ctx, "", pid, "note of A", domain.CommentOriginXG); err != nil {
		t.Fatalf("Add comment: %v", err)
	}
	if err := s.ImportBatches().SetStudied(ctx, "", pid, true); err != nil {
		t.Fatalf("SetStudied: %v", err)
	}
	if _, err := s.Training().Save(ctx, "", storage.TrainingSession{Exercise: "decision",
		Items: []storage.TrainingItem{{NumberType: "decision", Wrong: true, PositionID: &pid}}}); err != nil {
		t.Fatalf("Save training session: %v", err)
	}

	entry, err := trash.Match(ctx, s, "", a)
	if err != nil {
		t.Fatalf("trash.Match: %v", err)
	}
	if _, err := s.Positions().Load(ctx, "", pid); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("the position held by the deleted match alone: %v, want purged", err)
	}

	// B brings P back, under a new id, with its own analysis and note.
	q := provenancePos(51)
	qid, err := s.Positions().Save(ctx, "", &q)
	if err != nil || qid == pid {
		t.Fatalf("Save of P again = %d, %v; want an id other than %d", qid, err, pid)
	}
	newMatchWith("Cid", "Dan", "h-back-b", qid)
	if err := s.Analyses().Save(ctx, "", qid, verdictBy("GNU", time.Date(2025, 2, 3, 0, 0, 0, 0, time.UTC))); err != nil {
		t.Fatalf("Save analysis of B: %v", err)
	}
	if _, err := s.Comments().Add(ctx, "", qid, "note of B"); err != nil {
		t.Fatalf("Add comment of B: %v", err)
	}

	res, err := trash.Restore(ctx, s, "", entry)
	if err != nil || res.ID != a {
		t.Fatalf("Restore = %+v, %v; want match %d", res, err, a)
	}
	var reached []int64
	for mv, err := range ms.MovesByMatch(ctx, "", a) {
		if err != nil {
			t.Fatalf("MovesByMatch: %v", err)
		}
		reached = append(reached, mv.PositionID)
	}
	if !slices.Equal(reached, []int64{qid}) {
		t.Errorf("restored moves reach %v, want [%d]", reached, qid)
	}
	if mark, err := s.ImportBatches().StudyMark(ctx, "", qid); err != nil || mark == 0 {
		t.Errorf("study mark of the position back elsewhere = %d, %v; want restored", mark, err)
	}
	if items, err := s.Training().ItemsOfPosition(ctx, "", qid); err != nil || len(items) != 1 {
		t.Errorf("training answers of the position back elsewhere = %v, %v; want 1", items, err)
	}
	if an, err := s.Analyses().Load(ctx, "", qid); err != nil || an.CheckerAnalysis == nil ||
		an.CheckerAnalysis.Moves[0].AnalysisEngine != "GNU" {
		t.Errorf("analysis of the position back elsewhere = %+v, %v; want B's, untouched", an, err)
	}
	var notes []string
	for c, err := range s.Comments().ByPosition(ctx, "", qid) {
		if err != nil {
			t.Fatalf("ByPosition: %v", err)
		}
		notes = append(notes, c.Text)
	}
	slices.Sort(notes)
	if !slices.Equal(notes, []string{"note of A", "note of B"}) {
		t.Errorf("notes of the position back elsewhere = %v, want both", notes)
	}
	if m, err := ms.Get(ctx, "", a); err != nil || m.LastVisitedPosition != 0 {
		t.Errorf("last visited position after restore = %+v, %v; want 0", m, err)
	}
}

// directedSlot builds a Tournament directed by the service, with one Slot
// running, and the match of the library that fills it. It returns the
// service, the Tournament, the Slot and the match.
func directedSlot(t *testing.T, s storage.Storage, name string) (*service.Service, int64, string, int64) {
	t.Helper()
	ctx := context.Background()
	svc := service.New(s, "", nil)
	tid, err := s.Tournaments().Create(ctx, "", name, "2026-09-12", "Lyon")
	if err != nil {
		t.Fatalf("Create tournament: %v", err)
	}
	cfg := `{"name":"` + name + `","tables":{"count":4},"phases":[
		{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous"}]}`
	if err := svc.CreateDirection(ctx, tid, cfg, 7); err != nil {
		t.Fatalf("CreateDirection: %v", err)
	}
	var players []string
	for _, id := range []string{"aa", "bb", "cc", "dd"} {
		players = append(players, `{"id":"`+id+`","name":"Joueur `+id+`"}`)
	}
	if err := svc.EnterParticipants(ctx, tid, "["+strings.Join(players, ",")+"]"); err != nil {
		t.Fatalf("EnterParticipants: %v", err)
	}
	v, err := svc.ConfirmAllProposals(ctx, tid)
	if err != nil || len(v.Running) == 0 {
		t.Fatalf("ConfirmAllProposals = %v; want matches running", err)
	}
	sm := v.Running[0]
	mid := newTournamentMatchOf(t, s, tid, "Joueur "+string(sm.A), "Joueur "+string(sm.B), 7)
	if err := svc.AttachMatchToSlot(ctx, tid, string(sm.ID), mid); err != nil {
		t.Fatalf("AttachMatchToSlot: %v", err)
	}
	return svc, tid, string(sm.ID), mid
}

// testTrashMatchSlotGone: a match whose Slot left its Direction comes back
// without it, warned; one whose Tournament was deleted comes back outside
// any, warned.
func testTrashMatchSlotGone(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	svc, tid, _, mid := directedSlot(t, s, "Gone")
	entry, err := trash.Match(ctx, s, "", mid)
	if err != nil {
		t.Fatalf("trash.Match: %v", err)
	}
	if err := svc.DeleteDirection(ctx, tid); err != nil {
		t.Fatalf("DeleteDirection: %v", err)
	}
	res, err := trash.Restore(ctx, s, "", entry)
	if err != nil || res.ID != mid || len(res.Warnings) != 1 || res.Warnings[0].Code != domain.TrashWarnSlotGone {
		t.Errorf("Restore with the Direction gone = %+v, %v; want a %s warning", res, err, domain.TrashWarnSlotGone)
	}
	if _, slot, err := s.Directions().SlotOf(ctx, "", mid); err != nil || slot != "" {
		t.Errorf("Slot of the restored match = %q, %v; want none", slot, err)
	}

	if entry, err = trash.Match(ctx, s, "", mid); err != nil {
		t.Fatalf("trash.Match again: %v", err)
	}
	if err := s.Tournaments().Delete(ctx, "", tid); err != nil {
		t.Fatalf("Delete tournament: %v", err)
	}
	res, err = trash.Restore(ctx, s, "", entry)
	if err != nil || res.ID != mid || len(res.Warnings) != 1 || res.Warnings[0].Code != domain.TrashWarnTournamentGone {
		t.Errorf("Restore with the Tournament gone = %+v, %v; want a %s warning", res, err, domain.TrashWarnTournamentGone)
	}
	if m, err := s.Matches().Get(ctx, "", mid); err != nil || m.TournamentID != nil {
		t.Errorf("restored match's tournament = %+v, %v; want none", m, err)
	}
}
