package mets

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

const testdata = "../engine/testdata/met/"

func openStore(t *testing.T) storage.Storage {
	t.Helper()
	s, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func readFile(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(testdata + name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestImportStoresATableOnceAndNeverTheBuiltIn(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)

	builtIn, err := Import(ctx, s, "", readFile(t, "Kazaross-XG2.xml"), "kxg")
	if err != nil || builtIn.ID != 0 || builtIn.Name != BuiltInName {
		t.Fatalf("Import(Kazaross-XG2) = %+v, %v; want the built-in table, not stored", builtIn, err)
	}
	rk := readFile(t, "Rockwell-Kazaross.xml")
	first, err := Import(ctx, s, "", rk, "first")
	if err != nil || first.ID == 0 {
		t.Fatalf("Import(Rockwell-Kazaross) = %+v, %v", first, err)
	}
	again, err := Import(ctx, s, "", rk, "second")
	if err != nil || again.ID != first.ID || again.Name != first.Name {
		t.Errorf("second import = %+v, %v; want the held %+v", again, err, first)
	}
	list, err := s.MatchEquityTables().List(ctx, "")
	if err != nil || len(list) != 1 {
		t.Errorf("List = %v, %v; want one table", list, err)
	}

	if _, err := Import(ctx, s, "", []byte("<not a table/>"), "x"); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("Import(garbage) = %v, want ErrInvalid", err)
	}
	if _, err := Import(ctx, s, "", make([]byte, MaxSourceBytes+1), "x"); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("Import(oversized) = %v, want ErrInvalid", err)
	}
}

func TestCurrentOverviewAndStatus(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)

	if id, m, err := Current(ctx, s, ""); err != nil || id != 0 || m != nil {
		t.Fatalf("Current on a fresh library = %d, %v, %v; want the built-in table", id, m, err)
	}
	rk, err := Import(ctx, s, "", readFile(t, "Rockwell-Kazaross.xml"), "")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MatchEquityTables().SetCurrent(ctx, "", rk.ID); err != nil {
		t.Fatal(err)
	}
	id, m, err := Current(ctx, s, "")
	if err != nil || id != rk.ID || m == nil || m.Digest() != rk.Digest {
		t.Fatalf("Current = %d, %v, %v; want %d parsed", id, m, err, rk.ID)
	}
	tables, err := Overview(ctx, s, "")
	if err != nil || len(tables) != 2 || tables[0].Name != BuiltInName || tables[0].Current || !tables[1].Current {
		t.Fatalf("Overview = %+v, %v; want the built-in first, the imported one current", tables, err)
	}

	p := domain.InitializePosition()
	p.Score = [2]int{3, 5}
	pos, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatal(err)
	}
	st, err := AnalysisStatus(ctx, s, "", pos, false)
	if err != nil || st.Name != BuiltInName || st.Current != rk.Name || !st.Different {
		t.Errorf("untagged at a match score = %+v, %v; want different from %q", st, err, rk.Name)
	}
	if st, _ := AnalysisStatus(ctx, s, "", pos, true); st.Different {
		t.Errorf("money = %+v; never different", st)
	}
}

// variant is a valid table whose values differ from Rockwell-Kazaross.
func variant(t *testing.T) []byte {
	t.Helper()
	rk := string(readFile(t, "Rockwell-Kazaross.xml"))
	v := strings.Replace(rk, "<me>0.676888</me>", "<me>0.670000</me>", 1)
	if v == rk {
		t.Fatal("variant: value to change not found")
	}
	return []byte(v)
}

func digestOf(t *testing.T, data []byte) string {
	t.Helper()
	m, err := engine.ParseGnubgMET(data)
	if err != nil {
		t.Fatal(err)
	}
	return m.Digest()
}

