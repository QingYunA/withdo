package views

import (
	"strings"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// Finished rows are the done tasks of Mine (Done today), of the Done group of a list and of
// Completed: the same row everywhere, the title quiet, and at the right who finished it and when.

// finishedOpts say what a place wants of a finished row.
type finishedOpts struct {
	glyph   bool   // the status glyph, a way to reopen; the lists' Done group has none, its header says it
	project string // the project's name beside the title
	when    string // the day or the time of day it was finished
	summary bool   // what the closing report said, after the agent's name
	// quiet leaves out who finished it, which the avatar already says (the Agents screen), and note,
	// when it is not empty, goes before the time: "3 subtasks".
	quiet bool
	note  string
}

// completedBy says who finished a done task and what was said as it was: see data.CompletedBy.
func (a *App) completedBy(t data.Task) (by, said string) {
	return data.CompletedBy(t, a.latestReport(t.ID), a.githubReport(t.ID))
}

// AgentName is an agent as people write it, for the words outside the windows.
func AgentName(agent string) string { return agentName(agent) }

// agentName is an agent as people write it: the ids agents report under become their product's
// name, matched the way theme.AgentAvatar matches them, so a name and its avatar always agree.
func agentName(agent string) string {
	switch n := strings.ToLower(agent); {
	case n == "" || n == unnamedAgent:
		return "Agent"
	case n == data.GitHubAgent:
		return "GitHub"
	case strings.Contains(n, "claude"):
		return "Claude Code"
	case strings.Contains(n, "codex"):
		return "Codex"
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
	case strings.Contains(n, "cline"):
		return "Cline"
	}
	return agent
}

// agentApp is the macOS app of the desktop agent behind an id agents report under, matched the way
// agentName matches it; "" for agents that have none (they run in a terminal).
func agentApp(agent string) string {
	switch n := strings.ToLower(agent); {
	case strings.Contains(n, "claude"):
		return "Claude"
	case strings.Contains(n, "codex"):
		return "Codex"
	case strings.Contains(n, "cursor"):
		return "Cursor"
	case strings.Contains(n, "windsurf"):
		return "Windsurf"
	case n == "zed" || strings.HasPrefix(n, "zed ") || strings.HasPrefix(n, "zed-"):
		return "Zed"
	case strings.Contains(n, "copilot") || strings.Contains(n, "visual studio code"):
		return "Visual Studio Code"
	}
	return ""
}

// whoFinished is the words at the right of a finished row: "You", or an agent's name, and with
// summary what its closing report said ("Claude Code · PR #245 merged").
func (m *Main) whoFinished(task data.Task, summary bool) string {
	by, said := m.app.completedBy(task)
	switch {
	case by == "":
		return m.say.DoneByYou()
	case summary && said != "":
		return m.say.DoneByAgent(agentName(by), said)
	}
	return agentName(by)
}

// finishedRow is one done (or cancelled) task. Beside the inspector it keeps the title and the time
// and drops who finished it, then the summary, as the other rows drop their companions.
func (m *Main) finishedRow(c *ui.Context, t *theme.Tokens, task data.Task, o finishedOpts) {
	ph := m.rowPhase(c, task)
	row := m.rowShell(c, t, task.ID, plainRow, ph).AlignItems(ui.Center)
	row.Children(func() {
		if o.glyph {
			m.glyphSlot(c, t, task, ph.fill)
		} else if o.quiet {
			theme.AvatarIcon(c.Key("author"), m.app.taskWho(task), theme.AvatarRow) // as in the Working rows above
		} else {
			m.authorSlot(c, task) // under the group's name, as the list's rows
		}
		body(c, func() {
			ui.Row(c).Gap(8).MinWidth(0).AlignItems(ui.Center).Children(func() {
				theme.Text(c, theme.Body, t.TextSecondary, task.Title).Shrink(1).MinWidth(0).Ellipsis("…")
				if o.project != "" && m.width >= dropProjectBelow {
					theme.Text(c, theme.Meta, t.TextTertiary, o.project).Shrink(8).MinWidth(0).Ellipsis("…")
				}
			})
		})
		right(c, func() {
			if o.note != "" && m.width >= dropPillsBelow {
				theme.Text(c, theme.Meta, t.TextTertiary, o.note).Shrink(0)
			}
			if m.width >= dropPillsBelow && !o.quiet {
				who := m.whoFinished(task, o.summary && m.width >= dropProjectBelow)
				theme.Text(c, theme.Meta, t.TextTertiary, who).MaxWidth(300).Shrink(2).MinWidth(0).Ellipsis("…")
			}
			m.rowThumb(c, t, task.ID)
			plainText(c, t, o.when)
		})
	})
	if row.Clicked() {
		m.openRow(task.ID)
	}
}

// doneTodaySection is the last section of Mine: what was finished since midnight, by agents and by
// me, newest first, five rows and then a link to Completed.
func (m *Main) doneTodaySection(c *ui.Context, t *theme.Tokens, items []data.Item, more int) {
	if len(items) == 0 {
		return
	}
	m.section(c, "done-today", itemTasks(items), func() {
		sectionHeader(c, t, m.say.SectionDoneToday(), len(items)+more)
		for _, it := range items {
			m.finishedRow(c, t, it.Task, finishedOpts{glyph: true, project: it.Project, when: m.clock.Time(it.Since), summary: true})
		}
	})
	if more > 0 && link(c, t, m.say.NowShowAllInCompleted()).Margin(8, 0, 0, theme.GlyphSize+8).Clicked() {
		m.goTo(nav{screen: screenCompleted})
	}
}
