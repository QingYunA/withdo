package views

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/QingYunA/adhd-todo/internal/agentsetup"
)

// agentsRig is the Agents pane open, on a Mac with five of the fourteen agents: Claude Code and
// Codex with the MCP installed, Gemini CLI, Cursor and opencode found. DeepSeek Harness and the
// other eight are only in More.
func agentsRig(t *testing.T) *settingsRig {
	t.Helper()
	r := newSettingsRig(t)
	r.mac.has("claude", "codex", "gemini", "opencode")
	r.mac.write(".claude.json", `{"mcpServers":{"withdo":{"command":"x"}}}`)
	r.mac.write(".codex/config.toml", "[mcp_servers.withdo]\ncommand = \"x\"\n")
	r.mac.write(".cursor/mcp.json", "{}")
	r.pane(t, "Agents")
	return r
}

// emptyMacRig is the pane on a Mac where nothing was found and nothing ever reported: the seeded
// fake reports for Claude Code would otherwise put it in On this computer.
func emptyMacRig(t *testing.T) *settingsRig {
	t.Helper()
	r := newSettingsRig(t)
	r.app.Store = demoStore(t, fixedNow, false)
	r.app.Refresh()
	r.pane(t, "Agents")
	return r
}

func TestAgentsPanePutsFiveTilesInARowAndKeepsCatalogOrder(t *testing.T) {
	r := agentsRig(t)
	has(t, r.tt, "On this computer", "5", "9 more agents", "Connect by hand")
	local := []string{"Claude Code", "Codex", "Gemini CLI", "Cursor", "opencode"}
	x, y := map[string]float32{}, map[string]float32{}
	for _, n := range local {
		f, ok := r.tt.Find(n)
		if !ok {
			t.Fatalf("no tile for %s", n)
		}
		x[n], y[n] = f.X, f.Y
	}
	for i := 1; i < len(local); i++ {
		if x[local[i]] <= x[local[i-1]] {
			t.Errorf("%s (x %v) is not right of %s (x %v)", local[i], x[local[i]], local[i-1], x[local[i-1]])
		}
		if y[local[i]] != y[local[0]] {
			t.Errorf("%s (y %v) is not on the first tile row (y %v)", local[i], y[local[i]], y[local[0]])
		}
	}
	// More is closed until its row is clicked; opened, it reads two to a row in catalog order.
	if got := countText(r.tt, "Get it"); got != 0 {
		t.Errorf("%d Get it links while More is closed, want 0", got)
	}
	mustNil(t, r.tt.Click("9 more agents"))
	r.tt.Frame()
	more := []string{"DeepSeek Harness", "Copilot (VS Code)", "Qwen Code", "Kimi CLI", "Windsurf", "Zed", "Amp", "Kiro", "Cline"}
	for i := 0; i+1 < len(more); i += 2 {
		a, _ := r.tt.Find(more[i])
		b, _ := r.tt.Find(more[i+1])
		if a.X >= b.X {
			t.Errorf("%s (x %v) is not left of %s (x %v)", more[i], a.X, more[i+1], b.X)
		}
	}
	for i := 2; i < len(more); i++ {
		a, _ := r.tt.Find(more[i])
		b, _ := r.tt.Find(more[i-2])
		if a.Y <= b.Y {
			t.Errorf("%s (y %v) is not under %s (y %v)", more[i], a.Y, more[i-2], b.Y)
		}
	}
	if got := countText(r.tt, "Get it"); got != 9 {
		t.Errorf("%d Get it links, want 9", got)
	}
}

func TestFiveTilesAndBothDisclosuresClosedFitTheWindow(t *testing.T) {
	r := agentsRig(t)
	for _, n := range []string{
		"On this computer", "5", "Claude Code", "opencode", "9 more agents", "Connect by hand",
		"Open agent work in", "GitHub", "Recent changes", "The latest merged pull requests and issues, by project · gh signed in",
	} {
		f, ok := r.tt.Find(n)
		if !ok {
			t.Fatalf("no %s", n)
		}
		if f.Y+f.H > theme.SettingsHeight {
			t.Errorf("%s ends at %v, below the window (%v)", n, f.Y+f.H, theme.SettingsHeight)
		}
	}
	// Both disclosures are closed, so neither the More list nor the command well is drawn.
	hasNot(t, r.tt, "Get it", "Or run this once in a terminal, for Claude Code", "Copy setup guide")
}

