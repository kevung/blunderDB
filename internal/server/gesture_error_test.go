package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestGestureError_FailureIsNotARefusal: only a refusal is the client's 400; a failure of the
// database or of its connection is a 500 whose message stays in the daemon.
func TestGestureError_FailureIsNotARefusal(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"refusal", direction.Refusef("direction: table %d is taken", 3), http.StatusBadRequest},
		{"wrapped refusal", fmt.Errorf("rencontre: tournament 4: %w", direction.Refusef("direction: an entry needs a name")), http.StatusBadRequest},
		{"finished", direction.ErrFinished, http.StatusBadRequest},
		{"nothing proposed", direction.ErrNothingProposed, http.StatusBadRequest},
		{"internal", fmt.Errorf("sqlite: read log: %w", storage.ErrInternal), http.StatusInternalServerError},
		{"pgx conn closed", errors.New("conn closed"), http.StatusInternalServerError},
		{"sql closed", errors.New("sql: database is closed"), http.StatusInternalServerError},
		{"conn done", sql.ErrConnDone, http.StatusInternalServerError},
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		writeStorageError(w, gestureError(c.err))
		if w.Code != c.want {
			t.Errorf("%s: status %d, want %d", c.name, w.Code, c.want)
		}
		if c.want == http.StatusInternalServerError {
			var env struct {
				Error struct{ Message string } `json:"error"`
			}
			_ = json.Unmarshal(w.Body.Bytes(), &env)
			if env.Error.Message == c.err.Error() {
				t.Errorf("%s: the failure's message reached the client", c.name)
			}
		}
	}
}

// TestDirectionGestures_RefusalsAreInvalid: the refusals of the service and the engine, sent
// through the routes, answer 400 with their message.
func TestDirectionGestures_RefusalsAreInvalid(t *testing.T) {
	ts, srv := gestureServer(t)
	f := seedDirection(t, srv.opts.Storage, "1", "Open de Lyon")
	busy := f.running(t)[0].Table
	room := `{"id":` + strconv.FormatInt(f.rencontreID, 10)
	calls := []struct{ path, body string }{
		{"/v1/directions.addParticipant", f.body(`"name":"  "`)},
		{"/v1/directions.startMatch", f.body(`"a":"aa","b":"bb","table":` + strconv.Itoa(busy))},
		{"/v1/directions.setConfig", f.body(`"config":""`)},
		{"/v1/directions.attachMatch", f.body(`"matchId":1`)},
		{"/v1/rencontres.setTableOutOfService", room + `,"table":42,"out":true}`},
		{"/v1/rencontres.update", room + `,"name":"","tables":4}`},
	}
	for _, c := range calls {
		r := send(t, ts, "1", c.path, c.body, f.versionOf(t, ts), "")
		if r.status != http.StatusBadRequest {
			t.Errorf("%s: status %d, body %.200s; want 400", c.path, r.status, r.body)
		}
	}
	if r := send(t, ts, "1", "/v1/rencontres.create", `{"name":"","tables":4}`, "", ""); r.status != http.StatusBadRequest {
		t.Errorf("rencontres.create without a name: status %d, body %.200s; want 400", r.status, r.body)
	}
}
