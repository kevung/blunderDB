//go:build postgres

// Cross-backend parity for the stats engine: seed a real XG match into SQLite,
// migrate it into a PostgreSQL tenant, then assert both StatsStore backends
// compute the same aggregates over the same logical data. This is the Postgres
// counterpart of the SQLite-vs-legacy parity test in package database.
//
// Only ID- and order-independent values are compared: row ids, match-date
// *strings* and the rolling windows differ legitimately between backends
// (auto-increment ids per tenant, TIMESTAMPTZ vs TEXT storage, tie-break order
// in ORDER BY). MWC sums use a float tolerance because floating-point addition
// is not associative and the two backends may sum equal multisets in a
// different order; PR / Snowie / counts derive from integer sums and must match
// exactly.
package migrate_test

import (
	"context"
	"math"
	"sort"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/migrate"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
)

const mwcTol = 1e-6

func approxEqual(t *testing.T, label string, a, b float64) {
	t.Helper()
	if math.Abs(a-b) > mwcTol {
		t.Errorf("%s: %v vs %v (|Δ|=%g > %g)", label, a, b, math.Abs(a-b), mwcTol)
	}
}

func firstMatchID(t *testing.T, s storage.Storage, scope string) int64 {
	t.Helper()
	for m, err := range s.Matches().List(context.Background(), scope) {
		if err != nil {
			t.Fatalf("Matches().List: %v", err)
		}
		return m.ID
	}
	t.Fatalf("no match found for scope %q", scope)
	return 0
}

func TestStatsCrossBackendParity(t *testing.T) {
	ctx := context.Background()
	src, _, _ := seedSQLite(t) // SQLite storage holding the XG match (scope "")

	dsn := startPostgres(t)
	dst, err := postgres.Open(ctx, dsn, nil)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer dst.Close()
	if err := dst.Migrate(ctx); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}
	if _, err := migrate.Run(ctx, src, dst, "1", migrate.Options{}); err != nil {
		t.Fatalf("migrate data: %v", err)
	}

	filter := storage.StatsFilter{DecisionType: -1}
	sRes, err := src.Stats().Compute(ctx, "", filter)
	if err != nil {
		t.Fatalf("sqlite Compute: %v", err)
	}
	pRes, err := dst.Stats().Compute(ctx, "1", filter)
	if err != nil {
		t.Fatalf("postgres Compute: %v", err)
	}

	// Totals — exact.
	if sRes.Totals != pRes.Totals {
		t.Errorf("Totals: sqlite=%+v postgres=%+v", sRes.Totals, pRes.Totals)
	}
	// PR / Snowie — derived from integer sums, exact.
	if sRes.PRGlobal != pRes.PRGlobal || sRes.PRChecker != pRes.PRChecker || sRes.PRCube != pRes.PRCube {
		t.Errorf("PR: sqlite=(%v,%v,%v) postgres=(%v,%v,%v)",
			sRes.PRGlobal, sRes.PRChecker, sRes.PRCube, pRes.PRGlobal, pRes.PRChecker, pRes.PRCube)
	}
	if sRes.SnowieGlobal != pRes.SnowieGlobal {
		t.Errorf("SnowieGlobal: sqlite=%v postgres=%v", sRes.SnowieGlobal, pRes.SnowieGlobal)
	}
	// MWC — float tolerance (sum order may differ).
	if sRes.MWCAvailable != pRes.MWCAvailable {
		t.Errorf("MWCAvailable: sqlite=%v postgres=%v", sRes.MWCAvailable, pRes.MWCAvailable)
	}
	approxEqual(t, "MWCGlobal", sRes.MWCGlobal, pRes.MWCGlobal)
	approxEqual(t, "MWCChecker", sRes.MWCChecker, pRes.MWCChecker)
	approxEqual(t, "MWCCube", sRes.MWCCube, pRes.MWCCube)

	// Error histogram — bucket counts, ordered by bucket on both sides.
	if len(sRes.ErrorHistogram) != len(pRes.ErrorHistogram) {
		t.Fatalf("ErrorHistogram len: sqlite=%d postgres=%d", len(sRes.ErrorHistogram), len(pRes.ErrorHistogram))
	}
	for i := range sRes.ErrorHistogram {
		if sRes.ErrorHistogram[i] != pRes.ErrorHistogram[i] {
			t.Errorf("ErrorHistogram[%d]: sqlite=%+v postgres=%+v", i, sRes.ErrorHistogram[i], pRes.ErrorHistogram[i])
		}
	}

	// Cube-action breakdown — grouped by action (id-free). Sort both by action.
	sortCube := func(c []storage.CubeActionStats) {
		sort.Slice(c, func(i, j int) bool { return c[i].Action < c[j].Action })
	}
	sortCube(sRes.CubeActionBreakdown)
	sortCube(pRes.CubeActionBreakdown)
	if len(sRes.CubeActionBreakdown) != len(pRes.CubeActionBreakdown) {
		t.Fatalf("CubeActionBreakdown len: sqlite=%d postgres=%d",
			len(sRes.CubeActionBreakdown), len(pRes.CubeActionBreakdown))
	}
	for i := range sRes.CubeActionBreakdown {
		a, b := sRes.CubeActionBreakdown[i], pRes.CubeActionBreakdown[i]
		if a.Action != b.Action || a.NumDecisions != b.NumDecisions || a.BlunderCount != b.BlunderCount || a.PR != b.PR {
			t.Errorf("CubeActionBreakdown[%d]: sqlite=%+v postgres=%+v", i, a, b)
		}
		approxEqual(t, "CubeActionBreakdown["+a.Action+"].MWC", a.MWC, b.MWC)
	}

	// PlayerNames — names + counts, id-free.
	sPN, err := src.Stats().PlayerNames(ctx, "")
	if err != nil {
		t.Fatalf("sqlite PlayerNames: %v", err)
	}
	pPN, err := dst.Stats().PlayerNames(ctx, "1")
	if err != nil {
		t.Fatalf("postgres PlayerNames: %v", err)
	}
	if len(sPN) != len(pPN) {
		t.Fatalf("PlayerNames len: sqlite=%d postgres=%d", len(sPN), len(pPN))
	}
	for i := range sPN {
		if sPN[i] != pPN[i] {
			t.Errorf("PlayerNames[%d]: sqlite=%+v postgres=%+v", i, sPN[i], pPN[i])
		}
	}

	// DateRange — both normalised to YYYY-MM-DD.
	sDR, err := src.Stats().DateRange(ctx, "")
	if err != nil {
		t.Fatalf("sqlite DateRange: %v", err)
	}
	pDR, err := dst.Stats().DateRange(ctx, "1")
	if err != nil {
		t.Fatalf("postgres DateRange: %v", err)
	}
	if sDR != pDR {
		t.Errorf("DateRange: sqlite=%+v postgres=%+v", sDR, pDR)
	}

	// MatchDetail — per-player scalar stats (match ids differ per backend).
	sDetail, err := src.Stats().MatchDetail(ctx, "", firstMatchID(t, src, ""))
	if err != nil {
		t.Fatalf("sqlite MatchDetail: %v", err)
	}
	pDetail, err := dst.Stats().MatchDetail(ctx, "1", firstMatchID(t, dst, "1"))
	if err != nil {
		t.Fatalf("postgres MatchDetail: %v", err)
	}
	comparePlayer(t, "Player1", sDetail.Player1, pDetail.Player1)
	comparePlayer(t, "Player2", sDetail.Player2, pDetail.Player2)

	// Tenant isolation: an unrelated tenant sees no decisions.
	empty, err := dst.Stats().Compute(ctx, "2", filter)
	if err != nil {
		t.Fatalf("postgres Compute(tenant 2): %v", err)
	}
	if empty.Totals != (storage.StatsTotals{}) || empty.MWCAvailable {
		t.Errorf("tenant 2 not isolated: totals=%+v mwcAvail=%v", empty.Totals, empty.MWCAvailable)
	}
}

