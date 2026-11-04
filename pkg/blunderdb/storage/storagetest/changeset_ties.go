// Property of the changeset: when two libraries hold different contents of one
// object under the same date, every family settles them by a merge that does
// not depend on the side and drops nothing the other side holds. Each case
// makes 24 objects, lets the two sides diverge and tie, exchanges in both
// orders, and counts the objects that did not converge or lost something.
package storagetest

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

const tieN = 24

// tieRun numbers the runs, so that each makes objects of its own in libraries
// the previous ones filled.
var tieRun int

func tieBoard(k int) domain.Position { return syncPos(2000 + tieRun*100 + k) }

func tieName(what string, i int) string { return fmt.Sprintf("%s %d.%d", what, tieRun, i) }

// tieCase is one family under test.
type tieCase struct {
	family string
	// independent: both libraries make the object on their own, under a uuid
	// the name alone derives (no exchange has put it on both).
	independent bool
	prepare     func(p *csPair)
	create      func(p *csPair, i int) int64
	// editA and editB run on the library that is theirs, with its own id.
	editA, editB func(p *csPair, i int, id int64)
	// check says what the object lost on one side, "" when nothing.
	check func(p *csPair, i int, id int64) string
}

func checkChangeSetTies(t *testing.T, pair func() *csPair) {
	cases := []func() tieCase{tieTournament, tieDeck, tieCollection, tieMatch, tieCard, tiePosition, tieComment, tieLesson, tieTranscription}
	for _, mk := range cases {
		c := mk()
		for _, order := range []string{"a-first", "b-first"} {
			t.Run(fmt.Sprintf("Ties_%s_%s", c.family, order), func(t *testing.T) {
				p := pair()
				p.t = t
				runTie(p, mk, order == "b-first")
			})
		}
	}
	t.Run("Ties_MatchBeforeItsTournament", func(t *testing.T) { p := pair(); p.t = t; tieMatchBeforeTournament(p) })
	t.Run("Ties_TournamentOrderOnAFreshPeer", func(t *testing.T) { p := pair(); p.t = t; scenarioOrderOnFreshPeer(p) })
	t.Run("Ties_SkippedIsCounted", func(t *testing.T) { p := pair(); p.t = t; scenarioSkippedIsCounted(p) })
	t.Run("Ties_LivingQueryOfRows", func(t *testing.T) { p := pair(); p.t = t; scenarioLivingQueryOfRows(p) })
	t.Run("Ties_SlotDatesNothing", func(t *testing.T) { p := pair(); p.t = t; scenarioSlotDatesNothing(p) })
}

func runTie(p *csPair, mk func() tieCase, bFirst bool) {
	t := p.t
	tieRun++
	c := mk()
	p.catchUp()
	if c.prepare != nil {
		c.prepare(p)
	}
	var ua [tieN]string
	var ia, ib [tieN]int64
	for i := range tieN {
		ia[i] = c.create(p, i)
		ua[i] = famUUID(t, p.a, p.ca, p.sa, c.family, ia[i])
	}
	if c.independent {
		for i := range tieN {
			ib[i] = c.create(p.swapped(), i)
			// Two libraries that named the object alike (and so drew one uuid from
			// the name) before the journal: the same uuid, the same date.
			if err := p.b.(storage.ObjectUUIDProvider).ObjectUUIDs().SetObjectUUID(p.cb, p.sb, c.family, ib[i], ua[i]); err != nil {
				t.Fatal(err)
			}
		}
	} else {
		p.send()
		for i := range tieN {
			ib[i] = famIDMust(p, p.b, p.cb, p.sb, c.family, ua[i])
		}
	}
	tick()
	for i := range tieN {
		if c.editA != nil {
			c.editA(p, i, ia[i])
		}
		if c.editB != nil {
			c.editB(p.swapped(), i, ib[i])
		}
	}
	// The same date on both sides for every object: a tie to the millisecond.
	for i := range tieN {
		at := max(p.entry(p.a, p.ca, p.sa, ua[i]).ChangedAt, p.entry(p.b, p.cb, p.sb, ua[i]).ChangedAt)
		p.markAs(p.a, p.ca, p.sa, c.family, ua[i], at)
		p.markAs(p.b, p.cb, p.sb, c.family, ua[i], at)
	}
	if bFirst {
		p.back()
		p.replay()
		p.back()
	} else {
		p.replay()
		p.back()
		p.replay()
	}
	lost := map[string]int{}
	for i := range tieN {
		for side, v := range map[string]struct {
			pp *csPair
			id int64
		}{"a": {p, ia[i]}, "b": {p.swapped(), ib[i]}} {
			if why := c.check(v.pp, i, v.id); why != "" {
				lost[side+": "+why]++
			}
		}
	}
	for why, n := range lost {
		t.Errorf("%s: %d/%d objects: %s", c.family, tieN-n, tieN, why)
	}
	if diff := p.diverged(c.family); diff != 0 {
		t.Errorf("%s: %d/%d objects differ between the two libraries after the exchange", c.family, diff, tieN)
	}
}

