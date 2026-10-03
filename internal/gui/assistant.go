package gui

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/zalando/go-keyring"

	"github.com/kevung/blunderdb/pkg/blunderdb/assistant"
)

// keyringService names blunderDB's entries in the system keyring; the user
// of each entry is the provider preset. An API key never reaches the
// database nor the settings file.
const keyringService = "blunderDB assistant"

// AssistantRequest is one sentence for the in-app assistant, with the
// provider it goes to. The key is not in it: the Go side reads it from the
// keyring.
type AssistantRequest struct {
	Preset  string `json:"preset"`
	BaseURL string `json:"baseURL"`
	Model   string `json:"model"`
	// Write offers the tools that change the database — each call still
	// waits for the user's confirmation. The same switch as the MCP server's.
	Write bool   `json:"write"`
	Text  string `json:"text"`
}

// assistantState is the conversation the panel holds, rebuilt when the
// provider or the write switch changes.
type assistantState struct {
	mu          sync.Mutex
	sess        *assistant.Session
	fingerprint string

	// phraseMu guards phrase, the sentence being answered: the views the
	// assistant opens are named after it.
	phraseMu sync.Mutex
	phrase   string

	cancelMu sync.Mutex
	cancel   context.CancelFunc
}

func (s *assistantState) currentPhrase() string {
	s.phraseMu.Lock()
	defer s.phraseMu.Unlock()
	return s.phrase
}

// AssistantPresets lists the providers the settings offer.
func (a *App) AssistantPresets() []assistant.Preset {
	return assistant.Presets()
}

func (a *App) sessionFor(ctx context.Context, req AssistantRequest) (*assistant.Session, error) {
	preset := assistant.PresetByID(req.Preset)
	base, model := strings.TrimSpace(req.BaseURL), strings.TrimSpace(req.Model)
	if base == "" {
		base = preset.BaseURL
	}
	if model == "" {
		model = preset.Model
	}
	fp := strings.Join([]string{preset.ID, base, model, boolStr(req.Write)}, "|")
	if a.assistant.sess != nil && a.assistant.fingerprint == fp {
		return a.assistant.sess, nil
	}
	if a.assistant.sess != nil {
		_ = a.assistant.sess.Close()
		a.assistant.sess = nil
	}
	key, err := keyring.Get(keyringService, preset.ID)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return nil, newGUIError(CodeInternal, "keyring: %v", err)
	}
	srv := a.newMCPServer(req.Write, guiDisplay{a: a, viewName: a.assistant.currentPhrase})
	cs, err := assistant.Connect(ctx, srv)
	if err != nil {
		return nil, newGUIError(CodeInternal, err.Error())
	}
	sess, err := assistant.NewSession(ctx, assistant.Provider{BaseURL: base, Model: model, APIKey: key}, cs)
	if err != nil {
		_ = cs.Close()
		return nil, newGUIError(CodeInternal, err.Error())
	}
	a.assistant.sess, a.assistant.fingerprint = sess, fp
	return sess, nil
}

func boolStr(b bool) string {
	if b {
		return "w"
	}
	return "r"
}

func (a *App) assistantContext() (context.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	a.assistant.cancelMu.Lock()
	a.assistant.cancel = cancel
	a.assistant.cancelMu.Unlock()
	return ctx, func() {
		a.assistant.cancelMu.Lock()
		a.assistant.cancel = nil
		a.assistant.cancelMu.Unlock()
		cancel()
	}
}

func turnError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, assistant.ErrPending):
		return newGUIError(CodeConflict, err.Error())
	case errors.Is(err, context.Canceled):
		return newGUIError(CodeConflict, "cancelled")
	default:
		return newGUIError(CodeInternal, err.Error())
	}
}

// AssistantAsk sends one sentence. The tools read freely; a write comes back
// as the turn's Pending, run only once AssistantConfirm accepts it.
func (a *App) AssistantAsk(req AssistantRequest) (assistant.Turn, error) {
	a.assistant.mu.Lock()
	defer a.assistant.mu.Unlock()
	ctx, done := a.assistantContext()
	defer done()
	sess, err := a.sessionFor(ctx, req)
	if err != nil {
		return assistant.Turn{}, err
	}
	a.assistant.phraseMu.Lock()
	a.assistant.phrase = req.Text
	a.assistant.phraseMu.Unlock()
	turn, err := sess.Ask(ctx, req.Text)
	return turn, turnError(err)
}

// AssistantConfirm answers the write the last turn prepared.
func (a *App) AssistantConfirm(accept bool) (assistant.Turn, error) {
	a.assistant.mu.Lock()
	defer a.assistant.mu.Unlock()
	if a.assistant.sess == nil {
		return assistant.Turn{}, newGUIError(CodeInvalid, "no conversation")
	}
	ctx, done := a.assistantContext()
	defer done()
	turn, err := a.assistant.sess.Confirm(ctx, accept)
	return turn, turnError(err)
}

// AssistantCancel stops the sentence being answered.
func (a *App) AssistantCancel() {
	a.assistant.cancelMu.Lock()
	defer a.assistant.cancelMu.Unlock()
	if a.assistant.cancel != nil {
		a.assistant.cancel()
	}
}

// AssistantReset forgets the conversation.
func (a *App) AssistantReset() {
	a.AssistantCancel()
	a.assistant.mu.Lock()
	defer a.assistant.mu.Unlock()
	if a.assistant.sess != nil {
		_ = a.assistant.sess.Close()
		a.assistant.sess = nil
		a.assistant.fingerprint = ""
	}
}

// AssistantSetKey stores a provider's API key in the system keyring; an
// empty key removes it.
func (a *App) AssistantSetKey(preset, key string) error {
	preset = assistant.PresetByID(preset).ID
	key = strings.TrimSpace(key)
	var err error
	if key == "" {
		err = keyring.Delete(keyringService, preset)
		if errors.Is(err, keyring.ErrNotFound) {
			err = nil
		}
	} else {
		err = keyring.Set(keyringService, preset, key)
	}
	if err != nil {
		return newGUIError(CodeInternal, "keyring: %v", err)
	}
	// The conversation holds the old key: the next sentence starts afresh.
	a.AssistantReset()
	return nil
}

// AssistantHasKey reports whether the keyring holds a key for the preset.
func (a *App) AssistantHasKey(preset string) bool {
	_, err := keyring.Get(keyringService, assistant.PresetByID(preset).ID)
	return err == nil
}

// stopAssistant ends the conversation at shutdown.
func (a *App) stopAssistant() {
	a.AssistantReset()
}
