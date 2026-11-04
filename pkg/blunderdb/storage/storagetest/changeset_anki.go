// Scenarios of the changeset package for the Anki decks and cards, the
// transcriptions and the tournaments.
package storagetest

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// CheckChangeSetAnkiFamilies runs them on two libraries.
func CheckChangeSetAnkiFamilies(t *testing.T, a storage.Storage, ca context.Context, sa string, b storage.Storage, cb context.Context, sb string) {
	pair := func() *csPair { return &csPair{t: t, a: a, ca: ca, sa: sa, b: b, cb: cb, sb: sb} }
	t.Run("AnkiDeckAndCards", func(t *testing.T) { p := pair(); p.t = t; scenarioAnki(p) })
	t.Run("AnkiSameCardTwoUUIDs", func(t *testing.T) { p := pair(); p.t = t; scenarioAnkiTwoUUIDs(p) })
	t.Run("AnkiSearchDeck", func(t *testing.T) { p := pair(); p.t = t; scenarioAnkiSearch(p) })
	t.Run("Transcription", func(t *testing.T) { p := pair(); p.t = t; scenarioTranscription(p) })
	t.Run("Tournament", func(t *testing.T) { p := pair(); p.t = t; scenarioTournament(p) })
	t.Run("TrashIsADelete", func(t *testing.T) { p := pair(); p.t = t; scenarioTrashWithoutBin(p) })
	checkChangeSetReview(t, pair)
}

// sent imports one record of a peer into b, dated now.
func (p *csPair) sent(family, uuid, op string) changeset.Report {
	p.t.Helper()
	rep, err := changeset.Import(p.cb, p.b, p.sb, changeset.Set{Records: []changeset.Record{{
		Change: storage.Change{UUID: uuid, Family: family, Op: op, ChangedAt: time.Now().UTC().Format(storage.ChangedAtLayout)}}}})
	if err != nil {
		p.t.Fatal(err)
	}
	return rep
}

// deckCards reads a deck's cards keyed by the board they ask about, without
// the ids that differ between libraries.
func (p *csPair) deckCards(s storage.Storage, ctx context.Context, scope string, deck int64) map[string]domain.AnkiCard {
	p.t.Helper()
	out := map[string]domain.AnkiCard{}
	for c, err := range s.Anki().DeckCards(ctx, scope, deck) {
		if err != nil {
			p.t.Fatal(err)
		}
		q, err := s.Positions().Load(ctx, scope, c.PositionID)
		if err != nil {
			p.t.Fatal(err)
		}
		key := boardKey(q)
		card := *c
		card.ID, card.DeckID, card.PositionID, card.Key = 0, 0, 0, ""
		out[key] = card
	}
	return out
}

func boardKey(q *domain.Position) string { return strconv.FormatUint(storage.ZobristOf(q), 10) }

func deckOf(p *csPair, s storage.Storage, ctx context.Context, scope string, id int64) *domain.AnkiDeck {
	p.t.Helper()
	d, err := s.(storage.AnkiStateProvider).AnkiStates().Deck(ctx, scope, id)
	if err != nil {
		p.t.Fatal(err)
	}
	return d
}

