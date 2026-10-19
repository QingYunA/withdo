package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/mcpserver"
	"github.com/QingYunA/adhd-todo/internal/store"
	"github.com/QingYunA/adhd-todo/internal/testutil"
)

type harness struct {
	out, err bytes.Buffer
	d        deps
}

func newHarness(t *testing.T, dbPath string) *harness {
	t.Helper()
	h := &harness{}
	h.d = deps{
		getenv: func(k string) string {
			if k == "ADHD_TODO_DB" {
				return dbPath
			}
			return ""
		},
		home:   t.TempDir(),
		stdout: &h.out,
		stderr: &h.err,
		serve:  func(context.Context, *mcp.Server) error { return nil },
	}
	return h
}

func TestVersionPrintsTheServersVersion(t *testing.T) {
	for _, arg := range []string{"--version", "version"} {
		h := newHarness(t, "")
		if code := run(context.Background(), []string{arg}, h.d); code != 0 {
			t.Fatalf("%s: exit %d, stderr %q", arg, code, h.err.String())
		}
		if want := "adhd-core " + mcpserver.Version + "\n"; h.out.String() != want {
			t.Errorf("%s printed %q, want %q", arg, h.out.String(), want)
		}
	}
}

func TestUsageAndHelp(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		code   int
		stdout string
		stderr string
	}{
		{"no command", nil, 2, "", "Usage: adhd-core"},
		{"unknown command", []string{"bogus"}, 2, "", "Unknown command: bogus"},
		{"serve is gone", []string{"serve"}, 2, "", "Unknown command: serve"},
		{"--help", []string{"--help"}, 0, "overview", ""},
		{"-h", []string{"-h"}, 0, "mcp", ""},
		{"--help lists guide", []string{"--help"}, 0, "guide ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, filepath.Join(t.TempDir(), "x.db"))
			if code := run(context.Background(), tt.args, h.d); code != tt.code {
				t.Errorf("exit code = %d, want %d", code, tt.code)
			}
			if !strings.Contains(h.out.String(), tt.stdout) || !strings.Contains(h.err.String(), tt.stderr) {
				t.Errorf("stdout = %q, stderr = %q", h.out.String(), h.err.String())
			}
			if tt.code == 2 && h.out.Len() != 0 {
				t.Errorf("usage errors must not write stdout: %q", h.out.String())
			}
		})
	}
}

func TestHelpWordAndSubcommandHelpFlagsPrintUsageAndDoNothingElse(t *testing.T) {
	for _, args := range [][]string{
		{"help"}, {"overview", "--help"}, {"overview", "-h"}, {"mcp", "--help"}, {"mcp", "-h"}, {"guide", "--help"}, {"guide", "-h"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			h := newHarness(t, filepath.Join(t.TempDir(), "never-opened.db"))
			served := false
			h.d.serve = func(context.Context, *mcp.Server) error { served = true; return nil }
			if code := run(context.Background(), args, h.d); code != 0 {
				t.Errorf("exit code = %d, stderr = %q", code, h.err.String())
			}
			if out := h.out.String(); out != usage {
				t.Errorf("stdout = %q, want the usage", out)
			}
			if served || h.err.Len() != 0 {
				t.Errorf("the command ran: served = %v, stderr = %q", served, h.err.String())
			}
			if _, err := os.Stat(h.d.getenv("ADHD_TODO_DB")); err == nil {
				t.Error("the database was opened")
			}
		})
	}
}

func TestGuidePrintsTheSetupGuideWithTheCorePath(t *testing.T) {
	h := newHarness(t, filepath.Join(t.TempDir(), "never-opened.db"))
	if code := run(context.Background(), []string{"guide"}, h.d); code != 0 {
		t.Fatalf("exit code %d: %s", code, h.err.String())
	}
	out := h.out.String()
	for _, want := range []string{filepath.Join(h.d.home, ".local", "bin", "adhd-core"), `"mcp"`, "get_overview", "report_progress"} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "{{") || h.err.Len() != 0 {
		t.Errorf("placeholder left or stderr %q", h.err.String())
	}
	if _, err := os.Stat(h.d.getenv("ADHD_TODO_DB")); err == nil {
		t.Error("guide opened the database")
	}
}

func TestOverviewOnEmptyDatabase(t *testing.T) {
	h := newHarness(t, filepath.Join(t.TempDir(), "empty.db"))
	if code := run(context.Background(), []string{"overview"}, h.d); code != 0 {
		t.Fatalf("exit code %d: %s", code, h.err.String())
	}
	if !strings.Contains(h.out.String(), "No active tasks") {
		t.Errorf("stdout = %q", h.out.String())
	}
}

func TestOverviewListsActiveTasks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.db")
	repo := testutil.NewRepo(t, t.TempDir(), "widgets", testutil.RepoOptions{Remote: "git@github.com:acme/widgets.git", Branch: "feat/login"})
	db, err := store.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = domain.New(db, domain.Options{}).ReportProgress(context.Background(), domain.ProgressInput{
		Summary: "wiring up login", Cwd: repo, Agent: "claude-code", NextStep: "write tests"})
	_ = db.Close()
	if err != nil {
		t.Fatal(err)
	}

	h := newHarness(t, path)
	if code := run(context.Background(), []string{"overview"}, h.d); code != 0 {
		t.Fatalf("exit code %d: %s", code, h.err.String())
	}
	text := h.out.String()
	for _, want := range []string{"widgets (acme/widgets)", "feat/login", "wiring up login", "next: write tests"} {
		if !strings.Contains(text, want) {
			t.Errorf("overview lacks %q:\n%s", want, text)
		}
	}
	if !strings.HasSuffix(text, "\n") {
		t.Error("overview should end with a newline")
	}
}

