package service_test

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction/service"
)

// playUntil confirms proposals one at a time and enters every result, the lower identifier
// winning, until the queue proposes an action of kind stop.
func playUntil(t *testing.T, ctx context.Context, svc *service.Service, tid int64, stop tournoi.ActionKind) {
	t.Helper()
	for step := 0; step < 400; step++ {
		v, err := svc.GetDirection(ctx, tid)
		if err != nil {
			t.Fatal(err)
		}
		var next *tournoi.Action
		for i, a := range v.Proposals {
			if a.Kind == stop {
				return
			}
			if next == nil && a.Kind != tournoi.ActWait && a.Kind != tournoi.ActFinish && a.Reason != tournoi.ReasonWaitingTable {
				next = &v.Proposals[i]
			}
		}
		if next != nil {
			blob, err := json.Marshal(*next)
			if err != nil {
				t.Fatal(err)
			}
			if v, err = svc.ConfirmProposal(ctx, tid, string(blob)); err != nil {
				t.Fatal(err)
			}
			if next.Kind != tournoi.ActStartMatch {
				continue
			}
		}
		if next == nil && len(v.Running) == 0 {
			t.Fatalf("stuck before %s: %+v", stop, v.Proposals)
		}
		for _, m := range v.Running {
			w := m.A
			if string(m.B) < string(m.A) {
				w = m.B
			}
			if _, err := svc.EnterResult(ctx, tid, string(m.ID), string(w), m.Length, 2, ""); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Fatalf("never reached %s", stop)
}

// statusOf finds the wall's line for one player.
func statusOf(page, name string) string {
	i := strings.Index(page, `<section class="statuts">`)
	if i < 0 {
		return ""
	}
	sec := page[i:]
	sec = sec[:strings.Index(sec, `</section>`)]
	for _, li := range strings.Split(sec, "<li") {
		if strings.Contains(li, ">"+name+" — ") {
			return li[strings.Index(li, " — ")+len(" — ") : strings.Index(li, "</li>")]
		}
	}
	return ""
}

// The wall answers "am I playing?": at the end of the Swiss it names who goes through and who
// stops; once the bracket is drawn, who sits the first round out and when they come in.
func TestWallSaysWhoPlays(t *testing.T) {
	ctx, svc, raw := openService(t)
	tid, err := raw.Tournaments().Create(ctx, "", "Club", "2026-09-12", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Club","tables":{"count":4},"phases":[` +
		`{"kind":"swiss_lives","length":5,"lives":2,"mode":"continuous","name":"Suisse"},` +
		`{"kind":"bracket","length":5,"entry":"top:3","name":"Tableau"}]}`
	if err := svc.CreateDirection(ctx, tid, cfg, 5); err != nil {
		t.Fatal(err)
	}
	for i, p := range names("P", 6) {
		if _, err := svc.AddParticipant(ctx, tid, p, "", float64(1600-10*i)); err != nil {
			t.Fatal(err)
		}
	}
	r, err := svc.CreateRencontre(ctx, "Festival", "", "", 4)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AttachToRencontre(ctx, tid, r.ID); err != nil {
		t.Fatal(err)
	}
	pages := func() map[string]string {
		t.Helper()
		own, err := svc.DirectionPageHTML(ctx, tid)
		if err != nil {
			t.Fatal(err)
		}
		wall, err := svc.RencontrePageHTML(ctx, r.ID)
		if err != nil {
			t.Fatal(err)
		}
		return map[string]string{"event page": own, "wall page": wall}
	}

	playUntil(t, ctx, svc, tid, tournoi.ActNextPhase)
	for which, page := range pages() {
		qualified, eliminated := 0, 0
		for _, n := range names("P", 6) {
			switch s := statusOf(page, n); {
			case strings.HasPrefix(s, "qualifié(e)") && strings.Contains(s, "Tableau"):
				qualified++
			case s == "éliminé(e)":
				eliminated++
			default:
				t.Errorf("%s: %s reads %q at the end of the Swiss", which, n, s)
			}
		}
		if qualified != 3 || eliminated != 3 {
			t.Errorf("%s: %d qualified and %d eliminated, want 3 and 3", which, qualified, eliminated)
		}
	}

	playUntil(t, ctx, svc, tid, tournoi.ActStartMatch)
	for which, page := range pages() {
		byes, eliminated := 0, 0
		for _, n := range names("P", 6) {
			switch s := statusOf(page, n); s {
			case "exempté(e) — entre au tour 2":
				byes++
			case "éliminé(e)":
				eliminated++
			}
		}
		if byes != 1 || eliminated != 3 {
			t.Errorf("%s: %d byes and %d eliminated once the bracket is drawn, want 1 and 3\n%s",
				which, byes, eliminated, page)
		}
	}
}

// A Swiss by rounds gives an odd player out a bye: the wall says who sits the round out and
// when they play again.
func TestWallSaysWhoSitsASwissRoundOut(t *testing.T) {
	ctx, svc, raw := openService(t)
	tid, err := raw.Tournaments().Create(ctx, "", "Club", "2026-10-04", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Club","tables":{"count":4},"phases":[` +
		`{"kind":"swiss_lives","length":5,"lives":2,"mode":"rounds","name":"Suisse"}]}`
	if err := svc.CreateDirection(ctx, tid, cfg, 5); err != nil {
		t.Fatal(err)
	}
	for i, p := range names("P", 5) {
		if _, err := svc.AddParticipant(ctx, tid, p, "", float64(1600-10*i)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.ConfirmAllProposals(ctx, tid); err != nil {
		t.Fatal(err)
	}
	page, err := svc.DirectionPageHTML(ctx, tid)
	if err != nil {
		t.Fatal(err)
	}
	byes := 0
	for _, n := range names("P", 5) {
		if statusOf(page, n) == "exempté(e) — rejoue à la ronde 2" {
			byes++
		}
	}
	if byes != 1 {
		t.Errorf("%d players read as sitting round 1 out, want 1\n%s", byes, page)
	}
}

// Between a pool qualifier's withdrawal and the director's choice of a tied repechage, the wall
// must not tell the withdrawn player's room that he goes through, nor send a candidate home.
func TestWallDuringATiedRepechage(t *testing.T) {
	ctx, svc, raw := openService(t)
	tid, err := raw.Tournaments().Create(ctx, "", "Poule", "2026-10-04", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Poule","tables":{"count":4},"phases":[` +
		`{"kind":"round_robin","length":5,"group_size":4,"qualifiers":2,"entry":"survivors"},` +
		`{"kind":"bracket","length":5,"entry":"survivors"}]}`
	if err := svc.CreateDirection(ctx, tid, cfg, 5); err != nil {
		t.Fatal(err)
	}
	for i, p := range names("P", 4) {
		if _, err := svc.AddParticipant(ctx, tid, p, "", float64(1600-10*i)); err != nil {
			t.Fatal(err)
		}
	}
	v, err := svc.GetDirection(ctx, tid)
	if err != nil {
		t.Fatal(err)
	}
	var ids []tournoi.PlayerID
	for _, p := range v.Players {
		ids = append(ids, p.ID)
	}
	slices.Sort(ids)
	// 2-2-1-1: the first two go through, the last two tie on one win each.
	beats := map[[2]tournoi.PlayerID]bool{
		{ids[0], ids[1]}: true, {ids[0], ids[2]}: true, {ids[1], ids[2]}: true,
		{ids[1], ids[3]}: true, {ids[2], ids[3]}: true, {ids[3], ids[0]}: true,
	}
	playWith(t, ctx, svc, tid, tournoi.ActNextPhase, func(a, b tournoi.PlayerID) tournoi.PlayerID {
		if beats[[2]tournoi.PlayerID{a, b}] {
			return a
		}
		return b
	})
	if _, err := svc.WithdrawParticipant(ctx, tid, string(ids[0]), false); err != nil {
		t.Fatal(err)
	}
	page, err := svc.DirectionPageHTML(ctx, tid)
	if err != nil {
		t.Fatal(err)
	}
	if s := statusOf(page, playerName(v.Players, ids[0])); strings.HasPrefix(s, "qualifié") {
		t.Errorf("the wall reads %q for the withdrawn qualifier", s)
	}
	for _, c := range ids[2:] {
		if s := statusOf(page, playerName(v.Players, c)); !strings.HasPrefix(s, "pas encore fixé") {
			t.Errorf("the wall reads %q for the tied candidate %s, want it undecided", s, c)
		}
	}
}
