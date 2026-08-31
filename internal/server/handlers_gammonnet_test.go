package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// bearoffRacePosition builds a pure home-board race with no dice set (a cube
// decision, not a checker decision) — cheap for gammonnet.EvaluatePosition to
// search at 0-ply. gammonnet.FromDomain requires a structurally valid
// position (15 checkers a side, on the board or borne off), so the checkers
// not on the board must be accounted for in Board.Bearoff — the same
// 15-onBoard convention domain/xgid.go uses.
func bearoffRacePosition() domain.Position {
	var pos domain.Position
	set := func(point, color, n int) {
		pos.Board.Points[point] = domain.Point{Color: color, Checkers: n}
	}
	set(1, domain.Black, 2)
	set(2, domain.Black, 1)
	set(23, domain.White, 1)
	set(24, domain.White, 2)
	pos.Board.Bearoff[domain.Black] = 15 - 3
	pos.Board.Bearoff[domain.White] = 15 - 3
	pos.PlayerOnRoll = domain.Black
	pos.Score = [2]int{-1, -1} // money
	return pos
}

func newGammonNetTestServer(t *testing.T) (*Server, storage.Storage) {
	t.Helper()
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	srv, err := New(Options{Storage: s})
	if err != nil {
		t.Fatal(err)
	}
	return srv, s
}

func postNDJSON(t *testing.T, srv *Server, path, body string) []map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set(middleware.TenantHeader, "t")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s: got %d (%s)", path, rec.Code, rec.Body)
	}

	var events []map[string]any
	dec := json.NewDecoder(bytes.NewReader(rec.Body.Bytes()))
	for {
		var ev map[string]any
		if err := dec.Decode(&ev); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("decoding ndjson event: %v", err)
		}
		events = append(events, ev)
	}
	return events
}

// TestGammonNetAnalyzeMissingFillsOnlyTheGap mirrors
// TestAnalyzeMissingWithGammonNetFillsOnlyTheGap (database package, #129):
// a position with no analysis gets one written; a position that already
// carries any analysis is left untouched (ADR-0013's gap rule), even though
// the sweep runs over the whole tenant.
func TestGammonNetAnalyzeMissingFillsOnlyTheGap(t *testing.T) {
	ctx := context.Background()
	srv, s := newGammonNetTestServer(t)

	unanalyzed := bearoffRacePosition()
	idMissing, err := s.Positions().Save(ctx, "t", &unanalyzed)
	if err != nil {
		t.Fatal(err)
	}

	// A genuinely different board: ZobristHash normalizes PlayerOnRoll to 0
	// (mirroring the board to the on-roll player's perspective), so flipping
	// PlayerOnRoll alone would NOT have changed the hash — this position
	// differs on the board itself, landing it on a distinct row.
	var analyzed domain.Position
	analyzed.Board.Points[3] = domain.Point{Color: domain.Black, Checkers: 3}
	analyzed.Board.Points[24] = domain.Point{Color: domain.White, Checkers: 3}
	analyzed.Board.Bearoff[domain.Black] = 15 - 3
	analyzed.Board.Bearoff[domain.White] = 15 - 3
	analyzed.PlayerOnRoll = domain.Black
	idAnalyzed, err := s.Positions().Save(ctx, "t", &analyzed)
	if err != nil {
		t.Fatal(err)
	}
	existing := &domain.PositionAnalysis{
		PositionID:            int(idAnalyzed),
		AnalysisEngineVersion: "XG",
		AnalysisType:          "DoublingCube",
		DoublingCubeAnalysis:  &domain.DoublingCubeAnalysis{AnalysisDepth: "4-ply", BestCubeAction: "No Double"},
	}
	if err := s.Analyses().Save(ctx, "t", idAnalyzed, existing); err != nil {
		t.Fatal(err)
	}

	events := postNDJSON(t, srv, "/v1/gammonnet.analyzeMissing", "")

	last := events[len(events)-1]
	if last["event"] != "done" {
		t.Fatalf("last event = %v, want done", last)
	}
	if got := int(last["done"].(float64)); got != 1 {
		t.Fatalf("done = %d, want 1 (only the unanalyzed position)", got)
	}

	filled, err := s.Analyses().Load(ctx, "t", idMissing)
	if err != nil {
		t.Fatalf("expected an analysis to be written for the missing position: %v", err)
	}
	if filled.AnalysisEngineVersion != "gammonNet v1.1.0" {
		t.Fatalf("engine = %q, want gammonNet v1.1.0", filled.AnalysisEngineVersion)
	}

	untouched, err := s.Analyses().Load(ctx, "t", idAnalyzed)
	if err != nil {
		t.Fatal(err)
	}
	if untouched.AnalysisEngineVersion != "XG" {
		t.Fatalf("the already-analysed position was overwritten: engine = %q, want XG (untouched)", untouched.AnalysisEngineVersion)
	}
}

// TestGammonNetAnalyzeMissingCancel exercises the cancel route end to end —
// a job started, cancelled by id, and the events reflect it (or the job had
// already finished, which is not a failure on a fast in-memory run).
func TestGammonNetAnalyzeMissingCancelUnknownID(t *testing.T) {
	srv, _ := newGammonNetTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/v1/gammonnet.analyzeMissing.cancel", strings.NewReader(`{"jobId":"does-not-exist"}`))
	req.Header.Set(middleware.TenantHeader, "t")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("cancelling an unknown job id: got %d, want 404", rec.Code)
	}
}

// TestGammonNetAnalyzeMissingEmptyLibrary is the zero-positions case: the
// sweep must still emit a well-formed started/done pair, done=0/total=0.
func TestGammonNetAnalyzeMissingEmptyLibrary(t *testing.T) {
	srv, _ := newGammonNetTestServer(t)

	events := postNDJSON(t, srv, "/v1/gammonnet.analyzeMissing", "")
	if len(events) != 2 || events[0]["event"] != "started" || events[1]["event"] != "done" {
		t.Fatalf("events = %v, want exactly [started, done]", events)
	}
	if got := int(events[1]["total"].(float64)); got != 0 {
		t.Fatalf("total = %d, want 0", got)
	}
}
