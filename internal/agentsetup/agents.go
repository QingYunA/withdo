// Package agentsetup finds the coding agents on this Mac and adds adhd-todo's MCP server to them.
//
// Detect says whether an agent is here (its command is on PATH, or its config folder exists) and
// whether adhd-todo is already in its MCP config. Install adds it: through the agent's own
// `mcp add` command where it has one, otherwise by editing its config file, after a backup,
// leaving the rest of the file as it was. Nothing here runs on its own: the app calls Install
// from a button.
package agentsetup

import "strings"

// kind is how an agent takes an MCP server.
type kind int

const (
	viaCommand kind = iota // the agent's own `mcp add` command
	viaFile                // an entry added to the agent's JSON config
)

// Agent is a coding agent adhd-todo knows how to set up.
type Agent struct {
	ID   string // stable key: "claude", "codex", ...
	Name string // as people write it; the avatar and the reports' names follow it
	URL  string // where to get it

	// Commands are looked for on PATH, in order; Dirs are config folders under the home folder.
	Commands []string
	Dirs     []string

	// Files are the config files the agent reads its MCP servers from, under the home folder, and
	// Key the top-level key of those files that holds them ("" for Codex's TOML).
	Files []string
	Key   string

	kind kind
}

// Agents is every agent, in the order the app lists them.
var Agents = []Agent{
	{ID: "claude", Name: "Claude Code", URL: "https://claude.com/claude-code", Commands: []string{"claude"},
		Files: []string{".claude.json"}, Key: "mcpServers", kind: viaCommand},
	{ID: "codex", Name: "Codex", URL: "https://developers.openai.com/codex", Commands: []string{"codex"},
		Files: []string{".codex/config.toml"}, kind: viaCommand},
	{ID: "gemini", Name: "Gemini CLI", URL: "https://geminicli.com", Commands: []string{"gemini"},
		Files: []string{".gemini/settings.json"}, Key: "mcpServers", kind: viaCommand},
	{ID: "cursor", Name: "Cursor", URL: "https://cursor.com", Dirs: []string{".cursor"},
		Files: []string{".cursor/mcp.json"}, Key: "mcpServers", kind: viaFile},
	{ID: "opencode", Name: "opencode", URL: "https://opencode.ai", Commands: []string{"opencode"}, Dirs: []string{".config/opencode"},
		Files: []string{".config/opencode/opencode.json", ".config/opencode/opencode.jsonc"}, Key: "mcp", kind: viaFile},
	{ID: "copilot", Name: "Copilot (VS Code)", URL: "https://code.visualstudio.com", Commands: []string{"code"},
		Files: []string{"Library/Application Support/Code/User/mcp.json"}, Key: "servers", kind: viaCommand},
	{ID: "qwen", Name: "Qwen Code", URL: "https://qwenlm.github.io/qwen-code-docs", Commands: []string{"qwen"},
		Files: []string{".qwen/settings.json"}, Key: "mcpServers", kind: viaCommand},
	{ID: "kimi", Name: "Kimi CLI", URL: "https://www.kimi.com/code", Commands: []string{"kimi"},
		Files: []string{".kimi/mcp.json"}, Key: "mcpServers", kind: viaCommand},
	{ID: "windsurf", Name: "Windsurf", URL: "https://windsurf.com", Dirs: []string{".codeium/windsurf"},
		Files: []string{".codeium/windsurf/mcp_config.json"}, Key: "mcpServers", kind: viaFile},
	{ID: "zed", Name: "Zed", URL: "https://zed.dev", Dirs: []string{".config/zed"},
		Files: []string{".config/zed/settings.json"}, Key: "context_servers", kind: viaFile},
}

// ByID finds an agent.
func ByID(id string) (Agent, bool) {
	for _, a := range Agents {
		if a.ID == id {
			return a, true
		}
	}
	return Agent{}, false
}

// Match finds the agent a report's agent string names: "claude-code", "Claude Code" and "claude"
// are all Claude Code, "Visual Studio Code" is Copilot. It matches the way the app's avatars do.
func Match(reported string) (Agent, bool) {
	n := strings.ToLower(reported)
	for _, a := range Agents {
		switch {
		case a.ID == "zed":
			if n == "zed" || strings.HasPrefix(n, "zed ") || strings.HasPrefix(n, "zed-") {
				return a, true
			}
		case a.ID == "copilot":
			if strings.Contains(n, "copilot") || strings.Contains(n, "visual studio code") {
				return a, true
			}
		case strings.Contains(n, a.ID):
			return a, true
		}
	}
	return Agent{}, false
}

// serverName is the name adhd-todo goes by in every agent's config.
const serverName = "adhd-todo"
