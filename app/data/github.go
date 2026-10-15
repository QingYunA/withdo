package data

import (
	"sync"
	"time"
)

// PRState is where a task's pull request stands, as the poller last saw it.
type PRState string

// Pull request states. Empty means the pull request was never checked.
const (
	PROpen   PRState = "open"
	PRDraft  PRState = "draft"
	PRMerged PRState = "merged"
	PRClosed PRState = "closed" // without merging
)

// PRChecks sums up the CI checks of a pull request.
type PRChecks string

// Check summaries.
const (
	PRChecksPassing PRChecks = "passing"
	PRChecksFailing PRChecks = "failing"
	PRChecksPending PRChecks = "pending"
	PRChecksNone    PRChecks = "none"
)

// PRReview is the review decision on a pull request.
type PRReview string

// Review decisions.
const (
	PRReviewApproved         PRReview = "approved"
	PRReviewChangesRequested PRReview = "changes_requested"
	PRReviewRequired         PRReview = "review_required"
	PRReviewNone             PRReview = "none"
)

// GitHubAgent is the agent name of the reports the poller files.
const GitHubAgent = "github"

// PRClosedUnseen reports whether the pull request was closed without merging and the user has not
// dismissed that yet.
func (t Task) PRClosedUnseen() bool { return t.PRState == PRClosed && t.PRClosedSeenAt == nil }

// PRChecksFailing reports whether the checks fail on an open pull request. A draft is work in
// progress, so its failures are not news.
func (t Task) PRChecksFailing() bool { return t.PRState == PROpen && t.PRChecks == PRChecksFailing }

// PRChangesRequested reports whether a reviewer asked for changes on an open pull request.
func (t Task) PRChangesRequested() bool {
	return t.PRState == PROpen && t.PRReview == PRReviewChangesRequested
}

// GitHubState says whether pull requests can be followed.
type GitHubState int

const (
	GitHubUnknown   GitHubState = iota // the first check has not ended
	GitHubReady                        // gh is installed and signed in
	GitHubMissing                      // there is no gh
	GitHubSignedOut                    // gh has no login
	GitHubOffline                      // GitHub could not be reached
)

// GitHubStatus is what Settings > GitHub shows.
type GitHubStatus struct {
	State     GitHubState
	Account   string
	CheckedAt time.Time // when the last check ended; zero before the first
	Checking  bool
}

// GitHub is the pull request poller, as Settings sees it. nil leaves the section out.
type GitHub interface {
	Status() GitHubStatus
	// CheckNow asks for a check at once. It returns without waiting.
	CheckNow()
	// Activate is called when the window comes back to the front: a check starts when the last
	// one is old enough.
	Activate()
	// Subscribe calls fn, on an arbitrary goroutine, when the status changes.
	Subscribe(fn func()) (cancel func())
}

// FakeGitHub is a GitHub with a status set by hand, for the demo and for tests.
type FakeGitHub struct {
	mu     sync.Mutex
	status GitHubStatus
	subs   []func()
	// Checks counts the CheckNow calls, Activations the Activate calls.
	Checks, Activations int
}

var _ GitHub = (*FakeGitHub)(nil)

// NewFakeGitHub starts with status.
func NewFakeGitHub(status GitHubStatus) *FakeGitHub { return &FakeGitHub{status: status} }

// Set replaces the status and tells the subscribers.
func (g *FakeGitHub) Set(status GitHubStatus) {
	g.mu.Lock()
	g.status = status
	subs := append([]func(){}, g.subs...)
	g.mu.Unlock()
	for _, fn := range subs {
		fn()
	}
}

func (g *FakeGitHub) Status() GitHubStatus {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.status
}

func (g *FakeGitHub) CheckNow() {
	g.mu.Lock()
	g.Checks++
	g.mu.Unlock()
}

func (g *FakeGitHub) Activate() {
	g.mu.Lock()
	g.Activations++
	g.mu.Unlock()
}

func (g *FakeGitHub) Subscribe(fn func()) func() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.subs = append(g.subs, fn)
	return func() {}
}
