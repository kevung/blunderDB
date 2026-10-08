// Contract case for a match deleted through the trash: what the delete purges
// is what a plain DeleteCascade purges, and the restore gives the match back
// as it was, under its own id and import date. The table that runs it lives in contract.go.
package storagetest

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/trash"
)

// matchContent is a match as a reader sees it, stripped of the ids a restore
// cannot keep: games and moves by their order, positions by their content.
type matchContent struct {
	Header   domain.Match
	Games    []domain.Game
	Moves    []domain.Move
	Analyses []domain.MoveAnalysis
	Boards   []domain.Position // the position of each move, in move order
}

func readMatchContent(t *testing.T, s storage.Storage, matchID int64) matchContent {
	t.Helper()
	ctx := context.Background()
	m, err := s.Matches().Get(ctx, "", matchID)
	if err != nil {
		t.Fatalf("Get match %d: %v", matchID, err)
	}
	c := matchContent{Header: *m}
	// The import date as an instant: a backend may hand back another zone.
	c.Header.ImportDate = c.Header.ImportDate.UTC()
	for g, err := range s.Matches().Games(ctx, "", matchID) {
		if err != nil {
			t.Fatalf("Games: %v", err)
		}
		g.ID, g.MatchID = 0, 0
		c.Games = append(c.Games, *g)
	}
	// Read the moves before loading their positions: a store pinned to one
	// connection cannot run a query while the listing holds it.
	var moves []*domain.Move
	for mv, err := range s.Matches().MovesByMatch(ctx, "", matchID) {
		if err != nil {
			t.Fatalf("MovesByMatch: %v", err)
		}
		moves = append(moves, mv)
	}
	moveIndex := map[int64]int64{}
	for _, mv := range moves {
		moveIndex[mv.ID] = int64(len(c.Moves))
		if mv.PositionID != 0 {
			p, err := s.Positions().Load(ctx, "", mv.PositionID)
			if err != nil {
				t.Fatalf("Load position %d: %v", mv.PositionID, err)
			}
			p.ID = 0
			c.Boards = append(c.Boards, *p)
		} else {
			c.Boards = append(c.Boards, domain.Position{})
		}
		mv.ID, mv.GameID, mv.PositionID = 0, 0, 0
		c.Moves = append(c.Moves, *mv)
	}
	for ma, err := range s.Matches().MoveAnalysesByMatch(ctx, "", matchID) {
		if err != nil {
			t.Fatalf("MoveAnalysesByMatch: %v", err)
		}
		ma.ID, ma.MoveID = 0, moveIndex[ma.MoveID]
		c.Analyses = append(c.Analyses, *ma)
	}
	return c
}

