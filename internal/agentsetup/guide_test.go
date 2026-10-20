package agentsetup

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// guideCheck is how the guide tells an agent to check the command, without any shell's syntax.
const guideCheck = "check that the file at the command path exists, and that running it with the single argument `--help` prints its usage"

func TestGuideNamesTheServerAndFillsInTheCorePath(t *testing.T) {
	env := &Env{Home: "/Users/me", Core: "/opt/tools/adhd-core"}
	g := env.Guide()
	for _, want := range []string{
		"`/opt/tools/adhd-core`", `"command": "/opt/tools/adhd-core"`, `"mcp"`, "adhd-todo",
		"get_overview", "report_progress", guideCheck, ".adhd-todo.bak", "task_id",
	} {
		if !strings.Contains(g, want) {
			t.Errorf("the guide lacks %q:\n%s", want, g)
		}
	}
	if strings.Contains(g, "{{") || strings.Contains(g, "}}") {
		t.Errorf("an unfilled placeholder remains:\n%s", g)
	}
	// the generic entry is valid JSON
	start := strings.Index(g, "```json\n") + len("```json\n")
	end := strings.Index(g[start:], "\n```")
	var entry struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	if err := json.Unmarshal([]byte(g[start:start+end]), &entry); err != nil ||
		entry.Command != "/opt/tools/adhd-core" || len(entry.Args) != 1 || entry.Args[0] != "mcp" {
		t.Errorf("entry %+v, err %v", entry, err)
	}
}

func TestGuideKeepsAWindowsPathWithSpacesWhole(t *testing.T) {
	core := `C:\Users\me\AppData\Local\Programs\ADHD Todo\adhd-core.exe`
	g := (&Env{Home: `C:\Users\me`, Core: core}).Guide()
	if !strings.Contains(g, "`"+core+"`") {
		t.Errorf("the guide lacks the path:\n%s", g)
	}
	if strings.Contains(g, core+" --help") {
		t.Errorf("the guide glues --help to the path, which PowerShell cannot run:\n%s", g)
	}
	start := strings.Index(g, "```json\n") + len("```json\n")
	end := strings.Index(g[start:], "\n```")
	var entry struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(g[start:start+end]), &entry); err != nil || entry.Command != core {
		t.Errorf("entry %+v, err %v", entry, err)
	}
}

func TestGuideDefaultsToTheCoreTheKnownAgentsGet(t *testing.T) {
	env := New(filepath.FromSlash("/Users/me"))
	if g := env.Guide(); !strings.Contains(g, env.CorePath()) {
		t.Errorf("the guide does not use %s:\n%s", env.CorePath(), g)
	}
}
