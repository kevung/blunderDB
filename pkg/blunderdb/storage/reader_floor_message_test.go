package storage

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The frontend recognises the refusal by its wording and takes the required
// schema from it; both tests read this fixture, so rewording it here fails
// the frontend's test too.
func TestCheckReaderFloor_MessageMatchesFrontendFixture(t *testing.T) {
	raw, err := os.ReadFile("../../../testdata/reader_floor_refusal.json")
	if err != nil {
		t.Fatal(err)
	}
	var fx struct{ Floor, Message string }
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	err = CheckReaderFloor(fx.Floor)
	if err == nil {
		t.Fatalf("floor %s accepted", fx.Floor)
	}
	want := strings.ReplaceAll(fx.Message, "{schema}", domain.DatabaseVersion)
	if err.Error() != want {
		t.Errorf("refusal wording changed:\n got %q\nwant %q", err.Error(), want)
	}
}
