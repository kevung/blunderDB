// Contract cases for import batches: the counts an import stores and the half
// of the report that is MEASURED over the batch's matches rather than stored.
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testImportBatchLifecycle pins Begin → Finish → Load: a batch is open with no
// finish stamp, the counts an import observed are what Load reads back, and
// finishing a batch that does not exist is ErrNotFound rather than a silent
// no-op.
func testImportBatchLifecycle(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	batches := s.ImportBatches()

	id, err := batches.Begin(ctx, "", "/home/someone/matches", "mixed")
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if id == 0 {
		t.Fatal("Begin returned id 0")
	}

	open, err := batches.Load(ctx, "", id)
	if err != nil {
		t.Fatalf("Load (open): %v", err)
	}
	if open.Source != "/home/someone/matches" || open.Format != "mixed" {
		t.Errorf("Load (open): got source %q format %q", open.Source, open.Format)
	}
	if open.FinishedAt != "" {
		t.Errorf("a batch that has not finished carries a finish stamp: %q", open.FinishedAt)
	}
	if open.StartedAt == "" {
		t.Error("a batch carries no start stamp")
	}

	counts := domain.ImportReport{
		MatchesImported: 3, MatchesSkipped: 1, MatchesEnriched: 2,
		PositionsSaved: 400, FilesFailed: 1,
		Failures: []domain.ImportFailure{{Source: "broken.xg", Reason: "unexpected EOF"}},
	}
	if err := batches.Finish(ctx, "", id, counts); err != nil {
		t.Fatalf("Finish: %v", err)
	}

	done, err := batches.Load(ctx, "", id)
	if err != nil {
		t.Fatalf("Load (finished): %v", err)
	}
	if done.FinishedAt == "" {
		t.Error("a finished batch carries no finish stamp")
	}
	if done.Report.MatchesImported != 3 || done.Report.MatchesSkipped != 1 ||
		done.Report.MatchesEnriched != 2 || done.Report.PositionsSaved != 400 {
		t.Errorf("stored counts came back as %+v", done.Report)
	}
	if len(done.Report.Failures) != 1 || done.Report.Failures[0].Source != "broken.xg" {
		t.Errorf("stored failures came back as %+v", done.Report.Failures)
	}

	if err := batches.Finish(ctx, "", id+9999, counts); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("finishing an unknown batch: got %v, want ErrNotFound", err)
	}
	if _, err := batches.Load(ctx, "", id+9999); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("loading an unknown batch: got %v, want ErrNotFound", err)
	}

	list, err := batches.List(ctx, "", storage.ListOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].ID != id {
		t.Errorf("List: got %d batches, want the one just written", len(list))
	}
}

