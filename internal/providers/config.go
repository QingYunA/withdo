package providers

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ConfigFile is the name of the provider file inside the data folder.
const ConfigFile = "models.json"

// Entry is what the user changed for one preset. Empty fields mean "the preset's default".
type Entry struct {
	BaseURL  string   `json:"base_url,omitempty"`
	Model    string   `json:"model,omitempty"`
	Protocol Protocol `json:"protocol,omitempty"` // only the Custom preset
}

// Config is the provider file: which preset is active and each preset's overrides. It holds no
// keys; those live in the keychain.
type Config struct {
	Active    string           `json:"active"`
	Providers map[string]Entry `json:"providers,omitempty"`
}

// Resolved is a preset with the user's overrides applied: what a request would use.
type Resolved struct {
	Preset   Preset
	BaseURL  string
	Model    string
	Protocol Protocol
}

// Resolve applies the overrides stored for a preset. It reports false for an unknown preset.
func (c Config) Resolve(id string) (Resolved, bool) {
	p, ok := PresetByID(id)
	if !ok {
		return Resolved{}, false
	}
	e := c.Providers[id]
	r := Resolved{Preset: p, BaseURL: p.BaseURL, Protocol: p.Protocol}
	if len(p.Models) > 0 {
		r.Model = p.Models[0]
	}
	if u := strings.TrimSpace(e.BaseURL); u != "" {
		r.BaseURL = u
	}
	if m := strings.TrimSpace(e.Model); m != "" {
		r.Model = m
	}
	if p.PickProtocol && (e.Protocol == Anthropic || e.Protocol == OpenAI) {
		r.Protocol = e.Protocol
	}
	return r, true
}

// With returns a copy of the config with one entry replaced; the original is left alone.
func (c Config) With(id string, e Entry) Config {
	next := Config{Active: c.Active, Providers: make(map[string]Entry, len(c.Providers)+1)}
	for k, v := range c.Providers {
		next.Providers[k] = v
	}
	next.Providers[id] = e
	return next
}

// WithActive returns a copy of the config with another preset active.
func (c Config) WithActive(id string) Config {
	next := c.With(id, c.Providers[id])
	next.Active = id
	return next
}

// Store reads and writes the provider file.
type Store struct {
	path string
	mu   sync.Mutex
}

// NewStore keeps the provider file at path.
func NewStore(path string) *Store { return &Store{path: path} }

// Load reads the file. A missing file is an empty config: nothing is configured yet.
func (s *Store) Load() (Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read %s: %w", s.path, err)
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", s.path, err)
	}
	return c, nil
}

// Save writes the file atomically (write a sibling, then rename) with owner-only permissions.
func (s *Store) Save(c Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create data folder: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("replace %s: %w", s.path, err)
	}
	return nil
}
