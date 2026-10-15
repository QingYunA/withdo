package ghpoll

import (
	"context"
	"strings"
	"sync"
)

// fakeRunner answers gh calls from a function and records them.
type fakeRunner struct {
	mu    sync.Mutex
	calls []string
	reply func(args []string) (Output, error)
}

func (f *fakeRunner) Run(ctx context.Context, args ...string) (Output, error) {
	f.mu.Lock()
	f.calls = append(f.calls, strings.Join(args, " "))
	f.mu.Unlock()
	return f.reply(args)
}

func (f *fakeRunner) callsTo(prefix string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.calls {
		if strings.HasPrefix(c, prefix) {
			n++
		}
	}
	return n
}
