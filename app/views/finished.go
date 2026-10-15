package views

import (
	"strings"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// Finished rows are the done tasks of Now (Done today), of the Done group of a list and of
// Completed: the same row everywhere, the title quiet, and at the right who finished it and when.

// finishedOpts say what a place wants of a finished row.
type finishedOpts struct {
	glyph   bool   // the status glyph, a way to reopen; the lists' Done group has none, its header says it
	project string // the project's name beside the title
	when    string // the day or the time of day it was finished
	summary bool   // what the closing report said, after the agent's name
}

// completedBy says who finished a done task and what was said as it was: see data.CompletedBy.
func (a *App) completedBy(t data.Task) (by, said string) {
	return data.CompletedBy(t, a.latestReport(t.ID), a.githubReport(t.ID))
}

// agentName is an agent as people write it: the ids agents report under become their product's name.
func agentName(agent string) string {
	switch strings.ToLower(agent) {
	case "claude-code", "claude_code", "claude code":
		return "Claude Code"
	case data.GitHubAgent:
		return "GitHub"
	}
	return agent
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
	fill, opacity, _ := m.app.completionPhase(c, task.ID)
	row := m.rowShell(c, t, task.ID, plainRow, opacity).AlignItems(ui.Center)
	row.Children(func() {
		if o.glyph {
			m.glyphSlot(c, t, task, fill)
		} else {
			ui.Box(c).Width(listIndent - 12).Shrink(0) // the glyph's column, as under a group header
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
			if m.width >= dropPillsBelow {
				who := m.whoFinished(task, o.summary && m.width >= dropProjectBelow)
				theme.Text(c, theme.Meta, t.TextTertiary, who).MaxWidth(300).Shrink(2).MinWidth(0).Ellipsis("…")
			}
			plainText(c, t, o.when)
		})
	})
	if row.Clicked() {
		m.openRow(task.ID)
	}
}

// doneTodaySection is the last section of Now: what was finished since midnight, by agents and by
// me, newest first, five rows and then a link to Completed.
func (m *Main) doneTodaySection(c *ui.Context, t *theme.Tokens, now data.Now) {
	if len(now.Done) == 0 {
		return
	}
	sectionHeader(c, t, m.say.SectionDoneToday(), len(now.Done)+now.MoreDone)
	for _, it := range now.Done {
		m.finishedRow(c, t, it.Task, finishedOpts{glyph: true, project: it.Project, when: m.clock.Time(it.Since), summary: true})
	}
	if now.MoreDone > 0 && link(c, t, m.say.NowShowAllInCompleted()).Margin(8, 0, 0, theme.GlyphSize+8).Clicked() {
		m.goTo(nav{screen: screenCompleted})
	}
}
