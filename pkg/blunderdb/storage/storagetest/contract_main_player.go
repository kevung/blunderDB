package storagetest

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// mainPlayerMatch is everything a bulk swap must keep of one match: what
// takeSwapSnapshot reads per seat and per move, the header beyond its seats
// (id, tournament, comment, video, transcriber, file, hashes) and the
// comments of its positions.
type mainPlayerMatch struct {
	Swap     swapSnapshot
	Header   domain.Match
	Comments map[int64][]string
}

func takeMainPlayerMatch(ctx context.Context, s storage.Storage, id int64) (mainPlayerMatch, error) {
	var out mainPlayerMatch
	var err error
	if out.Swap, err = takeSwapSnapshot(ctx, s, id); err != nil {
		return out, err
	}
	m, err := s.Matches().Get(ctx, "", id)
	if err != nil {
		return out, fmt.Errorf("get match: %w", err)
	}
	h := *m
	// The seat-bound fields are in Swap; the badges are derived.
	h.Player1Name, h.Player2Name = "", ""
	h.Player1Elo, h.Player2Elo, h.Player1Experience, h.Player2Experience = nil, nil, nil, nil
	h.PR, h.PR2, h.MWCLoss, h.MWCLoss2 = 0, 0, 0, 0
	h.MWC7, h.MWC7P2 = domain.MWC7{}, domain.MWC7{}
	out.Header = h
	ids := make([]int64, 0, len(out.Swap.PosByMove))
	for _, p := range out.Swap.PosByMove {
		ids = append(ids, p)
	}
	byPos, err := s.Comments().ByPositions(ctx, "", ids)
	if err != nil {
		return out, fmt.Errorf("comments: %w", err)
	}
	out.Comments = map[int64][]string{}
	for p, list := range byPos {
		for _, c := range list {
			out.Comments[p] = append(out.Comments[p], c.Text)
		}
	}
	return out, nil
}

// seatsExchanged reports how got differs from was other than by its seats
// trading places, "" when it does not.
func seatsExchanged(got, was mainPlayerMatch) string {
	if !reflect.DeepEqual(got.Header, was.Header) {
		return fmt.Sprintf("header changed:\n got %+v\nwant %+v", got.Header, was.Header)
	}
	if !reflect.DeepEqual(got.Comments, was.Comments) {
		return "comments changed"
	}
	g, w := got.Swap, was.Swap
	if !reflect.DeepEqual(g.PosByMove, w.PosByMove) || !reflect.DeepEqual(g.Positions, w.Positions) ||
		!reflect.DeepEqual(g.Analyses, w.Analyses) || g.PositionRows != w.PositionRows {
		return "positions or analyses changed"
	}
	for id, mv := range w.Moves {
		gm := g.Moves[id]
		if gm.Player != -mv.Player {
			return fmt.Sprintf("move %d: player %d, want %d", id, gm.Player, -mv.Player)
		}
		gm.Player = mv.Player
		if !reflect.DeepEqual(gm, mv) {
			return fmt.Sprintf("move %d changed beyond its player", id)
		}
	}
	for seat := range 2 {
		gs, ws := g.Seats[seat], w.Seats[1-seat]
		gs.Row.Seat = ws.Row.Seat
		if !reflect.DeepEqual(gs, ws) {
			return fmt.Sprintf("seat %d is not the former seat %d:\n got %+v\nwant %+v", seat+1, 2-seat, gs, ws)
		}
	}
	return ""
}

// errInjectedSwap is the failure swapFailStorage injects.
var errInjectedSwap = errors.New("injected swap failure")

// swapFailStorage fails the failAt-th SwapPlayers run inside one of its
// transactions: a bulk swap stopped halfway.
type swapFailStorage struct {
	storage.Storage
	failAt int
}

func (f *swapFailStorage) BeginTx(ctx context.Context) (storage.Tx, error) {
	tx, err := f.Storage.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	return &swapFailTx{Tx: tx, failAt: f.failAt}, nil
}

