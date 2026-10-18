package agentsetup

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGuideNamesTheServerAndFillsInTheCorePath(t *testing.T) {
	env := &Env{Home: "/Users/me", Core: "/opt/tools/adhd-core"}
	g := env.Guide()
	for _, want := range []string{
		"`/opt/tools/adhd-core`", `"command": "/opt/tools/adhd-core"`, `"mcp"`, "adhd-todo",
		"get_overview", "report_progress", "/opt/tools/adhd-core --help", ".adhd-todo.bak", "task_id",
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

func TestGuideDefaultsToTheCoreTheKnownAgentsGet(t *testing.T) {
	env := New("/Users/me")
	if g := env.Guide(); !strings.Contains(g, env.CorePath()) || !strings.Contains(g, "/Users/me/.local/bin/adhd-core") {
		t.Errorf("the guide does not use %s:\n%s", env.CorePath(), g)
	}
}