// diverged counts the objects of a family whose content differs on the two
// sides (all of a's records are read from 0 on both).
func (p *csPair) diverged(family string) int {
	p.t.Helper()
	get := func(q *csPair) map[string]string {
		set, err := changeset.Export(q.ca, q.a, q.sa, storage.Cursor{}, 1_000_000)
		if err != nil {
			p.t.Fatal(err)
		}
		out := map[string]string{}
		for _, r := range set.Records {
			if r.Family == family {
				out[r.UUID] = string(r.Payload)
			}
		}
		return out
	}
	ma, mb := get(p), get(p.swapped())
	n := 0
	for u, pa := range ma {
		if pb, ok := mb[u]; !ok || pb != pa {
			if n == 0 {
				at := 0
				for at < len(pa) && at < len(pb) && pa[at] == pb[at] {
					at++
				}
				p.t.Logf("first difference, %s, from byte %d:\n  a: %.200s\n  b: %.200s", u, at, pa[max(at-80, 0):], pb[max(at-80, 0):])
			}
			n++
		}
	}
	for u := range mb {
		if _, ok := ma[u]; !ok {
			n++
		}
	}
	return n
}

func has(got, want string) bool { return strings.Contains(got, want) }

func tieTournament() tieCase {
	return tieCase{
		family: "tournament", independent: true,
		create: func(p *csPair, i int) int64 {
			id, err := p.a.Tournaments().Create(p.ca, p.sa, tieName("Cup", i), "", "")
			if err != nil {
				p.t.Fatal(err)
			}
			return id
		},
		editA: func(p *csPair, i int, id int64) {
			if i%3 != 1 {
				_ = p.a.Tournaments().UpdateComment(p.ca, p.sa, id, fmt.Sprintf("note-a-%d", i))
			}
		},
		editB: func(p *csPair, i int, id int64) {
			if i%3 != 0 {
				_ = p.a.Tournaments().UpdateComment(p.ca, p.sa, id, fmt.Sprintf("note-b-%d", i))
			}
			if i%3 == 2 {
				_ = p.a.Tournaments().Update(p.ca, p.sa, id, tieName("Cup", i), "", "Paris")
			}
		},
		check: func(p *csPair, i int, id int64) string {
			g, err := p.a.Tournaments().Get(p.ca, p.sa, id)
			if err != nil {
				return err.Error()
			}
			if i%3 != 1 && !has(g.Comment, fmt.Sprintf("note-a-%d", i)) {
				return "a note of a was lost"
			}
			if i%3 != 0 && !has(g.Comment, fmt.Sprintf("note-b-%d", i)) {
				return "a note of b was lost"
			}
			if i%3 == 2 && g.Location != "Paris" {
				return "the location was lost"
			}
			return ""
		},
	}
}

