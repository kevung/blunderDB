package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Entry kinds of a turn, in the order the panel shows them.
const (
	KindUser  = "user"  // the sentence the user wrote
	KindModel = "model" // the model's free text, shown marked as such
	KindTool  = "tool"  // a tool the assistant ran, and what it returned
	KindError = "error" // a tool refused or failed
)

// Entry is one line of the conversation as the panel shows it. A tool entry
// carries facts the database returned; a model entry is free text the model
// wrote, never a measurement — Free marks it so the panel can say so.
type Entry struct {
	Kind   string `json:"kind"`
	Text   string `json:"text,omitempty"`
	Tool   string `json:"tool,omitempty"`
	Args   string `json:"args,omitempty"`
	Result string `json:"result,omitempty"`
	Free   bool   `json:"free,omitempty"`
}

// Pending is a write the model prepared and the user has not yet confirmed.
type Pending struct {
	Tool  string `json:"tool"`
	Title string `json:"title"`
	Args  string `json:"args"`
}

// Turn is what one Ask or Confirm produced: the new entries, and the write
// waiting for the user's answer, if any (the turn is then suspended).
type Turn struct {
	Entries []Entry  `json:"entries"`
	Pending *Pending `json:"pending,omitempty"`
}

// maxSteps bounds the model round trips of one sentence: a model looping on a
// tool stops here rather than running until the user's patience does.
const maxSteps = 8

// maxToolResult bounds what one tool result puts in the model's context.
const maxToolResult = 12000

// SystemPrompt is the assistant's standing instruction. It names the tools by
// role, not by schema: the schemas travel with the request.
const SystemPrompt = `You are the assistant built into blunderDB, a backgammon blunder database.
You answer with the database's tools, never from memory: every number you give comes from a tool result.
When the user asks to see, find or list positions, build a query in the search_positions grammar,
check it with search_positions, then call open_view with that query so the positions appear on screen.
Use player_stats, recurring_errors, get_position and explain_error to answer questions about play.
A tool that changes the database is only run after the user confirms it; say what it will do.
Answer briefly, in the language the user wrote in.`

// Session is one conversation with a provider, over an MCP client session to
// the application's own tools: the assistant is a client of those tools, not a
// second way into the data.
type Session struct {
	provider Provider
	cs       *sdk.ClientSession
	tools    []toolDef
	writes   map[string]string // tool name → title, for the tools that write

	mu      sync.Mutex
	msgs    []Message
	pending *pendingCall
}

type pendingCall struct {
	call ToolCall
	rest []ToolCall
}

// Connect runs srv in process and returns a client session to it.
func Connect(ctx context.Context, srv *sdk.Server) (*sdk.ClientSession, error) {
	ct, st := sdk.NewInMemoryTransports()
	if _, err := srv.Connect(ctx, st, nil); err != nil {
		return nil, err
	}
	return sdk.NewClient(&sdk.Implementation{Name: "blunderdb-assistant", Version: "1"}, nil).Connect(ctx, ct, nil)
}

// NewSession lists the tools of cs and starts an empty conversation.
func NewSession(ctx context.Context, p Provider, cs *sdk.ClientSession) (*Session, error) {
	s := &Session{provider: p, cs: cs, writes: map[string]string{},
		msgs: []Message{{Role: "system", Content: SystemPrompt}}}
	for t, err := range cs.Tools(ctx, nil) {
		if err != nil {
			return nil, err
		}
		var d toolDef
		d.Type = "function"
		d.Function.Name = t.Name
		d.Function.Description = t.Description
		d.Function.Parameters = t.InputSchema
		s.tools = append(s.tools, d)
		if t.Annotations == nil || !t.Annotations.ReadOnlyHint {
			title := t.Title
			if title == "" {
				title = t.Name
			}
			s.writes[t.Name] = title
		}
	}
	return s, nil
}

// Close ends the MCP session.
func (s *Session) Close() error { return s.cs.Close() }

// ErrPending refuses a new sentence while a write awaits its confirmation.
var ErrPending = errors.New("a change awaits your confirmation")

