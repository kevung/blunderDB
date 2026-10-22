package storagetest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// RunElsewhereTests holds storage.Elsewhere (ADR-0103) on three libraries from
// factory: "here" judges, "club" holds the board with an analysis, a comment
// and a play, "empty" holds only a neighbour. The libraries are separate
// Storages, the shape the CLI reads; the daemon's tenants of one Storage reach
// the same function through SourcesOf.
func RunElsewhereTests(t *testing.T, factory func() storage.Storage) {
	t.Helper()
	ctx := context.Background()
	here, club, empty := factory(), factory(), factory()
	defer here.Close()
	defer club.Close()
	defer empty.Close()

	// The club's play costs 50 mp: an Error at the default thresholds the
	// club keeps, a Blunder at the ones here.
	if err := here.LibrarySettings().Save(ctx, "", storage.LibrarySettings{ErrorThresholdMP: 20, BlunderThresholdMP: 40}); err != nil {
		t.Fatalf("Save settings: %v", err)
	}
	judge, err := storage.JudgeOf(ctx, here, "")
	if err != nil {
		t.Fatalf("JudgeOf: %v", err)
	}

	_, posIDs := statsFixtureMatch(t, club, 0, "Alice", "Bob")
	target, err := club.Positions().Load(ctx, "", posIDs[0])
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := club.Comments().Add(ctx, "", posIDs[0], "too bold"); err != nil {
		t.Fatalf("Add comment: %v", err)
	}
	asked := *target
	asked.ID = 0
	hereID, err := here.Positions().Save(ctx, "", &asked)
	if err != nil {
		t.Fatalf("Save here: %v", err)
	}
	asked.ID = hereID

	near := neighbourOf(t, asked)
	for _, s := range []storage.Storage{club, empty} {
		n := near
		if _, err := s.Positions().Save(ctx, "", &n); err != nil {
			t.Fatalf("Save neighbour: %v", err)
		}
	}
	sources := []storage.Source{{Name: "club", Storage: club}, {Name: "empty", Storage: empty}}
	q := storage.ElsewhereQuery{Target: &asked, Neighbours: 3}

	t.Run("JudgedWithTheReadersThresholds", func(t *testing.T) {
		got, err := storage.Elsewhere(ctx, sources, q, judge)
		if err != nil {
			t.Fatalf("Elsewhere: %v", err)
		}
		if len(got) != 2 || got[0].Tenant != "club" || got[1].Tenant != "empty" {
			t.Fatalf("reports = %+v, want club then empty", got)
		}
		c := got[0].Item
		if c.PositionID != posIDs[0] || c.Analysis == nil || c.OtherMET {
			t.Errorf("club: position %d analysis %v otherMET %v, want %d, an analysis, same table", c.PositionID, c.Analysis != nil, c.OtherMET, posIDs[0])
		}
		if len(c.Comments) != 1 || c.Comments[0].Text != "too bold" {
			t.Errorf("club comments = %+v", c.Comments)
		}
		if len(c.Plays) != 1 || c.Plays[0].Player1 != "Alice" || c.Plays[0].Verdict != storage.VerdictBlunder {
			t.Errorf("club plays = %+v, want one play by Alice judged a blunder here", c.Plays)
		}
		if len(c.Neighbours) == 0 || c.Neighbours[0].Zobrist != storage.ZobristOf(&near) {
			t.Errorf("club neighbours = %+v, want the near board first", c.Neighbours)
		}
		for _, n := range c.Neighbours {
			if n.Zobrist == storage.ZobristOf(&asked) {
				t.Error("the board itself is listed among its neighbours")
			}
		}
		e := got[1].Item
		if e.PositionID != 0 || e.Analysis != nil || len(e.Plays) != 0 || len(e.Comments) != 0 {
			t.Errorf("empty: %+v, want nothing known", e)
		}
		if len(e.Neighbours) != 1 || e.Neighbours[0].Zobrist != storage.ZobristOf(&near) {
			t.Errorf("empty neighbours = %+v, want the near board, ranked against the asked one", e.Neighbours)
		}
	})

	t.Run("ByHashAlone", func(t *testing.T) {
		got, err := storage.Elsewhere(ctx, sources, storage.ElsewhereQuery{Zobrist: storage.ZobristOf(&asked), Neighbours: -1}, judge)
		if err != nil {
			t.Fatalf("Elsewhere: %v", err)
		}
		if got[0].Item.PositionID != posIDs[0] || len(got[0].Item.Neighbours) != 0 {
			t.Errorf("club by hash = %+v, want the held board and no neighbours", got[0].Item)
		}
	})

	t.Run("AnotherTableIsFlagged", func(t *testing.T) {
		metID, err := club.MatchEquityTables().Save(ctx, "", domain.MatchEquityTable{Name: "Club MET", Digest: "club", Source: "<met/>"})
		if err != nil {
			t.Fatalf("Save MET: %v", err)
		}
		if err := club.MatchEquityTables().TagAnalyses(ctx, "", metID, []int64{posIDs[0]}); err != nil {
			t.Fatalf("TagAnalyses: %v", err)
		}
		got, err := storage.Elsewhere(ctx, sources[:1], q, judge)
		if err != nil {
			t.Fatalf("Elsewhere: %v", err)
		}
		c := got[0].Item
		if !c.OtherMET || c.METName != "Club MET" {
			t.Errorf("otherMET %v name %q, want flagged, Club MET", c.OtherMET, c.METName)
		}
		if len(c.Plays) != 1 || c.Plays[0].Verdict != storage.VerdictOtherMET || c.Plays[0].ErrorMP == nil {
			t.Errorf("plays = %+v, want the cost shown and judged other-met", c.Plays)
		}
		// The same table here makes it comparable again.
		same := judge
		same.METDigest = "club"
		got, err = storage.Elsewhere(ctx, sources[:1], q, same)
		if err != nil {
			t.Fatalf("Elsewhere: %v", err)
		}
		if got[0].Item.OtherMET || got[0].Item.Plays[0].Verdict != storage.VerdictBlunder {
			t.Errorf("same table: %+v, want comparable", got[0].Item)
		}
	})

	t.Run("NegativeDistanceRefused", func(t *testing.T) {
		if _, err := storage.Elsewhere(ctx, sources, storage.ElsewhereQuery{Target: &asked, MaxDistance: -1}, judge); err == nil {
			t.Error("a negative maxDistance was accepted")
		}
	})

	t.Run("TruncationIsSaid", func(t *testing.T) {
		busy := factory()
		defer busy.Close()
		for range storage.ElsewhereMaxMatches + 1 {
			statsFixtureMatch(t, busy, 0, "Alice", "Bob")
		}
		got, err := storage.Elsewhere(ctx, []storage.Source{{Name: "busy", Storage: busy}}, storage.ElsewhereQuery{Target: &asked, Neighbours: -1}, judge)
		if err != nil {
			t.Fatalf("Elsewhere: %v", err)
		}
		r := got[0].Item
		if r.MatchesTotal != storage.ElsewhereMaxMatches+1 || !r.PlaysTruncated || len(r.Plays) != storage.ElsewhereMaxMatches {
			t.Errorf("total %d truncated %v plays %d, want %d, true, %d", r.MatchesTotal, r.PlaysTruncated, len(r.Plays), storage.ElsewhereMaxMatches+1, storage.ElsewhereMaxMatches)
		}
	})

	t.Run("NeedsABoard", func(t *testing.T) {
		if _, err := storage.Elsewhere(ctx, sources, storage.ElsewhereQuery{}, judge); err == nil {
			t.Error("a query naming no board was accepted")
		}
	})

	t.Run("ObjectPresence", func(t *testing.T) {
		testElsewhereObject(t, here, club, empty)
	})
}

