package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// TestMain doubles as the fake sidecar: the manager runs this very test binary with
// FAKE_SIDECAR set, and FAKE_MODE picks the behaviour.
func TestMain(m *testing.M) {
	if os.Getenv("FAKE_SIDECAR") != "" {
		fakeSidecar()
		return
	}
	os.Exit(m.Run())
}

func fakeSidecar() {
	dir := os.Getenv("FAKE_DIR")
	_ = os.WriteFile(filepath.Join(dir, "pid"), []byte(strconv.Itoa(os.Getpid())), 0o600)
	port, token := "4317", "tok"
	if p := os.Getenv("ADHD_CORE_PORT"); p != "" {
		port, token = p, os.Getenv("ADHD_CORE_TOKEN")
		_ = os.WriteFile(filepath.Join(dir, "reused"), []byte(port+" "+token), 0o600)
	}
	ready := fmt.Sprintf(`{"port":%s,"token":%q}`, port, token)
	switch os.Getenv("FAKE_MODE") {
	case "silent":
		fmt.Fprintln(os.Stderr, "booting slowly")
		waitForStdinEOF()
	case "junk":
		fmt.Println("starting up")
		fmt.Println(ready)
		waitForStdinEOF()
	case "crash":
		fmt.Println(ready)
		time.Sleep(30 * time.Millisecond)
		os.Exit(3)
	case "crash-once":
		marker := filepath.Join(dir, "crashed")
		if _, err := os.Stat(marker); err != nil {
			_ = os.WriteFile(marker, nil, 0o600)
			fmt.Println(ready)
			time.Sleep(30 * time.Millisecond)
			os.Exit(3)
		}
		fmt.Println(ready)
		waitForStdinEOF()
	default:
		fmt.Println(ready)
		waitForStdinEOF()
	}
}

func waitForStdinEOF() {
	buf := make([]byte, 64)
	for {
		if _, err := os.Stdin.Read(buf); err != nil {
			return
		}
	}
}

type fakeEnv struct {
	dir    string
	failed chan string
	logs   *lockedLog
}

type lockedLog struct {
	mu    sync.Mutex
	lines []string
}

func (l *lockedLog) add(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines = append(l.lines, fmt.Sprintf(format, args...))
}

func (l *lockedLog) joined() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return strings.Join(l.lines, "\n")
}

func newFake(t *testing.T, mode string, timeout time.Duration, maxStarts ...int) (*Sidecar, fakeEnv) {
	t.Helper()
	env := fakeEnv{dir: t.TempDir(), failed: make(chan string, 1), logs: &lockedLog{}}
	max := 0
	if len(maxStarts) > 0 {
		max = maxStarts[0]
	}
	s := NewSidecar(SidecarConfig{
		MaxStarts:    max,
		Path:         os.Args[0],
		Env:          []string{"FAKE_SIDECAR=1", "FAKE_MODE=" + mode, "FAKE_DIR=" + env.dir},
		ReadyTimeout: timeout,
		Logf:         env.logs.add,
		OnFailed:     func(reason string) { env.failed <- reason },
	})
	t.Cleanup(s.Stop)
	return s, env
}

func waitCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func readPID(t *testing.T, dir string) int {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "pid"))
	if err != nil {
		t.Fatalf("fake sidecar never wrote its pid: %v", err)
	}
	pid, _ := strconv.Atoi(string(raw))
	return pid
}

func alive(pid int) bool { return syscall.Kill(pid, 0) == nil }

func TestParseReadyLine(t *testing.T) {
	cases := []struct {
		line string
		want Endpoint
		ok   bool
	}{
		{`{"port":52341,"token":"abc"}` + "\n", Endpoint{52341, "abc"}, true},
		{`  {"port":1,"token":"x"}  `, Endpoint{1, "x"}, true},
		{"listening on 127.0.0.1", Endpoint{}, false},
		{"", Endpoint{}, false},
		{`{"port":0,"token":"abc"}`, Endpoint{}, false},
		{`{"port":5}`, Endpoint{}, false},
	}
	for _, c := range cases {
		got, ok := parseReadyLine(c.line)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("parseReadyLine(%q) = %v, %v; want %v, %v", c.line, got, ok, c.want, c.ok)
		}
	}
}

