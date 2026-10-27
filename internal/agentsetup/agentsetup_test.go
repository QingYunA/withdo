package agentsetup

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

// fakeRun records the commands it is asked to run and never starts one.
type fakeRun struct {
	calls [][]string
	out   string
	err   error
	block bool
}

func (f *fakeRun) run(ctx context.Context, path string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{path}, args...))
	if f.block {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return []byte(f.out), f.err
}

// rig is an Env on a temporary home folder with a fake PATH and a fake runner.
type rig struct {
	t    *testing.T
	home string
	env  *Env
	run  *fakeRun
	path map[string]string // the commands "on PATH"
}

func newRig(t *testing.T) *rig {
	t.Helper()
	home := t.TempDir()
	r := &rig{t: t, home: home, run: &fakeRun{}, path: map[string]string{}}
	r.env = &Env{Home: home, Core: r.core(), Run: r.run.run, LookPath: func(name string) (string, error) {
		if p, ok := r.path[name]; ok {
			return p, nil
		}
		return "", exec.ErrNotFound
	}}
	r.write(".local/bin/withdo", "#!/bin/sh\n")
	return r
}

func (r *rig) core() string { return filepath.Join(r.home, ".local", "bin", "withdo") }

func (r *rig) write(rel, content string) {
	r.t.Helper()
	p := filepath.Join(r.home, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

func (r *rig) read(rel string) string {
	r.t.Helper()
	b, err := os.ReadFile(filepath.Join(r.home, filepath.FromSlash(rel)))
	if err != nil {
		r.t.Fatal(err)
	}
	return string(b)
}

func (r *rig) agent(id string) Agent {
	a, ok := ByID(id)
	if !ok {
		r.t.Fatalf("no agent %q", id)
	}
	return a
}

func TestDetectFindsACommandOnPathOrAConfigFolder(t *testing.T) {
	r := newRig(t)
	r.path["gemini"] = "/usr/local/bin/gemini"
	r.write(".cursor/mcp.json", "{}")
	got := r.env.DetectAll()
	if s := got["gemini"]; !s.Found || s.Via != "gemini" || s.Installed {
		t.Errorf("gemini: %+v", s)
	}
	if s := got["cursor"]; !s.Found || s.Via != "~/.cursor" || s.Installed {
		t.Errorf("cursor: %+v", s)
	}
	for _, id := range []string{"claude", "codex", "deepseek", "opencode", "copilot", "qwen", "kimi", "windsurf", "zed", "amp", "kiro", "cline"} {
		if got[id].Found {
			t.Errorf("%s was found on an empty Mac: %+v", id, got[id])
		}
	}
	if len(got) != len(Agents) {
		t.Errorf("%d statuses for %d agents", len(got), len(Agents))
	}
}

func TestACommandAloneDoesNotCountForAnAgentWhoseFolderIsItsConfig(t *testing.T) {
	r := newRig(t)
	r.write(".claude/settings.json", "{}") // a folder, no `claude` command: nothing to run
	if r.env.Detect(r.agent("claude")).Found {
		t.Error("Claude Code was found by its folder, but it takes its MCP through its command")
	}
}

func TestInstalledReadsEachAgentsConfig(t *testing.T) {
	r := newRig(t)
	cases := map[string][2]string{ // agent: file, content
		"claude":   {".claude.json", `{"numStartups":3,"mcpServers":{"withdo":{"command":"x"}}}`},
		"codex":    {".codex/config.toml", "model = \"o3\"\n\n[mcp_servers.withdo]\ncommand = \"x\"\n"},
		"gemini":   {".gemini/settings.json", `{"mcpServers":{"withdo":{"command":"x"}}}`},
		"qwen":     {".qwen/settings.json", `{"mcpServers":{"withdo":{"command":"x"}}}`},
		"kimi":     {".kimi/mcp.json", `{"mcpServers":{"withdo":{"command":"x"}}}`},
		"cursor":   {".cursor/mcp.json", `{"mcpServers":{"withdo":{"command":"x"}}}`},
		"opencode": {".config/opencode/opencode.json", `{"mcp":{"withdo":{"type":"local"}}}`},
		"windsurf": {".codeium/windsurf/mcp_config.json", `{"mcpServers":{"withdo":{}}}`},
		"copilot":  {"Library/Application Support/Code/User/mcp.json", "{\n// mine\n\"servers\": {\"withdo\": {\"command\": \"x\",},},\n}"},
		"zed":      {".config/zed/settings.json", "// Zed settings\n{\n  /* servers */\n  \"context_servers\": {\n    \"withdo\": { \"command\": \"x\" }, // here\n  },\n}"},
		"amp":      {".config/amp/settings.json", `{"amp.mcpServers":{"withdo":{"command":"x"}}}`},
		"kiro":     {".kiro/settings/mcp.json", `{"mcpServers":{"withdo":{"command":"x"}}}`},
		"cline":    {"Library/Application Support/Code/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json", `{"mcpServers":{"withdo":{"command":"x"}}}`},
	}
	for id, c := range cases {
		a := r.agent(id)
		if r.env.Installed(a) {
			t.Errorf("%s: installed before its config exists", id)
		}
		r.write(c[0], c[1])
		if !r.env.Installed(a) {
			t.Errorf("%s: not seen as installed in %s", id, c[0])
		}
	}
}

func TestInstalledSaysNoToAnotherServerAndToABrokenFile(t *testing.T) {
	r := newRig(t)
	r.write(".gemini/settings.json", `{"mcpServers":{"other":{}}}`)
	r.write(".codex/config.toml", "[mcp_servers.other]\ncommand = \"x\"\n# [mcp_servers.withdo]\n")
	r.write(".kimi/mcp.json", `{"mcpServers": `)
	for _, id := range []string{"gemini", "codex", "kimi"} {
		if r.env.Installed(r.agent(id)) {
			t.Errorf("%s: seen as installed", id)
		}
	}
	r.write(".codex/config.toml", "[mcp_servers.\"withdo\".env]\nA = \"1\"\n")
	if !r.env.Installed(r.agent("codex")) {
		t.Error("a quoted table name was missed")
	}
}

func TestInstallRunsEachAgentsOwnCommandWithTheAbsolutePathOfTheCore(t *testing.T) {
	core := ""
	for id, want := range map[string][]string{
		"claude": {"mcp", "add", "--scope", "user", "withdo", "--", "CORE", "mcp"},
		"codex":  {"mcp", "add", "withdo", "--", "CORE", "mcp"},
		"gemini": {"mcp", "add", "-s", "user", "withdo", "CORE", "mcp"},
		"qwen":   {"mcp", "add", "--scope", "user", "withdo", "CORE", "mcp"},
		"kimi":   {"mcp", "add", "--transport", "stdio", "withdo", "--", "CORE", "mcp"},
		"amp":    {"mcp", "add", "withdo", "--", "CORE", "mcp"},
	} {
		r := newRig(t)
		core = r.core()
		r.path[id] = "/bin/" + id
		if err := r.env.Install(context.Background(), r.agent(id)); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		want = slices.Clone(want)
		for i, w := range want {
			if w == "CORE" {
				want[i] = core
			}
		}
		if len(r.run.calls) != 1 || !slices.Equal(r.run.calls[0], append([]string{"/bin/" + id}, want...)) {
			t.Errorf("%s ran %q, want %q", id, r.run.calls, want)
		}
	}
}

func TestInstallCopilotAddsTheServerToVSCodeAsJSON(t *testing.T) {
	r := newRig(t)
	r.path["code"] = "/bin/code"
	if err := r.env.Install(context.Background(), r.agent("copilot")); err != nil {
		t.Fatal(err)
	}
	call := r.run.calls[0]
	if len(call) != 3 || call[0] != "/bin/code" || call[1] != "--add-mcp" {
		t.Fatalf("ran %q", call)
	}
	var spec struct {
		Name, Command string
		Args          []string
	}
	if err := json.Unmarshal([]byte(call[2]), &spec); err != nil {
		t.Fatal(err)
	}
	if spec.Name != "withdo" || spec.Command != r.core() || !slices.Equal(spec.Args, []string{"mcp"}) {
		t.Errorf("spec %+v", spec)
	}
}

func TestInstallDoesNothingWhenItIsThereAlready(t *testing.T) {
	r := newRig(t)
	r.path["gemini"] = "/bin/gemini"
	r.write(".gemini/settings.json", `{"mcpServers":{"withdo":{}}}`)
	r.write(".cursor/mcp.json", `{"mcpServers":{"withdo":{"command":"mine"}}}`)
	for _, id := range []string{"gemini", "cursor"} {
		if err := r.env.Install(context.Background(), r.agent(id)); err != nil {
			t.Errorf("%s: %v", id, err)
		}
	}
	if len(r.run.calls) != 0 {
		t.Errorf("ran %q", r.run.calls)
	}
	if got := r.read(".cursor/mcp.json"); !strings.Contains(got, `"mine"`) {
		t.Errorf("the entry was rewritten: %s", got)
	}
	if _, err := os.Stat(filepath.Join(r.home, ".cursor", "mcp.json.withdo.bak")); err == nil {
		t.Error("a backup was made for a file that did not change")
	}
}

func TestInstallErrors(t *testing.T) {
	r := newRig(t)
	if err := r.env.Install(context.Background(), r.agent("codex")); !errors.Is(err, ErrNotFound) {
		t.Errorf("no command: %v", err)
	}
	r.path["codex"] = "/bin/codex"
	r.run.out, r.run.err = "warning: x\nerror: unknown option --scope\n", errors.New("exit status 2")
	var ce *CommandError
	if err := r.env.Install(context.Background(), r.agent("codex")); !errors.As(err, &ce) || ce.Output != "error: unknown option --scope" {
		t.Errorf("a failing command: %v", err)
	}
	if err := os.Remove(r.core()); err != nil {
		t.Fatal(err)
	}
	if err := r.env.Install(context.Background(), r.agent("codex")); !errors.Is(err, ErrNoCore) {
		t.Errorf("no core: %v", err)
	}
}

func TestInstallGivesUpOnACommandThatNeverEnds(t *testing.T) {
	r := newRig(t)
	r.path["claude"] = "/bin/claude"
	r.run.block = true
	r.env.Timeout = 20 * time.Millisecond
	if err := r.env.Install(context.Background(), r.agent("claude")); !errors.Is(err, ErrTimeout) {
		t.Errorf("got %v", err)
	}
}

func TestInstallEditsAFileKeepingTheRestInOrderAfterABackup(t *testing.T) {
	r := newRig(t)
	orig := "{\n\t\"theme\": \"dark\",\n\t\"big\": 12345678901234567890,\n\t\"mcpServers\": {\n\t\t\"other\": {\"url\": \"http://a/?x=1&y=<2>\"}\n\t},\n\t\"z\": [1, 2.50, null]\n}\n"
	r.write(".cursor/mcp.json", orig)
	if err := r.env.Install(context.Background(), r.agent("cursor")); err != nil {
		t.Fatal(err)
	}
	if got := r.read(".cursor/mcp.json.withdo.bak"); got != orig {
		t.Errorf("backup %q", got)
	}
	got := r.read(".cursor/mcp.json")
	for _, keep := range []string{"12345678901234567890", "2.50", "http://a/?x=1&y=<2>", "\t\"theme\": \"dark\""} {
		if !strings.Contains(got, keep) {
			t.Errorf("lost %q in\n%s", keep, got)
		}
	}
	if !(strings.Index(got, `"theme"`) < strings.Index(got, `"big"`) && strings.Index(got, `"big"`) < strings.Index(got, `"mcpServers"`) &&
		strings.Index(got, `"other"`) < strings.Index(got, `"withdo"`) && strings.Index(got, `"mcpServers"`) < strings.Index(got, `"z"`)) {
		t.Errorf("the order changed:\n%s", got)
	}
	var v struct {
		MCPServers map[string]struct {
			Type    string
			Command string
			Args    []string
		}
	}
	if err := json.Unmarshal([]byte(got), &v); err != nil {
		t.Fatal(err)
	}
	if e := v.MCPServers["withdo"]; e.Type != "stdio" || e.Command != r.core() || !slices.Equal(e.Args, []string{"mcp"}) {
		t.Errorf("entry %+v", e)
	}
	if !r.env.Installed(r.agent("cursor")) {
		t.Error("not installed after Install")
	}
}

func TestInstallCreatesAMissingFileEachAgentsWay(t *testing.T) {
	want := map[string]string{
		"cursor":   `"mcpServers": {"withdo": {"type":"stdio","command":"CORE","args":["mcp"]}}`,
		"opencode": `"mcp": {"withdo": {"type":"local","command":["CORE","mcp"],"enabled":true}}`,
		"windsurf": `"mcpServers": {"withdo": {"command":"CORE","args":["mcp"]}}`,
		"zed":      `"context_servers": {"withdo": {"command":"CORE","args":["mcp"],"env":{}}}`,
	}
	for id, w := range want {
		r := newRig(t)
		a := r.agent(id)
		if err := r.env.Install(context.Background(), a); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		var got, exp any
		if err := json.Unmarshal([]byte(r.read(a.Files[0])), &got); err != nil {
			t.Fatalf("%s wrote invalid JSON: %v", id, err)
		}
		coreJSON := strings.Trim(mustMarshal(t, r.core()), `"`)
		if err := json.Unmarshal([]byte("{"+strings.ReplaceAll(w, "CORE", coreJSON)+"}"), &exp); err != nil {
			t.Fatal(err)
		}
		if gj, _ := json.Marshal(got); string(gj) != mustMarshal(t, exp) {
			t.Errorf("%s wrote %s, want %s", id, gj, mustMarshal(t, exp))
		}
		if _, err := os.Stat(filepath.Join(r.home, filepath.FromSlash(a.Files[0])+".withdo.bak")); err == nil {
			t.Errorf("%s: a backup of a file that was not there", id)
		}
	}
}

func mustMarshal(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestInstallLeavesAFileWithCommentsAloneAndOffersTheSnippet(t *testing.T) {
	r := newRig(t)
	orig := "// Zed settings\n{\n  \"theme\": \"One\", // mine\n}\n"
	r.write(".config/zed/settings.json", orig)
	err := r.env.Install(context.Background(), r.agent("zed"))
	var me *ManualError
	if !errors.As(err, &me) {
		t.Fatalf("got %v", err)
	}
	if r.read(".config/zed/settings.json") != orig {
		t.Error("the file was rewritten")
	}
	for _, want := range []string{`"context_servers": {`, `"withdo": {`, `"command": ` + mustMarshal(t, r.core()), `"args": [`, `"env": {}`} {
		if !strings.Contains(me.Snippet, want) {
			t.Errorf("snippet lacks %q:\n%s", want, me.Snippet)
		}
	}
	if strings.HasPrefix(me.Snippet, "{") || strings.HasPrefix(me.Snippet, " ") {
		t.Errorf("the snippet is not what goes inside an existing object:\n%s", me.Snippet)
	}
	if _, err := os.Stat(filepath.Join(r.home, ".config", "zed", "settings.json.withdo.bak")); err == nil {
		t.Error("a backup for a file that did not change")
	}
}

func TestInstallAnEmptyFileAndAKeyThatIsNotAnObject(t *testing.T) {
	r := newRig(t)
	r.write(".cursor/mcp.json", "  \n")
	if err := r.env.Install(context.Background(), r.agent("cursor")); err != nil {
		t.Fatal(err)
	}
	r.write(".codeium/windsurf/mcp_config.json", `{"mcpServers": []}`)
	var me *ManualError
	if err := r.env.Install(context.Background(), r.agent("windsurf")); !errors.As(err, &me) {
		t.Errorf("got %v", err)
	}
	if r.read(".codeium/windsurf/mcp_config.json") != `{"mcpServers": []}` {
		t.Error("the file was rewritten")
	}
}

func TestInstallKeepsTheModeOfTheFileAndItsTwoSpaceIndent(t *testing.T) {
	r := newRig(t)
	r.write(".codeium/windsurf/mcp_config.json", "{\n  \"a\": 1\n}")
	p := filepath.Join(r.home, ".codeium", "windsurf", "mcp_config.json")
	if err := os.Chmod(p, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := r.env.Install(context.Background(), r.agent("windsurf")); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Stat(p); runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600 {
		t.Errorf("mode %v", fi.Mode().Perm())
	}
	if got := r.read(".codeium/windsurf/mcp_config.json"); !strings.HasPrefix(got, "{\n  \"a\": 1,\n  \"mcpServers\": {\n    \"withdo\"") {
		t.Errorf("indent changed:\n%s", got)
	}
}

func TestStripJSONCKeepsCommentLikeTextInStrings(t *testing.T) {
	in := `{"u": "http://x/*y*/", "a": [1,2,], /* c */ "b": "q\"//" // end
}`
	var v map[string]any
	if err := json.Unmarshal(stripJSONC([]byte(in)), &v); err != nil {
		t.Fatalf("%v in %q", err, stripJSONC([]byte(in)))
	}
	if v["u"] != "http://x/*y*/" || v["b"] != `q"//` {
		t.Errorf("%v", v)
	}
}

func TestAgentTableIsTheFourteenInOrder(t *testing.T) {
	var names []string
	for _, a := range Agents {
		names = append(names, a.Name)
		if a.URL == "" || len(a.Files) == 0 {
			t.Errorf("%s: no URL or config file", a.Name)
		}
	}
	want := []string{"Claude Code", "Codex", "DeepSeek Harness", "Gemini CLI", "Cursor", "opencode", "Copilot (VS Code)", "Qwen Code", "Kimi CLI", "Windsurf", "Zed", "Amp", "Kiro", "Cline"}
	if !slices.Equal(names, want) {
		t.Errorf("%q", names)
	}
}

// DeepSeek Harness has no `dsh` command on this Mac; its config folder is what says it is here.
func TestDeepSeekHarnessIsFoundByItsConfigFolder(t *testing.T) {
	r := newRig(t)
	if st := r.env.Detect(r.agent("deepseek")); st.Found {
		t.Fatalf("found on an empty Mac: %+v", st)
	}
	r.write(".dsh/profiles/desktop/cordis.patch.yml", "[]\n")
	if st := r.env.Detect(r.agent("deepseek")); !st.Found || st.Via != "~/.dsh" || st.Installed {
		t.Errorf("deepseek: %+v", st)
	}
}

// The reported harnesses that keep an MCP file of their own are found by it.
func TestKiroCursorStyleAgentsAreFoundByTheirFolder(t *testing.T) {
	r := newRig(t)
	r.path["amp"] = "/usr/local/bin/amp"
	r.write(".kiro/settings/mcp.json", "{}")
	r.write("Library/Application Support/Code/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json", "{}")
	got := r.env.DetectAll()
	for id, via := range map[string]string{
		"amp":   "amp",
		"kiro":  "~/.kiro",
		"cline": "~/Library/Application Support/Code/User/globalStorage/saoudrizwan.claude-dev",
	} {
		if st := got[id]; !st.Found || st.Via != via {
			t.Errorf("%s: %+v, want found via %q", id, st, via)
		}
	}
}

// The Cordis patch layers: the machine-wide one and each profile's own all count as installed,
// and a commented example does not.
func TestDeepSeekHarnessInstalledReadsEveryPatchLayer(t *testing.T) {
	r := newRig(t)
	a := r.agent("deepseek")
	if r.env.Installed(a) {
		t.Fatal("installed on an empty Mac")
	}
	r.write(".dsh/cordis.patch.yml", "# - insert:\n#     - name: '@deepseek-ai/dsh-mcp-client'\n#       config:\n#         serverName: withdo\n")
	if r.env.Installed(a) {
		t.Error("a commented example counted")
	}
	r.write(".dsh/profiles/web/cordis.patch.yml", "- insert:\n    - id: mcp-withdo\n      name: '@deepseek-ai/dsh-mcp-client'\n      config:\n        serverName: withdo\n        transport: stdio\n        command: '/x/withdo'\n        args: ['mcp']\n")
	if !r.env.Installed(a) {
		t.Error("the profile patch layer was not read")
	}
}

func TestInstallDeepSeekHarnessAppendsACordisPatchEntry(t *testing.T) {
	r := newRig(t)
	orig := "# my patches\n- id: ui-chat\n  config:\n    transcriptView: standard\n"
	r.write(".dsh/cordis.patch.yml", orig)
	if err := r.env.Install(context.Background(), r.agent("deepseek")); err != nil {
		t.Fatal(err)
	}
	got := r.read(".dsh/cordis.patch.yml")
	for _, want := range []string{
		"# my patches", "- id: ui-chat", "    transcriptView: standard",
		"- insert:", "id: withdo-mcp", "name: '@deepseek-ai/dsh-mcp-client'",
		"serverName: withdo", "transport: stdio", "command: '" + r.core() + "'",
		"args: ['mcp']", "failOnStartupError: false",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the patch lacks %q:\n%s", want, got)
		}
	}
	if r.read(".dsh/cordis.patch.yml.withdo.bak") != orig {
		t.Error("no backup of the patch layer")
	}
	if len(r.run.calls) != 0 {
		t.Errorf("ran %q", r.run.calls)
	}
	if !r.env.Installed(r.agent("deepseek")) {
		t.Error("not installed after Install")
	}
	if err := r.env.Install(context.Background(), r.agent("deepseek")); err != nil {
		t.Fatal(err)
	}
	if r.read(".dsh/cordis.patch.yml") != got {
		t.Error("a second install appended the server again")
	}
}

func TestInstallDeepSeekHarnessCreatesTheHomePatchOrFillsAnEmptyList(t *testing.T) {
	r := newRig(t)
	if err := r.env.Install(context.Background(), r.agent("deepseek")); err != nil {
		t.Fatal(err)
	}
	if got := r.read(".dsh/cordis.patch.yml"); !strings.HasPrefix(got, "- insert:\n") {
		t.Errorf("a created patch starts %q", got)
	}
	if _, err := os.Stat(filepath.Join(r.home, ".dsh", "cordis.patch.yml.withdo.bak")); err == nil {
		t.Error("a backup of a file that was not there")
	}
	r.write(".dsh/cordis.patch.yml", "[]\n")
	if err := r.env.Install(context.Background(), r.agent("deepseek")); err != nil {
		t.Fatal(err)
	}
	got := r.read(".dsh/cordis.patch.yml")
	if strings.Contains(got, "[]") || !strings.Contains(got, "serverName: withdo") {
		t.Errorf("the empty list was not replaced:\n%s", got)
	}
}

func TestInstallDeepSeekHarnessHandsBackAPatchItCannotAppendTo(t *testing.T) {
	r := newRig(t)
	orig := "some: mapping\n"
	r.write(".dsh/cordis.patch.yml", orig)
	var me *ManualError
	if err := r.env.Install(context.Background(), r.agent("deepseek")); !errors.As(err, &me) {
		t.Fatalf("got %v", err)
	}
	if r.read(".dsh/cordis.patch.yml") != orig {
		t.Error("the patch was rewritten")
	}
	for _, want := range []string{"- insert:", "@deepseek-ai/dsh-mcp-client", "serverName: withdo", "command: '" + r.core() + "'"} {
		if !strings.Contains(me.Snippet, want) {
			t.Errorf("snippet lacks %q:\n%s", want, me.Snippet)
		}
	}
	if me.File != filepath.Join(r.home, ".dsh", "cordis.patch.yml") {
		t.Errorf("manual file %q", me.File)
	}
}

// A list indented from column 0 would get an entry appended at column 0, which is not valid YAML.
func TestInstallDeepSeekHarnessHandsBackAnIndentedPatchList(t *testing.T) {
	r := newRig(t)
	orig := "  - insert:\n      - id: other\n"
	r.write(".dsh/cordis.patch.yml", orig)
	var me *ManualError
	if err := r.env.Install(context.Background(), r.agent("deepseek")); !errors.As(err, &me) {
		t.Fatalf("got %v", err)
	}
	if r.read(".dsh/cordis.patch.yml") != orig {
		t.Error("the patch was rewritten")
	}
	if !strings.Contains(me.Snippet, "- insert:") {
		t.Errorf("no snippet:\n%s", me.Snippet)
	}
}

// A document marker and comments before a list at column 0 do not stop the entry being appended.
func TestInstallDeepSeekHarnessAppendsAfterADocumentMarker(t *testing.T) {
	r := newRig(t)
	r.write(".dsh/cordis.patch.yml", "---\n# mine\n- insert:\n    - id: other\n")
	if err := r.env.Install(context.Background(), r.agent("deepseek")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.read(".dsh/cordis.patch.yml"), "serverName: withdo") {
		t.Error("the entry was not appended")
	}
}

// A comment after the serverName value is not part of it.
func TestDeepSeekHarnessServerNameMayHaveATrailingComment(t *testing.T) {
	for _, line := range []string{"serverName: withdo  # note", `serverName: "withdo" # x`} {
		text := "- insert:\n    - name: '@deepseek-ai/dsh-mcp-client'\n      config:\n        " + line + "\n"
		if !dshPatchHasServer(text, "withdo") {
			t.Errorf("%q did not count as installed", line)
		}
	}
	if dshPatchHasServer("- insert:\n    - name: '@deepseek-ai/dsh-mcp-client'\n      config:\n        serverName: other # withdo\n", "withdo") {
		t.Error("a name in the comment counted")
	}
}

func TestInstallKiroAndClineWriteTheirMcpJson(t *testing.T) {
	for _, id := range []string{"kiro", "cline"} {
		r := newRig(t)
		a := r.agent(id)
		if err := r.env.Install(context.Background(), a); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		var got struct {
			MCPServers map[string]struct {
				Command string
				Args    []string
			} `json:"mcpServers"`
		}
		if err := json.Unmarshal([]byte(r.read(a.Files[0])), &got); err != nil {
			t.Fatalf("%s wrote invalid JSON: %v", id, err)
		}
		e := got.MCPServers["withdo"]
		if e.Command != r.core() || !slices.Equal(e.Args, []string{"mcp"}) {
			t.Errorf("%s entry %+v", id, e)
		}
		if !r.env.Installed(a) {
			t.Errorf("%s not installed after Install", id)
		}
	}
}

func TestInstallAmpUsesItsOwnMCPCommand(t *testing.T) {
	r := newRig(t)
	r.path["amp"] = "/usr/local/bin/amp"
	if err := r.env.Install(context.Background(), r.agent("amp")); err != nil {
		t.Fatal(err)
	}
	want := []string{"/usr/local/bin/amp", "mcp", "add", "withdo", "--", r.core(), "mcp"}
	if len(r.run.calls) != 1 || !slices.Equal(r.run.calls[0], want) {
		t.Errorf("ran %q, want %q", r.run.calls, want)
	}
}

func TestDefaultCore(t *testing.T) {
	cases := []struct{ name, goos, home, exe, want string }{
		// filepath is the host's, so the Windows case uses slashes to run the same on every OS
		{"windows runs the one next to the app", "windows", "/home/me", "/home/me/Programs/Withdo/Withdo.exe", filepath.Join("/home/me", "Programs", "Withdo", "withdo.exe")},
		{"windows in the folder of the old name", "windows", "/home/me", "/home/me/Programs/ADHD Todo/ADHD Todo.exe", filepath.Join("/home/me", "Programs", "ADHD Todo", "withdo.exe")},
		{"windows without a known executable", "windows", "/home/me", "", filepath.Join("/home/me", ".local", "bin", "withdo")},
		{"darwin", "darwin", "/Users/me", "/Applications/Withdo.app/Contents/MacOS/Withdo", filepath.Join("/Users/me", ".local", "bin", "withdo")},
		{"linux", "linux", "/home/me", "/usr/bin/withdo", filepath.Join("/home/me", ".local", "bin", "withdo")},
	}
	for _, c := range cases {
		if got := defaultCore(c.goos, c.home, c.exe); got != c.want {
			t.Errorf("%s: defaultCore = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestSearchPathPutsTheDirsFirstAndKeepsTheOldPathIntact(t *testing.T) {
	sep := string(filepath.ListSeparator)
	old := strings.Join([]string{"first", "second"}, sep)
	got := searchPath([]string{"a", "b"}, old)
	if want := strings.Join([]string{"a", "b", "first", "second"}, sep); got != want {
		t.Errorf("searchPath = %q, want %q", got, want)
	}
	if got := searchPath([]string{"a"}, ""); got != "a"+sep {
		t.Errorf("with no old PATH: %q", got)
	}
}

// VS Code (Copilot, Cline) and Zed keep their settings in a different folder on each system: on
// Windows it is %APPDATA%, which is AppData/Roaming under the home folder.
func TestAgentConfigPathsFollowTheSystem(t *testing.T) {
	want := map[string]map[string][]string{
		"darwin": {
			"copilot": {"Library/Application Support/Code/User/mcp.json"},
			"cline":   {"Library/Application Support/Code/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json"},
			"zed":     {".config/zed/settings.json"},
		},
		"windows": {
			"copilot": {"AppData/Roaming/Code/User/mcp.json"},
			"cline":   {"AppData/Roaming/Code/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json"},
			"zed":     {"AppData/Roaming/Zed/settings.json"},
		},
	}
	wantDirs := map[string]map[string][]string{
		"darwin":  {"cline": {"Library/Application Support/Code/User/globalStorage/saoudrizwan.claude-dev"}, "zed": {".config/zed"}},
		"windows": {"cline": {"AppData/Roaming/Code/User/globalStorage/saoudrizwan.claude-dev"}, "zed": {"AppData/Roaming/Zed"}},
	}
	for goos, files := range want {
		agents := agentsFor(goos)
		for _, a := range agents {
			if f, ok := files[a.ID]; ok && !slices.Equal(a.Files, f) {
				t.Errorf("%s %s: Files = %q, want %q", goos, a.ID, a.Files, f)
			}
			if d, ok := wantDirs[goos][a.ID]; ok && !slices.Equal(a.Dirs, d) {
				t.Errorf("%s %s: Dirs = %q, want %q", goos, a.ID, a.Dirs, d)
			}
			if goos == "windows" {
				for _, p := range slices.Concat(a.Files, a.Dirs) {
					if strings.HasPrefix(p, "Library/") {
						t.Errorf("windows %s: macOS path %q", a.ID, p)
					}
				}
			}
		}
	}
	if !slices.EqualFunc(Agents, agentsFor(runtime.GOOS), func(a, b Agent) bool { return a.ID == b.ID && slices.Equal(a.Files, b.Files) }) {
		t.Error("Agents is not the table of this system")
	}
}

// A command line tool's usual places differ by system; Windows has no Homebrew or .app bundles.
func TestExtraDirsFollowTheSystem(t *testing.T) {
	home := filepath.Join("H", "me")
	mac := extraDirs("darwin", home)
	for _, d := range []string{"/opt/homebrew/bin", filepath.Join(home, ".local", "bin"), "/Applications/Visual Studio Code.app/Contents/Resources/app/bin"} {
		if !slices.Contains(mac, d) {
			t.Errorf("darwin: %q missing from %q", d, mac)
		}
	}
	win := extraDirs("windows", home)
	for _, d := range []string{
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, "AppData", "Roaming", "npm"),
		filepath.Join(home, "AppData", "Local", "Programs", "Microsoft VS Code", "bin"),
	} {
		if !slices.Contains(win, d) {
			t.Errorf("windows: %q missing from %q", d, win)
		}
	}
	for _, d := range win {
		if strings.HasPrefix(d, "/") {
			t.Errorf("windows: Unix path %q", d)
		}
	}
}