// testElsewhereObject holds storage.ElsewhereObject: a match is found by its
// fingerprint, a tournament, a collection and a lesson by their name, each
// with what it holds; a library without them answers ID 0.
func testElsewhereObject(t *testing.T, here, club, empty storage.Storage) {
	ctx := context.Background()
	// The club holds the same match under another format: its canonical
	// fingerprint names it.
	m := domain.Match{Player1Name: "Ann", Player2Name: "Ben", MatchLength: 5, MatchHash: "fp-presence", CanonicalHash: "canon-presence"}
	hereMatch, err := here.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match here: %v", err)
	}
	cross := domain.Match{Player1Name: "Ann", Player2Name: "Ben", MatchLength: 5, MatchHash: "other-format", CanonicalHash: "canon-presence"}
	clubMatch, err := club.Matches().Save(ctx, "", &cross)
	if err != nil {
		t.Fatalf("Save match club: %v", err)
	}
	hereT, err := here.Tournaments().Create(ctx, "", "Open de Lyon", "", "")
	if err != nil {
		t.Fatalf("Create tournament: %v", err)
	}
	clubT, err := club.Tournaments().Create(ctx, "", "Open de Lyon", "", "")
	if err != nil {
		t.Fatalf("Create tournament: %v", err)
	}
	if err := club.Tournaments().AddMatch(ctx, "", clubT, clubMatch); err != nil {
		t.Fatalf("AddMatch: %v", err)
	}
	hereC, err := here.Collections().Create(ctx, "", "À revoir", "")
	if err != nil {
		t.Fatalf("Create collection: %v", err)
	}
	clubC, err := club.Collections().Create(ctx, "", "À revoir", "")
	if err != nil {
		t.Fatalf("Create collection: %v", err)
	}
	pos := checkerPos()
	pid, err := club.Positions().Save(ctx, "", &pos)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	if err := club.Collections().AddPositions(ctx, "", clubC, []int64{pid}); err != nil {
		t.Fatalf("AddPositions: %v", err)
	}
	hereL, err := here.Lessons().Create(ctx, "", "Amorces", "")
	if err != nil {
		t.Fatalf("Create lesson: %v", err)
	}
	clubL, err := club.Lessons().Create(ctx, "", "Amorces", "")
	if err != nil {
		t.Fatalf("Create lesson: %v", err)
	}
	if _, err := club.Lessons().AddStep(ctx, "", clubL, domain.LessonStep{PositionID: pid}); err != nil {
		t.Fatalf("AddStep: %v", err)
	}

	sources := []storage.Source{{Name: "club", Storage: club}, {Name: "empty", Storage: empty}}
	for _, c := range []struct {
		kind  string
		id    int64
		held  int64
		count int
	}{
		{storage.ObjectMatch, hereMatch, clubMatch, 0},
		{storage.ObjectTournament, hereT, clubT, 1},
		{storage.ObjectCollection, hereC, clubC, 1},
		{storage.ObjectLesson, hereL, clubL, 1},
	} {
		q, err := storage.ObjectKey(ctx, here, "", c.kind, c.id)
		if err != nil {
			t.Fatalf("ObjectKey %s: %v", c.kind, err)
		}
		got, err := storage.ElsewhereObject(ctx, sources, q)
		if err != nil {
			t.Fatalf("ElsewhereObject %s: %v", c.kind, err)
		}
		if len(got) != 2 || got[0].Tenant != "club" || got[0].Item.ID != c.held || got[0].Item.Count != c.count {
			t.Errorf("%s in club = %+v, want id %d count %d", c.kind, got, c.held, c.count)
		}
		if len(got) == 2 && got[1].Item.ID != 0 {
			t.Errorf("%s in an empty library = %+v, want absent", c.kind, got[1].Item)
		}
	}
	if _, err := storage.ElsewhereObject(ctx, sources, storage.ObjectQuery{Kind: storage.ObjectTournament}); err == nil {
		t.Error("a tournament named by no name was accepted")
	}
	if _, err := storage.ElsewhereObject(ctx, sources, storage.ObjectQuery{Kind: "deck", Name: "x"}); err == nil {
		t.Error("an unknown kind was accepted")
	}
}

// neighbourOf moves one checker of the side on roll one point down onto an
// empty point: a board a few checker-pips away, of the same kind.
func neighbourOf(t *testing.T, p domain.Position) domain.Position {
	t.Helper()
	n := p
	n.ID = 0
	pts := &n.Board.Points
	for i := 2; i < len(pts)-1; i++ {
		if pts[i].Color == n.PlayerOnRoll && pts[i].Checkers >= 2 && pts[i-1].Checkers == 0 {
			pts[i].Checkers--
			pts[i-1] = domain.Point{Checkers: 1, Color: n.PlayerOnRoll}
			return n
		}
	}
	t.Fatal("neighbourOf: no checker to move")
	return n
}
