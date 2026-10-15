package ingest

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

const (
	srcCreated = "2018-02-03 04:05:06"
	srcUpdated = "2019-07-08 09:10:11"
)

func openMemTarget(t *testing.T) storage.Storage {
	t.Helper()
	target, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { target.Close() })
	return target
}

// TestDBImportKeepsDates: the tournament, draft and alias an import creates
// bear the dates their source rows bore, not the day of the import.
func TestDBImportKeepsDates(t *testing.T) {
	ctx := context.Background()
	srcPath := filepath.Join(t.TempDir(), "club.db")
	src, err := sqlite.Open(ctx, srcPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	tour, err := src.Tournaments().Create(ctx, "", "Open daté", "2018-02-03", "Nice")
	if err != nil {
		t.Fatal(err)
	}
	mid, err := src.Matches().Save(ctx, "", &domain.Match{Player1Name: "A", Player2Name: "B", MatchLength: 3, MatchHash: "h-dated"})
	if err != nil {
		t.Fatal(err)
	}
	if err := src.Tournaments().AddMatch(ctx, "", tour, mid); err != nil {
		t.Fatal(err)
	}
	draft, err := src.Transcriptions().Save(ctx, "", &storage.Transcription{FormatVersion: "1",
		Label: "Daté", Document: `{"header":{}}`})
	if err != nil {
		t.Fatal(err)
	}
	if err := src.Aliases().Set(ctx, "", storage.AliasEvent, "OD", "Open daté"); err != nil {
		t.Fatal(err)
	}
	for _, err := range []error{
		src.Tournaments().KeepDates(ctx, "", tour, srcCreated, srcUpdated),
		src.Transcriptions().KeepDates(ctx, "", draft, srcCreated, srcUpdated),
		src.Aliases().KeepCreatedAt(ctx, "", storage.AliasEvent, "OD", srcCreated),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}
	src.Close()

	target := openMemTarget(t)
	if _, err := (DBImporter{S: target}).Import(ctx, "", Source{Path: srcPath}, nil); err != nil {
		t.Fatal(err)
	}

	for tr, err := range target.Tournaments().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		if tr.CreatedAt != srcCreated || tr.UpdatedAt != srcUpdated {
			t.Errorf("tournament %q dates = %q / %q, want the source's", tr.Name, tr.CreatedAt, tr.UpdatedAt)
		}
	}
	n := 0
	for d, err := range target.Transcriptions().List(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		n++
		if d.CreatedAt != srcCreated || d.UpdatedAt != srcUpdated {
			t.Errorf("draft dates = %q / %q, want the source's", d.CreatedAt, d.UpdatedAt)
		}
	}
	if n != 1 {
		t.Errorf("%d drafts imported, want 1", n)
	}
	aliases, err := target.Aliases().List(ctx, "", storage.AliasEvent)
	if err != nil {
		t.Fatal(err)
	}
	if len(aliases) != 1 || aliases[0].CreatedAt != srcCreated {
		t.Errorf("aliases = %+v, want OD created %q", aliases, srcCreated)
	}
}

