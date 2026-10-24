package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/remote"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// seedPlayedPosition stores a position and a match whose moves were played
// there — a checker play and a cube action the analyses saved below do not
// name — and returns the position id.
func seedPlayedPosition(t *testing.T, st storage.Stores, scope string) int64 {
	t.Helper()
	ctx := context.Background()
	p := domain.InitializePosition()
	p.DecisionType = domain.CheckerAction
	p.Dice = [2]int{6, 3}
	pid, err := st.Positions().Save(ctx, scope, &p)
	if err != nil {
		t.Fatalf("save position: %v", err)
	}
	mid, err := st.Matches().Save(ctx, scope, &domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 5})
	if err != nil {
		t.Fatalf("save match: %v", err)
	}
	gid, err := st.Matches().CreateGame(ctx, scope, &domain.Game{MatchID: mid, GameNumber: 1})
	if err != nil {
		t.Fatalf("create game: %v", err)
	}
	for i, mv := range []domain.Move{
		{MoveType: "checker", CheckerMove: "24/18 13/10", Dice: [2]int32{6, 3}},
		{MoveType: "cube", CubeAction: "Double"},
	} {
		mv.GameID, mv.PositionID, mv.MoveNumber, mv.Player = gid, pid, int32(i+1), 1
		if _, err := st.Matches().CreateMove(ctx, scope, &mv); err != nil {
			t.Fatalf("create move: %v", err)
		}
	}
	return pid
}

// parityAnalyses are two saves of the same position: the second names other
// candidates and another play, which a merging save keeps beside the first's.
func parityAnalyses() []domain.PositionAnalysis {
	return []domain.PositionAnalysis{
		{
			AnalysisType: "CheckerMove", AnalysisEngineVersion: "XG",
			CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
				{Index: 0, Move: "24/18 13/10", Equity: 0.12},
				{Index: 1, Move: "24/15", Equity: 0.05},
			}},
			PlayedMoves:  []string{"24/15"},
			CreationDate: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		},
		{
			AnalysisType: "CheckerMove", AnalysisEngineVersion: "XG",
			CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
				{Index: 0, Move: "13/7 13/10", Equity: 0.2},
			}},
			PlayedMove: "13/7 13/10",
		},
	}
}

// comparableAnalysis is a's JSON without what differs by when and where it
// was written: its dates and its position id.
func comparableAnalysis(t *testing.T, a *domain.PositionAnalysis) string {
	t.Helper()
	c := *a
	c.PositionID, c.LastModifiedDate = 0, time.Time{}
	b, err := json.Marshal(&c)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// databaseAnalysisParity runs the parity scenario through the Database
// wrapper, the reference the daemon must match.
func databaseAnalysisParity(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "wrapper.db")
	db := database.NewDatabase()
	if err := db.SetupDatabase(path); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer db.Close()
	st, err := sqlite.Open(context.Background(), path, nil)
	if err != nil {
		t.Fatal(err)
	}
	pid := seedPlayedPosition(t, st, "")
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	for _, a := range parityAnalyses() {
		if err := db.SaveAnalysis(pid, a); err != nil {
			t.Fatalf("SaveAnalysis: %v", err)
		}
	}
	got, err := db.LoadAnalysis(pid)
	if err != nil {
		t.Fatalf("LoadAnalysis: %v", err)
	}
	return comparableAnalysis(t, got)
}

// daemonAnalysisParity runs the parity scenario through /v1/analyses.merge and
// /v1/analyses.load with withPlayed under tenant.
func daemonAnalysisParity(t *testing.T, ts *httptest.Server, st storage.Stores, tenant string) string {
	t.Helper()
	pid := seedPlayedPosition(t, st, tenant)
	for _, a := range parityAnalyses() {
		resp := postAsTenant(t, ts, tenant, "/v1/analyses.merge", analysisSaveReq{PositionID: pid, Analysis: &a})
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("analyses.merge status = %d, want 200", resp.StatusCode)
		}
	}
	resp := postAsTenant(t, ts, tenant, "/v1/analyses.load", analysisLoadReq{PositionID: pid, WithPlayed: true})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("analyses.load status = %d, want 200", resp.StatusCode)
	}
	var got domain.PositionAnalysis
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	return comparableAnalysis(t, &got)
}

