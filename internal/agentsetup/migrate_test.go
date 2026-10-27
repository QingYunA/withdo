package agentsetup

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
)

// legacyConfigs is each agent's config with only the old adhd-todo entry in it.
var legacyConfigs = map[string][2]string{ // agent: file, content
	"claude":   {".claude.json", `{"numStartups":3,"mcpServers":{"adhd-todo":{"command":"x"}}}`},
	"codex":    {".codex/config.toml", "model = \"o3\"\n\n[mcp_servers.\"adhd-todo\"]\ncommand = \"x\"\n"},
	"gemini":   {".gemini/settings.json", `{"mcpServers":{"adhd-todo":{"command":"x"}}}`},
	"qwen":     {".qwen/settings.json", `{"mcpServers":{"adhd-todo":{"command":"x"}}}`},
	"kimi":     {".kimi/mcp.json", `{"mcpServers":{"adhd-todo":{"command":"x"}}}`},
	"cursor":   {".cursor/mcp.json", `{"mcpServers":{"adhd-todo":{"command":"x"}}}`},
	"opencode": {".config/opencode/opencode.json", `{"mcp":{"adhd-todo":{"type":"local"}}}`},
	"windsurf": {".codeium/windsurf/mcp_config.json", `{"mcpServers":{"adhd-todo":{}}}`},
	"copilot":  {"Library/Application Support/Code/User/mcp.json", `{"servers":{"adhd-todo":{"command":"x"}}}`},
	"zed":      {".config/zed/settings.json", `{"context_servers":{"adhd-todo":{"command":"x"}}}`},
	"amp":      {".config/amp/settings.json", `{"amp.mcpServers":{"adhd-todo":{"command":"x"}}}`},
	"kiro":     {".kiro/settings/mcp.json", `{"mcpServers":{"adhd-todo":{"command":"x"}}}`},
	"cline":    {"Library/Application Support/Code/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json", `{"mcpServers":{"adhd-todo":{"command":"x"}}}`},
}

// legacyYAML is the patch entry the app wrote for the old name.
const legacyYAML = "- insert:\n    - id: adhd-todo-mcp\n      name: '@deepseek-ai/dsh-mcp-client'\n      config:\n        serverName: adhd-todo\n        transport: stdio\n        command: '/x/adhd-core'\n        args: ['mcp']\n        failOnStartupError: false\n"

// removing makes the fake runner act like an agent's `mcp remove`: the config loses its entry.
func (r *rig) removing(file, after string) {
	r.env.Run = func(ctx context.Context, path string, args ...string) ([]byte, error) {
		out, err := r.run.run(ctx, path, args...)
		if slices.Contains(args, "remove") && err == nil {
			r.write(file, after)
		}
		return out, err
	}
}

func TestDetectSeesTheOldEntryAsLegacyNotInstalled(t *testing.T) {
	r := newRig(t)
	for id, c := range legacyConfigs {
		a := r.agent(id)
		if st := r.env.Detect(a); st.Installed || st.Legacy {
			t.Errorf("%s: %+v before its config exists", id, st)
		}
		r.write(c[0], c[1])
		if st := r.env.Detect(a); st.Installed || !st.Legacy || !st.Found {
			t.Errorf("%s: %+v with only the old entry, want legacy and found by its config", id, st)
		}
	}
	r.write(".dsh/cordis.patch.yml", legacyYAML)
	if st := r.env.Detect(r.agent("deepseek")); st.Installed || !st.Legacy {
		t.Errorf("deepseek: %+v", st)
	}
}