func testTrashMatchRestores(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ms := s.Matches()

	tourID, err := s.Tournaments().Create(ctx, "", "Cup", "2025-01-01", "Paris")
	if err != nil {
		t.Fatalf("Create tournament: %v", err)
	}
	newMatch := func(p1, p2, hash string) (int64, int64) {
		elo := 1650.0
		video := "https://example.org/v"
		m := domain.Match{
			Player1Name: p1, Player2Name: p2, Event: "Open", Round: "R1", MatchLength: 7,
			MatchDate: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), FilePath: "m.xg", GameCount: 2,
			TournamentID: &tourID, Comment: "a good one", MatchHash: hash, CanonicalHash: "c" + hash,
			Player1Elo: &elo, Transcriber: "T", EngineVersion: "XG 2.19", VideoSource: &video,
		}
		id, err := ms.Save(ctx, "", &m)
		if err != nil {
			t.Fatalf("Save match: %v", err)
		}
		if err := ms.SetVideoSource(ctx, "", id, video); err != nil {
			t.Fatalf("SetVideoSource: %v", err)
		}
		g := domain.Game{MatchID: id, GameNumber: 1, InitialScore: [2]int32{0, 0}, Winner: 1, PointsWon: 2, MoveCount: 3}
		gameID, err := ms.CreateGame(ctx, "", &g)
		if err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
		return id, gameID
	}
	addMove := func(gameID int64, n int32, posID int64) int64 {
		luck := int32(-120)
		roll, tick := int64(1000*int64(n)), int64(1000*int64(n)+400)
		mv := domain.Move{GameID: gameID, MoveNumber: n, MoveType: "checker", PositionID: posID, Player: 1,
			Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5", LuckMP: &luck, RollTickMS: &roll, TickMS: &tick}
		id, err := ms.CreateMove(ctx, "", &mv)
		if err != nil {
			t.Fatalf("CreateMove: %v", err)
		}
		return id
	}
	savePos := func(n int) int64 {
		p := provenancePos(n)
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position: %v", err)
		}
		return id
	}

	before, _ := newMatch("Ann", "Ben", "h-before")
	matchID, gameID := newMatch("Alice", "Bob", "h-target")
	after, _ := newMatch("Cid", "Dan", "h-after")
	if err := s.Tournaments().ReorderMatches(ctx, "", tourID, []int64{before, matchID, after}); err != nil {
		t.Fatalf("ReorderMatches: %v", err)
	}

	orphan := savePos(31) // held by the match alone, with the file's note
	if err := s.Analyses().Save(ctx, "", orphan, verdictBy("XG", time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC))); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}
	if _, err := s.Comments().AddFrom(ctx, "", orphan, "from the file", domain.CommentOriginXG); err != nil {
		t.Fatalf("AddFrom: %v", err)
	}
	held := savePos(32) // held by a collection too
	colID, err := s.Collections().Create(ctx, "", "Keep", "")
	if err != nil {
		t.Fatalf("Create collection: %v", err)
	}
	if err := s.Collections().AddPositions(ctx, "", colID, []int64{held}); err != nil {
		t.Fatalf("AddPositions: %v", err)
	}

	if _, err := s.Comments().Add(ctx, "", held, "mine"); err != nil {
		t.Fatalf("Add comment: %v", err)
	}
	heldNote, err := s.Comments().AddFrom(ctx, "", held, "file note", domain.CommentOriginXG)
	if err != nil {
		t.Fatalf("AddFrom held: %v", err)
	}
	draftID, err := s.Transcriptions().Save(ctx, "", &storage.Transcription{MatchID: matchID, Label: "draft", Document: "{}"})
	if err != nil {
		t.Fatalf("Save transcription: %v", err)
	}
	m1 := addMove(gameID, 1, orphan)
	addMove(gameID, 2, held)
	addMove(gameID, 3, 0)
	ma := domain.MoveAnalysis{MoveID: m1, AnalysisType: "checker", Depth: "3-ply", Equity: 0.1, EquityError: -0.05, WinRate: 0.55}
	if _, err := ms.CreateMoveAnalysis(ctx, "", &ma); err != nil {
		t.Fatalf("CreateMoveAnalysis: %v", err)
	}
	if err := ms.SetLastVisitedPosition(ctx, "", matchID, 2); err != nil {
		t.Fatalf("SetLastVisitedPosition: %v", err)
	}
	if err := s.Duels().SetOrigin(ctx, "", &storage.MatchOrigin{MatchID: matchID, DiceSeed: "seed"}); err != nil {
		t.Fatalf("SetOrigin: %v", err)
	}
	// What names the match or its purged position from outside its cascade.
	if err := s.ImportBatches().SetStudied(ctx, "", orphan, true); err != nil {
		t.Fatalf("SetStudied: %v", err)
	}
	markedAt, err := s.ImportBatches().StudyMark(ctx, "", orphan)
	if err != nil || markedAt == 0 {
		t.Fatalf("StudyMark = %d, %v", markedAt, err)
	}
	if _, err := s.Training().Save(ctx, "", storage.TrainingSession{Exercise: "decision",
		Items: []storage.TrainingItem{{NumberType: "decision", Wrong: true, PositionID: &orphan}}}); err != nil {
		t.Fatalf("Save training session: %v", err)
	}
	items, err := s.Training().ItemsOfPosition(ctx, "", orphan)
	if err != nil || len(items) != 1 {
		t.Fatalf("ItemsOfPosition = %v, %v", items, err)
	}
	batchID, err := s.ImportBatches().Begin(ctx, "", "m.xg", "xg")
	if err != nil {
		t.Fatalf("Begin batch: %v", err)
	}
	if err := s.ImportBatches().RecordFiles(ctx, "", batchID,
		[]domain.ImportFileEntry{{Path: "m.xg", Outcome: domain.JournalNew, MatchID: matchID}}); err != nil {
		t.Fatalf("RecordFiles: %v", err)
	}
	files, err := s.ImportBatches().FilesOfMatch(ctx, "", matchID)
	if err != nil || len(files) != 1 {
		t.Fatalf("FilesOfMatch = %v, %v", files, err)
	}
	wantStats, err := s.Stats().MatchStats(ctx, "", []int64{matchID})
	if err != nil || len(wantStats) == 0 {
		t.Fatalf("MatchStats before delete = %v, %v", wantStats, err)
	}
	want := readMatchContent(t, s, matchID)

	entryID, err := trash.Match(ctx, s, "", matchID)
	if err != nil {
		t.Fatalf("trash.Match: %v", err)
	}
	if _, err := ms.Get(ctx, "", matchID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("match after trash delete: %v, want ErrNotFound", err)
	}
	// The retention rule is DeleteCascade's: the orphan goes, the held stays.
	if _, err := s.Positions().Load(ctx, "", orphan); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("orphan position after trash delete: %v, want ErrNotFound", err)
	}
	if _, err := s.Positions().Load(ctx, "", held); err != nil {
		t.Errorf("held position after trash delete: %v", err)
	}
	entries, err := s.Trash().List(ctx, "", domain.TrashMatch, storage.ListOpts{})
	if err != nil || len(entries) != 1 || entries[0].ID != entryID {
		t.Fatalf("trash list of matches: %v %v, want entry %d", entries, err, entryID)
	}

	// Deleted while the match sat in the trash: the held position stayed in
	// the library, so the note must not come back with the match.
	if err := s.Comments().Delete(ctx, "", heldNote); err != nil {
		t.Fatalf("Delete note: %v", err)
	}
	res, err := trash.Restore(ctx, s, "", entryID)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	restored := res.ID
	if restored != matchID {
		t.Errorf("restored match has id %d, want its own %d", restored, matchID)
	}
	got := readMatchContent(t, s, restored)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("restored match differs:\n got %+v\nwant %+v", got, want)
	}

	var order []int64
	for m, err := range s.Tournaments().Matches(ctx, "", tourID) {
		if err != nil {
			t.Fatalf("tournament matches: %v", err)
		}
		order = append(order, m.ID)
	}
	if !reflect.DeepEqual(order, []int64{before, restored, after}) {
		t.Errorf("tournament order after restore = %v, want [%d %d %d]", order, before, restored, after)
	}
	if o, err := s.Duels().Origin(ctx, "", restored); err != nil || o.DiceSeed != "seed" {
		t.Errorf("origin after restore = %+v, %v; want seed", o, err)
	}
	var restoredOrphan int64
	for mv, err := range ms.MovesByMatch(ctx, "", restored) {
		if err != nil {
			t.Fatalf("MovesByMatch: %v", err)
		}
		if mv.MoveNumber == 1 {
			restoredOrphan = mv.PositionID
		}
	}
	if restoredOrphan != orphan {
		t.Errorf("purged position restored as %d, want its own id %d", restoredOrphan, orphan)
	}
	if at, err := s.ImportBatches().StudyMark(ctx, "", orphan); err != nil || at != markedAt {
		t.Errorf("study mark after restore = %d, %v; want %d", at, err, markedAt)
	}
	if got, err := s.Training().ItemsOfPosition(ctx, "", orphan); err != nil || !reflect.DeepEqual(got, items) {
		t.Errorf("training items after restore = %v, %v; want %v", got, err, items)
	}
	if got, err := s.ImportBatches().FilesOfMatch(ctx, "", restored); err != nil || !reflect.DeepEqual(got, files) {
		t.Errorf("journal lines after restore = %v, %v; want %v", got, err, files)
	}
	if got, err := s.Stats().MatchStats(ctx, "", []int64{restored}); err != nil || !reflect.DeepEqual(got, wantStats) {
		t.Errorf("match stats after restore = %+v, %v; want %+v", got, err, wantStats)
	}
	if _, err := s.Analyses().Load(ctx, "", restoredOrphan); err != nil {
		t.Errorf("analysis of the purged position after restore: %v", err)
	}
	var notes []string
	for c, err := range s.Comments().ByPosition(ctx, "", restoredOrphan) {
		if err != nil {
			t.Fatalf("ByPosition: %v", err)
		}
		notes = append(notes, string(c.Origin)+":"+c.Text)
	}
	if !reflect.DeepEqual(notes, []string{"xg:from the file"}) {
		t.Errorf("comments of the purged position after restore = %v", notes)
	}
	if _, err := s.Trash().Load(ctx, "", entryID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("trash entry after restore: %v, want ErrNotFound", err)
	}
	// What named the match or its held positions by id finds them again.
	if d, err := s.Transcriptions().Get(ctx, "", draftID); err != nil || d.MatchID != matchID {
		t.Errorf("draft after restore names match %v (%v), want %d", d, err, matchID)
	}
	var mine []string
	for c, err := range s.Comments().ByPosition(ctx, "", held) {
		if err != nil {
			t.Fatalf("ByPosition: %v", err)
		}
		mine = append(mine, c.Text)
	}
	if !reflect.DeepEqual(mine, []string{"mine"}) {
		t.Errorf("comments of the held position after restore = %v, want [mine]", mine)
	}
	var members []int64
	for p, err := range s.Collections().Positions(ctx, "", colID, storage.ListOpts{}) {
		if err != nil {
			t.Fatalf("collection positions: %v", err)
		}
		members = append(members, p.ID)
	}
	if !reflect.DeepEqual(members, []int64{held}) {
		t.Errorf("collection after restore = %v, want [%d]", members, held)
	}
	for _, mv := range got.Moves {
		if mv.RollTickMS == nil || mv.TickMS == nil {
			t.Errorf("move %d lost its video marks", mv.MoveNumber)
		}
	}

	// A match imported again meanwhile is not restored over: that would be a
	// duplicate. The entry stays.
	entryID, err = trash.Match(ctx, s, "", restored)
	if err != nil {
		t.Fatalf("trash.Match again: %v", err)
	}
	newMatch("Alice", "Bob", "h-target")
	if _, err := trash.Restore(ctx, s, "", entryID); !errors.Is(err, trash.ErrMatchPresent) {
		t.Errorf("restore over a re-imported match: %v, want ErrMatchPresent", err)
	}
	if _, err := s.Trash().Load(ctx, "", entryID); err != nil {
		t.Errorf("trash entry after a refused restore: %v", err)
	}

	// The id taken by another match meanwhile: refused, the entry stays.
	if err := s.Trash().Discard(ctx, "", entryID); err != nil {
		t.Fatalf("Discard: %v", err)
	}
	third, _ := newMatch("Eve", "Fay", "h-third")
	entryID, err = trash.Match(ctx, s, "", third)
	if err != nil {
		t.Fatalf("trash.Match third: %v", err)
	}
	squatter := domain.Match{ID: third, Player1Name: "Gus", Player2Name: "Hal"}
	if err := ms.Reinstate(ctx, "", &squatter); err != nil {
		t.Fatalf("Reinstate squatter: %v", err)
	}
	if _, err := trash.Restore(ctx, s, "", entryID); !errors.Is(err, trash.ErrMatchPresent) {
		t.Errorf("restore over a taken id: %v, want ErrMatchPresent", err)
	}
	if _, err := s.Trash().Load(ctx, "", entryID); err != nil {
		t.Errorf("trash entry after a refused restore: %v", err)
	}
	if err := ms.Reinstate(ctx, "", &domain.Match{ID: third}); !errors.Is(err, storage.ErrConflict) {
		t.Errorf("Reinstate on a held id: %v, want ErrConflict", err)
	}
	if err := ms.Reinstate(ctx, "", &domain.Match{}); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("Reinstate without an id: %v, want ErrInvalid", err)
	}
}

