package data

import (
	"context"
	"sync"
	"time"
)

// FakeRecent is a Recent that answers from what was set, for the demo and for tests. A repository
// it knows nothing of is one with no changes.
type FakeRecent struct {
	mu      sync.Mutex
	changes map[string]RepoChanges
	errs    map[string]error
	fetched []string
	// Block, when non-nil, holds every Fetch until it is closed or the request is cancelled.
	Block chan struct{}
}

var _ Recent = (*FakeRecent)(nil)

// NewFakeRecent starts with nothing to report.
func NewFakeRecent() *FakeRecent {
	return &FakeRecent{changes: map[string]RepoChanges{}, errs: map[string]error{}}
}

// Set replaces what a repository reports.
func (f *FakeRecent) Set(ch RepoChanges) {
	f.mu.Lock()
	f.changes[ch.Repo] = ch
	delete(f.errs, ch.Repo)
	f.mu.Unlock()
}

// Fail makes a repository fail with err, a *RecentError.
func (f *FakeRecent) Fail(repo string, err error) {
	f.mu.Lock()
	f.errs[repo] = err
	f.mu.Unlock()
}

// Fetched is the repository of every Fetch, in order.
func (f *FakeRecent) Fetched() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.fetched...)
}

// Fetch implements Recent: what was set, cut to the period like gh's answer is.
func (f *FakeRecent) Fetch(ctx context.Context, repo string, since time.Time) (RepoChanges, error) {
	f.mu.Lock()
	f.fetched = append(f.fetched, repo)
	block, ch, err := f.Block, f.changes[repo], f.errs[repo]
	f.mu.Unlock()
	if block != nil {
		select {
		case <-block:
		case <-ctx.Done():
			return RepoChanges{}, ctx.Err()
		}
	}
	if err != nil {
		return RepoChanges{}, err
	}
	out := RepoChanges{Repo: repo, Open: ch.Open}
	for _, i := range ch.Merged {
		if inPeriod(i, since) {
			out.Merged = append(out.Merged, i)
		}
	}
	for _, i := range ch.Issues {
		if inPeriod(i, since) {
			out.Issues = append(out.Issues, i)
		}
	}
	return out, nil
}
