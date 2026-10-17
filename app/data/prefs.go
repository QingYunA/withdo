package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// Prefs are the choices the user made in Settings that the app keeps itself.
type Prefs struct {
	// OpenWith is where "go see" on a handed-over row takes the user: "agentApp", "terminal",
	// "editor" or "finder". Empty means the agent's app.
	OpenWith string `json:"openWith,omitempty"`
	// NotificationsOff is set when the user switched the system notifications off; they are on by
	// default, so a file without it, or none, means on.
	NotificationsOff bool `json:"notificationsOff,omitempty"`
}

// PrefsStore keeps Prefs between runs.
type PrefsStore interface {
	LoadPrefs() (Prefs, error)
	SavePrefs(Prefs) error
}

// MemoryPrefs is a PrefsStore that keeps nothing past the process: the demo's and the tests'.
type MemoryPrefs struct {
	mu    sync.Mutex
	prefs Prefs
}

// LoadPrefs implements PrefsStore.
func (m *MemoryPrefs) LoadPrefs() (Prefs, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.prefs, nil
}

// SavePrefs implements PrefsStore.
func (m *MemoryPrefs) SavePrefs(p Prefs) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.prefs = p
	return nil
}

// FilePrefs is a PrefsStore in a JSON file in the data folder: {"openWith": "terminal"}.
type FilePrefs struct{ Path string }

// LoadPrefs implements PrefsStore. A missing file is the defaults; an unreadable one is an error
// the caller may ignore: the defaults apply.
func (f FilePrefs) LoadPrefs() (Prefs, error) {
	raw, err := os.ReadFile(f.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return Prefs{}, nil
	}
	if err != nil {
		return Prefs{}, err
	}
	var p Prefs
	if err := json.Unmarshal(raw, &p); err != nil {
		return Prefs{}, fmt.Errorf("read %s: %w", f.Path, err)
	}
	return p, nil
}

// SavePrefs implements PrefsStore. It writes a temporary file and renames it, so a crash leaves
// the old file whole.
func (f FilePrefs) SavePrefs(p Prefs) error {
	raw, err := json.Marshal(p)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(f.Path), 0o755); err != nil {
		return err
	}
	tmp := f.Path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, f.Path)
}
