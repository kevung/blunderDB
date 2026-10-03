package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"iter"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// scopeLog records the scope of every store call a recordingStorage sees, read
// or write. SQLite ignores the scope, so what these tests check is not rows but
// the scopes the daemon hands its store: a tenant that reaches no store call is
// a tenant that is never read.
type scopeLog struct {
	mu     sync.Mutex
	reads  []string
	writes []string
	limits []int // the limit of every Search().Find call
}

func (l *scopeLog) read(scope string) { l.mu.Lock(); l.reads = append(l.reads, scope); l.mu.Unlock() }
func (l *scopeLog) write(scope string) {
	l.mu.Lock()
	l.writes = append(l.writes, scope)
	l.mu.Unlock()
}

func (l *scopeLog) snapshot() (reads, writes []string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.reads), slices.Clone(l.writes)
}

type recordingStorage struct {
	storage.Storage
	log *scopeLog
}

func (r recordingStorage) Matches() storage.MatchStore {
	return recordingMatches{r.Storage.Matches(), r.log}
}
func (r recordingStorage) Positions() storage.PositionStore {
	return recordingPositions{r.Storage.Positions(), r.log}
}
func (r recordingStorage) Search() storage.SearchStore {
	return recordingSearch{r.Storage.Search(), r.log}
}
func (r recordingStorage) Stats() storage.StatsStore { return recordingStats{r.Storage.Stats(), r.log} }
func (r recordingStorage) Analyses() storage.AnalysisStore {
	return recordingAnalyses{r.Storage.Analyses(), r.log}
}

type recordingAnalyses struct {
	storage.AnalysisStore
	log *scopeLog
}

func (a recordingAnalyses) LoadMany(ctx context.Context, scope string, ids []int64) (map[int64]*domain.PositionAnalysis, error) {
	a.log.read(scope)
	return a.AnalysisStore.LoadMany(ctx, scope, ids)
}

type recordingMatches struct {
	storage.MatchStore
	log *scopeLog
}

func (m recordingMatches) Save(ctx context.Context, scope string, x *domain.Match) (int64, error) {
	m.log.write(scope)
	return m.MatchStore.Save(ctx, scope, x)
}
func (m recordingMatches) Get(ctx context.Context, scope string, id int64) (*domain.Match, error) {
	m.log.read(scope)
	return m.MatchStore.Get(ctx, scope, id)
}
func (m recordingMatches) List(ctx context.Context, scope string, o storage.MatchListOpts) iterMatches {
	m.log.read(scope)
	return m.MatchStore.List(ctx, scope, o)
}

func (m recordingMatches) MovePositions(ctx context.Context, scope string, matchID int64) iter.Seq2[*domain.MatchMovePosition, error] {
	m.log.read(scope)
	return m.MatchStore.MovePositions(ctx, scope, matchID)
}

type recordingPositions struct {
	storage.PositionStore
	log *scopeLog
}

func (p recordingPositions) Save(ctx context.Context, scope string, x *domain.Position) (int64, error) {
	p.log.write(scope)
	return p.PositionStore.Save(ctx, scope, x)
}

type recordingSearch struct {
	storage.SearchStore
	log *scopeLog
}

func (s recordingSearch) Find(ctx context.Context, scope string, f domain.SearchFilters, o storage.ListOpts) iterPositions {
	s.log.read(scope)
	s.log.mu.Lock()
	s.log.limits = append(s.log.limits, o.Limit)
	s.log.mu.Unlock()
	return s.SearchStore.Find(ctx, scope, f, o)
}

type recordingStats struct {
	storage.StatsStore
	log *scopeLog
}

func (s recordingStats) Compute(ctx context.Context, scope string, f storage.StatsFilter) (*storage.StatsResult, error) {
	s.log.read(scope)
	return s.StatsStore.Compute(ctx, scope, f)
}
func (s recordingStats) PlayerTable(ctx context.Context, scope string, f storage.StatsFilter) ([]storage.PlayerRow, error) {
	s.log.read(scope)
	return s.StatsStore.PlayerTable(ctx, scope, f)
}

// newRecordingServer is a daemon that trusts X-Read-Tenants, over an
// in-memory SQLite store whose calls are recorded.
func newRecordingServer(t *testing.T, singleTenant bool) (*Server, *scopeLog) {
	t.Helper()
	return newRecordingServerWith(t, Options{SingleTenant: singleTenant, TrustReadTenants: true})
}

// newRecordingServerWith builds the recording daemon from o, its Storage and
// Logger filled in.
func newRecordingServerWith(t *testing.T, o Options) (*Server, *scopeLog) {
	t.Helper()
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	log := &scopeLog{}
	o.Storage = recordingStorage{st, log}
	o.Logger = slog.New(slog.DiscardHandler)
	srv, err := New(o)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return srv, log
}

