package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// unnamedAgent is what an agent that has not reported yet is called, and the name its avatar is
// drawn for.
const unnamedAgent = "agent"

// taskAgent is the name of the agent behind a task: the one its latest report came from, or
// "agent" when it has none yet. It is only meaningful for Source == agent.
func (a *App) taskAgent(t data.Task) string {
	if r := a.latestReport(t.ID); r != nil && r.Agent != "" {
		return r.Agent
	}
	return unnamedAgent
}

// taskWho is who wrote a task: me (the assistant writing it for me still counts as me) or the agent
// that made it.
func (a *App) taskWho(t data.Task) theme.Who {
	if t.Source == data.SourceAgent {
		return theme.AgentAvatar(a.taskAgent(t))
	}
	return theme.You
}

// taskWhoName is the words for taskWho: "You", or the agent's name as people write it.
func (m *Main) taskWhoName(t data.Task) string {
	if t.Source == data.SourceAgent {
		return agentName(m.app.taskAgent(t))
	}
	return m.say.DoneByYou()
}

// authorSlot is a row's avatar, under the group header's name. The avatar is 18 pt like the
// header's glyph, and the row's gap is wider than the header's, so the slot gives the difference
// back: the title sits as far from the avatar as the header's name from its glyph.
func (m *Main) authorSlot(c *ui.Context, task data.Task) {
	theme.AvatarIcon(c.Key("author"), m.app.taskWho(task), theme.AvatarRow).Margin(0, listIndent-theme.AvatarRow-rowGap, 0, 0)
}

// emptyBodyMax is the widest the quiet line of an empty state grows before it wraps.
const emptyBodyMax float32 = 480

// emptyState is the calm picture of a screen with nothing on it: you and the assistant side by side,
// a title and one quiet line, centered. The caller adds a link under it.
func emptyState(c *ui.Context, t *theme.Tokens, title, body string, top float32, extra func()) {
	ui.Column(c).Padding(top, 0, 0).Gap(6).AlignItems(ui.Center).Shrink(0).Children(func() {
		theme.DuoIcon(c, theme.AvatarLarge)
		theme.Paragraph(c, theme.BodyStrong, t.Text, title, 0).TextAlign(ui.Center).MaxWidth(emptyBodyMax).Margin(14, 0, 0)
		theme.Paragraph(c, theme.Body, t.TextSecondary, body, 0).TextAlign(ui.Center).MaxWidth(emptyBodyMax)
		if extra != nil {
			extra()
		}
	})
}