func tieDeck() tieCase {
	return tieCase{
		family: "anki_deck", independent: true,
		create: func(p *csPair, i int) int64 {
			id, err := p.a.Anki().CreateDeck(p.ca, p.sa, tieName("Deck", i), "", domain.AnkiSourceSearch, 0, "")
			if err != nil {
				p.t.Fatal(err)
			}
			return id
		},
		editA: func(p *csPair, i int, id int64) {
			if i%3 != 1 {
				_ = p.a.Anki().UpdateDeck(p.ca, p.sa, id, tieName("Deck", i), fmt.Sprintf("desc-a-%d", i))
			}
		},
		editB: func(p *csPair, i int, id int64) {
			if i%3 != 0 {
				_ = p.a.Anki().UpdateDeck(p.ca, p.sa, id, tieName("Deck", i), fmt.Sprintf("desc-b-%d", i))
			}
		},
		check: func(p *csPair, i int, id int64) string {
			d := deckOf(p, p.a, p.ca, p.sa, id)
			if i%3 != 1 && !has(d.Description, fmt.Sprintf("desc-a-%d", i)) {
				return "a description of a was lost"
			}
			if i%3 != 0 && !has(d.Description, fmt.Sprintf("desc-b-%d", i)) {
				return "a description of b was lost"
			}
			return ""
		},
	}
}

func tieCollection() tieCase {
	var pos [tieN][2]int64
	return tieCase{
		family: "collection",
		prepare: func(p *csPair) {
			for i := range tieN {
				for k := range 2 {
					q := tieBoard(2*i + k)
					pos[i][k], _ = p.a.Positions().Save(p.ca, p.sa, &q)
				}
			}
		},
		create: func(p *csPair, i int) int64 {
			id, err := p.a.Collections().Create(p.ca, p.sa, tieName("Col", i), "")
			if err != nil {
				p.t.Fatal(err)
			}
			return id
		},
		editA: func(p *csPair, i int, id int64) {
			if i%3 != 1 {
				_ = p.a.Collections().AddPositions(p.ca, p.sa, id, []int64{pos[i][0]})
				_ = p.a.Collections().Update(p.ca, p.sa, id, tieName("Col", i), fmt.Sprintf("desc-a-%d", i))
			}
		},
		editB: func(p *csPair, i int, id int64) {
			if i%3 != 0 {
				// p is the swapped pair: its "a" is the library b.
				bpos, _, _ := p.a.Positions().Exists(p.ca, p.sa, zobristOfPos(p, i, 1))
				_ = p.a.Collections().AddPositions(p.ca, p.sa, id, []int64{bpos})
				_ = p.a.Collections().Update(p.ca, p.sa, id, tieName("Col", i), fmt.Sprintf("desc-b-%d", i))
			}
		},
		check: func(p *csPair, i int, id int64) string {
			c, err := p.a.Collections().Get(p.ca, p.sa, id)
			if err != nil {
				return err.Error()
			}
			ids, _ := p.a.Collections().PositionIDs(p.ca, p.sa, id, storage.ListOpts{})
			want := 0
			if i%3 != 1 {
				want++
			}
			if i%3 != 0 {
				want++
			}
			if len(ids) != want {
				return "a member was lost"
			}
			if i%3 != 1 && !has(c.Description, fmt.Sprintf("desc-a-%d", i)) {
				return "a description of a was lost"
			}
			if i%3 != 0 && !has(c.Description, fmt.Sprintf("desc-b-%d", i)) {
				return "a description of b was lost"
			}
			return ""
		},
	}
}

// zobristOfPos is the hash of the board the case k-th gave to collection i.
func zobristOfPos(_ *csPair, i, k int) uint64 {
	q := tieBoard(2*i + k)
	return storage.ZobristOf(&q)
}