// A deck travels with its settings, its source collection and its cards with
// their scheduling; a review, a suspension, an edit and a delete follow, the
// later write winning.
func scenarioAnki(p *csPair) {
	t := p.t
	p.catchUp()
	var pa [3]int64
	for i := range pa {
		q := syncPos(600 + i)
		pa[i], _ = p.a.Positions().Save(p.ca, p.sa, &q)
	}
	col, _ := p.a.Collections().Create(p.ca, p.sa, "Deck source", "")
	deck, err := p.a.Anki().CreateDeck(p.ca, p.sa, "Openings", "to learn", domain.AnkiSourceCollection, col, "")
	if err != nil {
		t.Fatal(err)
	}
	limit := 12
	if err := p.a.Anki().UpdateDeckParams(p.ca, p.sa, deck, 0.85, 3650, false, &limit); err != nil {
		t.Fatal(err)
	}
	if err := p.a.Anki().SyncWithPositions(p.ca, p.sa, deck, pa[:]); err != nil {
		t.Fatal(err)
	}
	first, ok := cardOf(p, p.a, p.ca, p.sa, deck, pa[0])
	if !ok {
		t.Fatal("no card for the first board")
	}
	if _, err := p.a.Anki().ReviewCard(p.ca, p.sa, first, 3); err != nil {
		t.Fatal(err)
	}
	second, _ := cardOf(p, p.a, p.ca, p.sa, deck, pa[1])
	if err := p.a.Anki().SetCardSuspended(p.ca, p.sa, second, true); err != nil {
		t.Fatal(err)
	}
	du := famUUID(t, p.a, p.ca, p.sa, "anki_deck", deck)
	cu := famUUID(t, p.a, p.ca, p.sa, "anki_card", first)
	rep := p.send()
	if len(rep.Conflicts) != 0 || rep.Added < 4 {
		t.Fatalf("first send: %+v", rep)
	}
	bdeck, ok := famID(p.b, p.cb, p.sb, "anki_deck", du)
	if !ok {
		t.Fatal("the deck did not arrive under its uuid")
	}
	d := deckOf(p, p.b, p.cb, p.sb, bdeck)
	if d.Name != "Openings" || d.RequestRetention != 0.85 || d.MaximumInterval != 3650 || d.EnableFuzz || d.SessionLimit == nil || *d.SessionLimit != 12 {
		t.Fatalf("deck in b = %+v", d)
	}
	bcol, ok := famID(p.b, p.cb, p.sb, "collection", famUUID(t, p.a, p.ca, p.sa, "collection", col))
	if !ok || d.SourceType != domain.AnkiSourceCollection || d.SourceID != bcol {
		t.Errorf("the deck's source collection did not follow: %+v (collection %d)", d, bcol)
	}
	ca, cbs := p.deckCards(p.a, p.ca, p.sa, deck), p.deckCards(p.b, p.cb, p.sb, bdeck)
	if len(cbs) != 3 {
		t.Fatalf("cards in b: %d, want 3", len(cbs))
	}
	for k, c := range ca {
		if cbs[k] != c {
			t.Errorf("card %s differs: a %+v, b %+v", k, c, cbs[k])
		}
	}
	if bc, ok := famID(p.b, p.cb, p.sb, "anki_card", cu); !ok || bc == 0 {
		t.Error("the card did not arrive under its uuid")
	}
	if rep := p.replay(); rep.Applied != 0 {
		t.Errorf("a replay applied %+v", rep)
	}

	// A review on a travels.
	tick()
	if _, err := p.a.Anki().ReviewCard(p.ca, p.sa, first, 4); err != nil {
		t.Fatal(err)
	}
	p.send()
	if got := p.deckCards(p.b, p.cb, p.sb, bdeck); got[boardKey(mustLoad(p, p.a, p.ca, p.sa, pa[0]))].Reps != 2 {
		t.Errorf("the second review did not travel: %+v", got)
	}
	// b reviews after a edited the same card: b stays.
	tick()
	_ = p.a.Anki().SetCardSuspended(p.ca, p.sa, first, true)
	tick()
	bfirst, _ := famID(p.b, p.cb, p.sb, "anki_card", cu)
	if _, err := p.b.Anki().ReviewCard(p.cb, p.sb, bfirst, 1); err != nil {
		t.Fatal(err)
	}
	p.send()
	c, _ := ankiCard(p, p.b, p.cb, p.sb, bdeck, bfirst)
	if c.Suspended || c.Reps != 3 {
		t.Errorf("the newer local review was lost: %+v", c)
	}
	// A deck edit travels.
	tick()
	if err := p.a.Anki().UpdateDeck(p.ca, p.sa, deck, "Openings 2", "more"); err != nil {
		t.Fatal(err)
	}
	p.send()
	if d := deckOf(p, p.b, p.cb, p.sb, bdeck); d.Name != "Openings 2" {
		t.Errorf("the rename did not travel: %+v", d)
	}
	// A card removed on a leaves b; so does the whole deck, cards included.
	tick()
	if err := p.a.Anki().RemoveCard(p.ca, p.sa, second); err != nil {
		t.Fatal(err)
	}
	p.send()
	if len(p.deckCards(p.b, p.cb, p.sb, bdeck)) != 2 {
		t.Error("the removed card is still in b")
	}
	tick()
	if err := p.a.Anki().DeleteDeck(p.ca, p.sa, deck); err != nil {
		t.Fatal(err)
	}
	p.send()
	if _, ok := famID(p.b, p.cb, p.sb, "anki_deck", du); ok {
		t.Error("the deleted deck is still in b")
	}
	if _, ok := famID(p.b, p.cb, p.sb, "anki_card", cu); ok {
		t.Error("a card outlived its deck in b")
	}
	if e := p.entry(p.b, p.cb, p.sb, cu); e.Op != storage.ChangeDelete {
		t.Errorf("b's journal says %s for the card, want delete", e.Op)
	}
}