func TestCodexTomlTableNamesAreMatchedPerName(t *testing.T) {
	r := newRig(t)
	a := r.agent("codex")
	for _, header := range []string{"[mcp_servers.adhd-todo]", "[mcp_servers.\"adhd-todo\"]", "[mcp_servers.'adhd-todo'.env]", "[ mcp_servers . adhd-todo ]"} {
		r.write(".codex/config.toml", header+"\ncommand = \"x\"\n")
		if r.env.Installed(a) || !r.env.hasServer(a, legacyServerName) {
			t.Errorf("%s: installed %v, legacy %v", header, r.env.Installed(a), r.env.hasServer(a, legacyServerName))
		}
	}
	r.write(".codex/config.toml", "[mcp_servers.withdo-other]\ncommand = \"x\"\n[mcp_servers.adhd-todo-other]\n")
	if r.env.Installed(a) || r.env.hasServer(a, legacyServerName) {
		t.Error("a longer server name counted")
	}
}

func TestInstallOverTheOldEntryRunsTheAgentsOwnAddThenRemove(t *testing.T) {
	for id, want := range map[string][]string{
		"claude": {"mcp", "remove", "--scope", "user", "adhd-todo"},
		"codex":  {"mcp", "remove", "adhd-todo"},
		"gemini": {"mcp", "remove", "--scope", "user", "adhd-todo"},
		"qwen":   {"mcp", "remove", "--scope", "user", "adhd-todo"},
		"kimi":   {"mcp", "remove", "adhd-todo"},
		"amp":    {"mcp", "remove", "adhd-todo"},
	} {
		r := newRig(t)
		r.path[id] = "/bin/" + id
		c := legacyConfigs[id]
		r.write(c[0], c[1])
		r.removing(c[0], "{}")
		if err := r.env.Install(context.Background(), r.agent(id)); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		if len(r.run.calls) != 2 || !slices.Contains(r.run.calls[0], "add") || !slices.Contains(r.run.calls[0], "withdo") {
			t.Fatalf("%s: the first command should add withdo: %q", id, r.run.calls)
		}
		if got := r.run.calls[1]; !slices.Equal(got, append([]string{"/bin/" + id}, want...)) {
			t.Errorf("%s removed with %q, want %q", id, got, want)
		}
		if st := r.env.Detect(r.agent(id)); st.Legacy {
			t.Errorf("%s still has the old entry", id)
		}
	}
}

func TestInstallSaysSoWhenTheOldEntryCannotBeRemoved(t *testing.T) {
	r := newRig(t)
	r.path["claude"] = "/bin/claude"
	c := legacyConfigs["claude"]
	r.write(c[0], c[1])
	calls := 0
	r.env.Run = func(ctx context.Context, path string, args ...string) ([]byte, error) {
		calls++
		if slices.Contains(args, "remove") {
			return []byte("error: No MCP server found\n"), errors.New("exit status 1")
		}
		return nil, nil
	}
	err := r.env.Install(context.Background(), r.agent("claude"))
	var le *LegacyError
	var ce *CommandError
	if !errors.As(err, &le) || !errors.As(err, &ce) || ce.Output != "error: No MCP server found" || calls != 2 {
		t.Errorf("got %v after %d commands", err, calls)
	}
}

func TestInstallOverTheOldEntryOfAFileAgentSwapsItInOneWrite(t *testing.T) {
	for _, id := range []string{"cursor", "opencode", "windsurf", "zed", "kiro", "cline"} {
		r := newRig(t)
		a := r.agent(id)
		key := a.Key
		orig := `{"keep":1,"` + key + `":{"mine":{"command":"m"},"adhd-todo":{"command":"old"},"zzz":{}},"after":[1,2]}`
		r.write(a.Files[0], orig)
		if err := r.env.Install(context.Background(), a); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		if got := r.read(a.Files[0] + ".withdo.bak"); got != orig {
			t.Errorf("%s: backup %q", id, got)
		}
		var got map[string]json.RawMessage
		if err := json.Unmarshal([]byte(r.read(a.Files[0])), &got); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		var servers map[string]json.RawMessage
		if err := json.Unmarshal(got[key], &servers); err != nil {
			t.Fatal(err)
		}
		if _, ok := servers["adhd-todo"]; ok {
			t.Errorf("%s kept the old entry", id)
		}
		for _, name := range []string{"withdo", "mine", "zzz"} {
			if _, ok := servers[name]; !ok {
				t.Errorf("%s lost %s: %s", id, name, got[key])
			}
		}
		if string(got["keep"]) != "1" || string(got["after"]) == "" {
			t.Errorf("%s changed the rest: %v", id, got)
		}
		if len(r.run.calls) != 0 {
			t.Errorf("%s ran %q", id, r.run.calls)
		}
	}
}