// testTrashMatchDirectionSlot: a match comes back in the Slot it filled when
// the Slot is free, and without it, warned, when another match fills it now.
func testTrashMatchDirectionSlot(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ds := s.Directions()
	tid := newDirectedTournament(t, s, "Principal")
	mid := newTournamentMatch(t, s, tid, "Anna", "Bruno")
	if err := ds.AttachSlot(ctx, "", tid, "M1", mid); err != nil {
		t.Fatalf("AttachSlot: %v", err)
	}

	entry, err := trash.Match(ctx, s, "", mid)
	if err != nil {
		t.Fatalf("trash.Match: %v", err)
	}
	res, err := trash.Restore(ctx, s, "", entry)
	if err != nil {
		t.Fatalf("Restore with the Slot free: %v", err)
	}
	if res.ID != mid || len(res.Warnings) != 0 {
		t.Errorf("Restore with the Slot free = %+v, want id %d and no warning", res, mid)
	}
	if _, slot, err := ds.SlotOf(ctx, "", mid); err != nil || slot != "M1" {
		t.Errorf("Slot after restore = %q, %v; want M1", slot, err)
	}

	if entry, err = trash.Match(ctx, s, "", mid); err != nil {
		t.Fatalf("trash.Match again: %v", err)
	}
	other := newTournamentMatch(t, s, tid, "Carl", "Dora")
	if err := ds.AttachSlot(ctx, "", tid, "M1", other); err != nil {
		t.Fatalf("AttachSlot of another match: %v", err)
	}
	res, err = trash.Restore(ctx, s, "", entry)
	if err != nil {
		t.Fatalf("Restore with the Slot taken: %v", err)
	}
	if res.ID != mid || len(res.Warnings) != 1 || res.Warnings[0].Code != domain.TrashWarnSlotTaken {
		t.Errorf("Restore with the Slot taken = %+v, want id %d and a %s warning", res, mid, domain.TrashWarnSlotTaken)
	}
	if _, slot, err := ds.SlotOf(ctx, "", mid); err != nil || slot != "" {
		t.Errorf("Slot of the restored match = %q, %v; want none", slot, err)
	}
	if _, slot, err := ds.SlotOf(ctx, "", other); err != nil || slot != "M1" {
		t.Errorf("Slot of the match that took it = %q, %v; want M1", slot, err)
	}
	if m, err := s.Matches().Get(ctx, "", mid); err != nil || m.TournamentID == nil || *m.TournamentID != tid {
		t.Errorf("restored match's tournament = %+v, %v; want %d", m, err, tid)
	}
}