// serveAcross sends one POST as writer, with readTenants in X-Read-Tenants
// when it is not empty.
func serveAcross(t *testing.T, srv *Server, writer, readTenants, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set(middleware.TenantHeader, writer)
	if readTenants != "" {
		req.Header.Set(middleware.ReadTenantsHeader, readTenants)
	}
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

// acrossCalls is one call per across.* route that spans the whole read set;
// the routes naming one tenant are tested on their own.
var acrossCalls = []struct{ path, body string }{
	{"/v1/across.searchFind", `{}`},
	{"/v1/across.matchesList", `{}`},
	{"/v1/across.statsCompute", `{}`},
	{"/v1/across.playerTable", `{}`},
}

func TestAcross_ReadsTheReadSetInOrder(t *testing.T) {
	for _, c := range acrossCalls {
		t.Run(c.path, func(t *testing.T) {
			srv, log := newRecordingServer(t, false)
			rec := serveAcross(t, srv, "1", " 3, 2 ,3", c.path, c.body)
			if rec.Code != http.StatusOK {
				t.Fatalf("status %d: %s", rec.Code, rec.Body)
			}
			reads, writes := log.snapshot()
			if !slices.Equal(reads, []string{"1", "3", "2"}) {
				t.Errorf("store read %v, want [1 3 2]: the writer first, then the header's order, once each", reads)
			}
			if len(writes) != 0 {
				t.Errorf("a read across tenants wrote under %v", writes)
			}
		})
	}
}

func TestAcross_NoHeaderReadsTheWriterAlone(t *testing.T) {
	for _, c := range acrossCalls {
		srv, log := newRecordingServer(t, false)
		if rec := serveAcross(t, srv, "5", "", c.path, c.body); rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d: %s", c.path, rec.Code, rec.Body)
		}
		if reads, _ := log.snapshot(); !slices.Equal(reads, []string{"5"}) {
			t.Errorf("%s without %s read %v, want [5]", c.path, middleware.ReadTenantsHeader, reads)
		}
	}
}

func TestAcross_EveryItemSaysItsTenant(t *testing.T) {
	srv, _ := newRecordingServer(t, false)
	save := serveAcross(t, srv, "1", "", "/v1/matches.save", `{"match":{"player1_name":"A","player2_name":"B","match_length":5}}`)
	if save.Code != http.StatusOK {
		t.Fatalf("matches.save: %d %s", save.Code, save.Body)
	}
	rec := serveAcross(t, srv, "1", "2", "/v1/across.matchesList", `{}`)
	sc := bufio.NewScanner(rec.Body)
	var tenants []string
	for sc.Scan() {
		var item acrossMatch
		if err := json.Unmarshal(sc.Bytes(), &item); err != nil {
			t.Fatalf("line %q: %v", sc.Text(), err)
		}
		if item.Match == nil {
			t.Fatalf("line %q carries no match", sc.Text())
		}
		tenants = append(tenants, item.Tenant)
	}
	// SQLite ignores the scope, so tenant 2 answers the same row: what counts
	// here is that each line names the tenant it was read under.
	if !slices.Equal(tenants, []string{"1", "2"}) {
		t.Errorf("tenants on the wire %v, want [1 2]", tenants)
	}
}

