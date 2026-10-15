package views

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/internal/agentsetup"
)

// agentsRig is the Agents pane open, on a Mac with a few agents: Claude Code and Codex with the
// MCP installed, Gemini CLI, Cursor, opencode and VS Code found, the rest not there.
func agentsRig(t *testing.T) *settingsRig {
	t.Helper()
	r := newSettingsRig(t)
	r.mac.has("claude", "codex", "gemini", "opencode", "code")
	r.mac.write(".claude.json", `{"mcpServers":{"adhd-todo":{"command":"x"}}}`)
	r.mac.write(".codex/config.toml", "[mcp_servers.adhd-todo]\ncommand = \"x\"\n")
	r.mac.write(".cursor/mcp.json", "{}")
	r.pane(t, "Agents")
	return r
}

func TestAgentsPaneListsTheTenAgentsInTwoColumns(t *testing.T) {
	r := agentsRig(t)
	has(t, r.tt, "Agents on this Mac", "adds adhd-todo's MCP server to that agent", "does not install or start the agent")
	names := []string{"Claude Code", "Codex", "Gemini CLI", "Cursor", "opencode", "Copilot (VS Code)", "Qwen Code", "Kimi CLI", "Windsurf", "Zed"}
	var ys, xs []float32
	for _, n := range names {
		f, ok := r.tt.Find(n)
		if !ok {
			t.Fatalf("no row for %s", n)
		}
		ys, xs = append(ys, f.Y), append(xs, f.X)
	}
	for i := range names {
		if i%2 == 0 && xs[i] >= xs[i+1] {
			t.Errorf("%s (x %v) is not left of %s (x %v)", names[i], xs[i], names[i+1], xs[i+1])
		}
		if i >= 2 && ys[i] <= ys[i-2] {
			t.Errorf("%s (y %v) is not under %s (y %v)", names[i], ys[i], names[i-2], ys[i-2])
		}
	}
}

func TestAgentsPaneSaysWhereEachAgentStands(t *testing.T) {
	r := agentsRig(t)
	store := r.app.Store.(*data.Fake)
	store.ReportAs(1, "codex", "one", fixedNow().Add(-6*time.Minute))
	store.ReportAs(1, "codex", "two", fixedNow().Add(-50*time.Minute))
	r.app.Refresh()
	r.tt.Frame()
	has(t, r.tt,
		"2 reports today", // Codex
		"Found · gemini", "Found · ~/.cursor", "Found · opencode", "Found · code",
		"Not on this Mac")
	// two installed, three to install, four to get
	if got := countText(r.tt, "MCP installed"); got != 2 {
		t.Errorf("%d agents with the MCP installed, want 2", got)
	}
	if got := countText(r.tt, "Install MCP"); got != 4 {
		t.Errorf("%d Install MCP buttons, want 4 (Gemini CLI, Cursor, opencode, Copilot)", got)
	}
	if got := countText(r.tt, "Get it ↗"); got != 4 {
		t.Errorf("%d Get it links, want 4 (Qwen Code, Kimi CLI, Windsurf, Zed)", got)
	}
}

func TestAnAgentThatReportedHasItsMCPInstalledEvenIfItsConfigIsElsewhere(t *testing.T) {
	r := agentsRig(t)
	r.app.Store.(*data.Fake).ReportAs(1, "cursor-vscode", "one", fixedNow().Add(-30*time.Hour))
	r.app.Refresh()
	r.tt.Frame()
	has(t, r.tt, "No reports today")
	if got := countText(r.tt, "Install MCP"); got != 3 {
		t.Errorf("%d Install MCP buttons, want 3", got)
	}
}

