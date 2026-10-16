package agentsetup

import "testing"

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
