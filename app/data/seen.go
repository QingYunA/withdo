package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// Seen remembers which handed-over items the user has looked at: for each task id, the time it was
// handed over when the user last saw Mine. An item handed over again (a newer report) is unseen
// again.
type Seen map[int]time.Time

// Has reports whether the user has seen this handed-over row.
func (s Seen) Has(it Item) bool {
	at, ok := s[it.ID]
	return ok && !it.Since.After(at)
}

// Unseen are the items the user has not seen, in the order given.
func Unseen(items []Item, s Seen) []Item {
	var out []Item
	for _, it := range items {
		if !s.Has(it) {
			out = append(out, it)
		}
	}
	return out
}

// MarkSeen returns a copy of s in which every item is seen, without the entries of tasks for which
// alive says no (a nil alive keeps them all); changed tells whether the copy differs from s.
func (s Seen) MarkSeen(items []Item, alive func(id int) bool) (next Seen, changed bool) {
	next = Seen{}
	for id, at := range s {
		if alive == nil || alive(id) {
			next[id] = at
		} else {
			changed = true
		}
	}
	for _, it := range items {
		if at, ok := next[it.ID]; !ok || it.Since.After(at) {
			next[it.ID], changed = it.Since, true
		}
	}
	return next, changed
}

// SeenStore keeps Seen between runs.
type SeenStore interface {
	LoadSeen() (Seen, error)
	SaveSeen(Seen) error
}

// MemorySeen is a SeenStore that keeps nothing past the process: the demo's and the tests'.
type MemorySeen struct {
	mu   sync.Mutex
	seen Seen
}

// LoadSeen implements SeenStore.
func (m *MemorySeen) LoadSeen() (Seen, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := Seen{}
	for id, at := range m.seen {
		out[id] = at
	}
	return out, nil
}

// SaveSeen implements SeenStore.
func (m *MemorySeen) SaveSeen(s Seen) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seen = Seen{}
	for id, at := range s {
		m.seen[id] = at
	}
	return nil
}

// FileSeen is a SeenStore in a JSON file next to the database: {"12": "2026-10-15T12:00:00Z"}.
type FileSeen struct{ Path string }

// LoadSeen implements SeenStore. A missing file is an empty record; an unreadable one is an error
// the caller may ignore: every row then counts as unseen once.
func (f FileSeen) LoadSeen() (Seen, error) {
	raw, err := os.ReadFile(f.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return Seen{}, nil
	}
	if err != nil {
		return Seen{}, err
	}
	var byID map[string]time.Time
	if err := json.Unmarshal(raw, &byID); err != nil {
		return Seen{}, fmt.Errorf("read %s: %w", f.Path, err)
	}
	out := Seen{}
	for k, at := range byID {
		if id, err := strconv.Atoi(k); err == nil {
			out[id] = at
		}
	}
	return out, nil
}

// SaveSeen implements SeenStore. It writes a temporary file and renames it, so a crash leaves the
// old record whole.
func (f FileSeen) SaveSeen(s Seen) error {
	byID := make(map[string]time.Time, len(s))
	for id, at := range s {
		byID[strconv.Itoa(id)] = at.UTC()
	}
	raw, err := json.Marshal(byID)
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
