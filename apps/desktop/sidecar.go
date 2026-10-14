package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

// Endpoint is where adhd-core listens, from its one stdout line {"port":n,"token":"..."}.
type Endpoint struct {
	Port  int    `json:"port"`
	Token string `json:"token"`
}

// SidecarConfig says how to run `adhd-core serve`.
type SidecarConfig struct {
	Path         string
	Args         []string
	Env          []string // extra KEY=VALUE pairs on top of the app's environment
	ReadyTimeout time.Duration
	MaxStarts    int // starts before giving up; 2 means one restart
	Logf         func(format string, args ...any)
	OnFailed     func(reason string) // called once, when the last start has failed
}

type sidecarState int

const (
	stateStarting sidecarState = iota
	stateReady
	stateFailed
	stateStopped
)

var errStopped = errors.New("adhd-core was stopped")

// Sidecar runs adhd-core as a child process, tells callers where it listens, restarts it once
// when it dies, and kills it on Stop.
type Sidecar struct {
	cfg SidecarConfig

	mu       sync.Mutex
	state    sidecarState
	endpoint Endpoint
	reason   string
	changed  chan struct{} // closed and replaced on every state change
	cur      *proc
	starts   int
	done     chan struct{} // closed when the supervisor returns
}

// NewSidecar applies defaults; call Start to run it.
func NewSidecar(cfg SidecarConfig) *Sidecar {
	if cfg.ReadyTimeout <= 0 {
		cfg.ReadyTimeout = 15 * time.Second
	}
	if cfg.MaxStarts <= 0 {
		cfg.MaxStarts = 2
	}
	if cfg.Logf == nil {
		cfg.Logf = func(string, ...any) {}
	}
	return &Sidecar{cfg: cfg, changed: make(chan struct{}), done: make(chan struct{})}
}

// Start supervises the process in the background.
func (s *Sidecar) Start() { go s.supervise() }

// Starts is how many times the process was started so far.
func (s *Sidecar) Starts() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.starts
}

// Wait blocks until adhd-core is ready, has failed for good, or ctx ends.
func (s *Sidecar) Wait(ctx context.Context) (Endpoint, error) {
	for {
		s.mu.Lock()
		state, endpoint, reason, changed := s.state, s.endpoint, s.reason, s.changed
		s.mu.Unlock()
		switch state {
		case stateReady:
			return endpoint, nil
		case stateFailed:
			return Endpoint{}, errors.New(reason)
		case stateStopped:
			return Endpoint{}, errStopped
		}
		select {
		case <-changed:
		case <-ctx.Done():
			return Endpoint{}, ctx.Err()
		}
	}
}

// Stop kills the process and keeps the supervisor from restarting it. It waits briefly for the
// process to be gone, so nothing outlives the app.
func (s *Sidecar) Stop() {
	s.mu.Lock()
	if s.state != stateFailed {
		s.setLocked(stateStopped, "")
	}
	p := s.cur
	s.mu.Unlock()
	if p != nil {
		p.kill()
		select {
		case <-p.exited:
		case <-time.After(3 * time.Second):
		}
	}
	select {
	case <-s.done:
	case <-time.After(3 * time.Second):
	}
}

func (s *Sidecar) setLocked(state sidecarState, reason string) {
	s.state, s.reason = state, reason
	close(s.changed)
	s.changed = make(chan struct{})
}

func (s *Sidecar) publish(state sidecarState, ep Endpoint, reason string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == stateStopped {
		return false
	}
	s.endpoint = ep
	s.setLocked(state, reason)
	return true
}

// setCurrent registers the running process, or refuses when Stop has been called.
func (s *Sidecar) setCurrent(p *proc) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == stateStopped {
		return false
	}
	s.cur = p
	s.starts++
	return true
}

func (s *Sidecar) stopped() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state == stateStopped
}

func (s *Sidecar) supervise() {
	defer close(s.done)
	var previous *Endpoint
	reason := "adhd-core did not start"
	for attempt := 1; attempt <= s.cfg.MaxStarts; attempt++ {
		if !s.publish(stateStarting, Endpoint{}, "") {
			return
		}
		out := s.runOnce(previous)
		if s.stopped() {
			return
		}
		if out.endpoint != nil {
			previous = out.endpoint
		}
		reason = out.reason
		s.cfg.Logf("adhd-core stopped (start %d of %d): %s", attempt, s.cfg.MaxStarts, reason)
	}
	if s.publish(stateFailed, Endpoint{}, reason) && s.cfg.OnFailed != nil {
		s.cfg.OnFailed(reason)
	}
}

// restartEnv makes a restart reuse the port and token, so the page's cached endpoint stays valid
// and its EventSource just reconnects.
func restartEnv(previous *Endpoint) []string {
	if previous == nil {
		return nil
	}
	return []string{fmt.Sprintf("ADHD_CORE_PORT=%d", previous.Port), "ADHD_CORE_TOKEN=" + previous.Token}
}

func environ(extra ...[]string) []string {
	env := os.Environ()
	for _, e := range extra {
		env = append(env, e...)
	}
	return env
}