// The carrier names a table by the digest of its source, never by the one
// the source database declares.
func TestCarrierMergesByDigestAndMapsTheBuiltInToZero(t *testing.T) {
	ctx := context.Background()
	dst := openStore(t)
	rk, other, kxg := readFile(t, "Rockwell-Kazaross.xml"), variant(t), readFile(t, "Kazaross-XG2.xml")
	if digestOf(t, other) == digestOf(t, rk) {
		t.Fatal("variant has the digest of Rockwell-Kazaross")
	}
	held, err := dst.MatchEquityTables().Save(ctx, "", domain.MatchEquityTable{Name: "Held", Digest: digestOf(t, rk), Source: string(rk)})
	if err != nil {
		t.Fatal(err)
	}
	c := NewCarrier(dst.MatchEquityTables(), "", []*domain.MatchEquityTable{
		{ID: 7, Name: "Club", Digest: "declared", Source: string(rk)},
		{ID: 8, Name: "New", Digest: "new", Source: string(other)},
		{ID: 9, Name: "Kazaross copy", Digest: "copy", Source: string(kxg)},
		{ID: 10, Name: BuiltInName, Digest: engine.KazarossXG2Digest(), Source: string(other)},
		{ID: 11, Name: "Usurper", Digest: digestOf(t, rk), Source: string(other)},
		{ID: 12, Name: "Garbage", Digest: "garbage", Source: "<met/>"},
	})
	for src, want := range map[int64]int64{0: 0, 7: held, 9: 0} {
		if got, err := c.Target(ctx, src); err != nil || got != want {
			t.Errorf("Target(%d) = %d, %v; want %d", src, got, err, want)
		}
	}
	fresh, err := c.Target(ctx, 8)
	if err != nil || fresh == 0 || fresh == held {
		t.Fatalf("Target(new) = %d, %v; want a new id", fresh, err)
	}
	if again, _ := c.Target(ctx, 8); again != fresh {
		t.Errorf("Target(new) twice = %d then %d", fresh, again)
	}
	for _, src := range []int64{10, 11} {
		if got, err := c.Target(ctx, src); err != nil || got != fresh {
			t.Errorf("Target(forged %d) = %d, %v; want the table its source holds (%d)", src, got, err, fresh)
		}
	}
	if cur, err := dst.MatchEquityTables().Current(ctx, ""); err != nil || cur != nil {
		t.Errorf("carried tables made current: %+v, %v", cur, err)
	}
	if _, err := c.Target(ctx, 12); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("Target(unparseable) = %v, want ErrInvalid", err)
	}
	if _, err := c.Target(ctx, 99); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Target(unknown) = %v, want ErrNotFound", err)
	}
}

func verdict(engineLabel string, created time.Time) *domain.PositionAnalysis {
	return &domain.PositionAnalysis{
		AnalysisType: "CheckerMove", CreationDate: created,
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{{Move: "13/10 6/5", AnalysisEngine: engineLabel, AnalysisDepth: "2-ply"}}},
	}
}

func TestAfterMergeKeepsTheTableOfTheVerdictKept(t *testing.T) {
	t1 := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	t2 := t1.Add(time.Hour)
	gnOld, gnNew, xg := verdict("gammonNet v1.2.1", t1), verdict("gammonNet v1.2.1", t2), verdict("XG", t1)
	for _, tc := range []struct {
		name               string
		merged             *domain.PositionAnalysis
		existing, imported Side
		want               int64
	}{
		{"existing gammonNet kept", gnOld, SideOf(gnOld, 3), SideOf(gnNew, 4), 3},
		{"imported gammonNet taken", gnNew, SideOf(nil, 0), SideOf(gnNew, 4), 4},
		{"XG verdict", xg, SideOf(gnOld, 3), SideOf(xg, 0), 0},
		{"tie goes to the existing side", gnOld, SideOf(gnOld, 3), SideOf(gnOld, 4), 3},
		{"neither side", gnNew, SideOf(gnOld, 3), SideOf(xg, 0), 0},
	} {
		if got := AfterMerge(tc.merged, tc.existing, tc.imported); got != tc.want {
			t.Errorf("%s: %d, want %d", tc.name, got, tc.want)
		}
	}
}