func tieMatch() tieCase {
	var tour int64
	var tourU string
	return tieCase{
		family: "match",
		prepare: func(p *csPair) {
			tour, _ = p.a.Tournaments().Create(p.ca, p.sa, tieName("Ties", 0), "", "")
			tourU = famUUID(p.t, p.a, p.ca, p.sa, "tournament", tour)
		},
		create: func(p *csPair, i int) int64 {
			h := tieName("tie", i)
			id, err := p.a.Matches().Save(p.ca, p.sa, &domain.Match{Player1Name: "P", Player2Name: "Q", MatchLength: 3, FilePath: h + ".xg", MatchHash: h, CanonicalHash: h + "c"})
			if err != nil {
				p.t.Fatal(err)
			}
			if err := p.a.Tournaments().AddMatch(p.ca, p.sa, tour, id); err != nil {
				p.t.Fatal(err)
			}
			return id
		},
		editA: func(p *csPair, i int, id int64) {
			if i%3 == 0 {
				_ = p.a.Matches().UpdateComment(p.ca, p.sa, id, fmt.Sprintf("note-a-%d", i))
			}
		},
		editB: func(p *csPair, i int, id int64) {
			switch i % 3 {
			case 1:
				_ = p.a.Tournaments().RemoveMatch(p.ca, p.sa, id)
			case 2:
				_ = p.a.Matches().UpdateComment(p.ca, p.sa, id, fmt.Sprintf("note-b-%d", i))
			}
		},
		check: func(p *csPair, i int, id int64) string {
			// A detachment is a gesture like any other: at a tie to the
			// millisecond the sides agree on one outcome (checked for all), and
			// only the matches nobody detached must have stayed.
			if i%3 != 1 {
				tr, err := p.a.Tournaments().TournamentOf(p.ca, p.sa, id)
				if err != nil {
					return "the match left its tournament"
				}
				if got := famUUID(p.t, p.a, p.ca, p.sa, "tournament", tr.ID); got != tourU {
					return "the match is in another tournament"
				}
			}
			m, err := p.a.Matches().Get(p.ca, p.sa, id)
			if err != nil {
				return err.Error()
			}
			if i%3 == 0 && !has(m.Comment, fmt.Sprintf("note-a-%d", i)) {
				return "a note of a was lost"
			}
			if i%3 == 2 && !has(m.Comment, fmt.Sprintf("note-b-%d", i)) {
				return "a note of b was lost"
			}
			return ""
		},
	}
}

// Matches that reached b before their tournament did stay detached there; the
// tournament, once it arrives, takes them in whatever the dates tie on.
func tieMatchBeforeTournament(p *csPair) {
	t := p.t
	p.catchUp()
	var ids [tieN]int64
	for i := range tieN {
		h := tieName("late", i)
		ids[i], _ = p.a.Matches().Save(p.ca, p.sa, &domain.Match{Player1Name: "P", Player2Name: "Q", MatchLength: 3, FilePath: h + ".xg", MatchHash: h, CanonicalHash: h + "c"})
	}
	p.send()
	tid, _ := p.a.Tournaments().Create(p.ca, p.sa, tieName("Late", 0), "", "")
	tu := famUUID(t, p.a, p.ca, p.sa, "tournament", tid)
	for _, id := range ids {
		if err := p.a.Tournaments().AddMatch(p.ca, p.sa, tid, id); err != nil {
			t.Fatal(err)
		}
	}
	set, err := changeset.Export(p.ca, p.a, p.sa, p.cursorFrom, 1_000_000)
	if err != nil {
		t.Fatal(err)
	}
	// b has the matches, dated as a dates them, but not the tournament yet.
	var without []changeset.Record
	for _, r := range set.Records {
		if r.Family != "tournament" {
			without = append(without, r)
		}
	}
	if _, err := changeset.Import(p.cb, p.b, p.sb, changeset.Set{Records: without}); err != nil {
		t.Fatal(err)
	}
	p.replay()
	p.back()
	p.replay()
	count := func(q *csPair) int {
		n := 0
		if bt, ok := famID(q.a, q.ca, q.sa, "tournament", tu); ok {
			for _, err := range q.a.Tournaments().Matches(q.ca, q.sa, bt) {
				if err != nil {
					t.Fatal(err)
				}
				n++
			}
		}
		return n
	}
	if n := count(p); n != tieN {
		t.Errorf("a: %d/%d matches in the tournament", n, tieN)
	}
	if n := count(p.swapped()); n != tieN {
		t.Errorf("b: %d/%d matches in the tournament", n, tieN)
	}
	if got, want := tournamentOrder(p), tournamentOrder(p.swapped()); !slices.Equal(got, want) {
		t.Errorf("the order differs: %v vs %v", got, want)
	}
}

