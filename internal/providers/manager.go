package providers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/model"
)

// ErrNotConfigured means no provider is active, or the active one is missing something it needs.
var ErrNotConfigured = errors.New("no model configured")

// Manager is the one place the app asks "which model, and is it ready?". It reads the provider
// file and the keychain, and builds models on demand so a change in Settings applies to the next
// request.
type Manager struct {
	store  *Store
	keys   Keyring
	client *http.Client // nil for the default; tests point it at a local server

	mu         sync.Mutex
	cfg        Config
	configured bool
}

// NewManager loads the provider file. A missing or unreadable file is reported but the Manager
// still works, as "not configured".
func NewManager(store *Store, keys Keyring, client *http.Client) (*Manager, error) {
	m := &Manager{store: store, keys: keys, client: client}
	return m, m.Reload()
}

// Reload re-reads the provider file and recomputes whether the active provider is ready. The UI
// asks Configured every frame, so the keychain is only read here.
func (m *Manager) Reload() error {
	cfg, err := m.store.Load()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg, m.configured = cfg, err == nil && m.ready(cfg)
	return err
}

// ready is true when the active provider has everything a request needs.
func (m *Manager) ready(cfg Config) bool {
	r, ok := cfg.Resolve(cfg.Active)
	if !ok || r.Model == "" || (r.BaseURL == "" && r.Protocol == OpenAI) {
		return false
	}
	if !r.Preset.NeedsKey() {
		return true
	}
	k, err := m.keys.Get(r.Preset.ID)
	return err == nil && k != ""
}

// Config is the current provider file contents.
func (m *Manager) Config() Config {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cfg
}

// Configured reports whether a model is ready to use.
func (m *Manager) Configured() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.configured
}

// Save stores a new provider file, then reloads.
func (m *Manager) Save(cfg Config) error {
	if err := m.store.Save(cfg); err != nil {
		return err
	}
	return m.Reload()
}

// SetKey stores (or, when empty, removes) the key of one provider, then reloads.
func (m *Manager) SetKey(id, key string) error {
	var err error
	if key == "" {
		err = m.keys.Delete(id)
	} else {
		err = m.keys.Set(id, key)
	}
	if err != nil {
		return fmt.Errorf("keychain: %w", err)
	}
	return m.Reload()
}

// HasKey reports whether a key is stored for a provider, without returning it.
func (m *Manager) HasKey(id string) bool {
	k, err := m.keys.Get(id)
	return err == nil && k != ""
}

// Model builds the chat model of the active provider. It is the agent's ModelFunc.
func (m *Manager) Model(context.Context) (model.ToolCallingChatModel, error) {
	cfg := m.Config()
	r, ok := cfg.Resolve(cfg.Active)
	if !ok {
		return nil, ErrNotConfigured
	}
	return m.build(r)
}

func (m *Manager) build(r Resolved) (model.ToolCallingChatModel, error) {
	key, err := m.keys.Get(r.Preset.ID)
	if err != nil && !errors.Is(err, ErrNoKey) {
		return nil, fmt.Errorf("keychain: %w", err)
	}
	if r.Preset.NeedsKey() && key == "" {
		return nil, ErrNoKey
	}
	return NewModel(context.Background(), r, key, m.client)
}

// Test sends a one-token request with what is typed in Settings, before it is saved. A non-empty
// key replaces the stored one for this call only.
func (m *Manager) Test(ctx context.Context, id string, e Entry, key string) (time.Duration, error) {
	r, ok := m.Config().With(id, e).Resolve(id)
	if !ok {
		return 0, ErrNotConfigured
	}
	if key == "" {
		stored, err := m.keys.Get(id)
		if err != nil && !errors.Is(err, ErrNoKey) {
			return 0, fmt.Errorf("keychain: %w", err)
		}
		key = stored
	}
	if r.Preset.NeedsKey() && key == "" {
		return 0, ErrNoKey
	}
	mod, err := NewModel(ctx, r, key, m.client)
	if err != nil {
		return 0, err
	}
	return Probe(ctx, mod)
}
