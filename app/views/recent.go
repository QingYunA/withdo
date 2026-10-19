package views

import (
	"context"
	"errors"
	"log"
	"sort"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The Recent changes page (ui-spec, Recent changes): what changed in the projects' repositories
// lately. This file is what the page knows: what it read from GitHub, what the assistant wrote about
// it, and when it reads again. recent_page.go draws the page, recent_line.go its items.

const (
	recentEvery = 10 * time.Minute // GitHub is read again this often while the page is shown
	recentBeat  = 30 * time.Second // the page redraws, so that "Updated 2 min ago" keeps up
	recentChunk = 12               // items the assistant is asked about at once
)

// recentState is the Recent changes page: its choices, what was read, and the assistant's work.
type recentState struct {
	loaded bool
	repo   string // the repository shown, "" for all
	days   int    // 7 or 30

	shown    bool // the page was open last frame: it reads GitHub when it opens
	again    bool // a click on Updated asked for another read
	fetching bool
	// gen counts the reads, so that the answer to an older one is dropped; read is the one whose
	// answer the results are.
	gen, read int
	results   map[string]data.RepoResult
	fetchedAt time.Time
	lang      string // the language of the window, as the assistant is told

	// rev counts what a section is built from changing; built is the sections of one rev.
	rev   int
	built recentBuilt

	describing map[string]bool // repository and language, while the assistant writes its items
	attempted  map[string]int  // the read it was last tried for
	failed     map[string]int  // the read it last failed for
	lines      map[string]*recentWork
	explain    map[string]*recentWork
	open       map[string]bool // items unfolded, by URL
	moreOpen   map[string]bool // Other unfolded, by repository

	copied    time.Time
	copiedKey string
}

// recentWork is a request to the assistant: one that is on its way, or has failed.
type recentWork struct{ writing, failed bool }

// recentSec is a project's section: its items, or why they could not be read.
type recentSec struct {
	project data.Project
	data.RecentSection
	err *data.RecentError
}

// recentBuilt is the sections as built for one state of what they are made of.
type recentBuilt struct {
	ok    bool
	rev   int
	appRv int
	lang  string
	secs  []recentSec
}

// recentLoad reads the page's choices the first time.
func (m *Main) recentLoad() {
	st := &m.recent
	if st.loaded {
		return
	}
	st.loaded = true
	st.repo, st.days = m.app.recentChoice()
	st.describing, st.attempted, st.failed = map[string]bool{}, map[string]int{}, map[string]int{}
	st.lines, st.explain = map[string]*recentWork{}, map[string]*recentWork{}
	st.open, st.moreOpen = map[string]bool{}, map[string]bool{}
}

func (a *App) windowKey() bool { return a.WindowKey == nil || a.WindowKey() }

// recentCache is where the assistant's work is kept; in memory when the app has no file.
func (m *Main) recentCache() *data.RecentCache {
	if m.app.RecentCache == nil {
		m.app.RecentCache = data.NewRecentCache("")
	}
	return m.app.RecentCache
}

// recentProjects are the projects whose repository the page reads, each repository once, in the
// sidebar's order. Archived projects are not in the snapshot.
func (m *Main) recentProjects() []data.Project {
	seen := map[string]bool{}
	var out []data.Project
	for _, p := range m.app.snap.projects {
		if p.RepoSlug != "" && !seen[p.RepoSlug] {
			seen[p.RepoSlug] = true
			out = append(out, p)
		}
	}
	return out
}

// ShowRecent opens the Recent changes page on a repository ("" for all of them), as the link of a
// project page does.
func (m *Main) ShowRecent(repo string) {
	m.recentLoad()
	m.recent.repo = repo
	m.app.setRecentChoice(repo, m.recent.days)
	m.goTo(nav{screen: screenRecent})
}

// recentPick picks the repository shown, and keeps the choice.
func (m *Main) recentPick(repo string) {
	st := &m.recent
	st.repo = repo
	m.app.setRecentChoice(repo, st.days)
	m.scroll = ui.ScrollState{}
}

// recentPickDays picks the period, and keeps the choice. What was read is of the other period: the
// page reads again.
func (m *Main) recentPickDays(days int) {
	st := &m.recent
	if st.days == days {
		return
	}
	st.days, st.results, st.fetching, st.again = days, nil, false, true
	st.gen++
	st.rev++
	m.app.setRecentChoice(st.repo, days)
}

// recentTick reads GitHub when the page opens, again when Updated is clicked, and every ten minutes
// while the window is key. It runs every frame the page is drawn.
func (m *Main) recentTick(c *ui.Context, now time.Time) {
	st := &m.recent
	m.recentLoad()
	st.lang = languageName(c.Locale())
	due := !st.shown || st.again || st.fetchedAt.IsZero() || now.Sub(st.fetchedAt) >= recentEvery && m.app.windowKey()
	st.shown = true
	if due && !st.fetching {
		st.again = false
		m.recentFetch(now)
	}
	c.After(recentBeat)
}

// recentFetch reads every project's repository off the UI thread.
func (m *Main) recentFetch(now time.Time) {
	st := &m.recent
	if m.app.Recent == nil {
		return
	}
	var repos []string
	for _, p := range m.recentProjects() {
		repos = append(repos, p.RepoSlug)
	}
	if len(repos) == 0 {
		st.results, st.fetchedAt = map[string]data.RepoResult{}, now
		return
	}
	st.fetching = true
	st.gen++
	gen, since := st.gen, data.RecentSince(now, st.days)
	read, post, clock := m.app.Recent, m.app.Act.Post, m.app.Now
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), writeLimit)
		defer cancel()
		results := data.FetchRepos(ctx, read, repos, since)
		post(func() {
			if gen != st.gen {
				return // the period changed meanwhile
			}
			st.fetching, st.results, st.fetchedAt, st.read = false, results, clock(), gen
			st.rev++
		})
	}()
}