func tournamentOrder(p *csPair) []string {
	var out []string
	for tr, err := range p.a.Tournaments().List(p.ca, p.sa, storage.ListOpts{}) {
		if err != nil {
			p.t.Fatal(err)
		}
		if tr.Name != tieName("Late", 0) {
			continue
		}
		for m, err := range p.a.Tournaments().Matches(p.ca, p.sa, tr.ID) {
			if err != nil {
				p.t.Fatal(err)
			}
			out = append(out, m.MatchHash)
		}
	}
	return out
}

func tieCard() tieCase {
	var deck int64
	var pos [tieN]int64
	return tieCase{
		family: "anki_card",
		prepare: func(p *csPair) {
			for i := range tieN {
				q := tieBoard(i)
				pos[i], _ = p.a.Positions().Save(p.ca, p.sa, &q)
			}
			deck, _ = p.a.Anki().CreateDeck(p.ca, p.sa, tieName("Cards", 0), "", domain.AnkiSourceSearch, 0, "")
			if err := p.a.Anki().SyncWithPositions(p.ca, p.sa, deck, pos[:]); err != nil {
				p.t.Fatal(err)
			}
		},
		create: func(p *csPair, i int) int64 {
			id, _ := cardOf(p, p.a, p.ca, p.sa, deck, pos[i])
			return id
		},
		editA: func(p *csPair, i int, id int64) {
			if i%3 != 1 {
				review(p, p.a, p.ca, p.sa, id, 1)
			}
		},
		editB: func(p *csPair, i int, id int64) {
			if i%3 != 0 {
				review(p, p.a, p.ca, p.sa, id, 1+i%3-1)
			}
		},
		check: func(p *csPair, i int, id int64) string {
			c, err := p.a.(storage.AnkiStateProvider).AnkiStates().Card(p.ca, p.sa, id)
			if err != nil {
				return err.Error()
			}
			want := 1
			if i%3 == 2 {
				want = 2
			}
			if c.Reps != want {
				return fmt.Sprintf("the review was lost (reps %d, want %d)", c.Reps, want)
			}
			return ""
		},
	}
}

func tiePosition() tieCase {
	return tieCase{
		family: "position",
		create: func(p *csPair, i int) int64 {
			q := tieBoard(i)
			id, err := p.a.Positions().Save(p.ca, p.sa, &q)
			if err != nil {
				p.t.Fatal(err)
			}
			return id
		},
		editA: func(p *csPair, i int, id int64) {
			if i%3 != 1 {
				_ = p.a.Positions().SetIndividuallyImported(p.ca, p.sa, id, true)
			}
		},
		editB: func(p *csPair, i int, id int64) {
			if i%3 != 0 {
				_ = p.a.Positions().SetIndividuallyImported(p.ca, p.sa, id, true)
			}
		},
		check: func(p *csPair, i int, id int64) string {
			q, err := p.a.Positions().Load(p.ca, p.sa, id)
			if err != nil {
				return err.Error()
			}
			if !q.IndividuallyImported {
				return "a mark was lost"
			}
			return ""
		},
	}
}

func tieComment() tieCase {
	var pos [tieN]int64
	return tieCase{
		family: "comment",
		prepare: func(p *csPair) {
			for i := range tieN {
				q := tieBoard(i)
				pos[i], _ = p.a.Positions().Save(p.ca, p.sa, &q)
			}
		},
		create: func(p *csPair, i int) int64 {
			id, err := p.a.Comments().Add(p.ca, p.sa, pos[i], "first")
			if err != nil {
				p.t.Fatal(err)
			}
			return id
		},
		editA: func(p *csPair, i int, id int64) {
			_ = p.a.Comments().Update(p.ca, p.sa, id, fmt.Sprintf("from a %d", i))
		},
		editB: func(p *csPair, i int, id int64) {
			_ = p.a.Comments().Update(p.ca, p.sa, id, fmt.Sprintf("from b %d", i))
		},
		check: func(*csPair, int, int64) string { return "" },
	}
}

