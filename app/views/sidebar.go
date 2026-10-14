package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// sidebar is the source list over the window's material: Inbox, then the projects with their
// counts, then Settings at the bottom.
func (m *Main) sidebar(c *ui.Context, t *theme.Tokens, vibrant bool) {
	side := ui.Column(c).Width(theme.SidebarWidth).Shrink(0)
	if !vibrant {
		side.Background(t.BgSubtle)
	}
	side.Children(func() {
		ui.Box(c).Height(titleBarHeight).Shrink(0).DragWindow()
		m.inboxRow(c, t)
		ui.Row(c).Padding(18, 18, 6).Shrink(0).Children(func() {
			theme.Text(c, theme.Section, t.TextSecondary, "Projects")
		})
		for _, p := range m.app.snap.projects {
			m.projectRow(c, t, p)
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

func (m *Main) inboxRow(c *ui.Context, t *theme.Tokens) {
	n, _ := openCount(m.app.snap.tasks, func(t data.Task) bool { return t.ProjectID == nil })
	row := sidebarRow(c, t, "inbox", m.nav.inbox)
	row.Children(func() {
		ui.Icon(c, theme.IconInbox).Size(16, 16).TextColor(t.TextSecondary)
		sidebarLabel(c, t, "Inbox", m.nav.inbox)
		sidebarCount(c, t, n)
	})
	if row.Clicked() {
		m.nav, m.openID = nav{inbox: true}, 0
	}
}

func (m *Main) projectRow(c *ui.Context, t *theme.Tokens, p data.Project) {
	selected := !m.nav.inbox && m.nav.projectID == p.ID
	n, blocked := openCount(m.app.snap.tasks, func(task data.Task) bool { return task.ProjectID != nil && *task.ProjectID == p.ID })
	row := sidebarRow(c, t, p.ID, selected)
	row.Children(func() {
		ui.Box(c).Size(16, 16).Shrink(0).Center().Children(func() {
			theme.Dot(c, t.TextTertiary, 6)
		})
		sidebarLabel(c, t, p.Name, selected)
		if blocked {
			theme.Dot(c, t.Blocked(), 6).Shrink(0)
		}
		sidebarCount(c, t, n)
	})
	if row.Clicked() {
		m.nav, m.openID = nav{projectID: p.ID}, 0
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

// sidebarRow is a 32 pt row inset 8 pt, highlighted by selection or hover.
func sidebarRow(c *ui.Context, t *theme.Tokens, key any, selected bool) ui.Element {
	row := ui.Row(c.Key(key)).Height(theme.RowOneLine).Shrink(0).Margin(0, 8).Padding(0, 10).Gap(8).
		Radius(theme.RadiusCard).AlignItems(ui.Center).Cursor(ui.CursorPointer)
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
