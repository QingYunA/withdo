package data

import "github.com/QingYunA/adhd-todo/internal/ghpoll"

// PollerGitHub is the GitHub the app runs on: the pull request poller of internal/ghpoll.
type PollerGitHub struct{ Poller *ghpoll.Poller }

var _ GitHub = PollerGitHub{}

// Status implements GitHub.
func (g PollerGitHub) Status() GitHubStatus {
	st := g.Poller.Status()
	return GitHubStatus{State: githubState(st.State), Account: st.Account, CheckedAt: st.CheckedAt, Checking: st.Checking}
}

// CheckNow implements GitHub.
func (g PollerGitHub) CheckNow() { g.Poller.CheckNow() }

// Activate implements GitHub.
func (g PollerGitHub) Activate() { g.Poller.Activate() }

// Subscribe implements GitHub.
func (g PollerGitHub) Subscribe(fn func()) func() { return g.Poller.Subscribe(fn) }

func githubState(s ghpoll.AuthState) GitHubState {
	switch s {
	case ghpoll.AuthReady:
		return GitHubReady
	case ghpoll.AuthMissing:
		return GitHubMissing
	case ghpoll.AuthSignedOut:
		return GitHubSignedOut
	case ghpoll.AuthOffline:
		return GitHubOffline
	}
	return GitHubUnknown
}