// recentSections are the projects' sections, the one that changed last first, and those that could
// not be read after the others. They are built again only when something they are made of changed.
func (m *Main) recentSections() []recentSec {
	st := &m.recent
	b := &st.built
	if b.ok && b.rev == st.rev && b.appRv == m.app.rev && b.lang == st.lang {
		return b.secs
	}
	cache := m.recentCache()
	note := func(i data.RecentItem) (data.RecentNote, bool) { return cache.Note(st.lang, i) }
	var secs []recentSec
	for _, p := range m.recentProjects() {
		res, ok := st.results[p.RepoSlug]
		if !ok {
			continue
		}
		s := recentSec{project: p}
		if res.Err != nil {
			var re *data.RecentError
			if !errors.As(res.Err, &re) {
				re = &data.RecentError{Repo: p.RepoSlug, Text: res.Err.Error(), Err: res.Err}
			}
			s.err = re
		} else {
			s.RecentSection = data.BuildRecent(res.Changes, note)
		}
		secs = append(secs, s)
	}
	sort.SliceStable(secs, func(a, c int) bool { return secs[a].Last.After(secs[c].Last) })
	*b = recentBuilt{ok: true, rev: st.rev, appRv: m.app.rev, lang: st.lang, secs: secs}
	return secs
}

// recentOf is the section of a repository.
func recentOf(secs []recentSec, repo string) (recentSec, bool) {
	for _, s := range secs {
		if s.project.RepoSlug == repo {
			return s, true
		}
	}
	return recentSec{}, false
}

// describeIfDue has the assistant write a sentence for each item of a repository that has none kept,
// a few at a time, once for each read. It runs every frame the section is drawn.
func (m *Main) describeIfDue(repo string, ch data.RepoChanges) {
	st := &m.recent
	key := st.lang + "|" + repo
	if !m.hasModel() || st.describing[key] || st.attempted[key] == st.read {
		return
	}
	cache := m.recentCache()
	var need []data.RecentItem
	for _, list := range [][]data.RecentItem{ch.Merged, ch.Open, ch.Issues} {
		for _, i := range list {
			if _, ok := cache.Note(st.lang, i); !ok {
				need = append(need, i)
			}
		}
	}
	st.attempted[key] = st.read
	if len(need) == 0 {
		return
	}
	st.describing[key] = true
	asst, post, now, gen, lang := m.app.Assistant, m.app.Act.Post, m.app.Now(), st.read, st.lang
	go func() {
		var failure error
		for from := 0; from < len(need) && failure == nil; from += recentChunk {
			chunk := need[from:min(from+recentChunk, len(need))]
			ctx, cancel := context.WithTimeout(context.Background(), writeLimit)
			notes, err := asst.DescribeRecent(ctx, data.RecentDescribeRequest{Repo: repo, Language: lang, Items: chunk})
			cancel()
			if err != nil {
				failure = err
				break
			}
			var put []data.RecentNoteFor
			for _, i := range chunk {
				if n, ok := notes[i.Number]; ok {
					put = append(put, data.RecentNoteFor{Item: i, Note: n})
				}
			}
			cache.PutNotes(lang, now, put)
			post(func() { st.rev++ })
		}
		post(func() {
			delete(st.describing, key)
			if failure != nil {
				log.Printf("recent changes: %v", failure)
				st.failed[key] = gen
			}
			st.rev++
		})
	}()
}

