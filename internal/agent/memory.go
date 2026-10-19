package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"
)

// The two sections of the memory file. The headings are what the user reads when editing by hand.
const (
	memoryTitle  = "# 助手记忆"
	headingAbout = "## 关于我"
	headingPrefs = "## 纠正与偏好"

	// SectionAbout and SectionPrefs are the names the remember tool takes for the two sections.
	SectionAbout = "about"
	SectionPrefs = "prefs"

	maxNotes      = 40
	maxMemoryText = 4000
	maxNoteRunes  = 200
)

// Memory is the assistant's notes about the user: one Markdown file with a bullet per note. The
// file is read again on every call, so a hand edit is picked up and the last writer wins. A
// missing file means no memory. A nil *Memory is a memory that is switched off.
type Memory struct {
	path string
	mu   sync.Mutex
}

// NewMemory is the memory kept in the file at path.
func NewMemory(path string) *Memory { return &Memory{path: path} }

// Path is the file the notes are kept in.
func (m *Memory) Path() string { return m.path }

// MemoryEdit is one change to the file, with what is needed to take it back.
type MemoryEdit struct {
	Text    string // the file after the change
	Changed int    // notes added (0 or 1) or removed
	prev    string
	existed bool
}

// Text is the whole file; empty when there is none.
func (m *Memory) Text() string {
	if m == nil {
		return ""
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	s, _ := m.read()
	return s
}

func (m *Memory) read() (string, bool) {
	b, err := os.ReadFile(m.path)
	if err != nil {
		return "", false
	}
	return string(b), true
}

// write replaces the file through a temporary file in the same folder, so a reader never sees half of it.
func (m *Memory) write(text string) error {
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(m.path), ".assistant-memory-*")
	if err != nil {
		return err
	}
	_, werr := tmp.WriteString(text)
	if cerr := tmp.Close(); werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Chmod(tmp.Name(), 0o644)
	}
	if werr == nil {
		werr = os.Rename(tmp.Name(), m.path)
	}
	if werr != nil {
		_ = os.Remove(tmp.Name())
	}
	return werr
}

// Restore puts the file back as it was before e: its text, or no file when there was none.
func (m *Memory) Restore(e MemoryEdit) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !e.existed {
		if err := os.Remove(m.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	return m.write(e.prev)
}

// cleanNote is a note as it is stored: one line, trimmed, without a bullet of its own.
func cleanNote(s string) (string, error) {
	s = strings.Join(strings.Fields(s), " ")
	s = strings.TrimSpace(strings.TrimPrefix(s, "- "))
	if s == "" {
		return "", errors.New("note is empty")
	}
	if n := len([]rune(s)); n > maxNoteRunes {
		return "", fmt.Errorf("note is %d characters; keep it under %d and make it a short rule", n, maxNoteRunes)
	}
	return s, nil
}

// bulletOf is the text of a bullet line, and whether the line is one.
func bulletOf(line string) (string, bool) {
	t := strings.TrimSpace(line)
	if !strings.HasPrefix(t, "- ") {
		return "", false
	}
	return strings.TrimSpace(t[2:]), true
}

// normal is a note with spacing, punctuation and case taken away, to tell that two say the same.
func normal(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			return -1
		}
		return unicode.ToLower(r)
	}, s)
}

func countNotes(lines []string) int {
	n := 0
	for _, l := range lines {
		if _, ok := bulletOf(l); ok {
			n++
		}
	}
	return n
}

// Add appends a note to a section ("about" or "prefs"). A note that is already there, up to
// spacing, punctuation and case, is not added again (Changed is 0). Over the limits it is an error
// that asks to merge or forget first; nothing is cut off silently.
func (m *Memory) Add(section, note string) (MemoryEdit, error) {
	heading := headingPrefs
	switch section {
	case SectionAbout:
		heading = headingAbout
	case SectionPrefs:
	default:
		return MemoryEdit{}, fmt.Errorf("section must be %q or %q", SectionAbout, SectionPrefs)
	}
	note, err := cleanNote(note)
	if err != nil {
		return MemoryEdit{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	prev, existed := m.read()
	edit := MemoryEdit{Text: prev, prev: prev, existed: existed}
	lines := splitLines(prev)
	for _, l := range lines {
		if b, ok := bulletOf(l); ok && normal(b) == normal(note) {
			return edit, nil
		}
	}
	if countNotes(lines) >= maxNotes {
		return edit, fmt.Errorf("memory is full (%d notes): merge notes with forget and remember first", maxNotes)
	}
	text := insertNote(lines, heading, "- "+note)
	if len(text) > maxMemoryText {
		return edit, fmt.Errorf("memory is over %d characters: merge or forget notes first", maxMemoryText)
	}
	if err := m.write(text); err != nil {
		return edit, err
	}
	edit.Text, edit.Changed = text, 1
	return edit, nil
}

// Forget removes every note that contains match (ignoring case) and says how many went.
func (m *Memory) Forget(match string) (MemoryEdit, error) {
	match = strings.ToLower(strings.TrimSpace(match))
	if match == "" {
		return MemoryEdit{}, errors.New("match is empty")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	prev, existed := m.read()
	edit := MemoryEdit{Text: prev, prev: prev, existed: existed}
	var kept []string
	for _, l := range splitLines(prev) {
		if b, ok := bulletOf(l); ok && strings.Contains(strings.ToLower(b), match) {
			edit.Changed++
			continue
		}
		kept = append(kept, l)
	}
	if edit.Changed == 0 {
		return edit, nil
	}
	edit.Text = strings.Join(kept, "\n") + "\n"
	if err := m.write(edit.Text); err != nil {
		return edit, err
	}
	return edit, nil
}

func splitLines(text string) []string {
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

// insertNote puts a bullet at the end of the section with that heading, making the section (and
// the title of an empty file) when it is not there. Lines the user wrote are left as they are.
func insertNote(lines []string, heading, bullet string) string {
	at := -1 // the line after which the bullet goes
	in := false
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "#") {
			in = t == heading
			if in {
				at = i
			}
			continue
		}
		if _, bullet := bulletOf(l); in && bullet {
			at = i
		}
	}
	var out []string
	if at >= 0 {
		out = append(append(append(out, lines[:at+1]...), bullet), lines[at+1:]...)
	} else {
		out = append(out, lines...)
		if len(out) == 0 {
			out = append(out, memoryTitle)
		}
		out = append(out, "", heading, bullet)
	}
	return strings.Join(out, "\n") + "\n"
}
