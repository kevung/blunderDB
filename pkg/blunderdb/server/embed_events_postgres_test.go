//go:build postgres

// Needs Docker (testcontainers):
//
//	go test -tags postgres ./pkg/blunderdb/server/ -run TestBootstrapRelaysGestures -v
package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// transportBackends counts the connections of every event transport on the database.
func transportBackends(t *testing.T, conn *pgx.Conn) int {
	t.Helper()
	var n int
	if err := conn.QueryRow(context.Background(),
		`SELECT count(*) FROM pg_stat_activity WHERE application_name LIKE 'blunderdb-events-%' OR application_name LIKE 'blunderdb-notify-%'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func bootstrapOn(t *testing.T, dsn string) (*httptest.Server, func()) {
	t.Helper()
	h, closer, err := Bootstrap(context.Background(), Config{Backend: "postgres", DSN: dsn, Transcription: true})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	return httptest.NewServer(h), func() { _ = closer.Close() }
}

func post(t *testing.T, ts *httptest.Server, path string, rev int64, body any) []byte {
	t.Helper()
	buf, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+path, bytes.NewReader(buf))
	req.Header.Set(TenantHeader, "1")
	req.Header.Set("Content-Type", "application/json")
	if rev != 0 {
		req.Header.Set("If-Match", `"`+strconv.FormatInt(rev, 10)+`"`)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out bytes.Buffer
	_, _ = out.ReadFrom(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s: %d %s", path, resp.StatusCode, out.String())
	}
	return out.Bytes()
}

// Two embedded engines over one PostgreSQL database: a transcription gesture
// made on one reaches a subscriber of the other, and closing the engine
// leaves no connection of its transport behind.
func TestBootstrapRelaysGestures(t *testing.T) {
	ctx := context.Background()
	container, err := tcpg.Run(ctx, "postgres:16-alpine",
		tcpg.WithDatabase("blunderdb"), tcpg.WithUsername("test"), tcpg.WithPassword("test"),
		tcpg.BasicWaitStrategies())
	if err != nil {
		t.Skipf("postgres container unavailable (Docker required): %v", err)
	}
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(container) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })

	tsA, closeA := bootstrapOn(t, dsn)
	defer tsA.Close()
	defer closeA()
	tsB, closeB := bootstrapOn(t, dsn)
	defer tsB.Close()
	closedB := false
	defer func() {
		if !closedB {
			closeB()
		}
	}()

	var draft struct {
		ID       int64 `json:"id"`
		Revision int64 `json:"revision"`
	}
	if err := json.Unmarshal(post(t, tsA, "/v1/transcriptions.create", 0,
		map[string]any{"header": map[string]any{"MatchLength": 7}}), &draft); err != nil {
		t.Fatal(err)
	}

	sctx, cancel := context.WithCancel(ctx)
	defer cancel()
	req, _ := http.NewRequestWithContext(sctx, http.MethodGet,
		tsB.URL+"/v1/events?transcription="+strconv.FormatInt(draft.ID, 10), nil)
	req.Header.Set(TenantHeader, "1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("subscribe: %v", err)
	}
	defer resp.Body.Close()
	events := make(chan string, 16)
	subscribed := make(chan struct{})
	go func() {
		sc := bufio.NewScanner(resp.Body)
		var once bool
		for sc.Scan() {
			line := sc.Text()
			if strings.HasPrefix(line, ": subscribed") && !once {
				once = true
				close(subscribed)
			}
			if strings.HasPrefix(line, "event: ") {
				events <- strings.TrimPrefix(line, "event: ")
			}
		}
	}()
	select {
	case <-subscribed:
	case <-time.After(10 * time.Second):
		t.Fatal("subscription not registered")
	}

	post(t, tsA, "/v1/transcriptions.abandon", draft.Revision, map[string]any{"id": draft.ID})
	deadline := time.After(10 * time.Second)
	for {
		select {
		case ev := <-events:
			if ev == "transcription" {
				closedB = true
				cancel()
				both := transportBackends(t, conn)
				if both == 0 {
					t.Fatal("no transport connection while both instances run")
				}
				closeB()
				deadline := time.Now().Add(10 * time.Second)
				for transportBackends(t, conn) >= both {
					if time.Now().After(deadline) {
						t.Fatalf("closing B left its transport connections open (%d of %d)", both, both)
					}
					time.Sleep(50 * time.Millisecond)
				}
				return
			}
		case <-deadline:
			t.Fatal("instance B never heard the gesture made on A")
		}
	}
}
