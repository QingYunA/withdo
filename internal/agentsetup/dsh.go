package agentsetup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DeepSeek Harness takes its MCP servers as Cordis patch entries, not JSON. Its patch layers are
// YAML lists: a machine-wide one at $DSH_HOME/cordis.patch.yml that every profile reads after its
// own, and one in each profile. The home layer is the one adhd-todo edits; the profile ones are
// only read, so an install the user made there still counts as installed.
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
			if strings.Trim(strings.TrimSpace(rest), `'"`) == dshMCPServer {
				named = true
			}
		}
	}
	return client && named
}

// dshSnippet is the patch entry to add to a patch list, as a whole top-level list item. The
// server runs adhd-core, like every other agent's.
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
		"        failOnStartupError: true",
	}, "\n")
}

// installYAML adds adhd-todo to the home patch layer by appending one entry. It only ever writes
// a file that is an empty patch list (or []), or one whose first line is a list item; anything
// else is handed back as the manual snippet rather than guessed at.
func (e *Env) installYAML(a Agent) error {
	if e.dshInstalled() {
		return nil
	}
	if !e.exists(e.core()) {
		return ErrNoCore
	}
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
			// a mapping, a flow sequence, or a list with a leading anchor: not something to append to
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

// firstDSHLine is the first line of a patch list that is neither blank nor a comment.
func firstDSHLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			return line
		}
	}
	return ""
}
