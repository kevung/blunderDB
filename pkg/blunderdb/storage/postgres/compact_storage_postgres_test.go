//go:build postgres

package postgres_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
)

func compactPosition(n int) *domain.Position {
	p := domain.InitializePosition()
	p.DecisionType = domain.CheckerAction
	for i := range 7 {
		p.Board.Points[9+i] = domain.Point{Checkers: n % 4, Color: domain.White}
		n /= 4
	}
	return &p
}

func entryOf(t *testing.T, s *pg.Storage, ctx context.Context, scope string, id int64) storage.Change {
	t.Helper()
	u, err := s.ObjectUUIDs().ObjectUUID(ctx, scope, "position", id)
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.ChangeJournal().Entry(ctx, scope, u)
	if err != nil {
		t.Fatalf("Entry: %v", err)
	}
	return c
}

// A position and its analysis written in one transaction are one entry, a create:
// no reader saw the object between the two. A later transaction moves it.
func TestJournal_SecondWriteOfATransactionAddsNothing(t *testing.T) {
	bg := context.Background()
	s, _ := openRLS(t)
	const scope = "1"
	ctx := tctx(bg, scope)
	tx, err := s.BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	id, err := tx.Positions().Save(ctx, scope, compactPosition(1))
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Analyses().Save(ctx, scope, id, &domain.PositionAnalysis{AnalysisType: "CheckerMove"}); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	first := entryOf(t, s, ctx, scope, id)
	if first.Op != storage.ChangeCreate {
		t.Fatalf("op after one transaction = %q, want create", first.Op)
	}
	if err := s.Analyses().Save(ctx, scope, id, &domain.PositionAnalysis{AnalysisType: "DoublingCube"}); err != nil {
		t.Fatal(err)
	}
	second := entryOf(t, s, ctx, scope, id)
	if second.Op != storage.ChangeUpdate || second.Seq <= first.Seq || second.ChangedAt <= first.ChangedAt {
		t.Errorf("a later transaction left the entry at %+v (was %+v)", second, first)
	}
}

// A dated entry in the transaction does not swallow the edit that follows it.
func TestJournal_EditAfterADatedMarkStillDates(t *testing.T) {
	bg := context.Background()
	s, _ := openRLS(t)
	const scope = "1"
	ctx := tctx(bg, scope)
	id, err := s.Positions().Save(ctx, scope, compactPosition(2))
	if err != nil {
		t.Fatal(err)
	}
	u, err := s.ObjectUUIDs().ObjectUUID(ctx, scope, "position", id)
	if err != nil {
		t.Fatal(err)
	}
	const past = "2001-01-01T00:00:00.000Z"
	tx, err := s.BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	j := tx.(storage.ChangeJournalProvider).ChangeJournal()
	if err := j.Mark(ctx, scope, "position", u, storage.ChangeUpdate, past); err != nil {
		t.Fatal(err)
	}
	if err := tx.Analyses().Save(ctx, scope, id, &domain.PositionAnalysis{AnalysisType: "CheckerMove"}); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if c := entryOf(t, s, ctx, scope, id); c.ChangedAt == past {
		t.Errorf("the edit after a dated mark kept the date %s", c.ChangedAt)
	}
}

