package database

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

const copyMatchFile = "testdata/gnubg_roll_then_resign_1p.mat"

// tournamentFixture is a source library holding one match filed in a
// tournament, and an empty destination.
func tournamentFixture(t *testing.T) (src, dst *Database, matchID, tournamentID int64) {
	t.Helper()
	src, dst = newTestDB(t), newTestDB(t)
	matchID, err := src.ImportGnuBGMatch(copyMatchFile)
	if err != nil {
		t.Fatalf("ImportGnuBGMatch: %v", err)
	}
	if tournamentID, err = src.CreateTournament("Open", "2026-10-01", "Lyon"); err != nil {
		t.Fatalf("CreateTournament: %v", err)
	}
	if err := src.AddMatchToTournament(tournamentID, matchID); err != nil {
		t.Fatalf("AddMatchToTournament: %v", err)
	}
	return src, dst, matchID, tournamentID
}

func copyAndSeal(t *testing.T, src, dst *Database, opts ExportOptions) CopyReceipt {
	t.Helper()
	rep, err := src.TransferTo(context.Background(), dst, opts)
	if err != nil {
		t.Fatalf("TransferTo: %v", err)
	}
	r, err := SealCopy(context.Background(), dst, rep)
	if err != nil {
		t.Fatalf("SealCopy: %v", err)
	}
	return r
}

func tournamentOpts(id int64) ExportOptions {
	return ExportOptions{
		IncludeAnalysis: true, IncludeComments: true, IncludePlayedMoves: true,
		IncludeTournaments: true, TournamentIDs: []int64{id},
		IncludeTranscriptions: true, IncludeAliases: true,
	}
}

func counts(t *testing.T, d *Database) (positions, matches, tournaments int) {
	t.Helper()
	ctx := context.Background()
	positions, err := d.CountPositions()
	if err != nil {
		t.Fatalf("CountPositions: %v", err)
	}
	if matches, err = d.store.Matches().Count(ctx, "", storage.MatchListOpts{}); err != nil {
		t.Fatalf("Count matches: %v", err)
	}
	for _, err := range d.store.Tournaments().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatalf("List tournaments: %v", err)
		}
		tournaments++
	}
	return positions, matches, tournaments
}

func TestCopyTournament_TakesBackWhatItCreated(t *testing.T) {
	src, dst, _, tid := tournamentFixture(t)
	r := copyAndSeal(t, src, dst, tournamentOpts(tid))
	if len(r.Matches) != 1 || len(r.Tournaments) != 1 || len(r.Positions) == 0 {
		t.Fatalf("receipt holds %d matches, %d tournaments, %d positions", len(r.Matches), len(r.Tournaments), len(r.Positions))
	}
	if err := TakeBackCopy(dst, r); err != nil {
		t.Fatalf("TakeBackCopy: %v", err)
	}
	if p, m, tn := counts(t, dst); p+m+tn != 0 {
		t.Errorf("after the take-back the destination holds %d positions, %d matches, %d tournaments", p, m, tn)
	}
	if _, m, tn := counts(t, src); m != 1 || tn != 1 {
		t.Error("the take-back reached the source")
	}
}

func TestCopyMatch_KeepsThePositionTheDestinationHeld(t *testing.T) {
	src, dst, mid, _ := tournamentFixture(t)
	ctx := context.Background()
	var first *Position
	for p, err := range src.store.Positions().List(ctx, "", storage.ListOpts{Limit: 1}) {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		first = p
	}
	held := *first
	held.ID = 0
	if _, err := dst.SavePosition(&held); err != nil {
		t.Fatalf("SavePosition: %v", err)
	}
	r := copyAndSeal(t, src, dst, ExportOptions{
		IncludeAnalysis: true, IncludeComments: true, IncludePlayedMoves: true,
		IncludeMatches: true, MatchIDs: []int64{mid}, IncludeTournaments: true,
	})
	if len(r.Matches) != 1 || len(r.Tournaments) != 1 {
		t.Fatalf("a match copied alone: receipt holds %d matches, %d tournaments", len(r.Matches), len(r.Tournaments))
	}
	if err := TakeBackCopy(dst, r); err != nil {
		t.Fatalf("TakeBackCopy: %v", err)
	}
	if p, m, tn := counts(t, dst); p != 1 || m+tn != 0 {
		t.Errorf("after the take-back the destination holds %d positions (want its own 1), %d matches, %d tournaments", p, m, tn)
	}
	if _, found := heldIn(t, dst, *first); !found {
		t.Error("the destination lost the position it held before the copy")
	}
}

func TestTakeBackCopy_RefusesWhatChangedSince(t *testing.T) {
	src, dst, _, tid := tournamentFixture(t)
	r := copyAndSeal(t, src, dst, tournamentOpts(tid))
	if err := dst.AddComment(r.Positions[0].ID, "mine now"); err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if err := TakeBackCopy(dst, r); !errors.Is(err, ErrCopyChanged) {
		t.Fatalf("TakeBackCopy after a comment = %v, want ErrCopyChanged", err)
	}
	if p, m, tn := counts(t, dst); p == 0 || m != 1 || tn != 1 {
		t.Errorf("a refused take-back removed rows: %d positions, %d matches, %d tournaments", p, m, tn)
	}
}

func TestCopyCollection_IntoNamesakeTakesBackOnlyTheNewMembers(t *testing.T) {
	src, dst, id, pos := copyFixture(t)
	other := bearoffPosition()
	srcColl, err := src.CreateCollection("Revoir", "")
	if err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}
	if err := src.AddPositionToCollection(srcColl, id); err != nil {
		t.Fatalf("AddPositionToCollection: %v", err)
	}
	dstColl, err := dst.CreateCollection("Revoir", "")
	if err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}
	ownID, err := dst.SavePosition(&other)
	if err != nil {
		t.Fatalf("SavePosition: %v", err)
	}
	if err := dst.AddPositionToCollection(dstColl, ownID); err != nil {
		t.Fatalf("AddPositionToCollection: %v", err)
	}
	r := copyAndSeal(t, src, dst, ExportOptions{
		IncludeAnalysis: true, IncludeComments: true, IncludePlayedMoves: true,
		IncludeCollections: true, CollectionIDs: []int64{srcColl}, IncludeCollectionPositions: true,
	})
	if len(r.Collections) != 0 || len(r.Members) != 1 {
		t.Fatalf("receipt holds %d collections, %d member sets", len(r.Collections), len(r.Members))
	}
	if err := TakeBackCopy(dst, r); err != nil {
		t.Fatalf("TakeBackCopy: %v", err)
	}
	ids, err := dst.store.Collections().PositionIDs(context.Background(), "", dstColl, storage.ListOpts{})
	if err != nil {
		t.Fatalf("PositionIDs: %v", err)
	}
	if len(ids) != 1 || ids[0] != ownID {
		t.Errorf("the namesake holds %v after the take-back, want its own [%d]", ids, ownID)
	}
	if _, found := heldIn(t, dst, pos); found {
		t.Error("the copied position outlived its take-back")
	}
}

func TestSealCopy_NothingCreatedIsEmpty(t *testing.T) {
	src, dst, _, tid := tournamentFixture(t)
	copyAndSeal(t, src, dst, tournamentOpts(tid))
	r := copyAndSeal(t, src, dst, tournamentOpts(tid))
	if !r.Empty() {
		t.Fatalf("a second copy of the same tournament has a receipt: %+v", r)
	}
	if err := TakeBackCopy(dst, r); err == nil {
		t.Error("taking back a copy that created nothing succeeded")
	}
}