// TestPlanImportMatches: the preview counts what MergeMatches then does —
// a match the target holds by match_hash or by canonical_hash is known; the
// others are to import.
func TestPlanImportMatches(t *testing.T) {
	ctx := context.Background()
	srcPath := filepath.Join(t.TempDir(), "club.db")
	src, err := sqlite.Open(ctx, srcPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []domain.Match{
		{Player1Name: "A", Player2Name: "B", MatchLength: 5, MatchHash: "h-held"},
		{Player1Name: "C", Player2Name: "D", MatchLength: 5, MatchHash: "h-other-format", CanonicalHash: "c-held"},
		{Player1Name: "G", Player2Name: "H", MatchLength: 5, MatchHash: "h-new", CanonicalHash: "c-new"},
		{Player1Name: "E", Player2Name: "F", MatchLength: 3},
	} {
		if _, err := src.Matches().Save(ctx, "", &m); err != nil {
			t.Fatal(err)
		}
	}
	source := storage.Stores(src)

	target := openMemTarget(t)
	if _, err := target.Matches().Save(ctx, "", &domain.Match{Player1Name: "A", Player2Name: "B",
		MatchLength: 5, MatchHash: "h-held"}); err != nil {
		t.Fatal(err)
	}
	if _, err := target.Matches().Save(ctx, "", &domain.Match{Player1Name: "C", Player2Name: "D",
		MatchLength: 5, MatchHash: "h-this-format", CanonicalHash: "c-held"}); err != nil {
		t.Fatal(err)
	}

	plan, err := PlanImportMatches(ctx, target, "", source)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ToImport != 2 || plan.Known != 2 || plan.Unreadable {
		t.Errorf("plan = %+v, want 2 to import, 2 known", plan)
	}
	merged, err := MergeMatches(ctx, target, "", source, map[int64]int64{})
	if err != nil {
		t.Fatal(err)
	}
	if merged.Matches != plan.ToImport || merged.Duplicates != plan.Known {
		t.Errorf("merge = %d created, %d known; the plan said %+v", merged.Matches, merged.Duplicates, plan)
	}
	src.Close()
}

// TestDBImportLivingOntoManualList: a living collection arriving on the
// recipient's hand-made list of the same name leaves that list as it was,
// and the summary names it.
func TestDBImportLivingOntoManualList(t *testing.T) {
	ctx := context.Background()
	srcPath := filepath.Join(t.TempDir(), "club.db")
	src, err := sqlite.Open(ctx, srcPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	p := domain.InitializePosition()
	p.Dice = [2]int{5, 2}
	if _, err := src.Positions().Save(ctx, "", &p); err != nil {
		t.Fatal(err)
	}
	m, err := src.Matches().Save(ctx, "", &domain.Match{Player1Name: "A", Player2Name: "B", MatchLength: 5, MatchHash: "h-a"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := storage.CreateCollection(ctx, src, "", "Mes coups", "", fmt.Sprintf("ma%d", m)); err != nil {
		t.Fatal(err)
	}
	src.Close()

	target := openMemTarget(t)
	own := domain.InitializePosition()
	own.Dice = [2]int{4, 1}
	ownID, err := target.Positions().Save(ctx, "", &own)
	if err != nil {
		t.Fatal(err)
	}
	list, err := target.Collections().Create(ctx, "", "Mes coups", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := target.Collections().AddPositions(ctx, "", list, []int64{ownID}); err != nil {
		t.Fatal(err)
	}

	sum, err := DBImporter{S: target}.Import(ctx, "", Source{Path: srcPath}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(sum.LivingCollectionsOnManual, []string{"Mes coups"}) {
		t.Errorf("livingCollectionsOnManual = %v, want [Mes coups]", sum.LivingCollectionsOnManual)
	}
	if sum.Collections != 0 {
		t.Errorf("collections changed = %d, want 0", sum.Collections)
	}
	var members []int64
	for m, err := range target.Collections().Members(ctx, "", list) {
		if err != nil {
			t.Fatal(err)
		}
		members = append(members, m.PositionID)
	}
	if !slices.Equal(members, []int64{ownID}) {
		t.Errorf("list members = %v, want only the recipient's %d", members, ownID)
	}
	c, err := target.Collections().Get(ctx, "", list)
	if err != nil {
		t.Fatal(err)
	}
	if c.FilterQuery != "" {
		t.Errorf("the hand-made list became living: %q", c.FilterQuery)
	}
}

// TestDBImportMergedTournamentKeepsRecipientDates: a tournament merged into
// the target's namesake keeps its event date and created_at, and the source's
// dates apply only to a tournament the import creates.
func TestDBImportMergedTournamentKeepsRecipientDates(t *testing.T) {
	ctx := context.Background()
	srcPath := filepath.Join(t.TempDir(), "club.db")
	src, err := sqlite.Open(ctx, srcPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	tour, err := src.Tournaments().Create(ctx, "", "Open commun", "2018-02-03", "Nice")
	if err != nil {
		t.Fatal(err)
	}
	mid, err := src.Matches().Save(ctx, "", &domain.Match{Player1Name: "A", Player2Name: "B", MatchLength: 3, MatchHash: "h-shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := src.Tournaments().AddMatch(ctx, "", tour, mid); err != nil {
		t.Fatal(err)
	}
	if err := src.Tournaments().KeepDates(ctx, "", tour, srcCreated, srcUpdated); err != nil {
		t.Fatal(err)
	}
	src.Close()

	target := openMemTarget(t)
	const recvCreated, recvUpdated = "2024-05-06 07:08:09", "2025-01-02 03:04:05"
	rid, err := target.Tournaments().Create(ctx, "", "Open commun", "2024-05-06", "Lyon")
	if err != nil {
		t.Fatal(err)
	}
	if err := target.Tournaments().KeepDates(ctx, "", rid, recvCreated, recvUpdated); err != nil {
		t.Fatal(err)
	}
	if _, err := (DBImporter{S: target}).Import(ctx, "", Source{Path: srcPath}, nil); err != nil {
		t.Fatal(err)
	}

	n := 0
	for tr, err := range target.Tournaments().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		n++
		// A new match advances updated_at; the event's date and created_at stay.
		if tr.CreatedAt != recvCreated || tr.Date != "2024-05-06" {
			t.Errorf("tournament %q created/date = %q / %q, want the recipient's %q / 2024-05-06", tr.Name, tr.CreatedAt, tr.Date, recvCreated)
		}
		if tr.UpdatedAt == srcUpdated {
			t.Errorf("tournament %q took the source's updated_at", tr.Name)
		}
	}
	if n != 1 {
		t.Errorf("%d tournaments after the merge, want 1", n)
	}
}