func TestInstallOverTheOldEntryOfAFileThatIsNotPlainJSONLeavesItAlone(t *testing.T) {
	r := newRig(t)
	orig := "// mine\n{\"mcpServers\":{\"adhd-todo\":{}},}\n"
	r.write(".cursor/mcp.json", orig)
	var me *ManualError
	if err := r.env.Install(context.Background(), r.agent("cursor")); !errors.As(err, &me) {
		t.Fatalf("got %v", err)
	}
	if r.read(".cursor/mcp.json") != orig {
		t.Error("the file was rewritten")
	}
}

func TestInstallCopilotRemovesTheOldEntryFromMcpJsonItself(t *testing.T) {
	r := newRig(t)
	r.path["code"] = "/bin/code"
	file := legacyConfigs["copilot"][0]
	r.write(file, `{"servers":{"adhd-todo":{"command":"x"},"mine":{}},"inputs":[]}`)
	if err := r.env.Install(context.Background(), r.agent("copilot")); err != nil {
		t.Fatal(err)
	}
	if len(r.run.calls) != 1 || r.run.calls[0][1] != "--add-mcp" {
		t.Errorf("ran %q: VS Code has no remove command", r.run.calls)
	}
	got := r.read(file)
	if strings.Contains(got, "adhd-todo") || !strings.Contains(got, `"mine"`) || !strings.Contains(got, `"inputs"`) {
		t.Errorf("mcp.json is now %s", got)
	}
	if r.read(file+".withdo.bak") == got {
		t.Error("no backup of the original")
	}
}

func TestInstallCopilotWithCommentsInMcpJsonKeepsTheFileAndSaysSo(t *testing.T) {
	r := newRig(t)
	r.path["code"] = "/bin/code"
	file := legacyConfigs["copilot"][0]
	orig := "{\n// mine\n\"servers\": {\"adhd-todo\": {}}\n}"
	r.write(file, orig)
	var le *LegacyError
	var me *ManualError
	if err := r.env.Install(context.Background(), r.agent("copilot")); !errors.As(err, &le) || !errors.As(err, &me) {
		t.Fatalf("got %v", err)
	}
	if r.read(file) != orig {
		t.Error("the file was rewritten")
	}
}

