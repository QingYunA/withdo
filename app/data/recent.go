package data

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QingYunA/adhd-todo/internal/agent"
)

// The Recent changes page (ui-spec, Recent changes): what changed in the projects' repositories
// lately. This file is what the page knows: the pull requests and issues as GitHub reports them,
// how they group into a project's section, and what is shown when the assistant has said nothing.

// RecentKind says what an item is.
type RecentKind string

// Kinds of item.
const (
	RecentMerged RecentKind = agent.RecentMerged // a pull request that merged
	RecentOpen   RecentKind = agent.RecentOpen   // a pull request still open, drafts too
	RecentIssue  RecentKind = agent.RecentIssue  // an issue opened or closed in the period
)

// RecentCategory is the group a merged pull request is shown in.
type RecentCategory string

// Categories.
const (
	RecentNew   RecentCategory = agent.RecentNew
	RecentFixed RecentCategory = agent.RecentFixed
	RecentOther RecentCategory = agent.RecentOther
)

// RecentItem is a pull request or an issue as gh reports it.
type RecentItem struct {
	Repo   string
	Kind   RecentKind
	Number int
	Title  string
	Body   string
	URL    string
	// Author is the GitHub login, without the "app/" of a bot's.
	Author string
	// Bot says the author is an app: a dependency bot, or a coding agent.
	Bot    bool
	Draft  bool
	Branch string
	Labels []string
	// Files are the names of the files a pull request changed.
	Files     []string
	CreatedAt time.Time
	UpdatedAt time.Time
	// ClosedAt is when a pull request merged or an issue was closed; zero while open.
	ClosedAt time.Time
	// FixedBy is the number of the pull request that closed an issue, 0 for none or unknown.
	FixedBy int
}

// Solved reports whether an issue is closed.
func (i RecentItem) Solved() bool { return i.Kind == RecentIssue && !i.ClosedAt.IsZero() }

// At is the moment the item is dated by: when a pull request merged, when one was opened, when an
// issue was closed, else when it was opened.
func (i RecentItem) At() time.Time {
	if !i.ClosedAt.IsZero() {
		return i.ClosedAt
	}
	return i.CreatedAt
}

// RepoChanges is what changed in one repository since a moment.
type RepoChanges struct {
	Repo   string
	Merged []RecentItem // merged pull requests, newest first
	Open   []RecentItem // open pull requests, newest first
	Issues []RecentItem // issues opened or closed in the period, newest first
}

// RecentProblem says why a repository could not be read, so that the views can word it.
type RecentProblem int

// Problems.
const (
	RecentOtherProblem RecentProblem = iota // anything else: the text is gh's own
	RecentNoGH                              // there is no gh: brew install gh
	RecentSignedOut                         // gh has no login: gh auth login
	RecentOffline                           // GitHub could not be reached
	RecentNoAccess                          // the repository is not there for this account
)

// RecentError is a repository that could not be read.
type RecentError struct {
	Problem RecentProblem
	Repo    string
	Text    string
	Err     error
}

func (e *RecentError) Error() string { return e.Text }
func (e *RecentError) Unwrap() error { return e.Err }

// Recent reads the repositories. It blocks, so the UI calls it off its thread.
type Recent interface {
	// Fetch reads the pull requests that merged since, the ones that are open, and the issues
	// opened or closed since, of a repository ("owner/name"). The error is a *RecentError.
	Fetch(ctx context.Context, repo string, since time.Time) (RepoChanges, error)
}

// RepoResult is one repository's answer.
type RepoResult struct {
	Changes RepoChanges
	Err     error
}

// FetchRepos reads every repository, a few at a time.
func FetchRepos(ctx context.Context, r Recent, repos []string, since time.Time) map[string]RepoResult {
	out := make(map[string]RepoResult, len(repos))
	var mu sync.Mutex
	var wg sync.WaitGroup
	gate := make(chan struct{}, 4)
	for _, repo := range repos {
		wg.Add(1)
		go func() {
			defer wg.Done()
			gate <- struct{}{}
			defer func() { <-gate }()
			ch, err := r.Fetch(ctx, repo, since)
			mu.Lock()
			out[repo] = RepoResult{Changes: ch, Err: err}
			mu.Unlock()
		}()
	}
	wg.Wait()
	return out
}

// RecentSince is the start of a period of days: local midnight of the first of them, today being
// the last. 7 days is today and the six before it.
func RecentSince(now time.Time, days int) time.Time {
	y, m, d := now.Date()
	return time.Date(y, m, d-(days-1), 0, 0, 0, 0, now.Location())
}