func tieLesson() tieCase {
	return tieCase{
		family: "lesson",
		create: func(p *csPair, i int) int64 {
			id, err := p.a.Lessons().Create(p.ca, p.sa, tieName("Lesson", i), "")
			if err != nil {
				p.t.Fatal(err)
			}
			return id
		},
		editA: func(p *csPair, i int, id int64) {
			_ = p.a.Lessons().Update(p.ca, p.sa, id, tieName("Lesson", i), "from a")
		},
		editB: func(p *csPair, i int, id int64) {
			_ = p.a.Lessons().Update(p.ca, p.sa, id, tieName("Lesson", i), "from b")
		},
		check: func(*csPair, int, int64) string { return "" },
	}
}

func tieTranscription() tieCase {
	var match [tieN]int64
	return tieCase{
		family: "transcription",
		prepare: func(p *csPair) {
			for i := range tieN {
				h := tieName("trtie", i)
				match[i], _ = p.a.Matches().Save(p.ca, p.sa, &domain.Match{Player1Name: "P", Player2Name: "Q", MatchLength: 3, FilePath: h + ".xg", MatchHash: h, CanonicalHash: h + "c"})
			}
		},
		create: func(p *csPair, i int) int64 {
			id, err := p.a.Transcriptions().Save(p.ca, p.sa, &storage.Transcription{FormatVersion: "1", Label: tieName("draft", i), Document: "{}"})
			if err != nil {
				p.t.Fatal(err)
			}
			return id
		},
		editA: func(p *csPair, i int, id int64) {
			if i%2 == 0 {
				t, _ := p.a.Transcriptions().Get(p.ca, p.sa, id)
				t.MatchID, t.Revision = match[i], 0
				_, _ = p.a.Transcriptions().Save(p.ca, p.sa, t)
			}
		},
		editB: func(p *csPair, i int, id int64) {},
		check: func(p *csPair, i int, id int64) string {
			t, err := p.a.Transcriptions().Get(p.ca, p.sa, id)
			if err != nil {
				return err.Error()
			}
			if i%2 == 0 && t.MatchID == 0 {
				return "the link to the match was lost"
			}
			return ""
		},
	}
}

// Four matches joined in one order and moved twice reach a peer that has none
// of them in the order they end in, whatever order the journal lists them in.
func scenarioOrderOnFreshPeer(p *csPair) {
	t := p.t
	tieRun++
	p.catchUp()
	tid, _ := p.a.Tournaments().Create(p.ca, p.sa, tieName("Order", 0), "", "")
	var m [4]int64
	for i, h := range []string{"y", "z", "w", "x"} {
		m[i], _ = p.a.Matches().Save(p.ca, p.sa, &domain.Match{Player1Name: "P", Player2Name: "Q", MatchLength: 3,
			FilePath: tieName(h, 0) + ".xg", MatchHash: tieName(h, 0), CanonicalHash: tieName(h, 0) + "c"})
		if err := p.a.Tournaments().AddMatch(p.ca, p.sa, tid, m[i]); err != nil {
			t.Fatal(err)
		}
		tick()
	}
	y, z, w, x := m[0], m[1], m[2], m[3]
	for _, order := range [][]int64{{w, y, z, x}, {z, y, w, x}} {
		if err := p.a.Tournaments().ReorderMatches(p.ca, p.sa, tid, order); err != nil {
			t.Fatal(err)
		}
		tick()
	}
	p.replay()
	p.back()
	read := func(q *csPair) []string {
		id, ok := famID(q.a, q.ca, q.sa, "tournament", famUUID(t, p.a, p.ca, p.sa, "tournament", tid))
		if !ok {
			t.Fatal("the tournament did not arrive")
		}
		var out []string
		for mm, err := range q.a.Tournaments().Matches(q.ca, q.sa, id) {
			if err != nil {
				t.Fatal(err)
			}
			out = append(out, strings.TrimSuffix(mm.MatchHash, " "+fmt.Sprint(tieRun)+".0"))
		}
		return out
	}
	want := []string{"z", "y", "w", "x"}
	if got := read(p); !slices.Equal(got, want) {
		t.Errorf("a: %v, want %v", got, want)
	}
	if got := read(p.swapped()); !slices.Equal(got, want) {
		t.Errorf("b: %v, want %v", got, want)
	}
}

