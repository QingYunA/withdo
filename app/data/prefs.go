package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Prefs are the choices the user made in Settings that the app keeps itself.
type Prefs struct {
	// OpenWith is where "go see" on a handed-over row takes the user: "agentApp", "terminal",
	// "editor" or "finder". Empty means the agent's app.
	OpenWith string `json:"openWith,omitempty"`
	// NotificationsOff is set when the user switched the system notifications off; they are on by
	// default, so a file without it, or none, means on.
	NotificationsOff bool `json:"notificationsOff,omitempty"`
	// SoundOff is set when the user switched the Summary scene's sounds off; they are on by default.
	SoundOff bool `json:"soundOff,omitempty"`
	// EndOfDayOff is set when the user switched the end-of-day reminder off; it is on by default.
	EndOfDayOff bool `json:"endOfDayOff,omitempty"`
	// EndOfDayAt is the time of the reminder as "15:04"; empty means DefaultEndOfDay.
	EndOfDayAt string `json:"endOfDayAt,omitempty"`
	// EndOfDayLast is the date ("2006-01-02") the reminder was last sent or left out, so that it comes
	// once a day.
	EndOfDayLast string `json:"endOfDayLast,omitempty"`
	// RecentChanges is set when the user switched the Recent changes page on; it is off by default.
	RecentChanges bool `json:"recentChanges,omitempty"`
	// RecentRepo is the repository the Recent changes page was last set to ("owner/name"); empty is
	// all of them. RecentDays is its period, 7 or 30; 0 is the default, 7.
	RecentRepo string `json:"recentRepo,omitempty"`
	RecentDays int    `json:"recentDays,omitempty"`
	// MineOpen and AgentsOpen are set when the user unfolded the sub-rows of Mine or Agents in the
	// sidebar; they are folded by default.
	MineOpen   bool `json:"mineOpen,omitempty"`
	AgentsOpen bool `json:"agentsOpen,omitempty"`
}

// DefaultEndOfDay is when the end-of-day reminder comes unless the user picks another time.
const DefaultEndOfDay = "18:30"

// ClockOf reads a time written "15:04" as an hour and a minute.
func ClockOf(s string) (hour, minute int, ok bool) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, 0, false
	}
	return t.Hour(), t.Minute(), true
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
