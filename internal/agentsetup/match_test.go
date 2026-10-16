package agentsetup

import (
	"os/exec"
	"strings"
	"testing"
)

func TestMatchFindsTheAgentAReportNames(t *testing.T) {
	for reported, want := range map[string]string{
		"claude-code": "claude", "Claude Code": "claude", "claude": "claude", "CLAUDE-CODE": "claude",
		"codex": "codex", "Codex CLI": "codex", "gemini-cli": "gemini", "Qwen Code": "qwen",
		"Visual Studio Code": "copilot", "copilot": "copilot", "zed": "zed", "Zed Preview": "zed", "zed-agent": "zed",
		"cursor": "cursor", "opencode": "opencode",
	} {
		a, ok := Match(reported)
		if !ok || a.ID != want {
			t.Errorf("Match(%q) = %q, %v; want %q", reported, a.ID, ok, want)
		}
	}
	for _, reported := range []string{"", "github", "x", "Agent", "unzipped"} {
		if a, ok := Match(reported); ok {
			t.Errorf("Match(%q) = %q, want none", reported, a.ID)
		}
	}
}

func TestFindUsesTheEnvsLookPath(t *testing.T) {
	r := newRig(t)
	r.path["claude"] = "/somewhere/claude"
	if p, err := r.env.Find("claude"); err != nil || p != "/somewhere/claude" {
		t.Errorf("Find(claude) = %q, %v", p, err)
	}
	if _, err := r.env.Find("nothing"); err == nil || err != exec.ErrNotFound {
		t.Errorf("Find(nothing) err = %v, want ErrNotFound", err)
	}
}

func TestSearchPathStartsWithTheCommandsFolder(t *testing.T) {
	r := newRig(t)
	got := r.env.SearchPath("/opt/tool/bin/gemini")
	if !strings.HasPrefix(got, "/opt/tool/bin:/opt/homebrew/bin:") || !strings.Contains(got, r.home+"/.local/bin") {
		t.Errorf("SearchPath = %q", got)
	}
}
