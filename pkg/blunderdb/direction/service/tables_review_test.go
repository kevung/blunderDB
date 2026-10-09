package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction/service"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// relabel moves every table of room from into room to.
func relabel(settings []domain.TableSetting, from, to string) []domain.TableSetting {
	out := make([]domain.TableSetting, len(settings))
	for i, s := range settings {
		if s.Room == from {
			s.Room = to
		}
		out[i] = s
	}
	return out
}

func TestTablesThatLeaveAnEventNoTableAreRefused(t *testing.T) {
	ctx, svc, rid, _, _ := twoRoomFestival(t)
	r, err := svc.GetRencontre(ctx, rid)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.SetRencontreTables(ctx, rid, relabel(r.TableSettings, "B", "C"))
	if !errors.Is(err, direction.ErrRefused) || !strings.Contains(err.Error(), "DMP") {
		t.Fatalf("renaming room B, the DMP's only room: %v, want a refusal naming the DMP", err)
	}
	// With no room left at all the Rencontre is one room again: the DMP plays everywhere.
	if _, err := svc.SetRencontreTables(ctx, rid, relabel(relabel(r.TableSettings, "A", ""), "B", "")); err != nil {
		t.Errorf("clearing every room: %v", err)
	}
}

func playerIDs(t *testing.T, ctx context.Context, svc *service.Service, tid int64) []string {
	t.Helper()
	v, err := svc.GetDirection(ctx, tid)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, p := range v.Players {
		out = append(out, string(p.ID))
	}
	return out
}

func TestRestoringARencontreKeepsRunningMatchesInTheirRooms(t *testing.T) {
	ctx, svc, rid, _, side := twoRoomFestival(t)
	trashID, err := svc.TrashRencontre(ctx, rid)
	if err != nil {
		t.Fatal(err)
	}
	// Detached, the DMP plays on its own tables: one match goes to table 1, in room A.
	ids := playerIDs(t, ctx, svc, side)
	if _, err := svc.StartMatchManually(ctx, side, ids[0], ids[1], 7, 1); err != nil {
		t.Fatal(err)
	}
	_, err = svc.RestoreFromTrash(ctx, trashID)
	if !errors.Is(err, direction.ErrRefused) || !strings.Contains(err.Error(), "table 1") {
		t.Fatalf("restore with a DMP match on table 1: %v, want a refusal naming table 1", err)
	}
	if all, err := svc.ListRencontres(ctx); err != nil || len(all) != 0 {
		t.Errorf("a refused restore left %d Rencontres (%v)", len(all), err)
	}
	if of, _ := svc.RencontreOf(ctx, side); of != 0 {
		t.Errorf("a refused restore attached the DMP to %d", of)
	}

	ctx2, svc2, rid2, open2, side2 := twoRoomFestival(t)
	trash2, err := svc2.TrashRencontre(ctx2, rid2)
	if err != nil {
		t.Fatal(err)
	}
	restoredBack, err := svc2.RestoreFromTrash(ctx2, trash2)
	back := restoredBack.ID
	if err != nil {
		t.Fatal(err)
	}
	r, err := svc2.GetRencontre(ctx2, back)
	if err != nil || len(r.Members) != 2 {
		t.Fatalf("restored Rencontre %+v, %v", r, err)
	}
	for _, tid := range []int64{open2, side2} {
		if of, _ := svc2.RencontreOf(ctx2, tid); of != back {
			t.Errorf("tournament %d in %d after the restore, want %d", tid, of, back)
		}
	}
	if _, err := svc2.RestoreFromTrash(ctx2, trash2); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("restoring twice: %v, want ErrNotFound", err)
	}
}

func TestUpcomingSheetStaysInTheEventsRooms(t *testing.T) {
	ctx, svc, _, _, side := twoRoomFestival(t)
	sheet, err := svc.DirectionUpcomingSheetHTML(ctx, side, "lundi")
	if err != nil {
		t.Fatal(err)
	}
	for n := 1; n <= 3; n++ {
		if strings.Contains(sheet, fmt.Sprintf("<tr><td>%d</td>", n)) {
			t.Errorf("the DMP's sheet announces table %d, in room A", n)
		}
	}
	if !strings.Contains(sheet, "<tr><td>4</td>") || !strings.Contains(sheet, "<tr><td>—</td>") {
		t.Errorf("the DMP's sheet should seat three matches in room B and one waiting:\n%s", sheet)
	}
}