// testImportBatchReportIsMeasured is the point of the whole design: the half
// of the report that can be measured is measured on every call, and therefore
// tells the truth about the database as it is NOW — not as it was when the
// import ended.
//
// The case walks that: a batch whose positions have no analysis reports them
// as such; analysing one of them lowers the count without anything rewriting
// the batch's stored counts. A report that had been frozen at the end of the
// import would still claim the old figure.
func testImportBatchReportIsMeasured(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	batches := s.ImportBatches()

	id, err := batches.Begin(ctx, "", "corpus.xg", "xg")
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}

	// Two decisions of a match that belongs to the batch, one per player.
	matchID, posIDs := statsFixtureMatchInBatch(t, s, 0, "Alice", "Bob", id)
	// A third position of the same match, with no analysis at all.
	unjudged := statsDecisionPos(t, 4)
	unjudgedID, err := s.Positions().Save(ctx, "", &unjudged)
	if err != nil {
		t.Fatalf("Save unjudged position: %v", err)
	}
	var gameID int64
	for g, err := range s.Matches().Games(ctx, "", matchID) {
		if err != nil {
			t.Fatalf("Games: %v", err)
		}
		gameID = g.ID
		break
	}
	if gameID == 0 {
		t.Fatal("the fixture match has no game")
	}
	if _, err := s.Matches().CreateMove(ctx, "", &domain.Move{
		GameID: gameID, MoveNumber: 99, MoveType: "checker",
		PositionID: unjudgedID, Player: 1, CheckerMove: "13/11",
	}); err != nil {
		t.Fatalf("CreateMove (unjudged): %v", err)
	}

	if err := batches.Finish(ctx, "", id, domain.ImportReport{MatchesImported: 1}); err != nil {
		t.Fatalf("Finish: %v", err)
	}

	rep, err := batches.Report(ctx, "", id, nil)
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	if rep.Report.MatchesImported != 1 {
		t.Errorf("the stored half of the report was lost: %+v", rep.Report)
	}
	if rep.Report.PositionsWithoutAnalysis != 1 {
		t.Errorf("positions without analysis: got %d, want 1", rep.Report.PositionsWithoutAnalysis)
	}
	if rep.Report.Decisions != 2 {
		t.Errorf("counted decisions: got %d, want 2 (one per player)", rep.Report.Decisions)
	}
	if rep.Report.PR <= 0 {
		t.Errorf("PR over the batch: got %v, want a positive rate", rep.Report.PR)
	}
	if len(rep.Report.WorstDecisions) == 0 {
		t.Fatal("the report names no worst decision though both carry an error")
	}
	if rep.Report.WorstDecisions[0].Label == "" {
		t.Error("a worst decision carries no match label")
	}

	// Scoring one player is scoring one seat, not both: the fixture gave each
	// of them exactly one counted decision.
	one, err := batches.Report(ctx, "", id, []string{"Alice"})
	if err != nil {
		t.Fatalf("Report (one player): %v", err)
	}
	if one.Report.Decisions != 1 {
		t.Errorf("decisions of one player: got %d, want 1", one.Report.Decisions)
	}
	if one.Report.Player != "Alice" {
		t.Errorf("the report does not say whose PR it is: %q", one.Report.Player)
	}

	// Judge the third position: the measured half must follow.
	if err := s.Analyses().Save(ctx, "", unjudgedID, &domain.PositionAnalysis{}); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}
	rep, err = batches.Report(ctx, "", id, nil)
	if err != nil {
		t.Fatalf("Report (after analysis): %v", err)
	}
	if rep.Report.PositionsWithoutAnalysis != 0 {
		t.Errorf("positions without analysis after analysing it: got %d, want 0",
			rep.Report.PositionsWithoutAnalysis)
	}

	_ = posIDs
}

// testImportBatchReportIgnoresOtherBatches pins the narrowing: a report is
// about ONE import, and a second import into the same database must not appear
// in it. Without the import_batch_id predicate every report would be a report
// on the whole database, which is exactly the thing the panel replaced.
func testImportBatchReportIgnoresOtherBatches(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	batches := s.ImportBatches()

	first, err := batches.Begin(ctx, "", "first.xg", "xg")
	if err != nil {
		t.Fatalf("Begin (first): %v", err)
	}
	m1, _ := statsFixtureMatchInBatch(t, s, 0, "Alice", "Bob", first)
	if err := batches.Finish(ctx, "", first, domain.ImportReport{MatchesImported: 1}); err != nil {
		t.Fatalf("Finish (first): %v", err)
	}

	second, err := batches.Begin(ctx, "", "second.xg", "xg")
	if err != nil {
		t.Fatalf("Begin (second): %v", err)
	}
	_, _ = statsFixtureMatchInBatch(t, s, 2, "Carol", "Dave", second)
	if err := batches.Finish(ctx, "", second, domain.ImportReport{MatchesImported: 1}); err != nil {
		t.Fatalf("Finish (second): %v", err)
	}

	rep, err := batches.Report(ctx, "", first, nil)
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	if rep.Report.Decisions != 2 {
		t.Errorf("the first batch's report counts %d decisions; the second import leaked in", rep.Report.Decisions)
	}
	for _, d := range rep.Report.WorstDecisions {
		if d.MatchID != m1 {
			t.Errorf("the first batch's report names match %d, which belongs to another import", d.MatchID)
		}
	}
}