// remoteAnalysisParity runs the parity scenario through the remote backend,
// as `call --backend remote` does: the shared helpers over /v1.
func remoteAnalysisParity(t *testing.T, ts *httptest.Server, st storage.Stores, tenant string) string {
	t.Helper()
	ctx := context.Background()
	pid := seedPlayedPosition(t, st, tenant)
	rs, err := remote.Open(remote.Options{BaseURL: ts.URL, Tenant: tenant})
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range parityAnalyses() {
		if err := storage.SaveMergedAnalysis(ctx, rs.Analyses(), "", pid, a); err != nil {
			t.Fatalf("SaveMergedAnalysis over remote: %v", err)
		}
	}
	got, err := storage.LoadAnalysisWithPlayed(ctx, rs, "", pid)
	if err != nil {
		t.Fatalf("LoadAnalysisWithPlayed over remote: %v", err)
	}
	return comparableAnalysis(t, got)
}

// analysesMergeParity checks the daemon, directly and through the remote
// backend, against the Database wrapper.
func analysesMergeParity(t *testing.T, ts *httptest.Server, st storage.Stores, tenant string) {
	t.Helper()
	want := databaseAnalysisParity(t)
	if got := daemonAnalysisParity(t, ts, st, tenant); got != want {
		t.Errorf("daemon analysis differs from the Database wrapper's:\n daemon: %s\nwrapper: %s", got, want)
	}
	if got := remoteAnalysisParity(t, ts, st, tenant); got != want {
		t.Errorf("remote analysis differs from the Database wrapper's:\n remote: %s\nwrapper: %s", got, want)
	}
}

// TestAnalysesSaveLoad_DatabaseParity_SQLite: analyses.merge folds an
// analysis into the stored one and analyses.load with withPlayed adds the
// moves the matches played, exactly as Database.SaveAnalysis and
// Database.LoadAnalysis do.
func TestAnalysesSaveLoad_DatabaseParity_SQLite(t *testing.T) {
	ts, srv := newTestServerAndHandler(t)
	analysesMergeParity(t, ts, srv.opts.Storage, testTenant)
}

// TestAnalysesSaveLoad_DefaultsUnchanged: analyses.save replaces the stored
// analysis and analyses.load without withPlayed answers it as stored, with
// none of the moves the matches played — what a client that rewrites an
// analysis whole relies on.
func TestAnalysesSaveLoad_DefaultsUnchanged(t *testing.T) {
	ts, srv := newTestServerAndHandler(t)
	pid := seedPlayedPosition(t, srv.opts.Storage, testTenant)
	for _, a := range parityAnalyses() {
		resp := postAsTenant(t, ts, testTenant, "/v1/analyses.save", analysisSaveReq{PositionID: pid, Analysis: &a})
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("analyses.save status = %d, want 200", resp.StatusCode)
		}
	}
	resp := postAsTenant(t, ts, testTenant, "/v1/analyses.load", positionIDReq{PositionID: pid})
	defer resp.Body.Close()
	var got domain.PositionAnalysis
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.CheckerAnalysis == nil || len(got.CheckerAnalysis.Moves) != 1 || got.CheckerAnalysis.Moves[0].Move != "13/7 13/10" {
		t.Errorf("analyses.save did not replace: candidates %+v", got.CheckerAnalysis)
	}
	if len(got.PlayedMoves) != 0 || got.PlayedMove != "13/7 13/10" || len(got.PlayedCubeActions) != 0 {
		t.Errorf("analyses.load joined the played moves: %v / %q / %v", got.PlayedMoves, got.PlayedMove, got.PlayedCubeActions)
	}
}
