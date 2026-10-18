package agentsetup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DeepSeek Harness takes its MCP servers as Cordis patch entries, not JSON. Its patch layers are
// YAML lists: a machine-wide one at ~/.dsh/cordis.patch.yml that every profile reads after its
// own, and one in each profile. The home layer is the one adhd-todo edits; the profile ones are
// only read, so an install the user made there still counts as installed. The home layer is always
// ~/.dsh/cordis.patch.yml: DSH_HOME is not read, so a harness the user pointed elsewhere with it is
// not found.
//
// The server entry follows @deepseek-ai/dsh-mcp-client's own documentation (its README's minimal
// configuration): one insert of that plugin with serverName, transport, command and args.
const dshMCPServer = "adhd-todo"
const dshMCPClient = "@deepseek-ai/dsh-mcp-client"

// dshPatchFile is the machine-wide patch layer adhd-todo edits.
func (e *Env) dshPatchFile() string { return e.path(".dsh/cordis.patch.yml") }

// dshPatchFiles are every patch layer a server could be declared in: the home one first, then
// each profile's own.
func (e *Env) dshPatchFiles() []string {
	files := []string{e.dshPatchFile()}
	profiles := e.path(".dsh/profiles")
	entries, err := os.ReadDir(profiles)
	if err != nil {
		return files
	}
	for _, entry := range entries {
		if entry.IsDir() {
			files = append(files, filepath.Join(profiles, entry.Name(), "cordis.patch.yml"))
		}
	}
	return files
}

// dshInstalled reports whether any patch layer inserts adhd-todo's MCP client.
func (e *Env) dshInstalled() bool {
	for _, file := range e.dshPatchFiles() {
		b, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		if dshPatchHasServer(string(b)) {
			return true
		}
	}
	return false
}

// dshPatchHasServer reads a patch list without parsing YAML: a commented example does not count,
// and a line that inserts the client is paired with a serverName of adhd-todo.
func dshPatchHasServer(text string) bool {
	client, named := false, false
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, dshMCPClient) {
			client = true
		}
		if rest, ok := strings.CutPrefix(line, "serverName:"); ok {
			// a trailing YAML comment (a space, then a hash) is not part of the value
			if i := strings.Index(rest, " #"); i >= 0 {
				rest = rest[:i]
			}
			if strings.Trim(strings.TrimSpace(rest), `'"`) == dshMCPServer {
				named = true
			}
		}
	}
	return client && named
}

// dshSnippet is the patch entry to add to a patch list, as a whole top-level list item. The
// server runs adhd-core, like every other agent's. failOnStartupError is false on purpose: per the
// plugin's documentation true makes it reject activation when the server fails to start, while false
// (its default) degrades to zero tools, and adhd-todo must never stop DeepSeek Harness from starting
// when adhd-core is missing or broken.
func (e *Env) dshSnippet() string {
	command := strings.ReplaceAll(e.core(), "'", "''")
	return strings.Join([]string{
		"- insert:",
		"    - id: adhd-todo-mcp",
		"      name: '" + dshMCPClient + "'",
		"      config:",
		"        serverName: " + dshMCPServer,
		"        transport: stdio",
		"        command: '" + command + "'",
		"        args: ['mcp']",
		"        failOnStartupError: false",
	}, "\n")
}

// installYAML adds adhd-todo to the home patch layer by appending one entry. It only ever writes
// a file that is an empty patch list (or []), or one whose first line is a list item; anything
// else is handed back as the manual snippet rather than guessed at.
//
// Install has already returned for an agent that is installed and for a missing adhd-core, so
// neither is checked again here.
func (e *Env) installYAML(a Agent) error {
	file := e.dshPatchFile()
	old, err := os.ReadFile(file)
	exists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("cannot read %s: %w", file, err)
	}
	content := string(old)
	if strings.TrimSpace(content) != "" {
		first := firstDSHLine(content)
		if strings.TrimSpace(content) == "[]" {
			content = ""
		} else if first != "" && !strings.HasPrefix(first, "-") {
			// a mapping, a flow sequence, a list with a leading anchor, or a list indented from
			// column 0 (an entry appended at column 0 would not be valid YAML): not something to
			// append to
			return &ManualError{File: file, Snippet: e.dshSnippet()}
		}
	}
	snippet := e.dshSnippet()
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if content != "" {
		content += "\n"
	}
	content += snippet + "\n"
	if exists {
		if err := writeFile(file+".adhd-todo.bak", old, modeOf(file)); err != nil {
			return fmt.Errorf("cannot back up %s: %w", file, err)
		}
	}
	mode := os.FileMode(0o600)
	if exists {
		mode = modeOf(file)
	}
	return writeFile(file, []byte(content), mode)
}

// firstDSHLine is the first line of a patch list that is neither blank, a comment nor a `---`
// document marker, with its indentation kept so a list that does not start at column 0 can be told
// from one that does.
func firstDSHLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, " \t\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || line == "---" {
			continue
		}
		return line
	}
	return ""
}