func comparePlayer(t *testing.T, label string, a, b storage.MatchPlayerDetailStats) {
	t.Helper()
	// Integer-derived fields and PR — exact.
	if a.TotalDecisions != b.TotalDecisions || a.TotalErrors != b.TotalErrors || a.TotalBlunders != b.TotalBlunders ||
		a.CheckerDecisions != b.CheckerDecisions || a.CheckerErrors != b.CheckerErrors || a.CheckerBlunders != b.CheckerBlunders ||
		a.DoubleDecisions != b.DoubleDecisions || a.DoubleErrors != b.DoubleErrors || a.DoubleBlunders != b.DoubleBlunders ||
		a.TakeDecisions != b.TakeDecisions || a.TakeErrors != b.TakeErrors || a.TakeBlunders != b.TakeBlunders {
		t.Errorf("%s counts: sqlite=%+v postgres=%+v", label, a, b)
	}
	if a.PR != b.PR || a.PRChecker != b.PRChecker || a.PRCube != b.PRCube || a.SnowieER != b.SnowieER {
		t.Errorf("%s PR/Snowie: sqlite=(%v,%v,%v,%v) postgres=(%v,%v,%v,%v)",
			label, a.PR, a.PRChecker, a.PRCube, a.SnowieER, b.PR, b.PRChecker, b.PRCube, b.SnowieER)
	}
	if a.TotalEquityError != b.TotalEquityError || a.CheckerEquityError != b.CheckerEquityError ||
		a.DoubleEquityError != b.DoubleEquityError || a.TakeEquityError != b.TakeEquityError {
		t.Errorf("%s equity error: sqlite=%+v postgres=%+v", label, a, b)
	}
	// MWC losses — float tolerance.
	approxEqual(t, label+".MWCLoss", a.MWCLoss, b.MWCLoss)
	approxEqual(t, label+".CheckerMWCLoss", a.CheckerMWCLoss, b.CheckerMWCLoss)
	approxEqual(t, label+".DoubleMWCLoss", a.DoubleMWCLoss, b.DoubleMWCLoss)
	approxEqual(t, label+".TakeMWCLoss", a.TakeMWCLoss, b.TakeMWCLoss)
	approxEqual(t, label+".CubeMWCLoss", a.CubeMWCLoss, b.CubeMWCLoss)
}