// A library that cuts its journal writes none, refuses by name to synchronize, and
// leaves its neighbour's journal alone; restored, what it wrote meanwhile reaches a peer.
func TestJournalOff(t *testing.T) {
	bg := context.Background()
	s, _ := openRLS(t)
	pub, priv := tctx(bg, "1"), tctx(bg, "2")
	j := s.ChangeJournal()

	before, err := s.Positions().Save(pub, "1", compactPosition(3))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Positions().Save(priv, "2", compactPosition(4)); err != nil {
		t.Fatal(err)
	}
	if err := j.SetJournalOff(pub, "1", true); err != nil {
		t.Fatal(err)
	}
	during, err := s.Positions().Save(pub, "1", compactPosition(5))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Analyses().Save(pub, "1", during, &domain.PositionAnalysis{AnalysisType: "CheckerMove"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Positions().Delete(pub, "1", before); err != nil {
		t.Fatal(err)
	}
	if cs, err := j.Changes(pub, "1", 0, 100); err != nil || len(cs) != 0 {
		t.Errorf("journal of the cut library = %+v, %v; want empty", cs, err)
	}
	if _, err := changeset.Export(pub, s, "1", storage.Cursor{}, 10); !errors.Is(err, storage.ErrJournalOff) {
		t.Errorf("Export of a cut library = %v, want ErrJournalOff", err)
	}
	if _, err := changeset.Import(pub, s, "1", changeset.Set{}); !errors.Is(err, storage.ErrJournalOff) {
		t.Errorf("Import into a cut library = %v, want ErrJournalOff", err)
	}

	second, err := s.Positions().Save(priv, "2", compactPosition(6))
	if err != nil {
		t.Fatal(err)
	}
	if c := entryOf(t, s, priv, "2", second); c.Op != storage.ChangeCreate {
		t.Errorf("the neighbour's journal: %+v", c)
	}
	set, err := changeset.Export(priv, s, "2", storage.Cursor{}, 100)
	if err != nil || len(set.Records) != 2 {
		t.Errorf("the neighbour's export = %d records, %v; want 2", len(set.Records), err)
	}

	if err := j.SetJournalOff(pub, "1", false); err != nil {
		t.Fatal(err)
	}
	set, err = changeset.Export(pub, s, "1", storage.Cursor{}, 100)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	u, _ := s.ObjectUUIDs().ObjectUUID(pub, "1", "position", during)
	for _, r := range set.Records {
		found = found || r.UUID == u
	}
	if !found {
		t.Errorf("restored, the export of %d records misses what was written during the cut", len(set.Records))
	}
}

// A cursor taken before a cut passed nothing of what was written during it: restored, the journal
// refuses it (a full resumption, as for a journal from elsewhere) instead of answering from it.
func TestJournalOff_RestoredRefusesAnOlderCursor(t *testing.T) {
	bg := context.Background()
	s, _ := openRLS(t)
	pub := tctx(bg, "1")
	j := s.ChangeJournal()

	if _, err := s.Positions().Save(pub, "1", compactPosition(3)); err != nil {
		t.Fatal(err)
	}
	old, err := changeset.Export(pub, s, "1", storage.Cursor{}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if err := j.SetJournalOff(pub, "1", false); err != nil {
		t.Fatal(err)
	}
	if _, err := changeset.Export(pub, s, "1", old.Cursor, 100); err != nil {
		t.Errorf("a journal never cut refuses its cursor: %v", err)
	}
	if err := j.SetJournalOff(pub, "1", true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Positions().Save(pub, "1", compactPosition(5)); err != nil {
		t.Fatal(err)
	}
	if err := j.SetJournalOff(pub, "1", false); err != nil {
		t.Fatal(err)
	}
	if got, err := changeset.Export(pub, s, "1", old.Cursor, 100); !errors.Is(err, storage.ErrCursorStale) {
		t.Errorf("a cursor of before the cut = %d records, %v; want ErrCursorStale", len(got.Records), err)
	}
	fresh, err := changeset.Export(pub, s, "1", storage.Cursor{}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Positions().Save(pub, "1", compactPosition(6)); err != nil {
		t.Fatal(err)
	}
	if got, err := changeset.Export(pub, s, "1", fresh.Cursor, 100); err != nil || len(got.Records) != 1 {
		t.Errorf("a cursor of after the restore = %d records, %v; want 1", len(got.Records), err)
	}
}

// One tenant's cut is not another's, even set in its name by a statement that
// forgets to filter: the policy covers the setting.
func TestJournalOff_PolicyHoldsTheSetting(t *testing.T) {
	bg := context.Background()
	s, dsn := openRLS(t)
	if err := s.ChangeJournal().SetJournalOff(tctx(bg, "1"), "1", true); err != nil {
		t.Fatal(err)
	}
	conn, err := pgx.Connect(bg, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(bg)
	if _, err := conn.Exec(bg, `SELECT set_config('app.tenant_id', '2', false)`); err != nil {
		t.Fatal(err)
	}
	var off bool
	if err := conn.QueryRow(bg, `SELECT blunderdb_journal_off(1)`).Scan(&off); err != nil {
		t.Fatal(err)
	}
	if off {
		t.Error("tenant 2 reads tenant 1's setting")
	}
	if off, err = s.ChangeJournal().JournalOff(tctx(bg, "2"), "2"); err != nil || off {
		t.Errorf("tenant 2 cut by tenant 1's setting: %v, %v", off, err)
	}
}

// A value wider than its column is refused by name, on every way into it.
func TestNarrowColumn_WiderValueIsInvalid(t *testing.T) {
	bg := context.Background()
	s, _ := openRLS(t)
	ctx := tctx(bg, "1")
	far := compactPosition(8)
	far.Board.Points[3] = domain.Point{Checkers: 40000, Color: domain.White}
	if _, err := s.Positions().Save(ctx, "1", far); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("Save of a board past smallint = %v, want ErrInvalid", err)
	}
	many := []*domain.Position{far, compactPosition(9)}
	if _, _, err := s.Positions().SaveMany(ctx, "1", many); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("SaveMany of a board past smallint = %v, want ErrInvalid", err)
	}
}

// The largest values the types are chosen for go in and come out unchanged.
func TestNarrowColumn_BoundsRoundTrip(t *testing.T) {
	bg := context.Background()
	s, _ := openRLS(t)
	ctx := tctx(bg, "1")
	p := compactPosition(10)
	p.Cube.Value = 2000000
	p.Score = [2]int{99999, 99999}
	id, err := s.Positions().Save(ctx, "1", p)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Positions().Load(ctx, "1", id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Cube.Value != p.Cube.Value || got.Score != p.Score {
		t.Errorf("got cube %d score %v", got.Cube.Value, got.Score)
	}
}

// A stored value past its narrowed type refuses the rewrite by table, column, value and row,
// which the operator corrects before starting again, rather than a bare 22003.
func TestNarrowColumns_RefusalNamesTheValue(t *testing.T) {
	bg := context.Background()
	owner, conn, dsn := openAsRLSOwner(t)
	if err := owner.ApplyRLS(bg); err != nil {
		t.Fatal(err)
	}
	s, err := pg.Open(bg, dsn, &storage.Options{EnableRLS: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	id, err := s.Positions().Save(tctx(bg, "1"), "1", compactPosition(11))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(bg, `ALTER TABLE position ALTER COLUMN pip_2 TYPE bigint, ALTER COLUMN pip_1 TYPE bigint`); err != nil {
		t.Fatal(err)
	}
	asTenant(t, conn, "1", func() {
		if _, err := conn.Exec(bg, `UPDATE position SET pip_2 = 40000 WHERE id = $1`, id); err != nil {
			t.Fatal(err)
		}
	})
	_, err = conn.Exec(bg, `SELECT blunderdb_narrow_columns()`)
	if err == nil || !strings.Contains(err.Error(), "position.pip_2 holds 40000, past smallint") {
		t.Errorf("narrowing past a stored value = %v; want it named", err)
	}
}