func famIDMust(p *csPair, s storage.Storage, ctx context.Context, scope, family, uuid string) int64 {
	p.t.Helper()
	id, ok := famID(s, ctx, scope, family, uuid)
	if !ok {
		p.t.Fatalf("no %s %s", family, uuid)
	}
	return id
}

func mustLoad(p *csPair, s storage.Storage, ctx context.Context, scope string, id int64) *domain.Position {
	q, err := s.Positions().Load(ctx, scope, id)
	if err != nil {
		p.t.Fatal(err)
	}
	return q
}

func cardOf(p *csPair, s storage.Storage, ctx context.Context, scope string, deck, position int64) (int64, bool) {
	for c, err := range s.Anki().DeckCards(ctx, scope, deck) {
		if err != nil {
			p.t.Fatal(err)
		}
		if c.PositionID == position {
			return c.ID, true
		}
	}
	return 0, false
}

func ankiCard(p *csPair, s storage.Storage, ctx context.Context, scope string, deck, id int64) (domain.AnkiCard, bool) {
	for c, err := range s.Anki().DeckCards(ctx, scope, deck) {
		if err != nil {
			p.t.Fatal(err)
		}
		if c.ID == id {
			return *c, true
		}
	}
	return domain.AnkiCard{}, false
}

// Two libraries that drew the same card in the same deck before they met keep
// the older uuid and one card.
func scenarioAnkiTwoUUIDs(p *csPair) {
	t := p.t
	p.catchUp()
	q := syncPos(610)
	pa, _ := p.a.Positions().Save(p.ca, p.sa, &q)
	q2 := syncPos(610)
	pb, _ := p.b.Positions().Save(p.cb, p.sb, &q2)
	deck, _ := p.a.Anki().CreateDeck(p.ca, p.sa, "Shared", "", domain.AnkiSourceSearch, 0, "")
	du := famUUID(t, p.a, p.ca, p.sa, "anki_deck", deck)
	p.send()
	bdeck, ok := famID(p.b, p.cb, p.sb, "anki_deck", du)
	if !ok {
		t.Fatal("no deck in b")
	}
	tick()
	if err := p.a.Anki().SyncWithPositions(p.ca, p.sa, deck, []int64{pa}); err != nil {
		t.Fatal(err)
	}
	tick()
	if err := p.b.Anki().SyncWithPositions(p.cb, p.sb, bdeck, []int64{pb}); err != nil {
		t.Fatal(err)
	}
	ac, _ := cardOf(p, p.a, p.ca, p.sa, deck, pa)
	bc, _ := cardOf(p, p.b, p.cb, p.sb, bdeck, pb)
	ua, ub := famUUID(t, p.a, p.ca, p.sa, "anki_card", ac), famUUID(t, p.b, p.cb, p.sb, "anki_card", bc)
	older := min(ua, ub)
	rep := p.send()
	if len(rep.Conflicts) != 0 {
		t.Fatalf("the twin card conflicted: %+v", rep)
	}
	if n := len(p.deckCards(p.b, p.cb, p.sb, bdeck)); n != 1 {
		t.Fatalf("b holds %d cards of the board, want 1", n)
	}
	if got := famUUID(t, p.b, p.cb, p.sb, "anki_card", bc); got != older {
		t.Errorf("b kept %s, want the older %s", got, older)
	}
}