// recentLinesKey names the lines of a repository for this period and language.
func (m *Main) recentLinesKey(repo string) string {
	st := &m.recent
	return st.lang + "|" + itoa(st.days) + "|" + repo
}

// linesOf is the state of the lines of a repository.
func (m *Main) linesOf(repo string) *recentWork {
	st := &m.recent
	key := m.recentLinesKey(repo)
	w := st.lines[key]
	if w == nil {
		w = &recentWork{}
		st.lines[key] = w
	}
	return w
}

// linesIfDue has the assistant write the lines of a project the first time they are due: once its
// items have their sentences, for each repository and period. It runs every frame the section is
// drawn.
func (m *Main) linesIfDue(s recentSec) {
	st := &m.recent
	repo := s.project.RepoSlug
	key := st.lang + "|" + repo
	if !m.hasModel() || s.Count() == 0 || st.describing[key] || st.attempted[key] != st.read || st.failed[key] == st.read {
		return
	}
	if w := m.linesOf(repo); w.writing || w.failed {
		return
	}
	if _, ok := m.recentCache().Lines(repo, st.days, st.lang); ok {
		return
	}
	m.writeLines(s)
}

// writeLines asks the assistant for the lines of a project, off the UI thread. The ones being
// replaced show until the new ones arrive.
func (m *Main) writeLines(s recentSec) {
	st := &m.recent
	repo := s.project.RepoSlug
	w := m.linesOf(repo)
	if w.writing || !m.hasModel() {
		return
	}
	w.writing, w.failed = true, false
	req := data.RecentLinesRequest{Repo: repo, Language: st.lang, Period: "the last " + itoa(st.days) + " days", Lines: s.Lines()}
	asst, post, cache, now, days, lang := m.app.Assistant, m.app.Act.Post, m.recentCache(), m.app.Now(), st.days, st.lang
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), writeLimit)
		defer cancel()
		text, err := asst.WriteRecentLines(ctx, req)
		if err == nil {
			cache.PutLines(repo, days, lang, data.StoredLines{Text: text, WrittenAt: now})
		}
		post(func() {
			w.writing = false
			if err != nil {
				log.Printf("recent changes: %v", err)
				w.failed = true
			}
		})
	}()
}

// newSince reports whether something merged after the lines were written.
func (s recentSec) newSince(at time.Time) bool {
	for _, group := range [][]data.RecentLine{s.New, s.Fixed, s.Other} {
		for _, l := range group {
			if l.Item.ClosedAt.After(at) {
				return true
			}
		}
	}
	return false
}

// explanationOf is the state of the explanation of an item.
func (m *Main) explanationOf(url string) *recentWork {
	st := &m.recent
	w := st.explain[url]
	if w == nil {
		w = &recentWork{}
		st.explain[url] = w
	}
	return w
}

// explainIfDue has the assistant explain an item the first time it is unfolded, off the UI thread,
// and keeps the answer. It runs every frame an item is unfolded.
func (m *Main) explainIfDue(i data.RecentItem) {
	st := &m.recent
	if !m.hasModel() {
		return
	}
	if _, ok := m.recentCache().Explanation(st.lang, i); ok {
		return
	}
	w := m.explanationOf(i.URL)
	if w.writing || w.failed {
		return
	}
	w.writing = true
	asst, post, cache, now, lang := m.app.Assistant, m.app.Act.Post, m.recentCache(), m.app.Now(), st.lang
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), writeLimit)
		defer cancel()
		x, err := asst.ExplainRecent(ctx, data.RecentExplainRequest{Language: lang, Item: i})
		if err == nil {
			cache.PutExplanation(lang, now, i, x)
		}
		post(func() {
			w.writing = false
			if err != nil {
				log.Printf("recent changes: %v", err)
				w.failed = true
			}
		})
	}()
}
