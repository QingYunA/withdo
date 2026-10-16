package handoff

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// nasty is a prompt that tries every way out of a quoted string.
var nasty = []string{
	"plain",
	"it's a 'test'",
	`say "hi" and \ back`,
	"cost $HOME and $(rm -rf x) and ${PATH}",
	"`id` and !! and * and ?",
	"line one\nline two\n\nline four",
	"请完成 adhd-todo 里的任务 #44：交接 'Agent'\n开始时调用 report_progress（task_id: 44）",
	"-n --help",
	"",
	"tab\tand   spaces ",
}

func TestQuoteSurvivesAShell(t *testing.T) {
	for _, s := range nasty {
		out, err := exec.Command("/bin/sh", "-c", "printf %s "+Quote(s)).Output()
		if err != nil {
			t.Fatalf("%q: %v", s, err)
		}
		if string(out) != s {
			t.Errorf("Quote(%q) came back as %q", s, out)
		}
	}
	if got := Quote("a\x00b"); got != "'ab'" {
		t.Errorf("Quote with NUL = %q", got)
	}
}

func TestQuoteAllKeepsEveryWordApart(t *testing.T) {
	out, err := exec.Command("/bin/sh", "-c", "printf '%s|' "+QuoteAll(nasty)).Output()
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(nasty, "|") + "|"; string(out) != got {
		t.Errorf("got %q, want %q", out, got)
	}
}

func TestCommandPerAgent(t *testing.T) {
	cases := []struct {
		agent string
		want  []string
	}{
		{"claude", []string{"/bin/claude", "do it"}},
		{"codex", []string{"/bin/codex", "do it"}},
		{"gemini", []string{"/bin/gemini", "-i", "do it"}},
		{"qwen", []string{"/bin/qwen", "-i", "do it"}},
	}
	for _, c := range cases {
		got, ok := Command(c.agent, "/bin/"+c.agent, "do it")
		if !ok || !slices.Equal(got, c.want) {
			t.Errorf("Command(%s) = %q, %v; want %q", c.agent, got, ok, c.want)
		}
		if !Launchable(c.agent) {
			t.Errorf("%s should be launchable", c.agent)
		}
	}
	for _, agent := range []string{"cursor", "opencode", "kimi", "copilot", "windsurf", "zed", "", "nope"} {
		if got, ok := Command(agent, "/bin/x", "p"); ok || got != nil || Launchable(agent) {
			t.Errorf("%q should not be launchable, got %q", agent, got)
		}
	}
}

func TestScriptChangesDirectoryThenRunsTheAgent(t *testing.T) {
	got := Script(Request{
		Dir:      "/Users/me/my repo's",
		Argv:     []string{"/bin/claude", "fix 'it'\n$HOME"},
		PathDirs: []string{"/a b", "/c"},
	})
	want := "#!/bin/zsh -l\n" +
		"export PATH='/a b:/c':\"$PATH\"\n" +
		"cd '/Users/me/my repo'\\''s' || exit 1\n" +
		"exec '/bin/claude' 'fix '\\''it'\\''\n$HOME'\n"
	if got != want {
		t.Errorf("script =\n%s\nwant\n%s", got, want)
	}
	if strings.Contains(Script(Request{Dir: "/r", Argv: []string{"/bin/x"}}), "PATH") {
		t.Error("no PathDirs, no PATH line")
	}
}

func TestLaunchWritesAnExecutableScriptAndOpensIt(t *testing.T) {
	folder := filepath.Join(t.TempDir(), "tmp", "handoff")
	var opened []string
	term := Terminal{Folder: folder, Open: func(_ context.Context, f string) error {
		opened = append(opened, f)
		return nil
	}}
	req := Request{Dir: t.TempDir(), Argv: []string{"/bin/claude", nasty[6]}}
	if err := term.Launch(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if len(opened) != 1 || filepath.Dir(opened[0]) != folder || !strings.HasSuffix(opened[0], ".command") {
		t.Fatalf("opened %v", opened)
	}
	info, err := os.Stat(opened[0])
	if err != nil || info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("script mode = %v, %v; want executable", info, err)
	}
	if b, _ := os.ReadFile(opened[0]); string(b) != Script(req) {
		t.Errorf("script = %q", b)
	}
}

// The script, run for real, gets the prompt in one piece in the right folder.
func TestScriptRunsTheAgentWithTheExactPromptInTheRepo(t *testing.T) {
	if _, err := os.Stat("/bin/zsh"); err != nil {
		t.Skip("no zsh")
	}
	dir := t.TempDir()
	repo := filepath.Join(dir, "my repo's")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	stub := filepath.Join(dir, "stub")
	body := "#!/bin/sh\npwd -P > " + Quote(out) + "\nfor a in \"$@\"; do printf '[%s]' \"$a\" >> " + Quote(out) + "; done\n"
	if err := os.WriteFile(stub, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	term := Terminal{Folder: filepath.Join(dir, "scripts"), Open: func(ctx context.Context, f string) error {
		return exec.CommandContext(ctx, f).Run()
	}}
	prompt := nasty[6] + "\n`id` $HOME 'q'"
	if err := term.Launch(context.Background(), Request{Dir: repo, Argv: []string{stub, "-i", prompt}}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	real, _ := filepath.EvalSymlinks(repo)
	if want := real + "\n[-i][" + prompt + "]"; string(b) != want {
		t.Errorf("stub saw %q, want %q", b, want)
	}
}

func TestLaunchRemovesOldScriptsOnly(t *testing.T) {
	folder := t.TempDir()
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	old := filepath.Join(folder, "handoff-old.command")
	fresh := filepath.Join(folder, "handoff-fresh.command")
	other := filepath.Join(folder, "notes.txt")
	for _, f := range []string{old, fresh, other} {
		if err := os.WriteFile(f, []byte("x"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for f, age := range map[string]time.Duration{old: 25 * time.Hour, fresh: time.Hour, other: 48 * time.Hour} {
		if err := os.Chtimes(f, now.Add(-age), now.Add(-age)); err != nil {
			t.Fatal(err)
		}
	}
	term := Terminal{Folder: folder, Now: func() time.Time { return now }, Open: func(context.Context, string) error { return nil }}
	if err := term.Launch(context.Background(), Request{Dir: "/", Argv: []string{"/bin/x"}}); err != nil {
		t.Fatal(err)
	}
	for f, want := range map[string]bool{old: false, fresh: true, other: true} {
		if _, err := os.Stat(f); (err == nil) != want {
			t.Errorf("%s exists = %v, want %v", filepath.Base(f), err == nil, want)
		}
	}
}

func TestLaunchFailuresLeaveNoScript(t *testing.T) {
	folder := t.TempDir()
	boom := Terminal{Folder: folder, Open: func(context.Context, string) error { return os.ErrPermission }}
	if err := boom.Launch(context.Background(), Request{Dir: "/", Argv: []string{"/bin/x"}}); err == nil {
		t.Error("an open that fails should fail the launch")
	}
	if entries, _ := os.ReadDir(folder); len(entries) != 0 {
		t.Errorf("%d files left after a failed open", len(entries))
	}
	if err := boom.Launch(context.Background(), Request{Dir: "/"}); err == nil {
		t.Error("no command should fail")
	}
	file := filepath.Join(folder, "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := (Terminal{Folder: filepath.Join(file, "sub")}).Launch(context.Background(), Request{Dir: "/", Argv: []string{"x"}}); err == nil {
		t.Error("a folder below a file should fail")
	}
}
