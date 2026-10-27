package agentsetup

import (
	_ "embed"
	"encoding/json"
	"strings"
)

//go:embed guide.md
var guideTemplate string

// Guide is a self-contained setup guide for any coding agent: the user pastes it to an agent that
// Withdo does not know, and the agent adds the MCP server to itself. The path of withdo is
// the one the known agents get.
func (e *Env) Guide() string {
	core := e.core()
	entry, _ := json.MarshalIndent(struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}{core, []string{"mcp"}}, "", "  ")
	// A checkout on Windows may carry CRLF endings; render the guide in LF.
	template := strings.ReplaceAll(guideTemplate, "\r\n", "\n")
	return strings.NewReplacer("{{CORE}}", core, "{{ENTRY}}", string(entry)).Replace(template)
}
