package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Preset is a provider the settings offer by name: an OpenAI-compatible
// endpoint and a starting model. Remote is IsRemote of its address, given for
// the settings' first display; what decides is the address actually used,
// which the user may change (IsRemote).
type Preset struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	BaseURL  string `json:"baseURL"`
	Model    string `json:"model"`
	Remote   bool   `json:"remote"`
	NeedsKey bool   `json:"needsKey"`
}

// DefaultPreset is the provider the assistant starts on: a local Ollama, so
// that nothing leaves the machine until the user picks otherwise.
const DefaultPreset = "ollama"

// Presets lists the providers in the order the settings show them. Every one
// speaks the OpenAI chat-completions protocol with tool calls; "other" is any
// further endpoint that does.
func Presets() []Preset {
	ps := []Preset{
		// The Ollama model is a starting point, not a measured recommendation:
		// a recommendation comes from the assistantbench score.
		{ID: "ollama", Name: "Ollama", BaseURL: "http://localhost:11434/v1", Model: "qwen2.5:7b"},
		{ID: "groq", Name: "Groq", BaseURL: "https://api.groq.com/openai/v1", Model: "llama-3.3-70b-versatile", NeedsKey: true},
		{ID: "openrouter", Name: "OpenRouter", BaseURL: "https://openrouter.ai/api/v1", Model: "openai/gpt-4o-mini", NeedsKey: true},
		{ID: "gemini", Name: "Gemini", BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai", Model: "gemini-2.0-flash", NeedsKey: true},
		{ID: "anthropic", Name: "Anthropic", BaseURL: "https://api.anthropic.com/v1", Model: "claude-haiku-4-5", NeedsKey: true},
		{ID: "other", Name: "Other (OpenAI-compatible)"},
	}
	for i := range ps {
		ps[i].Remote = IsRemote(ps[i].BaseURL)
	}
	return ps
}

// IsRemote reports whether an endpoint is off this machine: any host that is
// not loopback, whatever preset it came from. An address that cannot be read
// counts as remote — the privacy warning errs on the side of asking.
func IsRemote(baseURL string) bool {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || u.Hostname() == "" {
		return true
	}
	host := u.Hostname()
	if strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".localhost") {
		return false
	}
	ip := net.ParseIP(host)
	return ip == nil || !ip.IsLoopback()
}

// keyLike matches what looks like a credential in a provider's message: the
// known key prefixes, a bearer token, and any long unbroken token.
var keyLike = regexp.MustCompile(`(?i)(bearer\s+\S+|\b(sk|gsk|pk|rk|xai)[-_][A-Za-z0-9_\-]{8,}|AIza[A-Za-z0-9_\-]{10,}|[A-Za-z0-9_\-]{32,})`)

// redact masks the key the request carried and anything that looks like a
// key, so an error shown in the panel or a log never repeats a credential a
// provider echoed back.
func redact(s, key string) string {
	if key != "" {
		s = strings.ReplaceAll(s, key, "[redacted]")
	}
	return keyLike.ReplaceAllString(s, "[redacted]")
}

// PresetByID returns the preset with that id, the default one when unknown.
func PresetByID(id string) Preset {
	ps := Presets()
	for _, p := range ps {
		if p.ID == id {
			return p
		}
	}
	return ps[0]
}

// Provider is the endpoint a conversation talks to.
type Provider struct {
	BaseURL string
	Model   string
	APIKey  string
	// HTTP is the client used; nil means one with a generous timeout, as a
	// local model on a laptop may take a minute to answer.
	HTTP *http.Client
}

// Message is one chat-completions message.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall is a model's request to run one tool.
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type toolDef struct {
	Type     string `json:"type"`
	Function struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Parameters  any    `json:"parameters"`
	} `json:"function"`
}

// ErrNoProvider says the endpoint could not be reached at all.
var ErrNoProvider = errors.New("the assistant's provider is unreachable")

func (p Provider) complete(ctx context.Context, msgs []Message, tools []toolDef) (Message, error) {
	base := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	if base == "" || strings.TrimSpace(p.Model) == "" {
		return Message{}, errors.New("the assistant has no provider address or model")
	}
	body, err := json.Marshal(map[string]any{"model": p.Model, "messages": msgs, "tools": tools, "temperature": 0})
	if err != nil {
		return Message{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Message{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}
	client := p.HTTP
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Minute}
	}
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return Message{}, ctx.Err()
		}
		return Message{}, fmt.Errorf("%w: %s", ErrNoProvider, redact(err.Error(), p.APIKey))
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return Message{}, err
	}
	if resp.StatusCode/100 != 2 {
		return Message{}, fmt.Errorf("provider: HTTP %d: %s", resp.StatusCode, redact(firstLine(raw), p.APIKey))
	}
	var out struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return Message{}, fmt.Errorf("provider: unreadable answer: %w", err)
	}
	if len(out.Choices) == 0 {
		return Message{}, errors.New("provider: empty answer")
	}
	m := out.Choices[0].Message
	m.Role = "assistant"
	return m, nil
}

func firstLine(b []byte) string {
	s := strings.TrimSpace(string(b))
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 300 {
		s = s[:300]
	}
	return s
}
