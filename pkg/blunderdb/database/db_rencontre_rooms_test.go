package database

import (
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestRencontreRoomsThroughTheFacade: the desktop and the CLI set a Rencontre's table
// properties and an event's rooms through the façade, and read back the plan they make.
func TestRencontreRoomsThroughTheFacade(t *testing.T) {
	d := newTestDB(t)
	a, b := roomOfTwo(t, d, 4, 4)
	rid, err := d.RencontreOf(a)
	if err != nil {
		t.Fatal(err)
	}
	settings := []domain.TableSetting{{Number: 1, Room: "A", Name: "Stream"}, {Number: 2, Room: "A"}, {Number: 3, Room: "B"}, {Number: 4, Room: "B"}}
	if _, err := d.SetRencontreTables(rid, settings); err != nil {
		t.Fatal(err)
	}
	v, err := d.SetEventRooms(rid, b, []string{"B"})
	if err != nil {
		t.Fatal(err)
	}
	if len(v.TableSettings) != 4 || !slices.Equal(v.EventRooms[b], []string{"B"}) {
		t.Fatalf("view: %+v", v.Rencontre)
	}
	plan, err := d.TablePlan(b)
	if err != nil || plan.Allowed(1) || !plan.Allowed(3) {
		t.Fatalf("plan of the event in room B: %+v, %v", plan, err)
	}
	if _, err := d.SetDirectionTables(a, settings); err == nil {
		t.Error("SetDirectionTables on an attached event: accepted, want refused")
	}
}