type swapFailTx struct {
	storage.Tx
	failAt, n int
}

func (t *swapFailTx) Matches() storage.MatchStore {
	return &swapFailMatches{MatchStore: t.Tx.Matches(), tx: t}
}

type swapFailMatches struct {
	storage.MatchStore
	tx *swapFailTx
}

func (m *swapFailMatches) SwapPlayers(ctx context.Context, scope string, id int64) error {
	m.tx.n++
	if m.tx.n == m.tx.failAt {
		return errInjectedSwap
	}
	return m.MatchStore.SwapPlayers(ctx, scope, id)
}

// importNamed imports the fixture under the given names, as a match of its
// own: the hashes that would make it a duplicate of the previous copy are
// cleared, and its dice hash is unique.
func importNamed(t *testing.T, s storage.Storage, p1, p2 string, keepSeats bool) (int64, *ingest.MatchGraph) {
	t.Helper()
	ctx := context.Background()
	g, err := ingest.MapMatchFile(repoTestdata("test.xg"))
	if err != nil {
		t.Fatalf("map fixture: %v", err)
	}
	g.Match.Player1Name, g.Match.Player2Name = p1, p2
	g.Match.MatchHash, g.Match.CanonicalHash = "", ""
	g.Match.DiceHash = "main-player-" + p1 + "-" + p2
	g.Match.Event = "Open " + p1
	g.KeepSeats = keepSeats
	tx, err := s.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	res, err := ingest.WriteMatch(ctx, tx, "", g, nil)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("import %s-%s: %v", p1, p2, err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	return res.MatchID, g
}

// testMainPlayerBulkSwapLosesNothing: designating the Main player swaps every
// match where they sit at seat 2 — recognised by their aliases — and nothing
// else; each swapped match keeps its id, analyses, errors, comments, video,
// transcriber and tournament, its seats only trading places; swapping again
// gives the library back; a failure halfway leaves no match swapped and no
// Main player; forgetting the Main player swaps nothing back.
func testMainPlayerBulkSwapLosesNothing(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ms := s.Matches()
	seat2, _ := importNamed(t, s, "Gildas", "Kevin", false)
	alias2, _ := importNamed(t, s, "Erin", "K. Unger", false)
	seat1, _ := importNamed(t, s, "Kevin", "Bob", false)
	draft2, _ := importNamed(t, s, "Yan", "Kevin", false)
	both, _ := importNamed(t, s, "K. Unger", "Kevin", false)
	strangers, _ := importNamed(t, s, "Zed", "Yan", false)
	all := []int64{seat2, alias2, seat1, draft2, both, strangers}
	// Merged after the imports: the matches keep the alias their file wrote.
	if err := s.Aliases().Set(ctx, "", storage.AliasPlayer, "K. Unger", "Kevin"); err != nil {
		t.Fatalf("alias: %v", err)
	}

	// What the user brought to the match swapped: a comment on it and on one
	// of its positions, its video, its transcriber.
	if err := ms.UpdateComment(ctx, "", seat2, "the final at Marseille"); err != nil {
		t.Fatalf("UpdateComment: %v", err)
	}
	if err := ms.SetVideoSource(ctx, "", seat2, "https://example.org/final.mp4"); err != nil {
		t.Fatalf("SetVideoSource: %v", err)
	}
	var firstPos int64
	for mv, err := range ms.MovesByMatch(ctx, "", seat2) {
		if err != nil {
			t.Fatalf("MovesByMatch: %v", err)
		}
		if mv.PositionID != 0 {
			firstPos = mv.PositionID
			break
		}
	}
	if _, err := s.Comments().Add(ctx, "", firstPos, "pick and pass"); err != nil {
		t.Fatalf("Add comment: %v", err)
	}
	header, err := ms.Get(ctx, "", seat2)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	header.Transcriber = "Kevin"
	if err := ms.ReplaceHeader(ctx, "", seat2, header); err != nil {
		t.Fatalf("ReplaceHeader: %v", err)
	}
	if _, err := s.Transcriptions().Save(ctx, "", &storage.Transcription{FormatVersion: "1",
		MatchID: draft2, Label: "Yan — Kevin", Document: "{}"}); err != nil {
		t.Fatalf("Save draft: %v", err)
	}

	snapAll := func(stage string) map[int64]mainPlayerMatch {
		t.Helper()
		out := map[int64]mainPlayerMatch{}
		for _, id := range all {
			sn, err := takeMainPlayerMatch(ctx, s, id)
			if err != nil {
				t.Fatalf("%s: match %d: %v", stage, id, err)
			}
			out[id] = sn
		}
		return out
	}
	mainIs := func(stage, want string) {
		t.Helper()
		got, err := storage.MainPlayer(ctx, s, "")
		if err != nil || got != want {
			t.Errorf("%s: MainPlayer = %q, %v; want %q", stage, got, err, want)
		}
	}
	before := snapAll("before")
	if before[seat2].Header.TournamentID == nil || before[seat2].Header.VideoSource == nil ||
		len(before[seat2].Comments) == 0 || len(before[seat2].Swap.Analyses) == 0 {
		t.Fatalf("the swapped match carries nothing to lose: %+v", before[seat2].Header)
	}

	plan, err := storage.PlanMainPlayer(ctx, s, "", "K. Unger")
	if err != nil {
		t.Fatalf("PlanMainPlayer: %v", err)
	}
	wantSwap := []int64{seat2, alias2}
	slices.Sort(plan.Swap)
	slices.Sort(wantSwap)
	if plan.Name != "Kevin" || !slices.Equal(plan.Swap, wantSwap) || !slices.Equal(plan.Refused, []int64{draft2}) {
		t.Fatalf("plan = %+v, want Kevin, swap %v, refused [%d]", plan, wantSwap, draft2)
	}
	if got := snapAll("after the plan"); !reflect.DeepEqual(got, before) {
		t.Errorf("drawing the plan wrote something")
	}

	// Stopped at its second swap: the first is rolled back with it.
	if _, err := storage.DesignateMainPlayer(ctx, &swapFailStorage{Storage: s, failAt: 2}, "", "Kevin"); !errors.Is(err, errInjectedSwap) {
		t.Fatalf("DesignateMainPlayer with a failing swap = %v, want the injected failure", err)
	}
	if got := snapAll("after the failed designation"); !reflect.DeepEqual(got, before) {
		t.Errorf("a designation that failed halfway left matches swapped")
	}
	mainIs("after the failed designation", "")

	applied, err := storage.DesignateMainPlayer(ctx, s, "", "Kevin")
	if err != nil {
		t.Fatalf("DesignateMainPlayer: %v", err)
	}
	slices.Sort(applied.Swap)
	if !slices.Equal(applied.Swap, wantSwap) {
		t.Errorf("applied swap %v, want %v", applied.Swap, wantSwap)
	}
	mainIs("after the designation", "Kevin")
	after := snapAll("after the designation")
	for _, id := range all {
		if slices.Contains(wantSwap, id) {
			if diff := seatsExchanged(after[id], before[id]); diff != "" {
				t.Errorf("match %d: %s", id, diff)
			}
			if after[id].Swap.Seats[0].Name == "Gildas" || after[id].Swap.Seats[0].Name == "Erin" {
				t.Errorf("match %d: seat 1 is still %q", id, after[id].Swap.Seats[0].Name)
			}
		} else if !reflect.DeepEqual(after[id], before[id]) {
			t.Errorf("match %d is not the Main player's at seat 2 alone, and changed", id)
		}
	}

	again, err := storage.PlanMainPlayer(ctx, s, "", "Kevin")
	if err != nil || len(again.Swap) != 0 {
		t.Errorf("plan after the designation = %+v, %v; want nothing left to swap", again, err)
	}

	// Forgetting the Main player swaps nothing back.
	if err := s.LibrarySettings().SetMainPlayer(ctx, "", ""); err != nil {
		t.Fatalf("SetMainPlayer(\"\"): %v", err)
	}
	mainIs("after forgetting", "")
	if got := snapAll("after forgetting"); !reflect.DeepEqual(got, after) {
		t.Errorf("forgetting the Main player changed matches")
	}

	for _, id := range wantSwap {
		if err := ms.SwapPlayers(ctx, "", id); err != nil {
			t.Fatalf("swap back %d: %v", id, err)
		}
	}
	if got := snapAll("after swapping back"); !reflect.DeepEqual(got, before) {
		t.Errorf("swapping the designation's matches again does not give the library back")
	}

	if _, err := storage.PlanMainPlayer(ctx, s, "", "  "); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("plan of a blank name = %v, want ErrInvalid", err)
	}
}