// A search deck's source lists local position ids; across libraries it names
// the boards instead.
func scenarioAnkiSearch(p *csPair) {
	t := p.t
	p.catchUp()
	var pa, pb [2]int64
	for i := range pa {
		q, q2 := syncPos(620+i), syncPos(620+i)
		pa[i], _ = p.a.Positions().Save(p.ca, p.sa, &q)
		pb[len(pb)-1-i], _ = p.b.Positions().Save(p.cb, p.sb, &q2) // opposite order: ids differ
	}
	deck, _ := p.a.Anki().CreateDeck(p.ca, p.sa, "Found", "", domain.AnkiSourceSearch,
		0, `{"command":"x","ids":[`+itoa(pa[0])+`,`+itoa(pa[1])+`]}`)
	du := famUUID(t, p.a, p.ca, p.sa, "anki_deck", deck)
	p.send()
	bdeck, ok := famID(p.b, p.cb, p.sb, "anki_deck", du)
	if !ok {
		t.Fatal("no deck in b")
	}
	d := deckOf(p, p.b, p.cb, p.sb, bdeck)
	want := []int64{pb[1], pb[0]} // the boards of a's first and second, as b numbered them
	got := domain.SearchSourceIDs(d.SourceCommand)
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("the search source names %v in b, want %v", got, want)
	}
}

