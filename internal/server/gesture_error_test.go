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
	tournoi "github.com/kevung/nicomaque"
)

// TestGestureError_FailureIsNotARefusal: a refusal of the rules or the engine is the client's
// 422 refused, a malformed request its 400 invalid; a failure of the database or of its
// connection is a 500 whose message stays in the daemon.
func TestGestureError_FailureIsNotARefusal(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
		code string
	}{
		{"refusal", direction.Refusef("direction: table %d is taken", 3), http.StatusUnprocessableEntity, CodeRefused},
		{"wrapped refusal", fmt.Errorf("rencontre: tournament 4: %w", direction.Refusef("direction: an entry needs a name")), http.StatusUnprocessableEntity, CodeRefused},
		{"engine config", &tournoi.ConfigRefusal{}, http.StatusUnprocessableEntity, CodeRefused},
		{"engine waiver", &tournoi.WaiverRefusal{Phase: 2, Reason: tournoi.RefusalStarted, Match: "m1"}, http.StatusUnprocessableEntity, CodeRefused},
		{"engine waiver applied", direction.RefusedAs(direction.ReasonEngineRefused, &tournoi.WaiverRefusal{Phase: 2, Match: "m1"}), http.StatusUnprocessableEntity, CodeRefused},
		{"finished", direction.ErrFinished, http.StatusUnprocessableEntity, CodeRefused},
		{"nothing proposed", direction.ErrNothingProposed, http.StatusUnprocessableEntity, CodeRefused},
		{"malformed", errNoTarget("tournamentId"), http.StatusBadRequest, CodeInvalid},
		{"malformed refusal", fmt.Errorf("%w: %w", storage.ErrInvalid, direction.Refusef("direction: until %q", "demain")), http.StatusBadRequest, CodeInvalid},
		{"internal", fmt.Errorf("sqlite: read log: %w", storage.ErrInternal), http.StatusInternalServerError, CodeInternal},
		{"pgx conn closed", errors.New("conn closed"), http.StatusInternalServerError, CodeInternal},
		{"sql closed", errors.New("sql: database is closed"), http.StatusInternalServerError, CodeInternal},
		{"conn done", sql.ErrConnDone, http.StatusInternalServerError, CodeInternal},
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		writeStorageError(w, gestureError(c.err))
		var got struct {
			Error struct{ Code string } `json:"error"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &got)
		if w.Code != c.want || got.Error.Code != c.code {
			t.Errorf("%s: status %d code %q, want %d %q", c.name, w.Code, got.Error.Code, c.want, c.code)
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

// TestDirectionGestures_RefusalsAreRefused: the refusals of the service and the engine, sent
// through the routes, answer 422 refused with their message.
func TestDirectionGestures_RefusalsAreRefused(t *testing.T) {
	ts, srv := gestureServer(t)
	f := seedDirection(t, srv.opts.Storage, "1", "Open de Lyon")
	busy := f.running(t)[0].Table
	room := `{"id":` + strconv.FormatInt(f.rencontreID, 10)
	calls := []struct{ path, body string }{
		{"/v1/directions.addParticipant", f.body(`"name":"  "`)},
		{"/v1/directions.startMatch", f.body(`"a":"aa","b":"bb","table":` + strconv.Itoa(busy))},
		{"/v1/directions.setConfig", f.body(`"config":""`)},
		{"/v1/directions.moveMatchToTable", f.body(`"matchId":"` + string(f.running(t)[0].ID) + `","table":99`)},
		{"/v1/directions.startMatch", f.body(`"a":"aa","b":"bb","table":99`)},
		{"/v1/directions.makeAbsent", f.body(`"id":"aa","until":"demain"`)},
		{"/v1/directions.attachMatch", f.body(`"matchId":1`)},
		{"/v1/rencontres.setTableOutOfService", room + `,"table":42,"out":true}`},
		{"/v1/rencontres.update", room + `,"name":"","tables":4}`},
	}
	for _, c := range calls {
		r := send(t, ts, "1", c.path, c.body, f.versionOf(t, ts), "")
		if r.status != http.StatusUnprocessableEntity {
			t.Errorf("%s: status %d, body %.200s; want 422", c.path, r.status, r.body)
		}
	}
	if r := send(t, ts, "1", "/v1/rencontres.create", `{"name":"","tables":4}`, "", ""); r.status != http.StatusUnprocessableEntity {
		t.Errorf("rencontres.create without a name: status %d, body %.200s; want 422", r.status, r.body)
	}
}