// Filling a Slot of a directed tournament is a gesture of the library: it
// dates the match no more than a look at it does.
func scenarioSlotDatesNothing(p *csPair) {
	t := p.t
	tieRun++
	p.catchUp()
	tid, _ := p.a.Tournaments().Create(p.ca, p.sa, tieName("Slots", 0), "", "")
	h := tieName("slot", 0)
	id, _ := p.a.Matches().Save(p.ca, p.sa, &domain.Match{Player1Name: "P", Player2Name: "Q", MatchLength: 3, FilePath: h + ".xg", MatchHash: h, CanonicalHash: h + "c"})
	if err := p.a.Tournaments().AddMatch(p.ca, p.sa, tid, id); err != nil {
		t.Fatal(err)
	}
	p.send()
	mu := famUUID(t, p.a, p.ca, p.sa, "match", id)
	before := p.entry(p.a, p.ca, p.sa, mu)
	tick()
	if err := p.a.Directions().AttachSlot(p.ca, p.sa, tid, "slot-1", id); err != nil {
		t.Fatal(err)
	}
	if after := p.entry(p.a, p.ca, p.sa, mu); after.ChangedAt != before.ChangedAt {
		t.Errorf("filling a slot dated the match: %s -> %s", before.ChangedAt, after.ChangedAt)
	}
	if rep := p.send(); rep.Applied != 0 {
		t.Errorf("filling a slot travelled: %+v", rep)
	}
}

// An entry whose object is gone is left out of the page, and the page says so.
func scenarioSkippedIsCounted(p *csPair) {
	t := p.t
	p.catchUp()
	ghost := "01a1a8d5-0000-7000-8000-000000000042"
	if err := p.a.(storage.ChangeJournalProvider).ChangeJournal().Mark(p.ca, p.sa, "lesson", ghost, storage.ChangeUpdate, "2026-11-05T00:00:00.000Z"); err != nil {
		t.Fatal(err)
	}
	set, err := changeset.Export(p.ca, p.a, p.sa, p.cursorFrom, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if set.Skipped != 1 {
		t.Errorf("a page that left one object out says Skipped = %d", set.Skipped)
	}
	for _, r := range set.Records {
		if r.UUID == ghost {
			t.Error("the ghost travelled")
		}
	}
}

// A living collection whose query names rows by number reaches the peer
// without that query (the numbers mean other rows there); one that does not,
// keeps it.
func scenarioLivingQueryOfRows(p *csPair) {
	t := p.t
	p.catchUp()
	byRows, _ := p.a.Collections().Create(p.ca, p.sa, tieName("Living rows", 0), "")
	plain, _ := p.a.Collections().Create(p.ca, p.sa, tieName("Living plain", 0), "")
	if err := p.a.Collections().SetFilterQuery(p.ca, p.sa, byRows, "ma7 tn3"); err != nil {
		t.Fatal(err)
	}
	if err := p.a.Collections().SetFilterQuery(p.ca, p.sa, plain, "S"); err != nil {
		t.Fatal(err)
	}
	p.send()
	queryOf := func(id int64) string {
		c, err := p.b.Collections().Get(p.cb, p.sb, id)
		if err != nil {
			t.Fatal(err)
		}
		return c.FilterQuery
	}
	if got := queryOf(famIDMust(p, p.b, p.cb, p.sb, "collection", famUUID(t, p.a, p.ca, p.sa, "collection", byRows))); got != "" {
		t.Errorf("the query that names rows reached the peer: %q", got)
	}
	if got := queryOf(famIDMust(p, p.b, p.cb, p.sb, "collection", famUUID(t, p.a, p.ca, p.sa, "collection", plain))); got != "S" {
		t.Errorf("the query that names no row changed on the way: %q", got)
	}
}
