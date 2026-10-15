package ghpoll

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestPollRunsHousekeepingWhateverGitHubSays(t *testing.T) {
	w := newWorld(t)
	var runs atomic.Int32
	w.auth = func() (Output, error) {
		return Output{}, &ExitError{Code: 1, Stderr: []byte("You are not logged into any GitHub hosts")}
	}
	w.poller = New(Config{Runner: w.runner, Store: w.core, Now: w.now, Interval: time.Hour,
		Housekeeping: func(context.Context) { runs.Add(1) }})

	w.poller.Poll(bg)
	if got := runs.Load(); got != 1 {
		t.Errorf("housekeeping ran %d times after one poll, want 1", got)
	}
	w.poller.Poll(bg)
	if got := runs.Load(); got != 2 {
		t.Errorf("housekeeping ran %d times after two polls, want 2", got)
	}
}

func TestPollWithoutHousekeepingIsFine(t *testing.T) {
	w := newWorld(t)
	w.poller.Poll(bg) // no Housekeeping configured: nothing to run, nothing to break
}