// testPositionReinstate: a deleted position comes back under its own id; a
// stored hash keeps its row; a taken id is refused without spoiling the
// transaction it ran in.
func testPositionReinstate(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ps := s.Positions()
	p := provenancePos(41)
	id, err := ps.Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := ps.Delete(ctx, "", id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	back := provenancePos(41)
	back.ID = id
	if got, created, err := ps.Reinstate(ctx, "", &back); err != nil || got != id || !created || back.ID != id {
		t.Fatalf("Reinstate of a deleted position = %d, %v, %v; want %d, created", got, created, err, id)
	}
	if _, err := ps.Load(ctx, "", id); err != nil {
		t.Errorf("Load of the reinstated position: %v", err)
	}

	// The hash stored under another id: that row is the position.
	q := provenancePos(42)
	qid, err := ps.Save(ctx, "", &q)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	stale := provenancePos(42)
	stale.ID = id
	if got, created, err := ps.Reinstate(ctx, "", &stale); err != nil || got != qid || created {
		t.Errorf("Reinstate of a stored hash = %d, %v, %v; want %d, not created", got, created, err, qid)
	}

	// The id held by another position: refused, and the transaction goes on.
	tx, err := s.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()
	squatter := provenancePos(43)
	squatter.ID = id
	if _, _, err := tx.Positions().Reinstate(ctx, "", &squatter); !errors.Is(err, storage.ErrConflict) {
		t.Errorf("Reinstate on a held id: %v, want ErrConflict", err)
	}
	if _, err := tx.Positions().Save(ctx, "", &squatter); err != nil {
		t.Errorf("Save after a refused Reinstate, same transaction: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Errorf("Commit after a refused Reinstate: %v", err)
	}
	if _, _, err := ps.Reinstate(ctx, "", &domain.Position{}); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("Reinstate without an id: %v, want ErrInvalid", err)
	}
}