// testImportStudyQueue pins the queue that follows the report (#259): the
// order it offers, the fact that a position appears once, and the narrowing to
// one batch.
//
// The order is the whole feature. A queue that offered the close cube
// decisions before the blunders would be a list of positions, not a study
// list — and the reason attached to each entry is what lets the interface say
// WHY it is showing this board rather than another.
func testImportStudyQueue(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	batches := s.ImportBatches()

	id, err := batches.Begin(ctx, "", "corpus.xg", "xg")
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	matchID, _ := statsFixtureMatchInBatch(t, s, 0, "Alice", "Bob", id)
	if err := batches.Finish(ctx, "", id, domain.ImportReport{MatchesImported: 1}); err != nil {
		t.Fatalf("Finish: %v", err)
	}

	// A second import must not leak into the first one's queue.
	other, err := batches.Begin(ctx, "", "other.xg", "xg")
	if err != nil {
		t.Fatalf("Begin (other): %v", err)
	}
	statsFixtureMatchInBatch(t, s, 10, "Carol", "Dave", other)
	if err := batches.Finish(ctx, "", other, domain.ImportReport{MatchesImported: 1}); err != nil {
		t.Fatalf("Finish (other): %v", err)
	}

	queue, err := batches.StudyQueue(ctx, "", id, nil, 0)
	if err != nil {
		if errors.Is(err, storage.ErrInternal) {
			t.Skip("ImportBatches not implemented on this backend")
		}
		t.Fatalf("StudyQueue: %v", err)
	}
	if len(queue) == 0 {
		t.Fatal("the queue is empty though the batch carries decisions that cost equity")
	}

	seen := map[int64]bool{}
	lastRank := -1
	rank := map[domain.StudyQueueReason]int{
		domain.StudyBlunder: 0, domain.StudyFlagged: 1, domain.StudyClose: 2,
	}
	for _, e := range queue {
		if seen[e.PositionID] {
			t.Fatalf("position %d appears twice: a flagged blunder is one entry, not two", e.PositionID)
		}
		seen[e.PositionID] = true
		if e.MatchID != matchID {
			t.Fatalf("the queue of batch %d carries a position of another import (match %d)", id, e.MatchID)
		}
		r, ok := rank[e.Reason]
		if !ok {
			t.Fatalf("unknown reason %q", e.Reason)
		}
		if r < lastRank {
			t.Fatalf("the queue is out of order: %q after a later group", e.Reason)
		}
		lastRank = r
		if e.Label == "" {
			t.Errorf("entry %d carries no match label", e.PositionID)
		}
		// Only a blunder's cost means anything: showing a 0 beside a flagged
		// position would read as a measurement rather than as an absence.
		if e.Reason != domain.StudyBlunder && e.ErrorMP != 0 {
			t.Errorf("a %q entry carries a cost of %d", e.Reason, e.ErrorMP)
		}
	}

	// Blunders come first and are ordered by what they cost.
	var lastCost = 1 << 30
	for _, e := range queue {
		if e.Reason != domain.StudyBlunder {
			break
		}
		if e.ErrorMP > lastCost {
			t.Fatalf("blunders are not ordered by cost: %d after %d", e.ErrorMP, lastCost)
		}
		lastCost = e.ErrorMP
	}

	// The limit is honoured, and a limit of one still gives the worst.
	short, err := batches.StudyQueue(ctx, "", id, nil, 1)
	if err != nil {
		t.Fatalf("StudyQueue (limit 1): %v", err)
	}
	if len(short) != 1 {
		t.Fatalf("limit 1 gave %d entries", len(short))
	}
	if short[0].PositionID != queue[0].PositionID {
		t.Errorf("limit 1 gave position %d, want the queue's first (%d)", short[0].PositionID, queue[0].PositionID)
	}
}