func TestAgentsPaneSaysWhereEachAgentStands(t *testing.T) {
	r := agentsRig(t)
	store := r.app.Store.(*data.Real)
	store.SeedReportAs(1, "codex", "one", fixedNow().Add(-6*time.Minute))
	store.SeedReportAs(1, "codex", "two", fixedNow().Add(-50*time.Minute))
	r.app.Refresh()
	r.tt.Frame()
	has(t, r.tt, "2 reports today", "Found", "Connected", "Connect")
	hasNot(t, r.tt, "Found · gemini", "Not on this Mac")
	// two connected, three to connect, nine only in More
	if got := countText(r.tt, "Connected"); got != 2 {
		t.Errorf("%d Connected pills, want 2", got)
	}
	if got := countText(r.tt, "Connect"); got != 3 {
		t.Errorf("%d Connect buttons, want 3 (Gemini CLI, Cursor, opencode)", got)
	}
	mustNil(t, r.tt.Click("9 more agents"))
	r.tt.Frame()
	if got := countText(r.tt, "Get it"); got != 9 {
		t.Errorf("%d Get it links, want 9", got)
	}
}

func TestAnAgentThatReportedHasItsMCPInstalledEvenIfItsConfigIsElsewhere(t *testing.T) {
	r := agentsRig(t)
	r.app.Store.(*data.Real).SeedReportAs(1, "cursor-vscode", "one", fixedNow().Add(-30*time.Hour))
	r.app.Refresh()
	r.tt.Frame()
	has(t, r.tt, "No reports today")
	if got := countText(r.tt, "Connect"); got != 2 {
		t.Errorf("%d Connect buttons, want 2", got)
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
	has(t, r.tt, "Amp", "Connected", "8 more agents")
	if got := countText(r.tt, "Amp"); got != 1 {
		t.Errorf("Amp is drawn %d times, want once (On this Mac, not More)", got)
	}
	mustNil(t, r.tt.Click("8 more agents"))
	r.tt.Frame()
	if got := countText(r.tt, "Get it"); got != 8 {
		t.Errorf("%d Get it links, want 8", got)
	}
}

func TestAnEmptyMacShowsTheEmptyRowAndAllAgentsUnderMore(t *testing.T) {
	r := emptyMacRig(t)
	has(t, r.tt, "On this computer", "0", "No agent found on this computer yet", "See how to connect", "14 more agents")
	if got := countText(r.tt, "Connect"); got != 0 {
		t.Errorf("%d Connect buttons on an empty Mac, want 0", got)
	}
	mustNil(t, r.tt.Click("See how to connect"))
	if len(r.rec.urls) != 1 || r.rec.urls[0] != repoURL+"#connect-claude-code" {
		t.Errorf("opened %v", r.rec.urls)
	}
	mustNil(t, r.tt.Click("14 more agents"))
	r.tt.Frame()
	if got := countText(r.tt, "Get it"); got != 14 {
		t.Errorf("%d Get it links, want 14", got)
	}
}

// The empty line follows the language, like the rest of the pane.
func TestAnEmptyMacSaysSoInChinese(t *testing.T) {
	r := emptyMacRig(t)
	inChinese(r.tt)
	r.s.Show(PaneAgents)
	r.tt.Frame()
	has(t, r.tt, "本机已有", "0", "还没在这台电脑上找到 Agent", "去看看怎么接", "还有 14 个 Agent")
	mustNil(t, r.tt.Click("还有 14 个 Agent"))
	r.tt.Frame()
	has(t, r.tt, "去下载")
}

func TestConnectRunsTheAgentsCommandThenShowsItConnected(t *testing.T) {
	r := agentsRig(t)
	gate := make(chan struct{})
	r.mac.mu.Lock()
	r.mac.gate = gate
	r.mac.run = func(string, []string) ([]byte, error) {
		r.mac.write(".gemini/settings.json", `{"mcpServers":{"withdo":{}}}`)
		return nil, nil
	}
	r.mac.mu.Unlock()
	if err := r.tt.Click("Connect Gemini CLI"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	has(t, r.tt, "Connecting…")
	if len(r.rec.urls) != 0 {
		t.Errorf("opened %v", r.rec.urls)
	}
	close(gate)
	r.rec.drain(t)
	r.tt.Frame()
	hasNot(t, r.tt, "Connecting…")
	if got := countText(r.tt, "Connected"); got != 3 {
		t.Errorf("%d Connected pills, want 3", got)
	}
	cmds := r.mac.commands()
	if len(cmds) != 1 || cmds[0][0] != "/usr/local/bin/gemini" || !strings.Contains(strings.Join(cmds[0], " "), "mcp add -s user withdo "+filepath.Join(r.mac.home, ".local/bin/withdo")+" mcp") {
		t.Errorf("ran %q", cmds)
	}
}

func TestConnectShowsAFailureUnderTheTile(t *testing.T) {
	r := agentsRig(t)
	r.mac.mu.Lock()
	r.mac.run = func(string, []string) ([]byte, error) {
		return []byte("noise\nerror: unknown option '-s'\n"), errors.New("exit status 1")
	}
	r.mac.mu.Unlock()
	mustNil(t, r.tt.Click("Connect Gemini CLI"))
	r.rec.drain(t)
	r.tt.Frame()
	has(t, r.tt, "Gemini CLI said: error: unknown option '-s'", "Connect")
	if got := countText(r.tt, "Connected"); got != 2 {
		t.Errorf("%d Connected pills, want 2", got)
	}
}

func TestConnectWithoutTheCoreSaysSo(t *testing.T) {
	r := agentsRig(t)
	mustNil(t, os.Remove(filepath.Join(r.mac.home, ".local", "bin", "withdo")))
	mustNil(t, r.tt.Click("Connect Gemini CLI"))
	r.rec.drain(t)
	r.tt.Frame()
	has(t, r.tt, "withdo isn't at ~/.local/bin/withdo. Build it first, then try again.")
	if len(r.mac.commands()) != 0 {
		t.Errorf("ran %q", r.mac.commands())
	}
}

func TestConnectEditsACursorFileAndShowsItConnected(t *testing.T) {
	r := agentsRig(t)
	mustNil(t, r.tt.Click("Connect Cursor"))
	r.rec.drain(t)
	r.tt.Frame()
	if got := countText(r.tt, "Connected"); got != 3 {
		t.Errorf("%d Connected pills, want 3", got)
	}
	b, err := os.ReadFile(filepath.Join(r.mac.home, ".cursor", "mcp.json"))
	mustNil(t, err)
	if !strings.Contains(string(b), `"withdo"`) {
		t.Errorf("not written: %s", b)
	}
	if len(r.mac.commands()) != 0 {
		t.Errorf("a command ran for a file-based agent: %q", r.mac.commands())
	}
}

// DeepSeek Harness has no JSON config: Connect appends a Cordis patch entry, in place.
func TestConnectAppendsADeepSeekHarnessCordisPatch(t *testing.T) {
	r := newSettingsRig(t)
	r.mac.write(".dsh/cordis.patch.yml", "# mine\n- id: ui-chat\n  config:\n    transcriptView: standard\n")
	r.pane(t, "Agents")
	has(t, r.tt, "DeepSeek Harness", "Found")
	mustNil(t, r.tt.Click("Connect DeepSeek Harness"))
	r.rec.drain(t)
	r.tt.Frame()
	has(t, r.tt, "Connected")
	b, err := os.ReadFile(filepath.Join(r.mac.home, ".dsh", "cordis.patch.yml"))
	mustNil(t, err)
	if !strings.Contains(string(b), "# mine") || !strings.Contains(string(b), "serverName: withdo") {
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
	mustNil(t, r.tt.Click("Connect DeepSeek Harness"))
	r.rec.drain(t)
	r.tt.Frame()
	has(t, r.tt, "won't rewrite it", "~/.dsh/cordis.patch.yml", "@deepseek-ai/dsh-mcp-client")
	b, err := os.ReadFile(filepath.Join(r.mac.home, ".dsh", "cordis.patch.yml"))
	mustNil(t, err)
	if string(b) != orig {
		t.Errorf("the patch was rewritten: %s", b)
	}
}

func TestConnectWritesAMCPJSONForKiroAndCline(t *testing.T) {
	for _, id := range []string{"Kiro", "Cline"} {
		r := newSettingsRig(t)
		r.app.Store = demoStore(t, fixedNow, false)
		r.app.Refresh()
		r.mac.write(".kiro/settings/mcp.json", "{}")
		r.mac.write("Library/Application Support/Code/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json", "{}")
		r.pane(t, "Agents")
		mustNil(t, r.tt.Click("Connect "+id))
		r.rec.drain(t)
		r.tt.Frame()
		if got := countText(r.tt, "Connected"); got != 1 {
			t.Errorf("%s: %d Connected pills, want 1", id, got)
		}
	}
}

func TestAZedFileWithCommentsIsNotRewrittenAndTheSnippetCanBeCopied(t *testing.T) {
	r := newSettingsRig(t)
	orig := "// my settings\n{\n  \"theme\": \"One\", // dark\n}\n"
	r.mac.write(".config/zed/settings.json", orig)
	r.pane(t, "Agents")
	mustNil(t, r.tt.Click("Connect Zed"))
	r.rec.drain(t)
	r.tt.Frame()
	has(t, r.tt, "won't rewrite it", "~/.config/zed/settings.json", `"context_servers": {`)
	b, err := os.ReadFile(filepath.Join(r.mac.home, ".config", "zed", "settings.json"))
	mustNil(t, err)
	if string(b) != orig {
		t.Errorf("the file was rewritten: %s", b)
	}
	mustNil(t, r.tt.Click("Copy command"))
	if clip := r.tt.Clipboard(); !strings.Contains(clip, `"withdo": {`) || strings.Contains(clip, "claude mcp add") {
		t.Errorf("clipboard %q", clip)
	}
}

func TestMoreOpensTheAgentsSite(t *testing.T) {
	r := agentsRig(t)
	mustNil(t, r.tt.Click("9 more agents"))
	r.tt.Frame()
	mustNil(t, r.tt.Click("Get Zed"))
	if len(r.rec.urls) != 1 || r.rec.urls[0] != "https://zed.dev" {
		t.Errorf("opened %v", r.rec.urls)
	}
}

func TestTheTwoDisclosuresOpenAndCloseInPlace(t *testing.T) {
	r := agentsRig(t)
	hasNot(t, r.tt, "Get it", "Or run this once in a terminal, for Claude Code", "Copy setup guide")
	mustNil(t, r.tt.Click("9 more agents"))
	r.tt.Frame()
	if got := countText(r.tt, "Get it"); got != 9 {
		t.Errorf("%d Get it links while More is open, want 9", got)
	}
	mustNil(t, r.tt.Click("9 more agents"))
	r.tt.Frame()
	hasNot(t, r.tt, "Get it")

	mustNil(t, r.tt.Click("Connect by hand"))
	r.tt.Frame()
	has(t, r.tt, "Or run this once in a terminal, for Claude Code", r.app.Config.ConnectCommand, "Copy setup guide")
	mustNil(t, r.tt.Click("Connect by hand"))
	r.tt.Frame()
	hasNot(t, r.tt, "Or run this once in a terminal, for Claude Code", "Copy setup guide")
}

// A connected avatar rises one sprite pixel for three beats and stands for three, and each tile
// is two beats behind the one before it. Reduce Motion keeps it standing.
func TestTheAvatarBobsThreeBeatsUpAndThreeDownTwoBeatsApart(t *testing.T) {
	base := time.UnixMilli(0)
	offsets := []float32{theme.AgentBobLift, theme.AgentBobLift, theme.AgentBobLift, 0, 0, 0}
	for beat, want := range offsets {
		at := base.Add(time.Duration(beat) * theme.AgentBobStep)
		if got := theme.AgentBobOffset(at, 0, false); got != want {
			t.Errorf("beat %d: offset %v, want %v", beat, got, want)
		}
		// Tile 1 is two beats behind tile 0.
		if got, want := theme.AgentBobOffset(at, 1, false), theme.AgentBobOffset(at.Add(2*theme.AgentBobStep), 0, false); got != want {
			t.Errorf("beat %d: tile 1 offset %v, want tile 0 two beats later (%v)", beat, got, want)
		}
	}
	if got := theme.AgentBobOffset(base.Add(time.Hour), 0, true); got != 0 {
		t.Errorf("Reduce Motion offset %v, want 0", got)
	}
	if got, want := theme.AgentBobNext(base.Add(40*time.Millisecond)), 60*time.Millisecond; got != want {
		t.Errorf("next beat in %v, want %v", got, want)
	}
}

func TestAgentsPaneWithoutAnEnvFindsNothingAndDoesNotPanic(t *testing.T) {
	r := newSettingsRig(t, func(a *App) { a.Config.AgentSetup = nil })
	r.s.app.Config.AgentSetup = nil
	r.app.Store = demoStore(t, fixedNow, false)
	r.app.Refresh()
	r.pane(t, "Agents")
	has(t, r.tt, "No agent found on this computer yet", "See how to connect", "14 more agents")
	if got := countText(r.tt, "Connect"); got != 0 {
		t.Errorf("%d Connect buttons without an Env, want 0", got)
	}
	mustNil(t, r.tt.Click("14 more agents"))
	r.tt.Frame()
	has(t, r.tt, "Claude Code", "Get it")
}

func TestAgentsPaneInChinese(t *testing.T) {
	r := newSettingsRig(t)
	inChinese(r.tt)
	r.mac.has("gemini")
	r.mac.write(".claude.json", `{"mcpServers":{"withdo":{}}}`)
	r.app.Store.(*data.Real).SeedReportAs(1, "codex", "one", fixedNow().Add(-6*time.Minute))
	r.app.Refresh()
	r.s.Show(PaneAgents)
	r.tt.Frame()
	has(t, r.tt, "本机已有", "3", "还有 11 个 Agent", "今天汇报 1 次", "找到了", "已接入", "接入", "手动接入")
	hasNot(t, r.tt, "这台 Mac 上没有", "去下载")
	mustNil(t, r.tt.Click("手动接入"))
	r.tt.Frame()
	has(t, r.tt, "也可以在终端里运行一次，给 Claude Code 安装", "复制接入说明")
	mustNil(t, r.tt.Click("还有 11 个 Agent"))
	r.tt.Frame()
	has(t, r.tt, "去下载")
}

func TestAgentsPaneDrawsInBothAppearances(t *testing.T) {
	r := agentsRig(t)
	for _, dark := range []bool{false, true} {
		r.tt.SetDark(dark)
		r.tt.Frame()
		has(t, r.tt, "Claude Code", "Connected", "Connect", "9 more agents")
		if got := countText(r.tt, "Get it"); got != 0 {
			t.Errorf("dark=%v: %d Get it links while More is closed, want 0", dark, got)
		}
	}
	mustNil(t, r.tt.Click("9 more agents"))
	r.tt.Frame()
	has(t, r.tt, "Get it")
}

func TestEveryAgentOfTheListHasAnAvatarAndAName(t *testing.T) {
	for _, a := range agentsetup.Agents {
		if got := agentName(a.Name); got != a.Name {
			t.Errorf("agentName(%q) = %q: a row's name and the name reports go by differ", a.Name, got)
		}
	}
	// A 103 pt tile drops the parenthetical; the catalog's full name stays in More and in the
	// accessibility label.
	if got := agentTileName(agentsetup.Agent{Name: "Copilot (VS Code)"}); got != "Copilot" {
		t.Errorf("tile name %q, want Copilot", got)
	}
}

// An agent that only has the pre-rename adhd-todo entry has not got Withdo yet, even though it
// reported before: it offers Connect, which adds withdo and takes the old entry out.
func TestAnAgentWithOnlyTheOldEntryOffersConnectAndMigrates(t *testing.T) {
	r := newSettingsRig(t)
	r.mac.has("claude")
	r.mac.write(".claude.json", `{"mcpServers":{"adhd-todo":{"command":"x"}}}`)
	r.app.Store.(*data.Real).SeedReportAs(1, "claude-code", "one", fixedNow().Add(-6*time.Minute))
	r.app.Refresh()
	r.pane(t, "Agents")
	has(t, r.tt, "Claude Code", "Connect")
	if got := countText(r.tt, "Connected"); got != 0 {
		t.Errorf("%d Connected pills before the install, want 0", got)
	}

	r.mac.mu.Lock()
	r.mac.run = func(_ string, args []string) ([]byte, error) {
		if slices.Contains(args, "add") {
			r.mac.write(".claude.json", `{"mcpServers":{"adhd-todo":{"command":"x"},"withdo":{"command":"y"}}}`)
		} else {
			r.mac.write(".claude.json", `{"mcpServers":{"withdo":{"command":"y"}}}`)
		}
		return nil, nil
	}
	r.mac.mu.Unlock()
	mustNil(t, r.tt.Click("Connect Claude Code"))
	r.rec.drain(t)
	r.tt.Frame()
	has(t, r.tt, "Connected")
	if got := countText(r.tt, "Connect"); got != 0 {
		t.Errorf("%d Connect buttons after the install, want 0", got)
	}
	cmds := r.mac.commands()
	if len(cmds) != 2 || !slices.Contains(cmds[0], "add") || !slices.Contains(cmds[1], "remove") || !slices.Contains(cmds[1], "adhd-todo") {
		t.Errorf("ran %q", cmds)
	}
}

// The Agents row of the sidebar carries the square while an agent is here without its MCP.
func TestTheSidebarMarksAnAgentFoundWithoutItsMCP(t *testing.T) {
	r := newSettingsRig(t)
	r.mac.has("gemini")
	r.s.setup.scannedAt = time.Time{}
	if !r.s.navAgentAlert() {
		t.Error("no alert with Gemini found and its MCP missing")
	}
	r.mac.write(".gemini/settings.json", `{"mcpServers":{"withdo":{"command":"x"}}}`)
	r.s.setup.scannedAt = time.Time{}
	if r.s.navAgentAlert() {
		t.Error("an alert with Gemini's MCP installed")
	}
}
