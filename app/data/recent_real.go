package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/internal/ghpoll"
)

// GHRecent reads the repositories through the gh the user is signed in to.
type GHRecent struct{ Runner ghpoll.Runner }

var _ Recent = GHRecent{}

// recentLimit is how many pull requests or issues a list asks for.
const recentLimit = "100"

const (
	prFields      = "number,title,body,url,author,createdAt,updatedAt,isDraft,headRefName,labels,files"
	issueFields   = "number,title,body,url,author,createdAt,updatedAt,closedAt,labels,closedByPullRequestsReferences"
	searchDateFmt = "2006-01-02"
)

// Fetch implements Recent: three lists with gh, one after the other.
func (g GHRecent) Fetch(ctx context.Context, repo string, since time.Time) (RepoChanges, error) {
	date := since.UTC().Format(searchDateFmt) // a superset of the local period: the rest is cut below
	out := RepoChanges{Repo: repo}
	lists := []struct {
		kind RecentKind
		args []string
		fill func([]RecentItem)
	}{
		{RecentMerged, []string{"pr", "list", "--state", "merged", "--search", "merged:>=" + date, "--json", prFields + ",mergedAt"}, func(v []RecentItem) { out.Merged = v }},
		{RecentOpen, []string{"pr", "list", "--state", "open", "--json", prFields}, func(v []RecentItem) { out.Open = v }},
		{RecentIssue, []string{"issue", "list", "--state", "all", "--search", "updated:>=" + date, "--json", issueFields}, func(v []RecentItem) { out.Issues = v }},
	}
	for _, l := range lists {
		args := append(l.args, "--repo", repo, "--limit", recentLimit)
		res, err := g.Runner.Run(ctx, args...)
		if err != nil {
			return RepoChanges{}, recentError(repo, err)
		}
		items, err := ParseRecent(l.kind, repo, res.Stdout, since)
		if err != nil {
			return RepoChanges{}, &RecentError{Repo: repo, Text: err.Error(), Err: err}
		}
		l.fill(items)
	}
	return out, nil
}

// gh's JSON.
type ghAuthor struct {
	Login string `json:"login"`
	IsBot bool   `json:"is_bot"`
}

type ghItem struct {
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	URL       string     `json:"url"`
	Author    ghAuthor   `json:"author"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	MergedAt  *time.Time `json:"mergedAt"`
	ClosedAt  *time.Time `json:"closedAt"`
	IsDraft   bool       `json:"isDraft"`
	Branch    string     `json:"headRefName"`
	Labels    []struct {
		Name string `json:"name"`
	} `json:"labels"`
	Files []struct {
		Path string `json:"path"`
	} `json:"files"`
	ClosedBy []struct {
		Number int `json:"number"`
	} `json:"closedByPullRequestsReferences"`
}

// ParseRecent reads the JSON of gh pr list or gh issue list. Items outside the period are left out:
// a merged pull request before since, an issue neither opened nor closed since. The rest come
// newest first.
func ParseRecent(kind RecentKind, repo string, raw []byte, since time.Time) ([]RecentItem, error) {
	var in []ghItem
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("read gh's answer for %s: %w", repo, err)
	}
	var out []RecentItem
	for _, g := range in {
		it := RecentItem{
			Repo: repo, Kind: kind, Number: g.Number, Title: g.Title, Body: g.Body, URL: g.URL,
			Author: strings.TrimPrefix(g.Author.Login, "app/"), Bot: g.Author.IsBot, Draft: g.IsDraft, Branch: g.Branch,
			CreatedAt: g.CreatedAt, UpdatedAt: g.UpdatedAt,
		}
		for _, l := range g.Labels {
			it.Labels = append(it.Labels, l.Name)
		}
		for _, f := range g.Files {
			it.Files = append(it.Files, f.Path)
		}
		switch {
		case kind == RecentMerged && g.MergedAt != nil:
			it.ClosedAt = *g.MergedAt
		case kind == RecentIssue && g.ClosedAt != nil:
			it.ClosedAt = *g.ClosedAt
		}
		if len(g.ClosedBy) > 0 {
			it.FixedBy = g.ClosedBy[0].Number
		}
		if !inPeriod(it, since) {
			continue
		}
		out = append(out, it)
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].At().After(out[b].At()) })
	return out, nil
}

// inPeriod reports whether an item belongs to the period starting at since.
func inPeriod(it RecentItem, since time.Time) bool {
	switch it.Kind {
	case RecentMerged:
		return !it.ClosedAt.IsZero() && !it.ClosedAt.Before(since)
	case RecentIssue:
		return !it.CreatedAt.Before(since) || (!it.ClosedAt.IsZero() && !it.ClosedAt.Before(since))
	}
	return true // an open pull request is open now, whenever it was opened
}

// recentError says in the views' terms why gh failed.
func recentError(repo string, err error) error {
	e := &RecentError{Repo: repo, Text: err.Error(), Err: err}
	var exit *ghpoll.ExitError
	switch {
	case errors.Is(err, ghpoll.ErrNotInstalled):
		e.Problem = RecentNoGH
	case errors.As(err, &exit):
		text := strings.ToLower(exit.Output())
		e.Text = strings.TrimSpace(exit.Output())
		e.Problem = classifyGH(text)
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		e.Problem = RecentOffline
	}
	return e
}

// classifyGH reads gh's own words.
func classifyGH(text string) RecentProblem {
	has := func(words ...string) bool {
		for _, w := range words {
			if strings.Contains(text, w) {
				return true
			}
		}
		return false
	}
	switch {
	case has("gh auth login", "not logged in", "authentication required", "http 401", "bad credentials"):
		return RecentSignedOut
	case has("could not resolve to a repository", "http 404", "not found"):
		return RecentNoAccess
	case has("error connecting", "dial tcp", "no such host", "timeout", "timed out", "network is unreachable", "connection refused"):
		return RecentOffline
	}
	return RecentOtherProblem
}
