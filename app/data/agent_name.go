package data

import "strings"

// unnamedAgent is what an agent that has not reported yet is called (views.unnamedAgent says the
// same).
const unnamedAgent = "agent"

// AgentName is an agent as people write it: the ids agents report under become their product's
// name, matched the way theme.AgentAvatar matches them, so a name and its avatar always agree.
func AgentName(agent string) string {
	switch n := strings.ToLower(agent); {
	case n == "" || n == unnamedAgent:
		return "Agent"
	case n == GitHubAgent:
		return "GitHub"
	case strings.Contains(n, "claude"):
		return "Claude Code"
	case strings.Contains(n, "codex"):
		return "Codex"
	case strings.Contains(n, "deepseek"):
		return "DeepSeek Harness"
	case n == "dsh" || strings.HasPrefix(n, "dsh-") || strings.HasPrefix(n, "dsh "):
		return "DeepSeek Harness"
	case strings.Contains(n, "cursor"):
		return "Cursor"
	case strings.Contains(n, "gemini"):
		return "Gemini CLI"
	case strings.Contains(n, "opencode"):
		return "opencode"
	case strings.Contains(n, "copilot") || strings.Contains(n, "visual studio code"):
		return "Copilot (VS Code)"
	case strings.Contains(n, "qwen"):
		return "Qwen Code"
	case strings.Contains(n, "kimi"):
		return "Kimi CLI"
	case strings.Contains(n, "windsurf"):
		return "Windsurf"
	case n == "zed" || strings.HasPrefix(n, "zed ") || strings.HasPrefix(n, "zed-"):
		return "Zed"
	case n == "amp" || strings.HasPrefix(n, "amp ") || strings.HasPrefix(n, "amp-"):
		return "Amp"
	case strings.Contains(n, "kiro"):
		return "Kiro"
	case strings.Contains(n, "cline"):
		return "Cline"
	}
	return agent
}