// testMainPlayerFirstAtImport: with a Main player designated, a match file
// that seats them second — under an alias too — is stored with its seats
// swapped exactly as SwapPlayers would; a match typed or played here
// (KeepSeats) keeps the seats it was given.
func testMainPlayerFirstAtImport(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	if err := s.Aliases().Set(ctx, "", storage.AliasPlayer, "K. Unger", "Kevin"); err != nil {
		t.Fatalf("alias: %v", err)
	}
	// Stored under an alias: the import resolves it as the designation does.
	if err := s.LibrarySettings().SetMainPlayer(ctx, "", "K. Unger"); err != nil {
		t.Fatalf("SetMainPlayer: %v", err)
	}

	id, g := importNamed(t, s, "Gildas", "K. Unger", false)
	m, err := s.Matches().Get(ctx, "", id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if m.Player1Name != "Kevin" || m.Player2Name != "Gildas" {
		t.Fatalf("seats %q / %q, want Kevin / Gildas", m.Player1Name, m.Player2Name)
	}
	// Collected first: an open cursor would hold the one connection an
	// in-memory database has.
	var games []*domain.Game
	for gm, err := range s.Matches().Games(ctx, "", id) {
		if err != nil {
			t.Fatalf("Games: %v", err)
		}
		games = append(games, gm)
	}
	if len(games) != len(g.Games) {
		t.Fatalf("%d games stored, the file has %d", len(games), len(g.Games))
	}
	for gi, gm := range games {
		want := g.Games[gi].Game
		if gm.InitialScore != [2]int32{want.InitialScore[1], want.InitialScore[0]} || gm.Winner != -want.Winner {
			t.Errorf("game %d: score %v winner %d, want the file's swapped (%v, %d)", gi+1, gm.InitialScore, gm.Winner, want.InitialScore, want.Winner)
		}
		mi := 0
		for mv, err := range s.Matches().Moves(ctx, "", gm.ID) {
			if err != nil {
				t.Fatalf("Moves: %v", err)
			}
			if fm := g.Games[gi].Moves[mi].Move; mv.Player != -fm.Player || mv.PositionID != fm.PositionID {
				t.Errorf("game %d move %d: player %d on %d, want %d on %d", gi+1, mi+1, mv.Player, mv.PositionID, -fm.Player, fm.PositionID)
			}
			mi++
		}
	}

	kept, _ := importNamed(t, s, "Yan", "Kevin", true)
	if k, err := s.Matches().Get(ctx, "", kept); err != nil || k.Player1Name != "Yan" {
		t.Errorf("a match that keeps its seats was swapped: %+v, %v", k, err)
	}
	seat1, _ := importNamed(t, s, "Kevin", "Bob", false)
	if k, err := s.Matches().Get(ctx, "", seat1); err != nil || k.Player1Name != "Kevin" {
		t.Errorf("a match with the Main player first was swapped: %+v, %v", k, err)
	}
}
