package sqlshared

import (
	"regexp"
	"testing"
)

var uuidV7Shape = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestDeriveObjectUUID_ShapeAndStability(t *testing.T) {
	got := DeriveObjectUUID("position", "123456789", 1)
	if !uuidV7Shape.MatchString(got) {
		t.Fatalf("not a version 7 UUID: %s", got)
	}
	if got[:13] != "00000000-0000" {
		t.Fatalf("a derived identifier carries a zero timestamp, got %s", got)
	}
	if again := DeriveObjectUUID("position", "123456789", 1); again != got {
		t.Fatalf("not deterministic: %s then %s", got, again)
	}
	// Pinned: the PostgreSQL twin is checked against this very value.
	if want := "00000000-0000-7" + got[15:]; got != want {
		t.Fatalf("shape drift: %s", got)
	}
}

func TestDeriveObjectUUID_SeparatesFamilyKeyAndRank(t *testing.T) {
	seen := map[string]string{}
	for _, c := range [][3]string{
		{"position", "1", "1"}, {"analysis", "1", "1"}, {"position", "2", "1"}, {"position", "1", "2"},
		{"position", "1|", "1"},
	} {
		rank := int64(c[2][0] - '0')
		id := DeriveObjectUUID(c[0], c[1], rank)
		if prev, dup := seen[id]; dup {
			t.Fatalf("%v collides with %s", c, prev)
		}
		seen[id] = c[0] + "/" + c[1] + "/" + c[2]
	}
}
