package agentsetup

import "context"

// Migration is what Migrate did for one agent: it had the old adhd-todo entry and no withdo one.
// Err is nil when the agent now has withdo and no adhd-todo; a *LegacyError when withdo was added
// and adhd-todo is still there.
type Migration struct {
	Agent Agent
	Err   error
}

// Migrate moves every agent that has only the old adhd-todo MCP entry to withdo, which is what
// Install does for one. An agent with withdo already, or with neither, is left alone; so is the old
// entry beside a withdo one. It runs the agents' own commands one after the other, which takes a
// while and can fail, so the app runs it in the background and logs the errors: nothing here
// stops at one.
func (e *Env) Migrate(ctx context.Context) []Migration {
	var out []Migration
	for _, a := range Agents {
		if ctx.Err() != nil {
			break
		}
		if e.Installed(a) || !e.hasServer(a, legacyServerName) {
			continue
		}
		out = append(out, Migration{Agent: a, Err: e.Install(ctx, a)})
	}
	return out
}
