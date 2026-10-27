package views

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/QingYunA/adhd-todo/internal/agentsetup"
)

// agentsRig is the Agents pane open, on a Mac with six of the fourteen agents: Claude Code and
// Codex with the MCP installed, Gemini CLI, Cursor, opencode and VS Code found; DeepSeek Harness
// and the other seven are only in More.
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

// emptyMacRig is the pane on a Mac where nothing was found and nothing ever reported: the seeded
// fake reports for Claude Code would otherwise put it in On this Mac.
func emptyMacRig(t *testing.T) *settingsRig {
	t.Helper()
	r := newSettingsRig(t)
	r.app.Store = demoStore(t, fixedNow, false)
	r.app.Refresh()
	r.pane(t, "Agents")
	return r
}

func TestAgentsPaneSplitsOnThisMacAndMoreInCatalogOrder(t *testing.T) {
	r := agentsRig(t)
	has(t, r.tt, "Coding agents", "On this computer", "More", "adds Withdo's MCP server to that agent")
	// The counts sit beside their labels: six on this Mac, eight in More.
	for label, count := range map[string]string{"On this computer": "6", "More": "8"} {
		head, ok := r.tt.Find(label)
		if !ok {
			t.Fatalf("no section label %q", label)
		}
		digits, ok := r.tt.Find(count)
		if !ok {
			t.Fatalf("no count %q beside %q", count, label)
		}
		if got := countText(r.tt, count); got != 1 {
			t.Errorf("%q is drawn %d times", count, got)
		}
		if digits.Y != head.Y {
			t.Errorf("the count %q is at y %v, the label %q at %v", count, digits.Y, label, head.Y)
		}
		if digits.X <= head.X+head.W {
			t.Errorf("the count %q is not after %q", count, label)
		}
	}
	local := []string{"Claude Code", "Codex", "Gemini CLI", "Cursor", "opencode", "Copilot (VS Code)"}
	more := []string{"DeepSeek Harness", "Qwen Code", "Kimi CLI", "Windsurf", "Zed", "Amp", "Kiro", "Cline"}
	x, y := map[string]float32{}, map[string]float32{}
	for _, n := range append(append([]string{}, local...), more...) {
		f, ok := r.tt.Find(n)
		if !ok {
			t.Fatalf("no row for %s", n)
		}
		x[n], y[n] = f.X, f.Y
	}
	// On this Mac: two columns, read left to right and then top to bottom, in catalog order.
	for i := 0; i+1 < len(local); i += 2 {
		if x[local[i]] >= x[local[i+1]] {
			t.Errorf("%s (x %v) is not left of %s (x %v)", local[i], x[local[i]], local[i+1], x[local[i+1]])
		}
	}
	for i := 2; i < len(local); i++ {
		if y[local[i]] <= y[local[i-2]] {
			t.Errorf("%s (y %v) is not under %s (y %v)", local[i], y[local[i]], local[i-2], y[local[i-2]])
		}
	}
	// More: the same reading order, two to a row.
	for i := 0; i+1 < len(more); i += 2 {
		if x[more[i]] >= x[more[i+1]] {
			t.Errorf("%s (x %v) is not left of %s (x %v)", more[i], x[more[i]], more[i+1], x[more[i+1]])
		}
	}
	for i := 2; i < len(more); i++ {
		if y[more[i]] <= y[more[i-2]] {
			t.Errorf("%s (y %v) is not under %s (y %v)", more[i], y[more[i]], more[i-2], y[more[i-2]])
		}
	}
	if y[more[0]] <= y[local[len(local)-1]] {
		t.Errorf("More (y %v) is not under the last row of On this Mac (y %v)", y[more[0]], y[local[len(local)-1]])
	}
}

func TestBothSectionsFitTheWindowBeforeItScrolls(t *testing.T) {
	r := agentsRig(t)
	for _, n := range []string{"On this computer", "6", "More", "8", "Kiro", "Cline"} {
		f, ok := r.tt.Find(n)
		if !ok {
			t.Fatalf("no %s", n)
		}
		if f.Y+f.H > theme.SettingsHeight {
			t.Errorf("%s ends at %v, below the first screen (%v)", n, f.Y+f.H, theme.SettingsHeight)
		}
	}
	// What stays below the fold stays below it, in the same order as before.
	for _, n := range []string{"Open agent work in", "Or run this once in a terminal, for Claude Code"} {
		f, ok := r.tt.Find(n)
		if !ok {
			t.Fatalf("no %s", n)
		}
		if f.Y < theme.SettingsHeight {
			t.Errorf("%s starts at %v, on the first screen", n, f.Y)
		}
	}
}

