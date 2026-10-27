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
	env := &Env{Home: "/Users/me", Core: "/opt/tools/withdo"}
	g := env.Guide()
	for _, want := range []string{
		"`/opt/tools/withdo`", `"command": "/opt/tools/withdo"`, `"mcp"`, "- Name: `withdo`",
		"get_overview", "report_progress", guideCheck, ".withdo.bak", "task_id",
		// an agent that has the old entry is told to replace it
		"named `adhd-todo`", "remove the `adhd-todo` entry",
	} {
		if !strings.Contains(g, want) {
			t.Errorf("the guide lacks %q:\n%s", want, g)
		}
	}
	if strings.Contains(g, "{{") || strings.Contains(g, "}}") {
		t.Errorf("an unfilled placeholder remains:\n%s", g)
	}
	if strings.Contains(g, ".adhd-todo.bak") {
		t.Errorf("the guide still names the old backup:\n%s", g)
	}
	// the generic entry is valid JSON
	start := strings.Index(g, "```json\n") + len("```json\n")
	end := strings.Index(g[start:], "\n```")
	var entry struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	if err := json.Unmarshal([]byte(g[start:start+end]), &entry); err != nil ||
		entry.Command != "/opt/tools/withdo" || len(entry.Args) != 1 || entry.Args[0] != "mcp" {
		t.Errorf("entry %+v, err %v", entry, err)
	}
}

func TestGuideKeepsAWindowsPathWithSpacesWhole(t *testing.T) {
	core := `C:\Users\me\AppData\Local\Programs\Withdo\withdo.exe`
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
