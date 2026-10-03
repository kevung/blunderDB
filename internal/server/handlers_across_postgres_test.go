//go:build postgres

// Needs Docker (testcontainers), like handlers_tenant_test.go:
//
//	go test -tags postgres ./internal/server/... -run TestAcross -v
package server

import (
	"bufio"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

// TestAcross_Postgres: on a real multi-tenant database, across.matchesList
// returns the writer's and the listed tenants' matches, each line tagged with
// its tenant, and never a row of a tenant the header leaves out.
func TestAcross_Postgres(t *testing.T) {
	_, srv := newPostgresTestServerAndHandlerWith(t, func(o *Options) { o.TrustReadTenants = true })
	for _, tenant := range []string{"1", "2", "3"} {
		body := `{"match":{"player1_name":"p` + tenant + `","player2_name":"x","match_length":5}}`
		if rec := serveAcross(t, srv, tenant, "", "/v1/matches.save", body); rec.Code != 200 {
			t.Fatalf("save under %s: %d %s", tenant, rec.Code, rec.Body)
		}
	}

	rec := serveAcross(t, srv, "1", "2", "/v1/across.matchesList", `{}`)
	if rec.Code != 200 {
		t.Fatalf("across.matchesList: %d %s", rec.Code, rec.Body)
	}
	got := map[string]string{}
	sc := bufio.NewScanner(rec.Body)
	for sc.Scan() {
		var item acrossMatch
		if err := json.Unmarshal(sc.Bytes(), &item); err != nil || item.Match == nil {
			t.Fatalf("line %q: %v", sc.Text(), err)
		}
		got[item.Tenant] += item.Match.Player1Name
	}
	if got["1"] != "p1" || got["2"] != "p2" || len(got) != 2 {
		t.Errorf("read across [1 2] = %v, want 1:p1 and 2:p2 only", got)
	}

	// Tenant 3's match is addressed by its id: unlisted, it is refused.
	var id struct{ ID int64 }
	list3 := serveAcross(t, srv, "3", "", "/v1/matches.list", `{}`)
	_ = json.NewDecoder(strings.NewReader(strings.SplitN(list3.Body.String(), "\n", 2)[0])).Decode(&id)
	if rec := serveAcross(t, srv, "1", "2", "/v1/across.matchesGet", `{"tenant":"3","id":`+jsonInt(id.ID)+`}`); rec.Code != 400 {
		t.Errorf("tenant 3 not listed: status %d, want 400", rec.Code)
	}
	// Listed, the same request reads it, tagged.
	rec = serveAcross(t, srv, "1", "3", "/v1/across.matchesGet", `{"tenant":"3","id":`+jsonInt(id.ID)+`}`)
	var one acrossMatch
	if err := json.Unmarshal(rec.Body.Bytes(), &one); err != nil || one.Tenant != "3" || one.Match == nil || one.Match.Player1Name != "p3" {
		t.Errorf("tenant 3 listed: %d %s", rec.Code, rec.Body)
	}

	// The coach's join: a board read in tenant 3 and saved in tenant 1 is
	// found in tenant 1 by the hash the read carried.
	pos := `{"position":` + initialPositionJSON(t) + `}`
	if rec := serveAcross(t, srv, "3", "", "/v1/positions.save", pos); rec.Code != 200 {
		t.Fatalf("positions.save under 3: %d %s", rec.Code, rec.Body)
	}
	found := serveAcross(t, srv, "1", "3", "/v1/across.searchFind", `{}`)
	var item acrossPosition
	for _, line := range strings.Split(strings.TrimSpace(found.Body.String()), "\n") {
		var it acrossPosition
		if json.Unmarshal([]byte(line), &it) == nil && it.Tenant == "3" {
			item = it
		}
	}
	if item.Zobrist == 0 {
		t.Fatalf("no position of tenant 3 with a hash: %s", found.Body)
	}
	if rec := serveAcross(t, srv, "1", "", "/v1/positions.save", pos); rec.Code != 200 {
		t.Fatalf("positions.save under 1: %d %s", rec.Code, rec.Body)
	}
	exists := serveAcross(t, srv, "1", "", "/v1/positions.exists", `{"zobrist":`+strconv.FormatUint(item.Zobrist, 10)+`}`)
	if !strings.Contains(exists.Body.String(), `"found":true`) {
		t.Errorf("tenant 1 does not find the board read in tenant 3 by its hash: %s", exists.Body)
	}
}

// TestAcross_IdOfAnotherListedTenant_Postgres: tenants 2 and 3 are both read,
// and each owns a match with one move and an analysed position. A (tenant, id)
// pair whose id belongs to the other listed tenant answers for the named
// tenant only: every line says the named tenant, and the other tenant's row
// never comes back under it.
func TestAcross_IdOfAnotherListedTenant_Postgres(t *testing.T) {
	_, srv := newPostgresTestServerAndHandlerWith(t, func(o *Options) { o.TrustReadTenants = true })
	type owned struct{ match, pos int64 }
	rows := map[string]owned{}
	call := func(tenant, path, body string) int64 {
		t.Helper()
		rec := serveAcross(t, srv, tenant, "", path, body)
		if rec.Code != 200 {
			t.Fatalf("%s under %s: %d %s", path, tenant, rec.Code, rec.Body)
		}
		var id struct{ ID int64 }
		_ = json.Unmarshal(rec.Body.Bytes(), &id)
		return id.ID
	}
	for _, tenant := range []string{"2", "3"} {
		matchID := call(tenant, "/v1/matches.save", `{"match":{"player1_name":"mover-`+tenant+`","player2_name":"x","match_length":5}}`)
		gameID := call(tenant, "/v1/matches.createGame", `{"game":{"match_id":`+jsonInt(matchID)+`,"game_number":1}}`)
		posID := call(tenant, "/v1/positions.save", `{"position":`+initialPositionJSON(t)+`}`)
		call(tenant, "/v1/matches.createMove", `{"move":{"game_id":`+jsonInt(gameID)+`,"move_number":1,"move_type":"checker","position_id":`+jsonInt(posID)+`,"player":1,"dice":[3,1],"checker_move":"8/5 6/5"}}`)
		call(tenant, "/v1/analyses.save", `{"positionId":`+jsonInt(posID)+`,"analysis":{"xgid":"analysis-`+tenant+`"}}`)
		rows[tenant] = owned{match: matchID, pos: posID}
	}

	movers := func(named string, matchID int64) []string {
		t.Helper()
		rec := serveAcross(t, srv, "1", "2,3", "/v1/across.matchMovePositions", `{"tenant":"`+named+`","matchId":`+jsonInt(matchID)+`}`)
		if rec.Code == 404 {
			return nil
		}
		if rec.Code != 200 {
			t.Fatalf("across.matchMovePositions %s/%d: %d %s", named, matchID, rec.Code, rec.Body)
		}
		var names []string
		for _, line := range strings.Split(strings.TrimSpace(rec.Body.String()), "\n") {
			if line == "" {
				continue
			}
			var it acrossMovePosition
			if err := json.Unmarshal([]byte(line), &it); err != nil || it.MovePosition == nil {
				t.Fatalf("line %q: %v", line, err)
			}
			if it.Tenant != named {
				t.Errorf("match %d asked in %s: a line tagged %s", matchID, named, it.Tenant)
			}
			names = append(names, it.MovePosition.Player1Name)
		}
		return names
	}
	analysed := func(named string, posID int64) []string {
		t.Helper()
		rec := serveAcross(t, srv, "1", "2,3", "/v1/across.analysesLoadByIds", `{"tenant":"`+named+`","ids":[`+jsonInt(posID)+`]}`)
		var got acrossAnalyses
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || rec.Code != 200 {
			t.Fatalf("across.analysesLoadByIds %s/%d: %d %s", named, posID, rec.Code, rec.Body)
		}
		if got.Tenant != named {
			t.Errorf("ids [%d] asked in %s: answer tagged %s", posID, named, got.Tenant)
		}
		var xgids []string
		for _, a := range got.Analyses {
			xgids = append(xgids, a.XGID)
		}
		return xgids
	}

	for named, other := range map[string]string{"2": "3", "3": "2"} {
		if got := movers(named, rows[named].match); len(got) != 1 || got[0] != "mover-"+named {
			t.Errorf("tenant %s's own match: moves of %v", named, got)
		}
		for _, name := range movers(named, rows[other].match) {
			if name == "mover-"+other {
				t.Errorf("match %d of %s resolved in %s", rows[other].match, other, named)
			}
		}
		if got := analysed(named, rows[named].pos); len(got) != 1 || got[0] != "analysis-"+named {
			t.Errorf("tenant %s's own position: analyses %v", named, got)
		}
		for _, xgid := range analysed(named, rows[other].pos) {
			if xgid == "analysis-"+other {
				t.Errorf("position %d of %s resolved in %s", rows[other].pos, other, named)
			}
		}
	}
}

// TestAcrossClub_Postgres: a coach (tenant 1) comments a student's board
// (tenant 2) in the coach's own tenant; across.commentsByZobrist with the
// student listed brings the coach's comment back on the student's hash, and
// an outsider's comment on the same board (tenant 3) never.
func TestAcrossClub_Postgres(t *testing.T) {
	_, srv := newPostgresTestServerAndHandlerWith(t, func(o *Options) { o.TrustReadTenants = true })
	pos := `{"position":` + initialPositionJSON(t) + `}`
	ids := map[string]int64{}
	for _, tenant := range []string{"1", "2", "3"} {
		rec := serveAcross(t, srv, tenant, "", "/v1/positions.save", pos)
		var id idResp
		if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &id) != nil {
			t.Fatalf("positions.save under %s: %d %s", tenant, rec.Code, rec.Body)
		}
		ids[tenant] = id.ID
	}
	for _, tenant := range []string{"1", "3"} {
		body := `{"positionId":` + strconv.FormatInt(ids[tenant], 10) + `,"text":"note-` + tenant + `"}`
		if rec := serveAcross(t, srv, tenant, "", "/v1/comments.add", body); rec.Code != 200 {
			t.Fatalf("comments.add under %s: %d %s", tenant, rec.Code, rec.Body)
		}
	}
	var student acrossPosition
	found := serveAcross(t, srv, "1", "2", "/v1/across.searchFind", `{}`)
	for _, line := range strings.Split(strings.TrimSpace(found.Body.String()), "\n") {
		var it acrossPosition
		if json.Unmarshal([]byte(line), &it) == nil && it.Tenant == "2" {
			student = it
		}
	}
	if student.Zobrist == 0 {
		t.Fatalf("no student position with a hash: %s", found.Body)
	}
	rec := serveAcross(t, srv, "1", "2", "/v1/across.commentsByZobrist", `{"zobrists":[`+strconv.FormatUint(student.Zobrist, 10)+`]}`)
	if rec.Code != 200 {
		t.Fatalf("across.commentsByZobrist: %d %s", rec.Code, rec.Body)
	}
	var texts []string
	sc := bufio.NewScanner(rec.Body)
	for sc.Scan() {
		var c acrossComment
		if err := json.Unmarshal(sc.Bytes(), &c); err != nil {
			t.Fatalf("line %q: %v", sc.Text(), err)
		}
		texts = append(texts, c.Tenant+":"+c.Comment.Text)
	}
	if len(texts) != 1 || texts[0] != "1:note-1" {
		t.Errorf("comments read %v, want the coach's alone [1:note-1]", texts)
	}
}