// RecentNote is the sentence written for an item, and for a merged pull request the group it is
// shown in. For an item in Other, Topic is a word or two saying what it is ("docs").
type RecentNote struct {
	Sentence string
	Category RecentCategory
	Topic    string
}

// conventional finds the type of a conventional commit title: "feat(scope)!: ".
var conventional = regexp.MustCompile(`^([A-Za-z]+)(\([^)]*\))?!?:\s*`)

// FallbackNote is what an item says with no assistant: the title without its conventional-commit
// prefix, and for a merged pull request New for feat, Fixed for fix and Other for the rest.
func FallbackNote(i RecentItem) RecentNote {
	note := RecentNote{Sentence: strings.TrimSpace(i.Title)}
	m := conventional.FindStringSubmatch(note.Sentence)
	if m == nil {
		note.Category = RecentOther
		return note.forKind(i.Kind)
	}
	if rest := strings.TrimSpace(note.Sentence[len(m[0]):]); rest != "" {
		note.Sentence = rest
	}
	switch strings.ToLower(m[1]) {
	case "feat":
		note.Category = RecentNew
	case "fix":
		note.Category = RecentFixed
	default:
		note.Category = RecentOther
	}
	return note.forKind(i.Kind)
}

// forKind drops the category of what is not a merged pull request.
func (n RecentNote) forKind(k RecentKind) RecentNote {
	if k != RecentMerged {
		n.Category = ""
	}
	return n
}

// RecentLine is an item as the page lists it.
type RecentLine struct {
	Item RecentItem
	Note RecentNote
	// Written says the assistant wrote the sentence, and not the title standing in for it.
	Written bool
}

// RecentSection is a project's part of the page. Each group is newest first.
type RecentSection struct {
	Repo                            string
	New, Fixed, Other, Open, Issues []RecentLine
	// Last is the newest moment of any item: sections are ordered by it.
	Last time.Time
}

// Merged counts the merged pull requests.
func (s RecentSection) Merged() int { return len(s.New) + len(s.Fixed) + len(s.Other) }

// Count is how many items the section lists.
func (s RecentSection) Count() int { return s.Merged() + len(s.Open) + len(s.Issues) }

// Lines lists every line of the section.
func (s RecentSection) Lines() []RecentLine {
	var out []RecentLine
	for _, g := range [][]RecentLine{s.New, s.Fixed, s.Other, s.Open, s.Issues} {
		out = append(out, g...)
	}
	return out
}

// OtherTopics are the distinct topics of what is folded under Other, in the order they come.
func (s RecentSection) OtherTopics() []string {
	var out []string
	seen := map[string]bool{}
	for _, l := range s.Other {
		if t := l.Note.Topic; t != "" && !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}

// BuildRecent groups a repository's changes. note says what the assistant wrote for an item, if
// anything; an item without a sentence shows its title (FallbackNote), and one the assistant gave
// no category shows under the fallback's.
func BuildRecent(ch RepoChanges, note func(RecentItem) (RecentNote, bool)) RecentSection {
	s := RecentSection{Repo: ch.Repo}
	line := func(i RecentItem) RecentLine {
		fb := FallbackNote(i)
		if n, ok := note(i); ok && n.Sentence != "" {
			if n.Category == "" {
				n.Category = fb.Category
			}
			return RecentLine{Item: i, Note: n.forKind(i.Kind), Written: true}
		}
		return RecentLine{Item: i, Note: fb}
	}
	for _, i := range ch.Merged {
		l := line(i)
		switch l.Note.Category {
		case RecentNew:
			s.New = append(s.New, l)
		case RecentFixed:
			s.Fixed = append(s.Fixed, l)
		default:
			s.Other = append(s.Other, l)
		}
	}
	for _, i := range ch.Open {
		s.Open = append(s.Open, line(i))
	}
	for _, i := range ch.Issues {
		s.Issues = append(s.Issues, line(i))
	}
	for _, g := range [][]RecentLine{s.New, s.Fixed, s.Other, s.Open, s.Issues} {
		sort.SliceStable(g, func(a, b int) bool { return g[a].Item.At().After(g[b].Item.At()) })
		for _, l := range g {
			if l.Item.At().After(s.Last) {
				s.Last = l.Item.At()
			}
		}
	}
	return s
}

// RecentOrder puts the sections with something in them first, the one that changed last on top.
func RecentOrder(sections []RecentSection) {
	sort.SliceStable(sections, func(a, b int) bool { return sections[a].Last.After(sections[b].Last) })
}
