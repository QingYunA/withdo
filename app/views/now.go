package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// nowScreen is the home screen: the date, the ask field, and up to three sections. A section with
// nothing in it is not drawn; with nothing in any of them the screen says All clear.
func (m *Main) nowScreen(c *ui.Context, t *theme.Tokens, now data.Now) {
	m.pageHeader(c, t, "Now", m.app.Now().Format("Monday, Jan 2"), nil)
	ui.Box(c).Height(18).Shrink(0)
	m.askField(c, t)
	if !m.app.snap.loaded {
		return
	}
	if now.Empty() {
		m.allClear(c, t)
		return
	}
	m.needsSection(c, t, now.Needs)
	m.agentsSection(c, t, now.Agents)
	m.yoursSection(c, t, now)
}

// pageHeader is the title with a quiet companion, and whatever controls sit at the right.
func (m *Main) pageHeader(c *ui.Context, t *theme.Tokens, title, sub string, controls func()) {
	ui.Row(c).Height(36).Shrink(0).Gap(10).AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.Title, t.Text, title).Shrink(0)
		if sub != "" {
			theme.Text(c, theme.Body, t.TextTertiary, sub).Shrink(1).MinWidth(0).Ellipsis("…")
		}
		ui.Spacer(c)
		if controls != nil {
			controls()
		}
	})
}

// sectionHeader is an uppercase title with its count.
func sectionHeader(c *ui.Context, t *theme.Tokens, title string, n int) {
	ui.Row(c).Margin(22, 0, 6).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.Section, t.TextSecondary, title)
		theme.Text(c, theme.Meta, t.TextTertiary, itoa(n))
	})
}

// allClear is the calm state: nothing needs me, nothing is running, nothing is mine.
func (m *Main) allClear(c *ui.Context, t *theme.Tokens) {
	ui.Column(c).Padding(96, 0, 0).Gap(6).AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.Input, t.Text, "All clear.")
		theme.Text(c, theme.Body, t.TextSecondary, "Agents report here through the adhd-todo MCP server.")
		if link(c, t, "How to connect").Margin(6, 0, 0).Clicked() {
			m.settings = true
		}
	})
}

// needsSection is the one boxed thing on the screen: what waits on me, blocked first.
func (m *Main) needsSection(c *ui.Context, t *theme.Tokens, items []data.NowItem) {
	if len(items) == 0 {
		return
	}
	sectionHeader(c, t, "Needs you", len(items))
	ui.Column(c).Shrink(0).Padding(4, 14).Radius(theme.RadiusPanel).Border(1, t.Border).Children(func() {
		for i, it := range items {
			if i > 0 {
				ui.Box(c.Key("rule-"+itoa(it.ID))).Height(1).Shrink(0).Margin(0, -needRow.horz).Background(t.Border)
			}
			m.needRow(c, t, it)
		}
	})
}

func (m *Main) needRow(c *ui.Context, t *theme.Tokens, it data.NowItem) {
	fill, opacity, _ := m.app.completionPhase(c, it.ID)
	row := m.rowShell(c, t, it.ID, needRow, opacity)
	row.Children(func() {
		if it.Kind == data.NeedReminder {
			ui.Icon(c, theme.IconBell).Size(16, 16).TextColor(t.Accent).Shrink(0).Margin(1, 0, 0)
		} else {
			m.app.glyphSlot(c, t, it.Task, fill)
		}
		body(c, func() {
			m.titleLine(c, t, it.Task, it.Project)
			summaryLine(c, t, it.Summary)
		})
		right(c, func() {
			if m.width >= dropPillsBelow { // beside the inspector the title needs the room more
				m.ageLabel(c, t, it)
			}
			text, label := m.actionFor(it)
			if actionButton(c, t, it.ID, text, label).Clicked() {
				m.act(it)
			}
		})
	})
	if row.Clicked() {
		m.openRow(it.ID)
	}
}

// actionFor names the button of a Needs you row, and what it says to assistive technology.
func (m *Main) actionFor(it data.NowItem) (text, label string) {
	switch {
	case it.Kind == data.NeedReminder:
		return "Done", "Mark done: " + it.Title
	case it.Kind == data.NeedReview && it.PRURL != "":
		return "Open PR", "Open pull request #" + itoa(it.PRNumber)
	}
	return "Open", "Open " + it.Title
}

// act runs the button of a Needs you row.
func (m *Main) act(it data.NowItem) {
	switch {
	case it.Kind == data.NeedReminder:
		m.app.SetStatus(it.Task, data.Done)
	case it.Kind == data.NeedReview && it.PRURL != "":
		m.app.Act.OpenURL(it.PRURL)
	default:
		m.openRow(it.ID)
	}
}

// agentsSection is what agents are doing now, newest report first, across projects.
func (m *Main) agentsSection(c *ui.Context, t *theme.Tokens, items []data.NowItem) {
	if len(items) == 0 {
		return
	}
	sectionHeader(c, t, "Agents at work", len(items))
	for _, it := range items {
		fill, opacity, _ := m.app.completionPhase(c, it.ID)
		row := m.rowShell(c, t, it.ID, plainRow, opacity)
		row.Children(func() {
			m.app.glyphSlot(c, t, it.Task, fill)
			body(c, func() {
				m.titleLine(c, t, it.Task, it.Project)
				summaryLine(c, t, it.Summary)
			})
			right(c, func() {
				if it.Live {
					liveDot(c, t)
				}
				m.ageLabel(c, t, it)
			})
		})
		if row.Clicked() {
			m.openRow(it.ID)
		}
	}
}

// yoursSection is my own work: due today and overdue first, then the week, then undated.
func (m *Main) yoursSection(c *ui.Context, t *theme.Tokens, now data.Now) {
	if len(now.Yours) == 0 {
		return
	}
	sectionHeader(c, t, "Yours", len(now.Yours))
	for _, it := range now.Yours {
		fill, opacity, _ := m.app.completionPhase(c, it.ID)
		row := m.rowShell(c, t, it.ID, plainRow, opacity).AlignItems(ui.Center)
		row.Children(func() {
			m.app.glyphSlot(c, t, it.Task, fill)
			body(c, func() { m.titleLine(c, t, it.Task, it.Project) })
			if it.Label != "" {
				theme.Text(c, theme.Meta, t.TextTertiary, it.Label).Shrink(0)
			}
		})
		if row.Clicked() {
			m.openRow(it.ID)
		}
	}
	if now.MoreUndated > 0 {
		label := "Show all in Inbox"
		if link(c, t, label).Margin(8, 0, 0).Clicked() {
			m.goTo(nav{screen: screenInbox})
		}
	}
}