// A draft travels with its document and the match it produced, by uuid; its
// edit and its deletion follow. A match this library lacks leaves the draft
// standing as one that produced none, and the report names it.
func scenarioTranscription(p *csPair) {
	t := p.t
	p.catchUp()
	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 5, FilePath: "t.xg", GameCount: 0,
		MatchHash: "tr-hash", CanonicalHash: "tr-canon"}
	mid, err := p.a.Matches().Save(p.ca, p.sa, &m)
	if err != nil {
		t.Fatal(err)
	}
	tid, err := p.a.Transcriptions().Save(p.ca, p.sa, &storage.Transcription{FormatVersion: "1", MatchID: mid, Label: "Alice-Bob", Document: `{"moves":[1,2]}`})
	if err != nil {
		t.Fatal(err)
	}
	loose, _ := p.a.Transcriptions().Save(p.ca, p.sa, &storage.Transcription{FormatVersion: "1", Label: "unfinished", Document: "{}"})
	tu, lu := famUUID(t, p.a, p.ca, p.sa, "transcription", tid), famUUID(t, p.a, p.ca, p.sa, "transcription", loose)
	rep := p.send()
	if len(rep.Conflicts) != 0 || len(rep.Missing) != 0 {
		t.Fatalf("first send: %+v", rep)
	}
	bid, ok := famID(p.b, p.cb, p.sb, "transcription", tu)
	if !ok {
		t.Fatal("the draft did not arrive under its uuid")
	}
	got, err := p.b.Transcriptions().Get(p.cb, p.sb, bid)
	bmatch, _ := famID(p.b, p.cb, p.sb, "match", famUUID(t, p.a, p.ca, p.sa, "match", mid))
	if err != nil || got.Label != "Alice-Bob" || got.Document != `{"moves":[1,2]}` || got.FormatVersion != "1" || got.MatchID != bmatch || bmatch == 0 {
		t.Fatalf("draft in b = %+v, %v (match %d)", got, err, bmatch)
	}
	if want, _ := p.a.Transcriptions().Get(p.ca, p.sa, tid); got.CreatedAt != want.CreatedAt || got.UpdatedAt != want.UpdatedAt {
		t.Errorf("the draft lost its dates: %s/%s vs %s/%s", got.CreatedAt, got.UpdatedAt, want.CreatedAt, want.UpdatedAt)
	}
	if rep := p.replay(); rep.Applied != 0 {
		t.Errorf("a replay applied %+v", rep)
	}

	tick()
	edit, _ := p.a.Transcriptions().Get(p.ca, p.sa, tid)
	edit.Document, edit.Revision = `{"moves":[1,2,3]}`, 0
	if _, err := p.a.Transcriptions().Save(p.ca, p.sa, edit); err != nil {
		t.Fatal(err)
	}
	p.send()
	if g, _ := p.b.Transcriptions().Get(p.cb, p.sb, bid); g.Document != `{"moves":[1,2,3]}` {
		t.Errorf("the edit did not travel: %+v", g)
	}
	// b types after a did: b stays.
	tick()
	_, _ = p.a.Transcriptions().Save(p.ca, p.sa, &storage.Transcription{ID: tid, FormatVersion: "1", MatchID: mid, Label: "from a", Document: "{}"})
	tick()
	_, _ = p.b.Transcriptions().Save(p.cb, p.sb, &storage.Transcription{ID: bid, FormatVersion: "1", MatchID: bmatch, Label: "from b", Document: "{}"})
	p.send()
	if g, _ := p.b.Transcriptions().Get(p.cb, p.sb, bid); g.Label != "from b" {
		t.Errorf("the newer local draft was lost: %+v", g)
	}
	// A draft whose match b never held: it stands alone and the report says so.
	tick()
	if err := p.a.Matches().DeleteCascade(p.ca, p.sa, mid); err != nil {
		t.Fatal(err)
	}
	loneMatch, _ := p.a.Matches().Save(p.ca, p.sa, &domain.Match{Player1Name: "C", Player2Name: "D", MatchLength: 3, FilePath: "x.xg", MatchHash: "tr2", CanonicalHash: "tr2c"})
	orphan, _ := p.a.Transcriptions().Save(p.ca, p.sa, &storage.Transcription{FormatVersion: "1", MatchID: loneMatch, Label: "orphan", Document: "{}"})
	ou := famUUID(t, p.a, p.ca, p.sa, "transcription", orphan)
	set, err := changeset.Export(p.ca, p.a, p.sa, p.cursorFrom, 1000)
	if err != nil {
		t.Fatal(err)
	}
	var only changeset.Set
	for _, r := range set.Records {
		if r.Family == "transcription" && r.UUID == ou {
			only.Records = append(only.Records, r)
		}
	}
	rep, err = changeset.Import(p.cb, p.b, p.sb, only)
	if err != nil || rep.Added != 1 || len(rep.Missing) != 1 {
		t.Fatalf("a draft of an unknown match: %+v, %v", rep, err)
	}
	if oid, _ := famID(p.b, p.cb, p.sb, "transcription", ou); oid == 0 {
		t.Error("the draft of the unknown match was not kept")
	} else if g, _ := p.b.Transcriptions().Get(p.cb, p.sb, oid); g.MatchID != 0 {
		t.Errorf("it points at match %d", g.MatchID)
	}

	// Everything else follows: the match is there now, and the draft takes it.
	tick()
	p.send()
	if g, _ := p.b.Transcriptions().Get(p.cb, p.sb, famIDMust(p, p.b, p.cb, p.sb, "transcription", ou)); g.MatchID == 0 {
		t.Error("the draft did not take its match once it arrived")
	}
	tick()
	_ = p.a.Transcriptions().Delete(p.ca, p.sa, loose)
	p.send()
	if _, ok := famID(p.b, p.cb, p.sb, "transcription", lu); ok {
		t.Error("the deleted draft is still in b")
	}
	if e := p.entry(p.b, p.cb, p.sb, lu); e.Op != storage.ChangeDelete {
		t.Errorf("b's journal says %s, want delete", e.Op)
	}
}