// Ask sends one sentence and runs the tools the model asks for, until it
// answers in text, prepares a write, or reaches the step bound.
func (s *Session) Ask(ctx context.Context, text string) (Turn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending != nil {
		return Turn{}, ErrPending
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return Turn{}, errors.New("empty sentence")
	}
	s.msgs = append(s.msgs, Message{Role: "user", Content: text})
	turn := Turn{Entries: []Entry{{Kind: KindUser, Text: text}}}
	return s.loop(ctx, turn)
}

// Confirm answers the pending write: run it, or tell the model it was
// declined, then let the conversation go on.
func (s *Session) Confirm(ctx context.Context, accept bool) (Turn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending == nil {
		return Turn{}, errors.New("nothing awaits confirmation")
	}
	p := s.pending
	s.pending = nil
	var turn Turn
	if accept {
		s.runCall(ctx, p.call, &turn)
	} else {
		s.msgs = append(s.msgs, Message{Role: "tool", ToolCallID: p.call.ID,
			Content: "The user declined this change; nothing was written."})
		turn.Entries = append(turn.Entries, Entry{Kind: KindError, Tool: p.call.Function.Name, Text: "declined"})
	}
	if s.runCalls(ctx, p.rest, &turn) {
		return turn, nil
	}
	return s.loop(ctx, turn)
}

func (s *Session) loop(ctx context.Context, turn Turn) (Turn, error) {
	for range maxSteps {
		m, err := s.provider.complete(ctx, s.msgs, s.tools)
		if err != nil {
			return turn, err
		}
		s.msgs = append(s.msgs, m)
		if txt := strings.TrimSpace(m.Content); txt != "" {
			turn.Entries = append(turn.Entries, Entry{Kind: KindModel, Text: txt, Free: true})
		}
		if len(m.ToolCalls) == 0 {
			return turn, nil
		}
		if s.runCalls(ctx, m.ToolCalls, &turn) {
			return turn, nil
		}
	}
	turn.Entries = append(turn.Entries, Entry{Kind: KindError, Text: "step limit reached"})
	return turn, nil
}

// runCalls runs calls in order and reports whether it stopped on a write,
// which it leaves pending with the calls after it.
func (s *Session) runCalls(ctx context.Context, calls []ToolCall, turn *Turn) bool {
	for i, c := range calls {
		if title, ok := s.writes[c.Function.Name]; ok {
			s.pending = &pendingCall{call: c, rest: calls[i+1:]}
			turn.Pending = &Pending{Tool: c.Function.Name, Title: title, Args: c.Function.Arguments}
			return true
		}
		s.runCall(ctx, c, turn)
	}
	return false
}

func (s *Session) runCall(ctx context.Context, c ToolCall, turn *Turn) {
	name := c.Function.Name
	var args map[string]any
	if a := strings.TrimSpace(c.Function.Arguments); a != "" && a != "null" {
		if err := json.Unmarshal([]byte(a), &args); err != nil {
			s.toolResult(c, turn, "", fmt.Sprintf("arguments are not a JSON object: %v", err))
			return
		}
	}
	res, err := s.cs.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		s.toolResult(c, turn, "", err.Error())
		return
	}
	if res.IsError {
		s.toolResult(c, turn, "", contentText(res))
		return
	}
	out := contentText(res)
	if res.StructuredContent != nil {
		if raw, err := json.Marshal(res.StructuredContent); err == nil {
			out = string(raw)
		}
	}
	s.toolResult(c, turn, out, "")
}

func (s *Session) toolResult(c ToolCall, turn *Turn, out, failure string) {
	content := out
	e := Entry{Kind: KindTool, Tool: c.Function.Name, Args: c.Function.Arguments, Result: truncate(out, 400)}
	if failure != "" {
		content = "error: " + failure
		e = Entry{Kind: KindError, Tool: c.Function.Name, Args: c.Function.Arguments, Text: failure}
	}
	s.msgs = append(s.msgs, Message{Role: "tool", ToolCallID: c.ID, Content: truncate(content, maxToolResult)})
	turn.Entries = append(turn.Entries, e)
}

func contentText(res *sdk.CallToolResult) string {
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*sdk.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	// Cut on a rune boundary: the panel and the model both read UTF-8.
	for n > 0 && n < len(s) && (s[n]&0xC0) == 0x80 {
		n--
	}
	return s[:n] + "…"
}
