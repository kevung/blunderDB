package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// A signed note's author is stored as sent, so a name no write would accept is
// refused rather than cut; an unsigned one is only normalised.
func TestCommentsAdd_SignedAuthorTooLongIsRefused(t *testing.T) {
	srv, _ := newRecordingServer(t, false)
	save := serveAcross(t, srv, "1", "", "/v1/positions.save", `{"position":`+initialPositionJSON(t)+`}`)
	var id idResp
	if err := json.Unmarshal(save.Body.Bytes(), &id); err != nil || id.ID == 0 {
		t.Fatalf("positions.save: %d %s", save.Code, save.Body)
	}
	long := strings.Repeat("a", 121)
	body := func(sig string) string {
		return `{"positionId":` + jsonInt(id.ID) + `,"text":"x","author":"` + long + `","signature":"` + sig + `"}`
	}
	if rec := serveAcross(t, srv, "1", "", "/v1/comments.add", body("sig")); rec.Code != http.StatusBadRequest {
		t.Errorf("signed, author of 121 characters: %d %s; want 400", rec.Code, rec.Body)
	}
	if rec := serveAcross(t, srv, "1", "", "/v1/comments.add", body("")); rec.Code != http.StatusOK {
		t.Errorf("unsigned, author of 121 characters: %d %s; want 200 (normalised)", rec.Code, rec.Body)
	}
}