func TestInstallDeepSeekHarnessDropsTheOldEntryOfTheHomePatch(t *testing.T) {
	r := newRig(t)
	orig := "# mine\n- id: ui-chat\n  config:\n    a: 1\n\n" + legacyYAML + "\n# trailing note\n- insert:\n    - id: other\n"
	r.write(".dsh/cordis.patch.yml", orig)
	if err := r.env.Install(context.Background(), r.agent("deepseek")); err != nil {
		t.Fatal(err)
	}
	got := r.read(".dsh/cordis.patch.yml")
	for _, want := range []string{"# mine", "- id: ui-chat", "# trailing note", "id: other", "serverName: withdo", "id: withdo-mcp"} {
		if !strings.Contains(got, want) {
			t.Errorf("lost %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "adhd-todo") || strings.Contains(got, "adhd-core") {
		t.Errorf("the old entry is still there:\n%s", got)
	}
	if r.read(".dsh/cordis.patch.yml.withdo.bak") != orig {
		t.Error("no backup")
	}
	if st := r.env.Detect(r.agent("deepseek")); !st.Installed || st.Legacy {
		t.Errorf("%+v", st)
	}
}

func TestInstallDeepSeekHarnessKeepsAnOldEntryItCannotTellFromTheUsersOwn(t *testing.T) {
	r := newRig(t)
	// one list item that inserts two plugins is not something to cut out of
	shared := "- insert:\n    - id: adhd-todo-mcp\n      name: '@deepseek-ai/dsh-mcp-client'\n      config:\n        serverName: adhd-todo\n    - id: other\n      name: x\n"
	r.write(".dsh/cordis.patch.yml", shared)
	var le *LegacyError
	if err := r.env.Install(context.Background(), r.agent("deepseek")); !errors.As(err, &le) {
		t.Fatalf("got %v", err)
	}
	if got := r.read(".dsh/cordis.patch.yml"); !strings.Contains(got, shared) || !strings.Contains(got, "serverName: withdo") {
		t.Errorf("patch is now:\n%s", got)
	}
	// and an old entry in a profile's patch layer is not edited at all
	r2 := newRig(t)
	profile := ".dsh/profiles/web/cordis.patch.yml"
	r2.write(profile, legacyYAML)
	if err := r2.env.Install(context.Background(), r2.agent("deepseek")); !errors.As(err, &le) {
		t.Fatalf("got %v", err)
	}
	if r2.read(profile) != legacyYAML {
		t.Error("a profile layer was rewritten")
	}
}

func TestMigrateMovesOnlyAgentsWithJustTheOldEntryAndNeverStops(t *testing.T) {
	r := newRig(t)
	// cursor and claude have just the old entry; gemini has both; kimi's command is gone; zed is new
	r.write(".cursor/mcp.json", legacyConfigs["cursor"][1])
	r.path["claude"] = "/bin/claude"
	r.write(legacyConfigs["claude"][0], legacyConfigs["claude"][1])
	r.removing(legacyConfigs["claude"][0], "{}")
	r.write(".gemini/settings.json", `{"mcpServers":{"adhd-todo":{},"withdo":{}}}`)
	r.write(legacyConfigs["kimi"][0], legacyConfigs["kimi"][1])
	r.write(".config/zed/settings.json", `{"context_servers":{"withdo":{}}}`)

	got := r.env.Migrate(context.Background())
	byID := map[string]error{}
	for _, m := range got {
		byID[m.Agent.ID] = m.Err
	}
	if len(got) != 3 || byID["cursor"] != nil || byID["claude"] != nil || !errors.Is(byID["kimi"], ErrNotFound) {
		t.Fatalf("migrations %v", byID)
	}
	if st := r.env.Detect(r.agent("cursor")); !st.Installed || st.Legacy {
		t.Errorf("cursor: %+v", st)
	}
	if b := r.read(".gemini/settings.json"); !strings.Contains(b, "adhd-todo") {
		t.Errorf("an agent that had both was touched: %s", b)
	}
	if again := r.env.Migrate(context.Background()); len(again) != 1 || again[0].Agent.ID != "kimi" {
		t.Errorf("a second run found %v, want only the one that failed", again)
	}
}

func TestMigrateDoesNothingWithoutTheCoreOrOnAnEmptyMac(t *testing.T) {
	r := newRig(t)
	if got := r.env.Migrate(context.Background()); len(got) != 0 {
		t.Errorf("on an empty Mac: %v", got)
	}
	r.write(".cursor/mcp.json", legacyConfigs["cursor"][1])
	if err := os.Remove(r.core()); err != nil {
		t.Fatal(err)
	}
	got := r.env.Migrate(context.Background())
	if len(got) != 1 || !errors.Is(got[0].Err, ErrNoCore) {
		t.Errorf("without withdo: %v", got)
	}
	if b := r.read(".cursor/mcp.json"); b != legacyConfigs["cursor"][1] {
		t.Errorf("cursor's config changed: %s", b)
	}
}
