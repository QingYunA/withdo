package syncserver_test

import (
	"bufio"
	"context"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

// sseStream reads one /v1/events response in the background: comments (connected, keepalive)
// are ignored, and every complete event is put on events.
type sseStream struct {
	resp   *http.Response
	ready  chan struct{}
	events chan string
	done   chan struct{}
}

func (h *harness) stream(t *testing.T, token, deviceID string) *sseStream {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		h.http.URL+"/v1/events?device_id="+url.QueryEscape(deviceID), nil)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := h.http.Client().Do(req)
	if err != nil {
		cancel()
		t.Fatalf("open event stream: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		cancel()
		_ = resp.Body.Close()
		t.Fatalf("open event stream = %d, want 200", resp.StatusCode)
	}
	s := &sseStream{
		resp:   resp,
		ready:  make(chan struct{}),
		events: make(chan string, 16),
		done:   make(chan struct{}),
	}
	go s.read()
	t.Cleanup(func() {
		cancel()
		_ = resp.Body.Close()
		select {
		case <-s.done:
		case <-time.After(time.Second):
		}
	})
	return s
}

func (s *sseStream) read() {
	defer close(s.done)
	var once sync.Once
	event := ""
	scanner := bufio.NewScanner(s.resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, ":"):
			if strings.Contains(line, "connected") {
				once.Do(func() { close(s.ready) })
			}
		case strings.HasPrefix(line, "event:"):
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case line == "":
			if event != "" {
				s.events <- event
				event = ""
			}
		}
	}
}

func (s *sseStream) waitReady(t *testing.T) {
	t.Helper()
	select {
	case <-s.ready:
	case <-time.After(2 * time.Second):
		t.Fatal("event stream did not become ready")
	}
}

func (s *sseStream) waitEvent(t *testing.T, want string, timeout time.Duration) {
	t.Helper()
	select {
	case got := <-s.events:
		if got != want {
			t.Fatalf("event = %q, want %q", got, want)
		}
	case <-time.After(timeout):
		t.Fatalf("no %q event within %s", want, timeout)
	}
}

func (s *sseStream) expectNoEvent(t *testing.T, timeout time.Duration) {
	t.Helper()
	select {
	case got := <-s.events:
		t.Fatalf("unexpected event %q", got)
	case <-time.After(timeout):
	}
}