func TestAcross_MatchesGetNamesItsTenant(t *testing.T) {
	srv, log := newRecordingServer(t, false)
	save := serveAcross(t, srv, "1", "", "/v1/matches.save", `{"match":{"player1_name":"A","player2_name":"B","match_length":5}}`)
	var id idResp
	_ = json.Unmarshal(save.Body.Bytes(), &id)

	rec := serveAcross(t, srv, "1", "2", "/v1/across.matchesGet", `{"tenant":"2","id":`+jsonInt(id.ID)+`}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	var got acrossMatch
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Tenant != "2" || got.Match == nil {
		t.Errorf("across.matchesGet = %s (%v), want tenant 2 and its match", rec.Body, err)
	}
	if reads, _ := log.snapshot(); !slices.Equal(reads, []string{"2"}) {
		t.Errorf("store read %v, want [2]", reads)
	}
}

func TestReadTenantsHeader_Malformed(t *testing.T) {
	others := make([]string, storage.MaxReadTenants)
	for i := range others {
		others[i] = strconv.Itoa(i + 2)
	}
	many := strings.Join(others, ",")
	for _, tc := range []struct {
		name, header string
		single       bool
	}{
		{"name", "alice", false},
		{"empty item", "2,,3", false},
		{"zero", "0", false},
		{"leading zero", "02", false},
		{"too many", many, false},
		{"single-tenant other", "2", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv, log := newRecordingServer(t, tc.single)
			// Any route, a write included: a broken list is refused before
			// the route is chosen.
			for _, path := range []string{"/v1/across.matchesList", "/v1/matches.list", "/v1/matches.save"} {
				rec := serveAcross(t, srv, "1", tc.header, path, `{}`)
				if rec.Code != http.StatusBadRequest {
					t.Errorf("%s with %s %q: status %d, want 400", path, middleware.ReadTenantsHeader, tc.header, rec.Code)
				}
				if body, _ := io.ReadAll(rec.Body); !strings.Contains(string(body), middleware.ReadTenantsHeader) {
					t.Errorf("%s: error does not name the header: %s", path, body)
				}
			}
			if reads, writes := log.snapshot(); len(reads)+len(writes) != 0 {
				t.Errorf("a refused request reached the store: reads %v writes %v", reads, writes)
			}
		})
	}
}

func TestReadTenantsHeader_SingleTenantAcceptsItsOwnTenant(t *testing.T) {
	srv, log := newRecordingServer(t, true)
	if rec := serveAcross(t, srv, "1", "1", "/v1/across.matchesList", `{}`); rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if reads, _ := log.snapshot(); !slices.Equal(reads, []string{"1"}) {
		t.Errorf("SQLite single-tenant read %v, want [1]: the header widens nothing", reads)
	}
}

func jsonInt(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}

// TestAcross_PositionsCarryTheirZobrist: each position read across tenants
// carries the hash it is stored under — the one positions.exists answers to —
// so a caller can join its own rows to it in any tenant.
func TestAcross_PositionsCarryTheirZobrist(t *testing.T) {
	srv, _ := newRecordingServer(t, false)
	if rec := serveAcross(t, srv, "1", "", "/v1/positions.save", `{"position":`+initialPositionJSON(t)+`}`); rec.Code != http.StatusOK {
		t.Fatalf("positions.save: %d %s", rec.Code, rec.Body)
	}
	rec := serveAcross(t, srv, "1", "", "/v1/across.searchFind", `{}`)
	var item acrossPosition
	if err := json.Unmarshal(bytes.TrimSpace(rec.Body.Bytes()), &item); err != nil || item.Zobrist == 0 {
		t.Fatalf("across.searchFind = %s (%v), want a position with its hash", rec.Body, err)
	}
	exists := serveAcross(t, srv, "1", "", "/v1/positions.exists", `{"zobrist":`+strconv.FormatUint(item.Zobrist, 10)+`}`)
	var found existsResp
	if err := json.Unmarshal(exists.Body.Bytes(), &found); err != nil || !found.Found || found.ID != item.Position.ID {
		t.Errorf("positions.exists(%d) = %s, want the position read across", item.Zobrist, exists.Body)
	}
}

// TestAcross_SearchIsBounded: a stream read across tenants has no unbounded
// form — limit 0 reads maxPageSize per tenant — and more is refused.
func TestAcross_SearchIsBounded(t *testing.T) {
	srv, log := newRecordingServer(t, false)
	if rec := serveAcross(t, srv, "1", "2", "/v1/across.searchFind", `{}`); rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	log.mu.Lock()
	limits := slices.Clone(log.limits)
	log.mu.Unlock()
	if !slices.Equal(limits, []int{maxPageSize, maxPageSize}) {
		t.Errorf("Find limits %v, want maxPageSize per tenant", limits)
	}
	if rec := serveAcross(t, srv, "1", "", "/v1/across.searchFind", `{"limit":`+strconv.Itoa(maxPageSize+1)+`}`); rec.Code != http.StatusBadRequest {
		t.Errorf("limit over maxPageSize: status %d, want 400", rec.Code)
	}
}

// TestAcross_MatchMovePositionsAndAnalyses: a coach reads a student's match
// move by move, each position with its hash, then the analyses by id — both
// in the named tenant only.
func TestAcross_MatchMovePositionsAndAnalyses(t *testing.T) {
	srv, log := newRecordingServer(t, false)
	// No such match: 404 is the store's answer, and it was asked of tenant 2.
	rec := serveAcross(t, srv, "1", "2", "/v1/across.matchMovePositions", `{"tenant":"2","matchId":1}`)
	if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
		t.Fatalf("across.matchMovePositions: %d %s", rec.Code, rec.Body)
	}
	rec = serveAcross(t, srv, "1", "2", "/v1/across.analysesLoadByIds", `{"tenant":"2","ids":[1,2]}`)
	var got acrossAnalyses
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Tenant != "2" {
		t.Errorf("across.analysesLoadByIds = %d %s", rec.Code, rec.Body)
	}
	if reads, _ := log.snapshot(); !slices.Equal(reads, []string{"2", "2"}) {
		t.Errorf("store read %v, want tenant 2 twice", reads)
	}
	for _, c := range []struct{ path, body string }{
		{"/v1/across.matchMovePositions", `{"tenant":"9","matchId":1}`},
		{"/v1/across.analysesLoadByIds", `{"tenant":"9","ids":[1]}`},
	} {
		srv, log := newRecordingServer(t, false)
		if rec := serveAcross(t, srv, "1", "2", c.path, c.body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s, unlisted tenant: status %d, want 400", c.path, rec.Code)
		}
		if reads, _ := log.snapshot(); len(reads) != 0 {
			t.Errorf("%s: an unlisted tenant reached the store: %v", c.path, reads)
		}
	}
}
