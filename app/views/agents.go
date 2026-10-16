package views

import (
	"strings"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// subRowStyle is the padding of a subtask row. Its highlight bleeds to the right of the column, and
// to the left only as far as the rail.
var subRowStyle = rowStyle{5, 8}

// subRail is the room between the rail and the text of a subtask.
const subRail float32 = 6

// agentsScreen is what the agents are doing: the title with how many are at work, the composer, then
// their tasks at work with the subtasks under each, and what they finished today.
func (m *Main) agentsScreen(c *ui.Context, t *theme.Tokens) {
	sub := ""
	if n := m.agentsPage.Count(); n > 0 {
		sub = m.say.AgentsWorking(n)
	}
	m.pageHeader(c, t, m.agentsTitle(), sub, nil)
	ui.Box(c).Height(18).Shrink(0)
	m.pageComposer(c, t)
	if !m.app.snap.loaded {
		return
	}
	m.belowComposer(c, func() {
		if m.agentsPage.Empty() {
			emptyState(c, t, m.say.AgentsEmpty(), m.say.AgentsEmptyBody(), 72, func() {
				if link(c, t, m.say.LinkHowToConnect()).Margin(6, 0, 0).Clicked() {
					m.openSettings(PaneAgents)
				}
			})
			return
		}
		if len(m.agentsPage.Working) > 0 {
			sectionHeader(c, t, m.say.SectionWorking(), len(m.agentsPage.Working))
			for _, r := range m.agentsPage.Working {
				m.workingRow(c, t, r)
			}
		}
		m.agentsDoneToday(c, t)
	})
}

// workingRow is an agent's task at work: its avatar, title, who and where and what it said, and at
// the right the live dot and how long ago; under it a rail with its subtasks.
func (m *Main) workingRow(c *ui.Context, t *theme.Tokens, r data.AgentRow) {
	_, opacity, _ := m.app.completionPhase(c, r.ID)
	row := m.rowShell(c, t, r.ID, plainRow, opacity)
	row.Children(func() {
		theme.AvatarIcon(c.Key("author"), m.app.taskWho(r.Task), theme.AvatarRow).Margin(2, 0, 0, 0)
		body(c, func() {
			m.titleLine(c, t, r.Task, "", false)
			summaryLine(c, t, m.workingMeta(r))
		})
		right(c, func() {
			switch {
			case r.Status == data.Blocked:
				m.statusPill(c, t, "pill", r.Status)
			case r.Live:
				liveDot(c, t)
			}
			m.ageLabel(c, t, r.Item)
		})
	})
	if row.Clicked() {
		m.openRow(r.ID)
	}
	if len(r.Subs) == 0 {
		return
	}
	ui.Row(c.Key("subs-"+itoa(r.ID))).Shrink(0).AlignItems(ui.Stretch).Margin(2, 0, 4, theme.AvatarRow/2).Children(func() {
		ui.Box(c.Key("rail")).Width(1).Shrink(0).Background(t.Border)
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			for _, s := range r.Subs {
				m.subtaskRow(c, t, s)
			}
		})
	})
}

// workingMeta is the second line of an agent's task: who, where and what it said, as far as it
// knows them: "Claude Code · main · three subagents at work".
func (m *Main) workingMeta(r data.AgentRow) string {
	parts := []string{agentName(m.app.taskAgent(r.Task))}
	if r.Branch != "" && m.width >= dropPillsBelow {
		parts = append(parts, r.Branch)
	}
	if said := m.agentSummary(r.Item); said != "" {
		parts = append(parts, said)
	}
	return strings.Join(parts, " · ")
}

// subtaskRow is a subtask under its parent: a small avatar, the title, how far along it is, and how
// long ago it moved.
func (m *Main) subtaskRow(c *ui.Context, t *theme.Tokens, s data.Item) {
	_, opacity, _ := m.app.completionPhase(c, s.ID)
	row := m.rowShell(c, t, s.ID, subRowStyle, opacity).Margin(0, -subRowStyle.horz, 0, subRail).AlignItems(ui.Center)
	row.Children(func() {
		theme.AvatarIcon(c.Key("author"), m.app.taskWho(s.Task), theme.AvatarSmall).Shrink(0)
		theme.Text(c, theme.Body, t.Text, s.Title).Grow(1).Shrink(1).MinWidth(0).Ellipsis("…")
		right(c, func() {
			m.statusPill(c, t, "pill", s.Status)
			if m.width >= dropPillsBelow {
				m.ageLabel(c, t, s)
			}
		})
	})
	if row.Clicked() {
		m.openRow(s.ID)
	}
}

// statusPill is where a subtask stands, in a small tinted label: Done, Doing, Handed back to the main
// agent (Review), and the other statuses by their own names.
func (m *Main) statusPill(c *ui.Context, t *theme.Tokens, key string, s data.Status) ui.Element {
	ink, well := t.Status(s), t.Status(s).Alpha(0.12)
	switch s {
	case data.Review, data.Todo, data.Backlog, data.Cancelled:
		ink, well = t.TextSecondary, t.BgSubtle
	}
	return ui.Row(c.Key(key)).Shrink(0).Padding(1, 7).Radius(theme.RadiusControl).Background(well).AlignItems(ui.Center).
		Children(func() { theme.Text(c, theme.Meta, ink, m.pillLabel(s)).Shrink(0) })
}

// pillLabel is the words of a status pill.
func (m *Main) pillLabel(s data.Status) string {
	switch s {
	case data.Done:
		return m.say.PillDone()
	case data.Doing:
		return m.say.PillDoing()
	case data.Review:
		return m.say.PillHandedBack()
	}
	return statusLabel(m.say, s)
}

// agentsDoneToday is what the agents finished since midnight: the task, how many subtasks it had, and
// when. Its subtasks are not rows.
func (m *Main) agentsDoneToday(c *ui.Context, t *theme.Tokens) {
	if len(m.agentsPage.Done) == 0 {
		return
	}
	sectionHeader(c, t, m.say.SectionDoneToday(), len(m.agentsPage.Done)+m.agentsPage.MoreDone)
	for _, r := range m.agentsPage.Done {
		o := finishedOpts{when: m.clock.Time(r.Since), quiet: true}
		if n := len(r.Subs); n > 0 {
			o.note = m.say.SubtasksCount(n)
		}
		m.finishedRow(c, t, r.Task, o)
	}
	if m.agentsPage.MoreDone > 0 && link(c, t, m.say.NowShowAllInCompleted()).Margin(8, 0, 0, theme.GlyphSize+8).Clicked() {
		m.goTo(nav{screen: screenCompleted})
	}
}
