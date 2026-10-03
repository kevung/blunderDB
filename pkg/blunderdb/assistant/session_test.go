package assistant_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/assistant"
	"github.com/kevung/blunderdb/pkg/blunderdb/mcp"
)

type display struct{ views []string }

func (d *display) OpenView(name, query string) error {
	d.views = append(d.views, name+"|"+query)
	return nil
}
func (d *display) ShowPosition(int64) error { return nil }

// scripted is an OpenAI-compatible endpoint that answers from a script, one
// reply per request, and records what it was sent.
type scripted struct {
	mu      sync.Mutex
	replies []map[string]any
	got     []map[string]any
}

func (s *scripted) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	s.got = append(s.got, body)
	if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer k" || len(s.replies) == 0 {
		http.Error(w, "unexpected", http.StatusBadRequest)
		return
	}
	m := s.replies[0]
	s.replies = s.replies[1:]
	_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": m}}})
}

func toolCall(id, name string, args map[string]any) map[string]any {
	raw, _ := json.Marshal(args)
	return map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{
		"id": id, "type": "function", "function": map[string]any{"name": name, "arguments": string(raw)}}}}
}

func session(t *testing.T, write bool, s *scripted, d *display) *assistant.Session {
	t.Helper()
	llm := httptest.NewServer(s)
	t.Cleanup(llm.Close)
	ctx := context.Background()
	srv := mcp.NewServer(demoEngine(t), mcp.Options{Tenant: "1", Write: write,
		Extensions: []mcp.Extension{mcp.DisplayTools(d)}})
	cs, err := assistant.Connect(ctx, srv)
	if err != nil {
		t.Fatal(err)
	}
	sess, err := assistant.NewSession(ctx, assistant.Provider{BaseURL: llm.URL + "/v1", Model: "m", APIKey: "k"}, cs)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sess.Close() })
	return sess
}

func TestAskRunsReadToolsAndMarksFreeText(t *testing.T) {
	s := &scripted{replies: []map[string]any{
		toolCall("1", "search_positions", map[string]any{"query": "cube E>80"}),
		toolCall("2", "open_view", map[string]any{"name": "x", "query": "cube E>80"}),
		{"role": "assistant", "content": "Voici vos erreurs de videau."},
	}}
	d := &display{}
	turn, err := session(t, false, s, d).Ask(context.Background(), "mes erreurs de videau")
	if err != nil {
		t.Fatal(err)
	}
	kinds := []string{}
	for _, e := range turn.Entries {
		kinds = append(kinds, e.Kind)
	}
	if strings.Join(kinds, ",") != "user,tool,tool,model" || turn.Pending != nil {
		t.Fatalf("entries %v pending %v", turn.Entries, turn.Pending)
	}
	if last := turn.Entries[3]; !last.Free {
		t.Fatal("the model's text must be marked as free text")
	}
	if len(d.views) != 1 || d.views[0] != "x|cube E>80" {
		t.Fatalf("views %v", d.views)
	}
	// The read-only server offers no write tool to the model.
	for _, tool := range s.got[0]["tools"].([]any) {
		if name := tool.(map[string]any)["function"].(map[string]any)["name"]; name == "create_collection" {
			t.Fatal("a write tool was offered on a read-only server")
		}
	}
	// The tool result reached the model.
	msgs := s.got[1]["messages"].([]any)
	if last := msgs[len(msgs)-1].(map[string]any); last["role"] != "tool" || !strings.Contains(last["content"].(string), "canonical") {
		t.Fatalf("tool result not sent back: %v", last)
	}
}

func TestWriteWaitsForConfirmation(t *testing.T) {
	s := &scripted{replies: []map[string]any{
		toolCall("1", "create_collection", map[string]any{"name": "Videau"}),
		{"role": "assistant", "content": "Créée."},
		toolCall("2", "create_collection", map[string]any{"name": "Autre"}),
		{"role": "assistant", "content": "Bien, rien n'est créé."},
	}}
	sess := session(t, true, s, &display{})
	ctx := context.Background()
	turn, err := sess.Ask(ctx, "crée une collection Videau")
	if err != nil {
		t.Fatal(err)
	}
	if turn.Pending == nil || turn.Pending.Tool != "create_collection" {
		t.Fatalf("want a pending write, got %+v", turn)
	}
	if _, err := sess.Ask(ctx, "autre chose"); !errors.Is(err, assistant.ErrPending) {
		t.Fatalf("a sentence during a pending write: %v", err)
	}
	turn, err = sess.Confirm(ctx, true)
	if err != nil || len(turn.Entries) != 2 || turn.Entries[0].Kind != assistant.KindTool {
		t.Fatalf("confirm: %v %+v", err, turn.Entries)
	}
	if _, err := sess.Ask(ctx, "et une autre"); err != nil {
		t.Fatal(err)
	}
	turn, err = sess.Confirm(ctx, false)
	if err != nil || turn.Entries[0].Kind != assistant.KindError || turn.Entries[0].Text != "declined" {
		t.Fatalf("decline: %v %+v", err, turn.Entries)
	}
}

func TestUnreachableProvider(t *testing.T) {
	ctx := context.Background()
	srv := mcp.NewServer(demoEngine(t), mcp.Options{Tenant: "1"})
	cs, err := assistant.Connect(ctx, srv)
	if err != nil {
		t.Fatal(err)
	}
	sess, err := assistant.NewSession(ctx, assistant.Provider{BaseURL: "http://127.0.0.1:1/v1", Model: "m"}, cs)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	if _, err := sess.Ask(ctx, "bonjour"); err == nil || !strings.Contains(err.Error(), "unreachable") {
		t.Fatalf("want unreachable, got %v", err)
	}
}

func TestPresets(t *testing.T) {
	if p := assistant.PresetByID("nope"); p.ID != assistant.DefaultPreset || p.Remote {
		t.Fatalf("default preset must be the local one: %+v", p)
	}
	for _, p := range assistant.Presets() {
		if p.ID != "ollama" && !p.Remote {
			t.Fatalf("%s: only the local preset may skip the privacy warning", p.ID)
		}
	}
}

func TestIsRemoteFollowsTheAddressNotThePreset(t *testing.T) {
	for url, want := range map[string]bool{
		"http://localhost:11434/v1": false, "http://127.0.0.1:8080/v1": false, "http://[::1]:1/v1": false,
		"https://api.groq.com/openai/v1": true, "http://192.168.1.20:11434/v1": true, "": true, "::nope": true,
	} {
		if got := assistant.IsRemote(url); got != want {
			t.Errorf("IsRemote(%q) = %v, want %v", url, got, want)
		}
	}
}

func TestProviderErrorNeverRepeatsAKey(t *testing.T) {
	llm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"Incorrect API key my-secret-key-1 (also sk-proj-abcdefghij1234, `+r.Header.Get("Authorization")+`)"}`, http.StatusUnauthorized)
	}))
	defer llm.Close()
	ctx := context.Background()
	cs, err := assistant.Connect(ctx, mcp.NewServer(demoEngine(t), mcp.Options{Tenant: "1"}))
	if err != nil {
		t.Fatal(err)
	}
	sess, err := assistant.NewSession(ctx, assistant.Provider{BaseURL: llm.URL, Model: "m", APIKey: "my-secret-key-1"}, cs)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	_, err = sess.Ask(ctx, "bonjour")
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("want the HTTP status, got %v", err)
	}
	for _, leak := range []string{"my-secret-key-1", "sk-proj-abcdefghij1234", "Bearer"} {
		if strings.Contains(err.Error(), leak) {
			t.Fatalf("error repeats %q: %v", leak, err)
		}
	}
}
