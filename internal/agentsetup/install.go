package agentsetup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Why an install did not happen. The app turns each into a sentence of its own language.
var (
	// ErrNoCore is when the adhd-core binary the agent would run is not where it should be.
	ErrNoCore = errors.New("adhd-core is not installed")
	// ErrNotFound is when the agent's command is not on this Mac.
	ErrNotFound = errors.New("the agent is not installed")
	// ErrTimeout is when the agent's own command took too long.
	ErrTimeout = errors.New("the command took too long")
)

// CommandError is the agent's own command failing: Output is the last of what it printed.
type CommandError struct {
	Agent  string
	Output string
	Err    error
}

func (e *CommandError) Error() string {
	if e.Output != "" {
		return e.Agent + ": " + e.Output
	}
	return e.Agent + ": " + e.Err.Error()
}

func (e *CommandError) Unwrap() error { return e.Err }

// ManualError is a config file that is not plain JSON (it has comments, for one): it is not
// rewritten, and Snippet is what to add to File by hand.
type ManualError struct {
	File    string
	Snippet string
}

func (e *ManualError) Error() string {
	return "cannot edit " + e.File + " safely, it is not plain JSON"
}

// Install adds adhd-todo's MCP server to the agent, unless it is there already. It runs the
// agent's own command or edits its config file (after a copy to <file>.adhd-todo.bak).
func (e *Env) Install(ctx context.Context, a Agent) error {
	if e.Installed(a) {
		return nil
	}
	if !e.exists(e.core()) {
		return ErrNoCore
	}
	if a.kind == viaFile {
		return e.installFile(a)
	}
	return e.installCommand(ctx, a)
}

// commandLine is the agent's own `mcp add` command, without the program's path.
func (e *Env) commandLine(a Agent) (program string, args []string) {
	core := e.core()
	switch a.ID {
	case "claude":
		return "claude", []string{"mcp", "add", "--scope", "user", serverName, "--", core, "mcp"}
	case "codex":
		return "codex", []string{"mcp", "add", serverName, "--", core, "mcp"}
	case "gemini":
		return "gemini", []string{"mcp", "add", "-s", "user", serverName, core, "mcp"}
	case "qwen":
		return "qwen", []string{"mcp", "add", "--scope", "user", serverName, core, "mcp"}
	case "kimi":
		return "kimi", []string{"mcp", "add", "--transport", "stdio", serverName, "--", core, "mcp"}
	case "copilot":
		spec, _ := json.Marshal(struct {
			Name    string   `json:"name"`
			Command string   `json:"command"`
			Args    []string `json:"args"`
		}{serverName, core, []string{"mcp"}})
		return "code", []string{"--add-mcp", string(spec)}
	}
	return "", nil
}

func (e *Env) installCommand(ctx context.Context, a Agent) error {
	program, args := e.commandLine(a)
	if program == "" {
		return fmt.Errorf("%s has no install command", a.Name)
	}
	path, err := e.lookPath(program)
	if err != nil {
		return ErrNotFound
	}
	ctx, cancel := context.WithTimeout(ctx, e.timeout())
	defer cancel()
	out, err := e.run(ctx, path, args...)
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return ErrTimeout
	case err != nil:
		return &CommandError{Agent: a.Name, Output: lastLine(out), Err: err}
	}
	return nil
}

// lastLine is the last non-empty line of a command's output, which is where it says what failed.
func lastLine(out []byte) string {
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// entry is the JSON of adhd-todo's server as the agent's config spells it.
func (e *Env) entry(a Agent) []byte {
	core := e.core()
	var v any
	switch a.ID {
	case "cursor":
		v = struct {
			Type    string   `json:"type"`
			Command string   `json:"command"`
			Args    []string `json:"args"`
		}{"stdio", core, []string{"mcp"}}
	case "opencode":
		v = struct {
			Type    string   `json:"type"`
			Command []string `json:"command"`
			Enabled bool     `json:"enabled"`
		}{"local", []string{core, "mcp"}, true}
	case "windsurf":
		v = struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		}{core, []string{"mcp"}}
	default: // zed
		v = struct {
			Command string            `json:"command"`
			Args    []string          `json:"args"`
			Env     map[string]string `json:"env"`
		}{core, []string{"mcp"}, map[string]string{}}
	}
	b, _ := json.Marshal(v)
	return b
}

// Snippet is what to add by hand to the agent's config file: the key with adhd-todo under it.
func (e *Env) Snippet(a Agent) string {
	entry, err := parseJSON(e.entry(a))
	if err != nil {
		return ""
	}
	root := &jval{kind: 'o'}
	_ = root.setPath([]string{a.Key, serverName}, entry)
	lines := strings.Split(string(root.render("  ")), "\n")
	// the object's own braces are not part of what is pasted into an existing file
	lines = lines[1 : len(lines)-1]
	for i, l := range lines {
		lines[i] = strings.TrimPrefix(l, "  ")
	}
	return strings.Join(lines, "\n")
}

// configFile is the file to edit: the first of the agent's that exists, else the first.
func (e *Env) configFile(a Agent) string {
	for _, f := range a.Files {
		if e.exists(e.path(f)) {
			return e.path(f)
		}
	}
	return e.path(a.Files[0])
}

func (e *Env) installFile(a Agent) error {
	file := e.configFile(a)
	manual := &ManualError{File: file, Snippet: e.Snippet(a)}
	old, err := os.ReadFile(file)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("cannot read %s: %w", file, err)
	}
	exists := err == nil
	root := &jval{kind: 'o'}
	if strings.TrimSpace(string(old)) != "" {
		if root, err = parseJSON(old); err != nil || root.kind != 'o' {
			return manual
		}
	}
	entry, err := parseJSON(e.entry(a))
	if err != nil {
		return err
	}
	if err := root.setPath([]string{a.Key, serverName}, entry); err != nil {
		return manual
	}
	out := root.render(indentOf(old))
	out = append(out, '\n')
	if exists {
		if err := writeFile(file+".adhd-todo.bak", old, 0o600); err != nil {
			return fmt.Errorf("cannot back up %s: %w", file, err)
		}
	}
	return writeFile(file, out, modeOf(file))
}

func modeOf(file string) os.FileMode {
	if fi, err := os.Stat(file); err == nil {
		return fi.Mode().Perm()
	}
	return 0o644
}

// writeFile writes through a temporary file in the same folder, so that a failure never leaves a
// half-written config.
func writeFile(file string, b []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(file), ".adhd-todo-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), mode); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), file)
}