func TestOverviewPrintsLocalTimes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.db")
	db, err := store.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 18, 3, 55, 24, 502e6, time.UTC)
	_, err = domain.New(db, domain.Options{Now: func() time.Time { return at }}).ReportProgress(context.Background(),
		domain.ProgressInput{Summary: "wiring up login", Agent: "claude-code"})
	_ = db.Close()
	if err != nil {
		t.Fatal(err)
	}

	h := newHarness(t, path)
	h.d.location = time.FixedZone("CST", 8*3600)
	if code := run(context.Background(), []string{"overview"}, h.d); code != 0 {
		t.Fatalf("exit code %d: %s", code, h.err.String())
	}
	if want := "(claude-code, 2026-10-18T11:55:24+08:00)"; !strings.Contains(h.out.String(), want) {
		t.Errorf("overview lacks %q:\n%s", want, h.out.String())
	}
}

func TestMCPCommandServesAndReportsServeErrors(t *testing.T) {
	h := newHarness(t, filepath.Join(t.TempDir(), "m.db"))
	served := false
	h.d.serve = func(_ context.Context, s *mcp.Server) error { served = s != nil; return nil }
	if code := run(context.Background(), []string{"mcp"}, h.d); code != 0 || !served {
		t.Errorf("code = %d, served = %v", code, served)
	}
	if h.out.Len() != 0 {
		t.Errorf("mcp must keep stdout for the protocol, got %q", h.out.String())
	}

	h = newHarness(t, filepath.Join(t.TempDir(), "m.db"))
	h.d.serve = func(context.Context, *mcp.Server) error { return errors.New("pipe broke") }
	if code := run(context.Background(), []string{"mcp"}, h.d); code != 1 || !strings.Contains(h.err.String(), "pipe broke") {
		t.Errorf("code = %d, stderr = %q", code, h.err.String())
	}
}

func TestDatabaseOpenFailureExitsOne(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, filepath.Join(blocker, "sub", "x.db"))
	if code := run(context.Background(), []string{"overview"}, h.d); code != 1 || !strings.HasPrefix(h.err.String(), "adhd-core:") {
		t.Errorf("code = %d, stderr = %q", code, h.err.String())
	}
}

// TestBuiltBinaryServesMCPOverStdio runs the real binary the way Claude Code
// does: spawn it, speak MCP over its stdio, close stdin and expect it to exit.
func TestBuiltBinaryServesMCPOverStdio(t *testing.T) {
	testutil.IsolateGit(t)
	bin := filepath.Join(t.TempDir(), "adhd-core")
	if runtime.GOOS == "windows" {
		bin += ".exe" // Windows LookPath requires the extension
	}
	if out, err := exec.Command("go", "build", "-o", bin, "../../cmd/adhd-core").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	dbPath := filepath.Join(t.TempDir(), "stdio.db")
	repo := testutil.NewRepo(t, t.TempDir(), "widgets", testutil.RepoOptions{Remote: "git@github.com:acme/widgets.git", Branch: "feat/stdio"})

	cmd := exec.Command(bin, "mcp")
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "ADHD_TODO_DB="+dbPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "smoke", Version: "0"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("connect: %v\n%s", err, stderr.String())
	}
	tools, err := session.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != 10 {
		t.Fatalf("tools = %v, err = %v", tools, err)
	}
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "report_progress",
		Arguments: map[string]any{"summary": "smoke", "title": "Smoke task"}})
	if err != nil || res.IsError {
		t.Fatalf("report_progress: %v %+v", err, res)
	}
	_ = session.Close() // closes stdin; the server must exit on its own

	h := newHarness(t, dbPath)
	if code := run(context.Background(), []string{"overview"}, h.d); code != 0 || !strings.Contains(h.out.String(), "Smoke task") {
		t.Errorf("task not persisted: code %d, %q", code, h.out.String())
	}
	if !strings.Contains(stderr.String(), "ready on stdio") {
		t.Errorf("stderr = %q", stderr.String())
	}
}

func TestDefaultDepsUseTheProcessEnvironment(t *testing.T) {
	t.Setenv("ADHD_TODO_DB", "/tmp/from-env.db")
	d := defaultDeps()
	if d.getenv("ADHD_TODO_DB") != "/tmp/from-env.db" || d.stdout != os.Stdout || d.stderr != os.Stderr || d.serve == nil {
		t.Errorf("defaultDeps = %+v", d)
	}
}

func TestIsCommandKnowsEveryCommandRunAnswersTo(t *testing.T) {
	for _, arg := range []string{"mcp", "overview", "guide", "version", "--version", "help", "-h", "--help"} {
		if !IsCommand(arg) {
			t.Errorf("IsCommand(%q) = false", arg)
		}
	}
	for _, arg := range []string{"", "-fake", "-psn_0_12345", "-NSDocumentRevisionsDebugMode", "bogus"} {
		if IsCommand(arg) {
			t.Errorf("IsCommand(%q) = true", arg)
		}
	}
}