func TestAgentsPaneSaysWhereEachAgentStands(t *testing.T) {
	r := agentsRig(t)
	store := r.app.Store.(*data.Real)
	store.SeedReportAs(1, "codex", "one", fixedNow().Add(-6*time.Minute))
	store.SeedReportAs(1, "codex", "two", fixedNow().Add(-50*time.Minute))
	r.app.Refresh()
	r.tt.Frame()
	has(t, r.tt,
		"2 reports today", // Codex
		"Found · gemini", "Found · ~/.cursor", "Found · opencode", "Found · code")
	hasNot(t, r.tt, "Not on this Mac")
	// two installed, four to install, eight only in More
	if got := countText(r.tt, "MCP installed"); got != 2 {
		t.Errorf("%d agents with the MCP installed, want 2", got)
	}
	if got := countText(r.tt, "Install MCP"); got != 4 {
		t.Errorf("%d Install MCP buttons, want 4 (Gemini CLI, Cursor, opencode, Copilot)", got)
	}
	if got := countText(r.tt, "Get it"); got != 8 {
		t.Errorf("%d Get it links, want 8 (the whole of More)", got)
	}
}

func TestAnAgentThatReportedHasItsMCPInstalledEvenIfItsConfigIsElsewhere(t *testing.T) {
	r := agentsRig(t)
	r.app.Store.(*data.Real).SeedReportAs(1, "cursor-vscode", "one", fixedNow().Add(-30*time.Hour))
	r.app.Refresh()
	r.tt.Frame()
	has(t, r.tt, "No reports today")
	if got := countText(r.tt, "Install MCP"); got != 3 {
		t.Errorf("%d Install MCP buttons, want 3", got)
	}
}

func TestAgentsPaneCountsOneReportInTheSingular(t *testing.T) {
	r := agentsRig(t)
	r.app.Store.(*data.Real).SeedReportAs(1, "cursor-agent", "one", fixedNow().Add(-time.Minute))
	r.app.Refresh()
	r.tt.Frame()
	has(t, r.tt, "Cursor", "1 report today")
}

// A report alone puts an agent in On this Mac, even when this look does not find a command or a
// config folder: its MCP is installed somewhere.
func TestAnAgentThatOnlyReportedStaysOnThisMac(t *testing.T) {
	r := agentsRig(t)
	r.app.Store.(*data.Real).SeedReportAs(1, "amp", "one", fixedNow().Add(-time.Hour))
	r.app.Refresh()
	r.tt.Frame()
	has(t, r.tt, "Amp", "MCP installed")
	if got := countText(r.tt, "Amp"); got != 1 {
		t.Errorf("Amp is drawn %d times, want once (On this Mac, not More)", got)
	}
	if got := countText(r.tt, "7"); got != 2 {
		t.Errorf("%d counts read seven, want 7 on this Mac and 7 in More", got)
	}
	if got := countText(r.tt, "Get it"); got != 7 {
		t.Errorf("%d Get it links, want 7", got)
	}
}

func TestAnEmptyMacShowsTheEmptyRowAndAllAgentsUnderMore(t *testing.T) {
	r := emptyMacRig(t)
	has(t, r.tt, "On this computer", "No agent found on this computer yet", "See how to connect", "More", "14")
	hasNot(t, r.tt, "Install MCP")
	if got := countText(r.tt, "Get it"); got != 14 {
		t.Errorf("%d Get it links, want 14", got)
	}
	mustNil(t, r.tt.Click("See how to connect"))
	if len(r.rec.urls) != 1 || r.rec.urls[0] != repoURL+"#connect-claude-code" {
		t.Errorf("opened %v", r.rec.urls)
	}
}

