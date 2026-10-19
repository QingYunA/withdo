package data

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/QingYunA/adhd-todo/internal/agent"
)

// RecentExplanation is what an item did, why and where, as the assistant wrote it.
type RecentExplanation = agent.RecentExplanation

// RecentCache keeps what the assistant wrote for the Recent changes page: a sentence and a category
// for every item, the explanation of those opened, and the lines of a repository's period. It is a
// file in the data folder, not the database: deleting it only costs the assistant's time. A
// sentence is kept by the item's URL and its updatedAt, so a changed title or description is
// written again; the lines are kept by repository, period and language.
type RecentCache struct {
	mu     sync.Mutex
	path   string // "" keeps nothing past the process
	loaded bool
	file   recentFile
}

type recentFile struct {
	Items map[string]cachedItem  `json:"items"`
	Lines map[string]StoredLines `json:"lines"`
}

type cachedItem struct {
	UpdatedAt   time.Time          `json:"updatedAt"`
	Sentence    string             `json:"sentence,omitempty"`
	Category    RecentCategory     `json:"category,omitempty"`
	Topic       string             `json:"topic,omitempty"`
	Explanation *RecentExplanation `json:"explanation,omitempty"`
}

// StoredLines are the sentences written about a repository's period, and when.
type StoredLines struct {
	Text      string    `json:"text"`
	WrittenAt time.Time `json:"writtenAt"`
}

// recentKeep is how long an item stays in the file after it was last updated.
const recentKeep = 90 * 24 * time.Hour

// NewRecentCache keeps the cache in the file at path; "" keeps it in memory only.
func NewRecentCache(path string) *RecentCache { return &RecentCache{path: path} }

// load reads the file the first time; one that cannot be read starts the cache empty.
func (c *RecentCache) load() {
	if c.loaded {
		return
	}
	c.loaded = true
	c.file = recentFile{Items: map[string]cachedItem{}, Lines: map[string]StoredLines{}}
	if c.path == "" {
		return
	}
	raw, err := os.ReadFile(c.path)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Printf("recent changes cache: %v", err)
		}
		return
	}
	var f recentFile
	if err := json.Unmarshal(raw, &f); err != nil {
		log.Printf("recent changes cache: %v", err)
		return
	}
	if f.Items != nil {
		c.file.Items = f.Items
	}
	if f.Lines != nil {
		c.file.Lines = f.Lines
	}
}

// save writes the file through a temporary one, so a crash leaves the old one whole.
func (c *RecentCache) save(now time.Time) {
	for k, it := range c.file.Items {
		if now.Sub(it.UpdatedAt) > recentKeep {
			delete(c.file.Items, k)
		}
	}
	if c.path == "" {
		return
	}
	raw, err := json.Marshal(c.file)
	if err == nil {
		if err = os.MkdirAll(filepath.Dir(c.path), 0o755); err == nil {
			tmp := c.path + ".tmp"
			if err = os.WriteFile(tmp, raw, 0o600); err == nil {
				err = os.Rename(tmp, c.path)
			}
		}
	}
	if err != nil {
		log.Printf("recent changes cache: %v", err)
	}
}

func itemKey(lang string, i RecentItem) string { return lang + " " + i.URL }

// entry is the item's stored entry, when it was kept for the item as it is now.
func (c *RecentCache) entry(lang string, i RecentItem) (cachedItem, bool) {
	c.load()
	e, ok := c.file.Items[itemKey(lang, i)]
	return e, ok && e.UpdatedAt.Equal(i.UpdatedAt)
}

// Note is the sentence kept for an item in a language, when it was written for the item as it is.
func (c *RecentCache) Note(lang string, i RecentItem) (RecentNote, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entry(lang, i)
	if !ok || e.Sentence == "" {
		return RecentNote{}, false
	}
	return RecentNote{Sentence: e.Sentence, Category: e.Category, Topic: e.Topic}, true
}

// PutNotes keeps the sentences written for items, which replace what was kept for them.
func (c *RecentCache) PutNotes(lang string, now time.Time, notes []RecentNoteFor) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.load()
	for _, n := range notes {
		c.file.Items[itemKey(lang, n.Item)] = cachedItem{
			UpdatedAt: n.Item.UpdatedAt, Sentence: n.Note.Sentence, Category: n.Note.Category, Topic: n.Note.Topic,
		}
	}
	c.save(now)
}

// RecentNoteFor is a sentence with the item it was written for.
type RecentNoteFor struct {
	Item RecentItem
	Note RecentNote
}

// Explanation is the explanation kept for an item in a language.
func (c *RecentCache) Explanation(lang string, i RecentItem) (RecentExplanation, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entry(lang, i)
	if !ok || e.Explanation == nil {
		return RecentExplanation{}, false
	}
	return *e.Explanation, true
}

// PutExplanation keeps the explanation of an item, beside its sentence when it has one.
func (c *RecentCache) PutExplanation(lang string, now time.Time, i RecentItem, x RecentExplanation) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entry(lang, i)
	if !ok {
		e = cachedItem{UpdatedAt: i.UpdatedAt}
	}
	e.Explanation = &x
	c.file.Items[itemKey(lang, i)] = e
	c.save(now)
}

func linesKey(repo string, days int, lang string) string {
	return lang + " " + strconv.Itoa(days) + " " + repo
}

// Lines are the sentences kept for a repository and a period of days, in a language.
func (c *RecentCache) Lines(repo string, days int, lang string) (StoredLines, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.load()
	l, ok := c.file.Lines[linesKey(repo, days, lang)]
	return l, ok
}

// PutLines keeps the sentences of a repository and a period, replacing the ones it had.
func (c *RecentCache) PutLines(repo string, days int, lang string, l StoredLines) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.load()
	c.file.Lines[linesKey(repo, days, lang)] = l
	c.save(l.WrittenAt)
}