// roomlessPair is a Rencontre of six tables and no room, where a first event of eight players
// already plays on tables 1 to 4, and a second of four players is still to be proposed.
func roomlessPair(t *testing.T) (context.Context, *service.Service, int64) {
	t.Helper()
	ctx, svc, raw := openService(t)
	r, err := svc.CreateRencontre(ctx, "Club", "", "", 6)
	if err != nil {
		t.Fatal(err)
	}
	first := swissOf(t, ctx, svc, raw, "Premier", 6, names("P", 8)...)
	second := swissOf(t, ctx, svc, raw, "Second", 6, names("S", 4)...)
	for _, tid := range []int64{first, second} {
		if _, err := svc.AttachToRencontre(ctx, tid, r.ID); err != nil {
			t.Fatal(err)
		}
	}
	if v, err := svc.ConfirmAllProposals(ctx, first); err != nil || len(v.Running) != 4 {
		t.Fatalf("first event: %v", err)
	}
	return ctx, svc, second
}

func TestPageProposesAroundTheSisterEvents(t *testing.T) {
	ctx, svc, second := roomlessPair(t)
	page, err := svc.DirectionPageHTML(ctx, second)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(page, "<li>Table 1 — ") || !strings.Contains(page, "<li>Table 5 — ") {
		t.Errorf("the page should propose tables 5 and 6, which no sister plays on:\n%s", page)
	}
}

func proposalOn(t *testing.T, v *service.DirectionView, table int) string {
	t.Helper()
	for _, a := range v.Proposals {
		if a.Kind == tournoi.ActStartMatch && a.Table == table {
			blob, err := json.Marshal(a)
			if err != nil {
				t.Fatal(err)
			}
			return string(blob)
		}
	}
	t.Fatalf("no proposal on table %d in %+v", table, v.Proposals)
	return ""
}

func TestConfirmingAProposalOnAClosedTableIsRefused(t *testing.T) {
	ctx, svc, raw := openService(t)
	tid := swissOf(t, ctx, svc, raw, "Open", 4, names("P", 8)...)
	v, err := svc.GetDirection(ctx, tid)
	if err != nil {
		t.Fatal(err)
	}
	shown := proposalOn(t, v, 2)
	// Table 2 is reserved after the panel showed it: a client that states no version confirms
	// what it was shown.
	if _, err := svc.SetDirectionTables(ctx, tid, []domain.TableSetting{{Number: 2, Reserved: true}}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConfirmProposal(ctx, tid, shown); !errors.Is(err, direction.ErrRefused) {
		t.Errorf("confirming on the reserved table 2: %v, want refused", err)
	}

	ctx, svc, raw = openService(t)
	tid = swissOf(t, ctx, svc, raw, "Open", 6, "Alice", "Bob")
	v, err = svc.SetDirectionTables(ctx, tid, []domain.TableSetting{{Number: 5, AssignedTo: []string{"Alice"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConfirmProposal(ctx, tid, proposalOn(t, v, 5)); err != nil {
		t.Errorf("confirming Alice's match on her own table: %v", err)
	}
}

var errSettings = errors.New("table settings unreadable")

// unreadableSettings is a storage whose table properties cannot be read, in or out of a
// transaction.
type unreadableSettings struct{ storage.Storage }

func (u unreadableSettings) Rencontres() storage.RencontreStore {
	return unreadableRencontres{u.Storage.Rencontres()}
}

func (u unreadableSettings) BeginTx(ctx context.Context) (storage.Tx, error) {
	tx, err := u.Storage.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	return unreadableTx{tx}, nil
}

type unreadableTx struct{ storage.Tx }

func (u unreadableTx) Rencontres() storage.RencontreStore {
	return unreadableRencontres{u.Tx.Rencontres()}
}

type unreadableRencontres struct{ storage.RencontreStore }

func (unreadableRencontres) TournamentTableSettings(context.Context, string, int64) ([]domain.TableSetting, error) {
	return nil, errSettings
}

func TestUnreadableTablePropertiesAreReported(t *testing.T) {
	ctx, svc, raw := openService(t)
	tid := swissOf(t, ctx, svc, raw, "Open", 4, names("P", 4)...)
	ids := playerIDs(t, ctx, svc, tid)
	broken := service.New(unreadableSettings{raw}, "", nil)
	if _, err := broken.TableGrid(ctx, tid); !errors.Is(err, errSettings) {
		t.Errorf("TableGrid: %v, want the storage's error", err)
	}
	if _, err := broken.StartMatchManually(ctx, tid, ids[0], ids[1], 7, 0); !errors.Is(err, errSettings) {
		t.Errorf("StartMatchManually on the first free table: %v, want the storage's error", err)
	}
}