func TestReadyLineIsParsed(t *testing.T) {
	s, env := newFake(t, "ok", 5*time.Second)
	s.Start()
	ep, err := s.Wait(waitCtx(t))
	if err != nil || ep != (Endpoint{4317, "tok"}) {
		t.Fatalf("Wait = %v, %v", ep, err)
	}
	if !alive(readPID(t, env.dir)) {
		t.Fatal("sidecar should be running")
	}
	if s.Starts() != 1 {
		t.Fatalf("starts = %d, want 1", s.Starts())
	}
}

func TestNoiseBeforeTheReadyLineIsIgnored(t *testing.T) {
	s, _ := newFake(t, "junk", 5*time.Second)
	s.Start()
	ep, err := s.Wait(waitCtx(t))
	if err != nil || ep.Port != 4317 {
		t.Fatalf("Wait = %v, %v", ep, err)
	}
}

func TestTimeoutWithoutReadyLineFails(t *testing.T) {
	s, env := newFake(t, "silent", 200*time.Millisecond, 1)
	s.Start()
	_, err := s.Wait(waitCtx(t))
	if err == nil || !strings.Contains(err.Error(), "did not report a port") {
		t.Fatalf("Wait error = %v", err)
	}
	select {
	case <-env.failed:
	case <-time.After(2 * time.Second):
		t.Fatal("OnFailed was not called")
	}
	if alive(readPID(t, env.dir)) {
		t.Fatal("a sidecar that timed out must be killed")
	}
	if !strings.Contains(env.logs.joined(), "[adhd-core] booting slowly") {
		t.Fatalf("stderr was not logged: %q", env.logs.joined())
	}
}

func TestRestartsOnceOnTheSamePortAndToken(t *testing.T) {
	s, env := newFake(t, "crash-once", 5*time.Second)
	s.Start()
	ctx := waitCtx(t)
	if _, err := s.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	// The first run dies after 30 ms; wait until the second one is up.
	deadline := time.Now().Add(5 * time.Second)
	for s.Starts() < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	ep, err := s.Wait(ctx)
	if err != nil || ep != (Endpoint{4317, "tok"}) {
		t.Fatalf("after restart Wait = %v, %v", ep, err)
	}
	if s.Starts() != 2 {
		t.Fatalf("starts = %d, want 2", s.Starts())
	}
	reused, err := os.ReadFile(filepath.Join(env.dir, "reused"))
	if err != nil || string(reused) != "4317 tok" {
		t.Fatalf("restart did not reuse port and token: %q, %v", reused, err)
	}
}

func TestSecondFailureIsFinal(t *testing.T) {
	s, env := newFake(t, "crash", 5*time.Second)
	s.Start()
	select {
	case reason := <-env.failed:
		if !strings.Contains(reason, "exit status 3") {
			t.Fatalf("reason = %q", reason)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("OnFailed was not called")
	}
	if _, err := s.Wait(waitCtx(t)); err == nil {
		t.Fatal("Wait should report the failure")
	}
	if s.Starts() != 2 {
		t.Fatalf("starts = %d, want 2 (one restart, no more)", s.Starts())
	}
}

func TestStopKillsTheSidecarAndDoesNotRestart(t *testing.T) {
	s, env := newFake(t, "ok", 5*time.Second)
	s.Start()
	if _, err := s.Wait(waitCtx(t)); err != nil {
		t.Fatal(err)
	}
	pid := readPID(t, env.dir)
	s.Stop()
	if alive(pid) {
		t.Fatal("sidecar still running after Stop")
	}
	time.Sleep(100 * time.Millisecond)
	if s.Starts() != 1 {
		t.Fatalf("Stop must not trigger a restart, starts = %d", s.Starts())
	}
	if _, err := s.Wait(waitCtx(t)); err == nil {
		t.Fatal("Wait after Stop should fail")
	}
}

func TestWaitHonoursContext(t *testing.T) {
	s, _ := newFake(t, "silent", 5*time.Second)
	s.Start()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := s.Wait(ctx); err != context.DeadlineExceeded {
		t.Fatalf("Wait = %v, want deadline exceeded", err)
	}
}

func TestMissingBinaryFails(t *testing.T) {
	failed := make(chan string, 1)
	s := NewSidecar(SidecarConfig{Path: "/nonexistent/adhd-core", OnFailed: func(r string) { failed <- r }})
	s.Start()
	select {
	case r := <-failed:
		if !strings.Contains(r, "cannot start adhd-core") {
			t.Fatalf("reason = %q", r)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("OnFailed was not called")
	}
}