func TestInstallMCPRunsTheAgentsCommandThenShowsItInstalled(t *testing.T) {
	r := agentsRig(t)
	gate := make(chan struct{})
	r.mac.mu.Lock()
	r.mac.gate = gate
	r.mac.run = func(string, []string) ([]byte, error) {
		r.mac.write(".gemini/settings.json", `{"mcpServers":{"adhd-todo":{}}}`)
		return nil, nil
	}
	r.mac.mu.Unlock()
	if err := r.tt.Click("Install the MCP in Gemini CLI"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	has(t, r.tt, "Installing…")
	if len(r.rec.urls) != 0 {
		t.Errorf("opened %v", r.rec.urls)
	}
	close(gate)
	r.rec.drain(t)
	r.tt.Frame()
	hasNot(t, r.tt, "Installing…")
	if got := countText(r.tt, "MCP installed"); got != 3 {
		t.Errorf("%d agents with the MCP installed, want 3", got)
	}
	cmds := r.mac.commands()
	if len(cmds) != 1 || cmds[0][0] != "/usr/local/bin/gemini" || !strings.Contains(strings.Join(cmds[0], " "), "mcp add -s user adhd-todo "+filepath.Join(r.mac.home, ".local/bin/adhd-core")+" mcp") {
		t.Errorf("ran %q", cmds)
	}
}

func TestInstallMCPShowsAFailureInTheRow(t *testing.T) {
	r := agentsRig(t)
	r.mac.mu.Lock()
	r.mac.run = func(string, []string) ([]byte, error) {
		return []byte("noise\nerror: unknown option '-s'\n"), errors.New("exit status 1")
	}
	r.mac.mu.Unlock()
	mustNil(t, r.tt.Click("Install the MCP in Gemini CLI"))
	r.rec.drain(t)
	r.tt.Frame()
	has(t, r.tt, "Gemini CLI said: error: unknown option '-s'", "Install MCP")
	if got := countText(r.tt, "MCP installed"); got != 2 {
		t.Errorf("%d agents with the MCP installed, want 2", got)
	}
}

func TestInstallMCPWithoutTheCoreSaysSo(t *testing.T) {
	r := agentsRig(t)
	mustNil(t, os.Remove(filepath.Join(r.mac.home, ".local", "bin", "adhd-core")))
	mustNil(t, r.tt.Click("Install the MCP in Gemini CLI"))
	r.rec.drain(t)
	r.tt.Frame()
	has(t, r.tt, "adhd-core isn't at ~/.local/bin/adhd-core. Build it first, then try again.")
	if len(r.mac.commands()) != 0 {
		t.Errorf("ran %q", r.mac.commands())
	}
}

func TestInstallMCPEditsACursorFileAndShowsItInstalled(t *testing.T) {
	r := agentsRig(t)
	mustNil(t, r.tt.Click("Install the MCP in Cursor"))
	r.rec.drain(t)
	r.tt.Frame()
	if got := countText(r.tt, "MCP installed"); got != 3 {
		t.Errorf("%d agents with the MCP installed, want 3", got)
	}
	b, err := os.ReadFile(filepath.Join(r.mac.home, ".cursor", "mcp.json"))
	mustNil(t, err)
	if !strings.Contains(string(b), `"adhd-todo"`) {
		t.Errorf("not written: %s", b)
	}
	if len(r.mac.commands()) != 0 {
		t.Errorf("a command ran for a file-based agent: %q", r.mac.commands())
	}
}

func TestAZedFileWithCommentsIsNotRewrittenAndTheSnippetCanBeCopied(t *testing.T) {
	r := newSettingsRig(t)
	orig := "// my settings\n{\n  \"theme\": \"One\", // dark\n}\n"
	r.mac.write(".config/zed/settings.json", orig)
	r.pane(t, "Agents")
	mustNil(t, r.tt.Click("Install the MCP in Zed"))
	r.rec.drain(t)
	r.tt.Frame()
	has(t, r.tt, "has comments, so adhd-todo won't rewrite it", "~/.config/zed/settings.json", `"context_servers": {`)
	b, err := os.ReadFile(filepath.Join(r.mac.home, ".config", "zed", "settings.json"))
	mustNil(t, err)
	if string(b) != orig {
		t.Errorf("the file was rewritten: %s", b)
	}
	r.tt.Scroll(340, 300, 0, 3000)
	r.tt.Frame()
	mustNil(t, r.tt.Click("Copy command"))
	if clip := r.tt.Clipboard(); !strings.Contains(clip, `"adhd-todo": {`) || strings.Contains(clip, "claude mcp add") {
		t.Errorf("clipboard %q", clip)
	}
}

func TestGetItOpensTheAgentsSite(t *testing.T) {
	r := agentsRig(t)
	mustNil(t, r.tt.Click("Get Zed"))
	if len(r.rec.urls) != 1 || r.rec.urls[0] != "https://zed.dev" {
		t.Errorf("opened %v", r.rec.urls)
	}
}

func TestAgentsPaneCountsOneReportInTheSingular(t *testing.T) {
	r := newSettingsRig(t)
	r.app.Store.(*data.Fake).ReportAs(1, "cursor-agent", "one", fixedNow().Add(-time.Minute))
	r.app.Refresh()
	r.pane(t, "Agents")
	has(t, r.tt, "Cursor", "1 report today")
}

func TestAgentsPaneWithoutAnEnvFindsNothingAndDoesNotPanic(t *testing.T) {
	r := newSettingsRig(t, func(a *App) { a.Config.AgentSetup = nil })
	r.s.app.Config.AgentSetup = nil
	r.pane(t, "Agents")
	has(t, r.tt, "Claude Code", "Not on this Mac", "Get it ↗")
	hasNot(t, r.tt, "Install MCP")
}

func TestAgentsPaneInChinese(t *testing.T) {
	r := newSettingsRig(t)
	inChinese(r.tt)
	r.mac.has("gemini")
	r.mac.write(".claude.json", `{"mcpServers":{"adhd-todo":{}}}`)
	r.app.Store.(*data.Fake).ReportAs(1, "codex", "one", fixedNow().Add(-6*time.Minute))
	r.app.Refresh()
	r.s.Show(PaneAgents)
	r.tt.Frame()
	has(t, r.tt, "这台 Mac 上的 Agent", "今天汇报 1 次", "找到了 · gemini", "这台 Mac 上没有", "安装 MCP", "MCP 已安装", "去下载 ↗",
		"也可以在终端里运行一次，给 Claude Code 安装")
	hasFold(t, r.tt, "安装 MCP")
}

func TestEveryAgentOfTheListHasAnAvatarAndAName(t *testing.T) {
	for _, a := range agentsetup.Agents {
		if got := agentName(a.Name); got != a.Name {
			t.Errorf("agentName(%q) = %q: a row's name and the name reports go by differ", a.Name, got)
		}
	}
}
