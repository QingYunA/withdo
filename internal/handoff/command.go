// Package handoff starts a coding agent on a task: it writes a small shell script that changes to
// the project's folder and runs the agent with the task's prompt, and opens it in Terminal.
package handoff

import "strings"

// Command is the argument list that starts agentID's interactive session at exe with prompt as its
// first message; ok is false for an agent that cannot be started this way. A prompt given as the
// argument starts an interactive session in claude and codex; gemini and qwen need -i for that,
// since their plain prompt argument is the non-interactive mode.
func Command(agentID, exe, prompt string) (argv []string, ok bool) {
	switch agentID {
	case "claude", "codex":
		return []string{exe, prompt}, true
	case "gemini", "qwen":
		return []string{exe, "-i", prompt}, true
	}
	return nil, false
}

// Launchable reports whether Command knows how to start the agent.
func Launchable(agentID string) bool {
	_, ok := Command(agentID, "", "")
	return ok
}

// Quote makes s one word for a POSIX shell: inside single quotes nothing is special, so a quote is
// the only character that needs care. A NUL cannot be in a command line and is dropped.
func Quote(s string) string {
	s = strings.ReplaceAll(s, "\x00", "")
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// QuoteAll quotes every word and joins them with spaces.
func QuoteAll(words []string) string {
	quoted := make([]string, len(words))
	for i, w := range words {
		quoted[i] = Quote(w)
	}
	return strings.Join(quoted, " ")
}
