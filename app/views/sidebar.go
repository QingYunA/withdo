package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// sidebar is the source list over the window's material: Now, Inbox, then the projects, then
// Settings at the bottom.
func (m *Main) sidebar(c *ui.Context, t *theme.Tokens, vibrant bool, now data.Now) {
	side := ui.Column(c).Width(theme.SidebarWidth).Shrink(0)
	if !vibrant {
		side.Background(t.BgSubtle)
	}
	side.Children(func() {
		ui.Box(c).Height(titleBarHeight).Shrink(0).DragWindow()
		m.navRow(c, t, "now", nav{}, theme.IconNow, "Now", now.Count())
		inbox, _ := openCount(m.app.snap.tasks, func(task data.Task) bool { return task.ProjectID == nil })
		m.navRow(c, t, "inbox", nav{screen: screenInbox}, theme.IconInbox, "Inbox", inbox)
		ui.Row(c).Padding(18, 18, 6).Shrink(0).Children(func() {
			theme.Text(c, theme.Section, t.TextTertiary, "Projects")
		})
		for _, p := range m.app.snap.projects {
			n, blocked := openCount(m.app.snap.tasks, func(task data.Task) bool { return task.ProjectID != nil && *task.ProjectID == p.ID })
			m.projectRow(c, t, p, n, blocked)
		}
		if len(m.app.snap.projects) == 0 {
			theme.Paragraph(c, theme.Meta, t.TextTertiary, "Projects appear when an agent reports from a repository.", 0).Padding(0, 18)
		}
		ui.Spacer(c)
		m.settingsRow(c, t)
		ui.Box(c).Height(8).Shrink(0)
	})
}

// openCount counts what is on the plate: not done, not cancelled, not parked in the backlog.
func openCount(tasks []data.Task, match func(data.Task) bool) (n int, blocked bool) {
	for _, t := range tasks {
		if !match(t) || t.Status.Terminal() || t.Status == data.Backlog {
			continue
		}
		n++
		blocked = blocked || t.Status == data.Blocked
	}
	return n, blocked
}

func (m *Main) navRow(c *ui.Context, t *theme.Tokens, key string, to nav, icon *ui.SVG, label string, count int) {
	selected := m.nav == to
	row := sidebarRow(c, t, key, selected)
	row.Children(func() {
		ui.Icon(c, icon).Size(16, 16).TextColor(t.TextSecondary)
		sidebarLabel(c, t, label, selected)
		sidebarCount(c, t, count)
	})
	if row.Clicked() {
		m.goTo(to)
	}
}

func (m *Main) projectRow(c *ui.Context, t *theme.Tokens, p data.Project, n int, blocked bool) {
	to := nav{screen: screenProject, projectID: p.ID}
	selected := m.nav == to
	row := sidebarRow(c, t, p.ID, selected)
	row.Children(func() {
		ui.Box(c).Size(16, 16).Shrink(0).Center().Children(func() {
			theme.Text(c, theme.Body, t.TextTertiary, "#")
		})
		sidebarLabel(c, t, p.Name, selected)
		if blocked {
			theme.Dot(c, t.Blocked(), 6).Shrink(0).Margin(0, 2, 0, 0)
		} else {
			sidebarCount(c, t, n)
		}
	})
	if row.Clicked() {
		m.goTo(to)
	}
}

func (m *Main) settingsRow(c *ui.Context, t *theme.Tokens) {
	row := sidebarRow(c, t, "settings", false)
	row.Children(func() {
		ui.Icon(c, theme.IconGear).Size(16, 16).TextColor(t.TextSecondary)
		sidebarLabel(c, t, "Settings", false)
	})
	if row.Clicked() {
		m.settings = true
	}
}

// sidebarRow is a 30 pt row inset 8 pt, highlighted by selection or hover.
func sidebarRow(c *ui.Context, t *theme.Tokens, key any, selected bool) ui.Element {
	row := ui.Row(c.Key(key)).Height(30).Shrink(0).Margin(0, 10).Padding(0, 10).Gap(10).
		Radius(7).AlignItems(ui.Center).Cursor(ui.CursorPointer)
	switch {
	case selected:
		row.Background(t.BgSelected)
	case row.Hovered():
		row.Background(t.BgHover)
	}
	return row.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
}

func sidebarLabel(c *ui.Context, t *theme.Tokens, text string, selected bool) {
	style := theme.Body
	if selected {
		style = theme.BodyStrong
	}
	theme.Text(c, style, t.Text, text).Grow(1).MinWidth(0).Ellipsis("…")
}

func sidebarCount(c *ui.Context, t *theme.Tokens, n int) {
	if n > 0 {
		theme.Text(c, theme.Meta, t.TextTertiary, itoa(n)).Shrink(0)
	}
}
