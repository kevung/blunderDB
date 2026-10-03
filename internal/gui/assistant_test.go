package gui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
)

func TestAssistantKeyLivesInTheKeyring(t *testing.T) {
	keyring.MockInit()
	a := NewApp(database.NewDatabase())
	if a.AssistantHasKey("groq") {
		t.Fatal("no key before one is set")
	}
	if err := a.AssistantSetKey("groq", "  secret "); err != nil {
		t.Fatal(err)
	}
	if got, _ := keyring.Get(keyringService, "groq"); got != "secret" || !a.AssistantHasKey("groq") {
		t.Fatalf("keyring holds %q", got)
	}
	if err := a.AssistantSetKey("groq", ""); err != nil || a.AssistantHasKey("groq") {
		t.Fatalf("an empty key must remove it (err %v)", err)
	}
	if err := a.AssistantSetKey("groq", ""); err != nil {
		t.Fatalf("removing an absent key: %v", err)
	}
}

func TestAssistantAskSendsTheKeyringKeyAndReadsTheOpenFile(t *testing.T) {
	keyring.MockInit()
	var auth string
	llm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		var body struct {
			Messages []map[string]any `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		msg := map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"id": "1", "type": "function",
			"function": map[string]any{"name": "database_overview", "arguments": "{}"}}}}
		if len(body.Messages) > 2 {
			msg = map[string]any{"role": "assistant", "content": "ok"}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": msg}}})
	}))
	defer llm.Close()

	db := database.NewDatabase()
	a := NewApp(db)
	t.Cleanup(a.stopAssistant)
	path, err := a.PrepareDemoDatabase()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	if err := db.OpenDatabase(path); err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := a.AssistantSetKey("other", "k"); err != nil {
		t.Fatal(err)
	}
	turn, err := a.AssistantAsk(AssistantRequest{Preset: "other", BaseURL: llm.URL, Model: "m", Text: "bonjour"})
	if err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer k" {
		t.Fatalf("Authorization %q", auth)
	}
	if len(turn.Entries) != 3 || turn.Entries[1].Kind != "tool" || turn.Entries[2].Text != "ok" {
		t.Fatalf("entries %+v", turn.Entries)
	}
}

func TestAssistantRefusesARemoteAddressWithoutItsConsent(t *testing.T) {
	keyring.MockInit()
	a := NewApp(database.NewDatabase())
	t.Cleanup(a.stopAssistant)
	accepted := "https://api.groq.com/openai/v1"
	a.assistant.consent = func() string { return accepted }
	// Consent is tied to the address: the preset's own one is accepted, a
	// changed one is not, whatever preset carries it.
	for _, req := range []AssistantRequest{
		{Preset: "ollama", BaseURL: "http://192.168.1.5:11434/v1", Model: "m", Text: "x"},
		{Preset: "groq", BaseURL: "https://evil.example/v1", Model: "m", Text: "x"},
	} {
		if _, err := a.AssistantAsk(req); err == nil || !strings.Contains(err.Error(), "privacy") {
			t.Fatalf("%+v: want a privacy refusal, got %v", req, err)
		}
	}
	a.assistant.consent = nil
	if _, err := a.AssistantAsk(AssistantRequest{Preset: "groq", Text: "x"}); err == nil || !strings.Contains(err.Error(), "privacy") {
		t.Fatalf("no consent recorded: got %v", err)
	}
}
