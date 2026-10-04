package migrate_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/migrate"
	"github.com/kevung/blunderdb/pkg/blunderdb/rollouts"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// checkMETsMigrated migrates a library holding two tables, one current and
// one citing a gammonNet analysis, into dst, and checks that both tables,
// the current one and the analysis's table arrive (ADR-0068).
func checkMETsMigrated(t *testing.T, dst storage.Storage, scope string) {
	t.Helper()
	ctx := context.Background()
	src, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	if err := src.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	mt := src.MatchEquityTables()
	rk, err := os.ReadFile("../engine/testdata/met/Rockwell-Kazaross.xml")
	if err != nil {
		t.Fatal(err)
	}
	otherSource := strings.Replace(string(rk), "<me>0.676888</me>", "<me>0.670000</me>", 1)
	clubDigest, otherDigest := metDigest(t, rk), metDigest(t, []byte(otherSource))
	club, err := mt.Save(ctx, "", domain.MatchEquityTable{Name: "Club", Digest: clubDigest, Source: string(rk)})
	if err != nil {
		t.Fatal(err)
	}
	other, err := mt.Save(ctx, "", domain.MatchEquityTable{Name: "Other", Digest: otherDigest, Source: otherSource})
	if err != nil {
		t.Fatal(err)
	}
	if err := mt.SetCurrent(ctx, "", other); err != nil {
		t.Fatal(err)
	}
	pos := domain.InitializePosition()
	pos.DecisionType = domain.CheckerAction
	srcID, err := src.Positions().Save(ctx, "", &pos)
	if err != nil {
		t.Fatal(err)
	}
	a := &domain.PositionAnalysis{
		AnalysisType: "CheckerMove",
		CreationDate: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Index: 0, Move: "13/10 6/5", Equity: 0.1, AnalysisEngine: "gammonNet v1.2.1", AnalysisDepth: "2-ply"},
		}},
	}
	if err := rollouts.SaveValuedAnalysis(ctx, src, "", srcID, a, club); err != nil {
		t.Fatal(err)
	}

	if _, err := migrate.Run(ctx, src, dst, scope, migrate.Options{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	list, err := dst.MatchEquityTables().List(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]int64{}
	for _, tb := range list {
		ids[tb.Digest] = tb.ID
	}
	if len(list) != 2 || ids[clubDigest] == 0 || ids[otherDigest] == 0 {
		t.Fatalf("migrated tables %+v, want Club and Other", list)
	}
	if cur, err := dst.MatchEquityTables().Current(ctx, scope); err != nil || cur == nil || cur.ID != ids[otherDigest] {
		t.Errorf("current table %+v, %v; want Other", cur, err)
	}
	dstID, err := dst.Positions().Save(ctx, scope, &pos)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := dst.MatchEquityTables().OfAnalysis(ctx, scope, dstID); err != nil || got != ids[clubDigest] {
		t.Errorf("migrated analysis names table %d, %v; want Club (%d)", got, err, ids[clubDigest])
	}
}

func metDigest(t *testing.T, data []byte) string {
	t.Helper()
	m, err := engine.ParseGnubgMET(data)
	if err != nil {
		t.Fatal(err)
	}
	return m.Digest()
}

func TestMigrateCarriesMETs_SQLite(t *testing.T) {
	ctx := context.Background()
	dst, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()
	if err := dst.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	checkMETsMigrated(t, dst, "")
}
