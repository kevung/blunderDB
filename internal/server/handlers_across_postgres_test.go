//go:build postgres

// Needs Docker (testcontainers), like handlers_tenant_test.go:
//
//	go test -tags postgres ./internal/server/... -run TestAcross -v
package server

import (
	"bufio"
	"encoding/json"
	"strings"
	"testing"
)

// TestAcross_Postgres: on a real multi-tenant database, across.matchesList
// returns the writer's and the listed tenants' matches, each line tagged with
// its tenant, and never a row of a tenant the header leaves out.
func TestAcross_Postgres(t *testing.T) {
	_, srv := newPostgresTestServerAndHandler(t)
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
}