// The empty line follows the language, like the rest of the pane.
func TestAnEmptyMacSaysSoInChinese(t *testing.T) {
	r := emptyMacRig(t)
	inChinese(r.tt)
	r.s.Show(PaneAgents)
	r.tt.Frame()
	has(t, r.tt, "本机已有", "还没在这台电脑上找到 Agent", "去看看怎么接", "更多", "14", "去下载")
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
	has(t, r.tt, "withdo isn't at ~/.local/bin/adhd-core. Build it first, then try again.")
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

// DeepSeek Harness has no JSON config: Install MCP appends a Cordis patch entry, in place.
func TestInstallMCPAppendsADeepSeekHarnessCordisPatch(t *testing.T) {
	r := newSettingsRig(t)
	r.mac.write(".dsh/cordis.patch.yml", "# mine\n- id: ui-chat\n  config:\n    transcriptView: standard\n")
	r.pane(t, "Agents")
	has(t, r.tt, "DeepSeek Harness", "Found · ~/.dsh")
	mustNil(t, r.tt.Click("Install the MCP in DeepSeek Harness"))
	r.rec.drain(t)
	r.tt.Frame()
	has(t, r.tt, "MCP installed")
	b, err := os.ReadFile(filepath.Join(r.mac.home, ".dsh", "cordis.patch.yml"))
	mustNil(t, err)
	if !strings.Contains(string(b), "# mine") || !strings.Contains(string(b), "serverName: adhd-todo") {
		t.Errorf("the patch layer was not appended to:\n%s", b)
	}
	if len(r.mac.commands()) != 0 {
		t.Errorf("ran %q", r.mac.commands())
	}
}

// A DeepSeek patch that cannot be appended to falls back to the snippet under the pane.
func TestADeepSeekPatchThatCannotBeAppendedOffersTheSnippet(t *testing.T) {
	r := newSettingsRig(t)
	orig := "some: mapping\n"
	r.mac.write(".dsh/cordis.patch.yml", orig)
	r.pane(t, "Agents")
	mustNil(t, r.tt.Click("Install the MCP in DeepSeek Harness"))
	r.rec.drain(t)
	r.tt.Frame()
	has(t, r.tt, "won't rewrite it", "~/.dsh/cordis.patch.yml", "@deepseek-ai/dsh-mcp-client")
	b, err := os.ReadFile(filepath.Join(r.mac.home, ".dsh", "cordis.patch.yml"))
	mustNil(t, err)
	if string(b) != orig {
		t.Errorf("the patch was rewritten: %s", b)
	}
}

func TestInstallMCPWritesAMCPJSONForKiroAndCline(t *testing.T) {
	for _, id := range []string{"Kiro", "Cline"} {
		r := newSettingsRig(t)
		r.app.Store = demoStore(t, fixedNow, false)
		r.app.Refresh()
		r.mac.write(".kiro/settings/mcp.json", "{}")
		r.mac.write("Library/Application Support/Code/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json", "{}")
		r.pane(t, "Agents")
		mustNil(t, r.tt.Click("Install the MCP in "+id))
		r.rec.drain(t)
		r.tt.Frame()
		if got := countText(r.tt, "MCP installed"); got != 1 {
			t.Errorf("%s: %d agents with the MCP installed, want 1", id, got)
		}
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
	has(t, r.tt, "won't rewrite it", "~/.config/zed/settings.json", `"context_servers": {`)
	b, err := os.ReadFile(filepath.Join(r.mac.home, ".config", "zed", "settings.json"))
	mustNil(t, err)
	if string(b) != orig {
		t.Errorf("the file was rewritten: %s", b)
	}
	r.tt.Scroll(340, 300, 0, 330) // the snippet's well comes into view
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

func TestAgentsPaneWithoutAnEnvFindsNothingAndDoesNotPanic(t *testing.T) {
	r := newSettingsRig(t, func(a *App) { a.Config.AgentSetup = nil })
	r.s.app.Config.AgentSetup = nil
	r.app.Store = demoStore(t, fixedNow, false)
	r.app.Refresh()
	r.pane(t, "Agents")
	has(t, r.tt, "No agent found on this computer yet", "See how to connect", "Claude Code", "Get it")
	hasNot(t, r.tt, "Install MCP")
}

func TestAgentsPaneInChinese(t *testing.T) {
	r := newSettingsRig(t)
	inChinese(r.tt)
	r.mac.has("gemini")
	r.mac.write(".claude.json", `{"mcpServers":{"adhd-todo":{}}}`)
	r.app.Store.(*data.Real).SeedReportAs(1, "codex", "one", fixedNow().Add(-6*time.Minute))
	r.app.Refresh()
	r.s.Show(PaneAgents)
	r.tt.Frame()
	has(t, r.tt, "编程 Agent", "本机已有", "3", "更多", "11", "今天汇报 1 次", "找到了 · gemini", "安装 MCP", "MCP 已安装", "去下载",
		"也可以在终端里运行一次，给 Claude Code 安装")
	hasNot(t, r.tt, "这台 Mac 上没有")
	hasFold(t, r.tt, "本机已有")
}

func TestAgentsPaneDrawsInBothAppearances(t *testing.T) {
	r := agentsRig(t)
	for _, dark := range []bool{false, true} {
		r.tt.SetDark(dark)
		r.tt.Frame()
		has(t, r.tt, "Claude Code", "DeepSeek Harness", "MCP installed", "Install MCP", "Get it")
	}
}

func TestEveryAgentOfTheListHasAnAvatarAndAName(t *testing.T) {
	for _, a := range agentsetup.Agents {
		if got := agentName(a.Name); got != a.Name {
			t.Errorf("agentName(%q) = %q: a row's name and the name reports go by differ", a.Name, got)
		}
	}
}
