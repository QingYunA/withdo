package agentsetup

import (
	"encoding/json"
	"os"
	"regexp"
)

// Status is what is known of one agent on this Mac.
type Status struct {
	Found     bool   // its command is on PATH or its config folder exists (or adhd-todo is already in its config)
	Via       string // what was found: the command ("gemini") or the folder ("~/.cursor")
	Installed bool   // adhd-todo is in its MCP config
}

// Detect looks at one agent: reads its config, runs nothing.
func (e *Env) Detect(a Agent) Status {
	st := Status{Installed: e.Installed(a)}
	for _, c := range a.Commands {
		if _, err := e.lookPath(c); err == nil {
			st.Found, st.Via = true, c
			return st
		}
	}
	for _, d := range a.Dirs {
		if fi, err := e.stat(e.path(d)); err == nil && fi.IsDir() {
			st.Found, st.Via = true, "~/"+d
			return st
		}
	}
	if st.Installed { // its config is all there is to show
		for _, f := range a.Files {
			if e.exists(e.path(f)) {
				st.Found, st.Via = true, "~/"+f
				break
			}
		}
	}
	return st
}

// DetectAll is Detect for every agent, by ID.
func (e *Env) DetectAll() map[string]Status {
	out := make(map[string]Status, len(Agents))
	for _, a := range Agents {
		out[a.ID] = e.Detect(a)
	}
	return out
}

// tomlServer matches the header of Codex's [mcp_servers.adhd-todo] table (and its sub-tables).
var tomlServer = regexp.MustCompile(`(?m)^[ \t]*\[[ \t]*mcp_servers[ \t]*\.[ \t]*("adhd-todo"|'adhd-todo'|adhd-todo)[ \t]*(\.[^\]\n]*)?\]`)

// Installed reports whether adhd-todo is in the agent's MCP config. A config it cannot read or
// parse says no.
func (e *Env) Installed(a Agent) bool {
	for _, f := range a.Files {
		b, err := os.ReadFile(e.path(f))
		if err != nil {
			continue
		}
		if a.Key == "" {
			if tomlServer.Match(b) {
				return true
			}
			continue
		}
		var top map[string]json.RawMessage
		if json.Unmarshal(stripJSONC(b), &top) != nil {
			continue
		}
		var servers map[string]json.RawMessage
		if json.Unmarshal(top[a.Key], &servers) != nil {
			continue
		}
		if _, ok := servers[serverName]; ok {
			return true
		}
	}
	return false
}

// stripJSONC drops the comments and the trailing commas of a JSON-with-comments file (Zed's
// settings, VS Code's mcp.json), so that it can be read. It never writes back: a file that is not
// plain JSON is not rewritten.
func stripJSONC(b []byte) []byte {
	out := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		c := b[i]
		switch {
		case c == '"':
			j := i + 1
			for j < len(b) && b[j] != '"' {
				if b[j] == '\\' {
					j++
				}
				j++
			}
			if j >= len(b) {
				j = len(b) - 1
			}
			out = append(out, b[i:j+1]...)
			i = j
		case c == '/' && i+1 < len(b) && b[i+1] == '/':
			for i < len(b) && b[i] != '\n' {
				i++
			}
			out = append(out, '\n')
		case c == '/' && i+1 < len(b) && b[i+1] == '*':
			i += 2
			for i+1 < len(b) && !(b[i] == '*' && b[i+1] == '/') {
				i++
			}
			i++
		case c == ',':
			// a comma followed (past spaces and comments) by a closing bracket is dropped
			j := i + 1
			for j < len(b) {
				switch {
				case b[j] == ' ' || b[j] == '\t' || b[j] == '\n' || b[j] == '\r':
					j++
				case b[j] == '/' && j+1 < len(b) && b[j+1] == '/':
					for j < len(b) && b[j] != '\n' {
						j++
					}
				case b[j] == '/' && j+1 < len(b) && b[j+1] == '*':
					j += 2
					for j+1 < len(b) && !(b[j] == '*' && b[j+1] == '/') {
						j++
					}
					j += 2
				default:
					goto done
				}
			}
		done:
			if j < len(b) && (b[j] == '}' || b[j] == ']') {
				continue
			}
			out = append(out, c)
		default:
			out = append(out, c)
		}
	}
	return out
}
