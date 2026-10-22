package database

import "testing"

func TestIsPositionStudied_FollowsTheMark(t *testing.T) {
	db := newTestDB(t)
	pos := InitializePosition()
	id, err := db.SavePosition(&pos)
	if err != nil {
		t.Fatalf("SavePosition: %v", err)
	}

	for _, step := range []struct {
		mark bool
		want bool
	}{{false, false}, {true, true}, {false, false}} {
		if step.mark {
			if err := db.SetPositionStudied(id, true); err != nil {
				t.Fatalf("SetPositionStudied: %v", err)
			}
		} else if step.want == false && id != 0 {
			if err := db.SetPositionStudied(id, false); err != nil {
				t.Fatalf("SetPositionStudied(false): %v", err)
			}
		}
		got, err := db.IsPositionStudied(id)
		if err != nil {
			t.Fatalf("IsPositionStudied: %v", err)
		}
		if got != step.want {
			t.Errorf("IsPositionStudied after mark=%v: got %v, want %v", step.mark, got, step.want)
		}
	}
}
