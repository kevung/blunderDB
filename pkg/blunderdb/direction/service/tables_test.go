package service_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction/service"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func openService(t *testing.T) (context.Context, *service.Service, storage.Storage) {
	t.Helper()
	ctx := context.Background()
	st, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return ctx, service.New(st, "", nil), st
}

// swissOf directs a continuous Swiss event of the given players on that many tables.
func swissOf(t *testing.T, ctx context.Context, svc *service.Service, raw storage.Storage, name string, tables int, players ...string) int64 {
	t.Helper()
	tid, err := raw.Tournaments().Create(ctx, "", name, "2026-09-12", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := fmt.Sprintf(`{"name":%q,"tables":{"count":%d},"phases":[{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous"}]}`, name, tables)
	if err := svc.CreateDirection(ctx, tid, cfg, 7); err != nil {
		t.Fatal(err)
	}
	for i, p := range players {
		if _, err := svc.AddParticipant(ctx, tid, p, "", float64(1600-10*i)); err != nil {
			t.Fatal(err)
		}
	}
	return tid
}

func names(prefix string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("%s %d", prefix, i+1)
	}
	return out
}

func startTables(acts []tournoi.Action) (placed []int, waiting int) {
	for _, a := range acts {
		if a.Kind != tournoi.ActStartMatch {
			continue
		}
		if a.Table > 0 {
			placed = append(placed, a.Table)
		} else if a.Reason == tournoi.ReasonWaitingTable {
			waiting++
		}
	}
	return placed, waiting
}

