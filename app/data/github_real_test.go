package data

import (
	"context"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/ghpoll"
)

type stubRunner struct{ out ghpoll.Output }

func (s stubRunner) Run(context.Context, ...string) (ghpoll.Output, error) { return s.out, nil }

func TestPollerGitHubReportsThePollersStatus(t *testing.T) {
	_, core := realPair(t, nil)
	checked := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)
	p := ghpoll.New(ghpoll.Config{
		Runner: stubRunner{ghpoll.Output{Stdout: []byte("github.com\n  ✓ Logged in to github.com account QingYunA (keyring)\n")}},
		Store:  core, Now: func() time.Time { return checked },
	})
	g := PollerGitHub{Poller: p}
	if st := g.Status(); st.State != GitHubUnknown || !st.CheckedAt.IsZero() {
		t.Errorf("before the first check: %+v", st)
	}
	notified := make(chan struct{}, 4)
	cancel := g.Subscribe(func() { notified <- struct{}{} })
	defer cancel()
	p.Poll(context.Background())
	<-notified
	st := g.Status()
	if st.State != GitHubReady || st.Account != "QingYunA" || !st.CheckedAt.Equal(checked) || st.Checking {
		t.Errorf("after: %+v", st)
	}
}

func TestGitHubStatesMapOneToOne(t *testing.T) {
	for in, want := range map[ghpoll.AuthState]GitHubState{
		ghpoll.AuthUnknown: GitHubUnknown, ghpoll.AuthReady: GitHubReady, ghpoll.AuthMissing: GitHubMissing,
		ghpoll.AuthSignedOut: GitHubSignedOut, ghpoll.AuthOffline: GitHubOffline,
	} {
		if got := githubState(in); got != want {
			t.Errorf("%v -> %v, want %v", in, got, want)
		}
	}
}