// A tournament travels with its header, its note and its dates; an edit and a
// delete follow, the later write winning. Its matches stay where each library
// put them.
func scenarioTournament(p *csPair) {
	t := p.t
	p.catchUp()
	tid, err := p.a.Tournaments().Create(p.ca, p.sa, "Monte Carlo", "2026-05-01", "Monaco")
	if err != nil {
		t.Fatal(err)
	}
	if err := p.a.Tournaments().UpdateComment(p.ca, p.sa, tid, "final in 3 days"); err != nil {
		t.Fatal(err)
	}
	u := famUUID(t, p.a, p.ca, p.sa, "tournament", tid)
	p.send()
	bid, ok := famID(p.b, p.cb, p.sb, "tournament", u)
	if !ok {
		t.Fatal("the tournament did not arrive under its uuid")
	}
	got, err := p.b.Tournaments().Get(p.cb, p.sb, bid)
	if err != nil || got.Name != "Monte Carlo" || got.Date != "2026-05-01" || got.Location != "Monaco" || got.Comment != "final in 3 days" {
		t.Fatalf("tournament in b = %+v, %v", got, err)
	}
	if want, _ := p.a.Tournaments().Get(p.ca, p.sa, tid); got.CreatedAt != want.CreatedAt || got.UpdatedAt != want.UpdatedAt {
		t.Errorf("the tournament lost its dates: %s/%s vs %s/%s", got.CreatedAt, got.UpdatedAt, want.CreatedAt, want.UpdatedAt)
	}
	if rep := p.replay(); rep.Applied != 0 {
		t.Errorf("a replay applied %+v", rep)
	}
	tick()
	_ = p.a.Tournaments().Update(p.ca, p.sa, tid, "Monte-Carlo Open", "2026-05-02", "Monaco")
	p.send()
	if g, _ := p.b.Tournaments().Get(p.cb, p.sb, bid); g.Name != "Monte-Carlo Open" || g.Date != "2026-05-02" {
		t.Errorf("the edit did not travel: %+v", g)
	}
	tick()
	_ = p.a.Tournaments().UpdateComment(p.ca, p.sa, tid, "from a")
	tick()
	_ = p.b.Tournaments().UpdateComment(p.cb, p.sb, bid, "from b")
	p.send()
	if g, _ := p.b.Tournaments().Get(p.cb, p.sb, bid); g.Comment != "from b" {
		t.Errorf("the newer local note was lost: %+v", g)
	}
	tick()
	_ = p.a.Tournaments().Delete(p.ca, p.sa, tid)
	p.send()
	if _, ok := famID(p.b, p.cb, p.sb, "tournament", u); ok {
		t.Error("the deleted tournament is still in b")
	}
}

// These families have no bin: a trash from a peer removes the object here and
// the journal says trash, so an older write cannot bring it back.
func scenarioTrashWithoutBin(p *csPair) {
	t := p.t
	p.catchUp()
	for _, fam := range []string{"tournament", "anki_deck", "transcription"} {
		var id int64
		switch fam {
		case "tournament":
			id, _ = p.a.Tournaments().Create(p.ca, p.sa, "Bin", "", "")
		case "anki_deck":
			id, _ = p.a.Anki().CreateDeck(p.ca, p.sa, "Bin", "", domain.AnkiSourceSearch, 0, "")
		default:
			id, _ = p.a.Transcriptions().Save(p.ca, p.sa, &storage.Transcription{FormatVersion: "1", Label: "Bin", Document: "{}"})
		}
		u := famUUID(t, p.a, p.ca, p.sa, fam, id)
		p.send()
		if _, ok := famID(p.b, p.cb, p.sb, fam, u); !ok {
			t.Fatalf("%s did not arrive", fam)
		}
		tick()
		if rep := p.sent(fam, u, storage.ChangeTrash); rep.Removed != 1 {
			t.Errorf("%s: a peer's trash applied %+v", fam, rep)
		}
		if _, ok := famID(p.b, p.cb, p.sb, fam, u); ok {
			t.Errorf("the trashed %s is still in b", fam)
		}
		if e := p.entry(p.b, p.cb, p.sb, u); e.Op != storage.ChangeTrash {
			t.Errorf("%s: b's journal says %s, want trash", fam, e.Op)
		}
		// An older update of it, sent again, does not bring it back.
		if rep := p.replay(); rep.Added != 0 {
			t.Errorf("%s: a replay brought it back: %+v", fam, rep)
		}
	}
}