// twoRoomFestival is a Rencontre of six tables in two rooms, A (1-3) and B (4-6), with an open
// event playing everywhere and a doubles-like side event restricted to room B.
func twoRoomFestival(t *testing.T) (context.Context, *service.Service, int64, int64, int64) {
	t.Helper()
	ctx, svc, raw := openService(t)
	r, err := svc.CreateRencontre(ctx, "Festival", "", "", 6)
	if err != nil {
		t.Fatal(err)
	}
	var settings []domain.TableSetting
	for n := 1; n <= 6; n++ {
		room := "A"
		if n > 3 {
			room = "B"
		}
		settings = append(settings, domain.TableSetting{Number: n, Room: room})
	}
	if _, err := svc.SetRencontreTables(ctx, r.ID, settings); err != nil {
		t.Fatal(err)
	}
	open := swissOf(t, ctx, svc, raw, "Principal", 6, names("Open", 4)...)
	side := swissOf(t, ctx, svc, raw, "DMP", 6, names("Side", 8)...)
	for _, tid := range []int64{open, side} {
		if _, err := svc.AttachToRencontre(ctx, tid, r.ID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.SetEventRooms(ctx, r.ID, side, []string{"B"}); err != nil {
		t.Fatal(err)
	}
	return ctx, svc, r.ID, open, side
}

func TestRestrictedEventIsProposedOnlyItsRoom(t *testing.T) {
	ctx, svc, rid, _, side := twoRoomFestival(t)
	v, err := svc.GetDirection(ctx, side)
	if err != nil {
		t.Fatal(err)
	}
	placed, waiting := startTables(v.Proposals)
	if len(placed) != 3 || waiting != 1 {
		t.Fatalf("DMP proposals on tables %v with %d waiting, want three in room B and one waiting", placed, waiting)
	}
	for _, n := range placed {
		if n < 4 {
			t.Errorf("DMP proposed table %d, outside room B", n)
		}
	}
	if strings.Join(v.Rooms, ",") != "B" || len(v.TableSettings) != 6 {
		t.Errorf("view rooms %v, %d settings", v.Rooms, len(v.TableSettings))
	}
	hall, err := svc.RencontreTableGrid(ctx, rid)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(hall.Rooms, ",") != "A,B" || hall.Cells[4].Room != "B" {
		t.Errorf("hall rooms %v, cell 5 in %q", hall.Rooms, hall.Cells[4].Room)
	}
}

func TestMoveOutsideTheRoomsIsRefused(t *testing.T) {
	ctx, svc, _, open, side := twoRoomFestival(t)
	v, err := svc.ConfirmAllProposals(ctx, side)
	if err != nil || len(v.Running) != 3 {
		t.Fatalf("ConfirmAllProposals: %d running, %v", len(v.Running), err)
	}
	_, err = svc.MoveMatchToTable(ctx, side, string(v.Running[0].ID), 1)
	if !errors.Is(err, direction.ErrRefused) || !strings.Contains(err.Error(), "table 1") {
		t.Fatalf("move to room A: %v, want a refusal naming table 1", err)
	}
	if _, err := svc.StartMatchManually(ctx, side, "x", "y", 7, 2); !errors.Is(err, direction.ErrRefused) {
		t.Errorf("manual start on table 2: %v, want refused", err)
	}
	// A swap with the open event is refused for its far end: the open event plays everywhere,
	// but the DMP match would go to the open event's table in room A.
	o, err := svc.ConfirmAllProposals(ctx, open)
	if err != nil || len(o.Running) == 0 {
		t.Fatalf("open: %v", err)
	}
	if _, err := svc.MoveMatchToTable(ctx, open, string(o.Running[0].ID), v.Running[0].Table); !errors.Is(err, direction.ErrRefused) {
		t.Errorf("a swap sending the DMP match to room A: %v, want refused", err)
	}
}

func TestRemovingARoomThatHoldsARunningMatchIsRefused(t *testing.T) {
	ctx, svc, rid, _, side := twoRoomFestival(t)
	v, err := svc.ConfirmAllProposals(ctx, side)
	if err != nil {
		t.Fatal(err)
	}
	table := v.Running[0].Table
	_, err = svc.SetRencontreTables(ctx, rid, []domain.TableSetting{{Number: 1, Room: "A"}, {Number: table, Room: "A"}})
	if !errors.Is(err, direction.ErrRefused) || !strings.Contains(err.Error(), fmt.Sprintf("table %d", table)) {
		t.Errorf("moving table %d to room A: %v, want refused naming it", table, err)
	}
	if _, err := svc.SetEventRooms(ctx, rid, side, []string{"A"}); !errors.Is(err, direction.ErrRefused) {
		t.Errorf("taking room B from the DMP: %v, want refused", err)
	}
	if _, err := svc.SetEventRooms(ctx, rid, side, []string{"A", "B"}); err != nil {
		t.Errorf("adding a room: %v", err)
	}
	if _, err := svc.SetEventRooms(ctx, rid, side, []string{"C"}); !errors.Is(err, direction.ErrRefused) {
		t.Errorf("an unknown room: %v, want refused", err)
	}
}

func TestReservedTableIsNeverProposed(t *testing.T) {
	ctx, svc, raw := openService(t)
	tid := swissOf(t, ctx, svc, raw, "Open", 4, names("P", 8)...)
	v, err := svc.SetDirectionTables(ctx, tid, []domain.TableSetting{{Number: 2, Name: "Stream", Reserved: true}})
	if err != nil {
		t.Fatal(err)
	}
	placed, waiting := startTables(v.Proposals)
	if fmt.Sprint(placed) != "[1 3 4]" || waiting != 1 {
		t.Errorf("proposals on %v, %d waiting; want [1 3 4] and the fourth waiting", placed, waiting)
	}
	if _, err := svc.SetDirectionTables(ctx, tid, []domain.TableSetting{{Number: 5}}); !errors.Is(err, direction.ErrRefused) {
		t.Errorf("table 5 of 4: %v, want refused", err)
	}
	// A reserved table accepts the director's own gesture.
	run, err := svc.ConfirmAllProposals(ctx, tid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MoveMatchToTable(ctx, tid, string(run.Running[0].ID), 2); err != nil {
		t.Errorf("moving onto the reserved table: %v", err)
	}
}

func TestKeptTableGoesToItsHolder(t *testing.T) {
	ctx, svc, raw := openService(t)
	tid := swissOf(t, ctx, svc, raw, "Open", 6, "Alice", "Bob")
	v, err := svc.SetDirectionTables(ctx, tid, []domain.TableSetting{{Number: 5, AssignedTo: []string{"Alice"}}})
	if err != nil {
		t.Fatal(err)
	}
	if placed, _ := startTables(v.Proposals); fmt.Sprint(placed) != "[5]" {
		t.Fatalf("Alice's match proposed on %v, want her table 5", placed)
	}
	v, err = svc.SetDirectionTables(ctx, tid, []domain.TableSetting{
		{Number: 5, AssignedTo: []string{"Alice"}}, {Number: 3, AssignedTo: []string{"Bob"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if placed, _ := startTables(v.Proposals); fmt.Sprint(placed) != "[3]" {
		t.Fatalf("Alice v Bob proposed on %v, want the smaller kept table 3", placed)
	}
	run, err := svc.ConfirmAllProposals(ctx, tid)
	if err != nil || len(run.Running) != 1 || run.Running[0].Table != 3 {
		t.Fatalf("confirmed: %+v, %v; want the match on table 3", run.Running, err)
	}
}

func TestAttachedEventSetsItsTablesOnTheRencontre(t *testing.T) {
	ctx, svc, _, open, _ := twoRoomFestival(t)
	if _, err := svc.SetDirectionTables(ctx, open, []domain.TableSetting{{Number: 1, Name: "Stream"}}); !errors.Is(err, direction.ErrRefused) {
		t.Errorf("SetDirectionTables on an attached event: %v, want refused", err)
	}
}

func TestPagesNameTablesAndGroupByRoom(t *testing.T) {
	ctx, svc, rid, _, _ := twoRoomFestival(t)
	r, err := svc.GetRencontre(ctx, rid)
	if err != nil {
		t.Fatal(err)
	}
	settings := r.TableSettings
	settings[4].Name = "Stream"
	if _, err := svc.SetRencontreTables(ctx, rid, settings); err != nil {
		t.Fatal(err)
	}
	wall, err := svc.RencontrePageHTML(ctx, rid)
	if err != nil {
		t.Fatal(err)
	}
	a, b := strings.Index(wall, "<h2>A</h2>"), strings.Index(wall, "<h2>B</h2>")
	if a < 0 || b < a || !strings.Contains(wall, "Table 5 — Stream") {
		t.Errorf("wall page not grouped by room with the table's name:\n%s", wall)
	}

	ctx, svc2, raw := openService(t)
	tid := swissOf(t, ctx, svc2, raw, "Open", 4, names("P", 4)...)
	if _, err := svc2.SetDirectionTables(ctx, tid, []domain.TableSetting{{Number: 1, Name: "Stream"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc2.ConfirmAllProposals(ctx, tid); err != nil {
		t.Fatal(err)
	}
	page, err := svc2.DirectionPageHTML(ctx, tid)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(page, "Table 1 — Stream") {
		t.Errorf("event page does not name table 1:\n%s", page)
	}
}
