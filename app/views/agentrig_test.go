package views

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/agentsetup"
)

// agentRig is this Mac as the tests make it: a temporary home folder, a PATH of its own and a
// command runner that starts nothing. Nothing of the real machine is looked at or touched.
type agentRig struct {
	t    *testing.T
	home string
	env  *agentsetup.Env

	mu    sync.Mutex
	path  map[string]string                                // the commands on PATH
	calls [][]string                                       // the commands that were asked to run
	run   func(path string, args []string) ([]byte, error) // what running one does; nil: succeeds quietly
	gate  chan struct{}                                    // when set, a command waits for it
}

func newAgentRig(t *testing.T) *agentRig {
	t.Helper()
	r := &agentRig{t: t, home: t.TempDir(), path: map[string]string{}}
	r.env = &agentsetup.Env{Home: r.home, LookPath: r.lookPath, Run: r.runCommand}
	r.write(".local/bin/adhd-core", "#!/bin/sh\n")
	return r
}

func (r *agentRig) lookPath(name string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.path[name]; ok {
		return p, nil
	}
	return "", exec.ErrNotFound
}

func (r *agentRig) runCommand(ctx context.Context, path string, args ...string) ([]byte, error) {
	r.mu.Lock()
	r.calls = append(r.calls, append([]string{path}, args...))
	gate, run := r.gate, r.run
	r.mu.Unlock()
	if gate != nil {
		select {
		case <-gate:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if run != nil {
		return run(path, args)
	}
	return nil, nil
}

// has puts the commands on PATH.
func (r *agentRig) has(names ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, n := range names {
		r.path[n] = "/usr/local/bin/" + n
	}
}

func (r *agentRig) write(rel, content string) {
	r.t.Helper()
	p := filepath.Join(r.home, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

func (r *agentRig) commands() [][]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([][]string(nil), r.calls...)
}
