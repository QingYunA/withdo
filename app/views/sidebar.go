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
		m.searchField(c, t)
		m.navRow(c, t, "now", nav{}, theme.IconNow, m.say.NavNow(), now.Count())
		inbox, _ := openCount(m.app.snap.tasks, func(task data.Task) bool { return task.ProjectID == nil })
		m.navRow(c, t, "inbox", nav{screen: screenInbox}, theme.IconInbox, m.say.NavInbox(), inbox)
		m.navRow(c, t, "calendar", nav{screen: screenCalendar}, theme.IconCalendar, m.say.NavCalendar(), 0)
		m.navRow(c, t, "completed", nav{screen: screenCompleted}, theme.IconCheck, m.say.ManageNavCompleted(), 0)
		m.projectsHeader(c, t)
		for _, p := range m.app.snap.projects {
			n, blocked := openCount(m.app.snap.tasks, func(task data.Task) bool { return task.ProjectID != nil && *task.ProjectID == p.ID })
			m.projectRow(c, t, p, n, blocked)
		}
		if len(m.app.snap.projects) == 0 {
			theme.Paragraph(c, theme.Meta, t.TextTertiary, m.say.NavNoProjects(), 0).Padding(0, 18)
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
	selected := m.nav == to && !m.searching()
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
	selected := m.nav == to && !m.searching()
	renaming := m.manage.rename.id == p.ID
	row := sidebarRow(c, t, p.ID, selected)
	row.ContextMenu(func(mn *ui.Menu) { m.projectMenu(mn, p) })
	row.Children(func() {
		ui.Box(c).Size(16, theme.Compact.Line).Shrink(0).Center().Children(func() {
			theme.Text(c, theme.Compact, t.TextTertiary, "#")
		})
		if renaming {
			m.renameInput(c, t)
		} else {
			sidebarLabel(c, t, p.Name, selected)
		}
		if blocked {
			theme.Dot(c, t.Blocked(), 6).Shrink(0).Margin(0, 2, 0, 0)
		} else {
			sidebarCount(c, t, n)
		}
	})
	if row.Clicked() && !renaming {
		m.goTo(to)
	}
}

// projectMenu is the context menu of a project row.
func (m *Main) projectMenu(mn *ui.Menu, p data.Project) {
	if p.RepoSlug != "" {
		if mn.Item(m.say.ManageOpenRepo(p.RepoSlug)).Chosen() {
			m.app.Act.OpenURL("https://github.com/" + p.RepoSlug)
		}
	}
	if mn.Item(m.say.ManageMilestoneNew()).Chosen() {
		m.newMilestone(p.ID)
	}
	mn.Separator()
	if mn.Item(m.say.ManageRename()).Chosen() {
		m.startRename(p.ID, p.Name)
	}
	if mn.Item(m.say.ManageArchive()).Chosen() {
		if m.app.ArchiveProject(p.ID) == nil {
			m.leaveProject(p.ID)
		}
	}
	mn.Separator()
	if mn.Item(m.say.ManageDelete()).Chosen() {
		m.askDeleteProject(p.ID)
	}
}

// projectsHeader is PROJECTS with a quiet plus that opens the New project sheet.
func (m *Main) projectsHeader(c *ui.Context, t *theme.Tokens) {
	ui.Row(c).Padding(12, 12, 2, 18).Shrink(0).AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.Section, t.TextTertiary, m.say.NavProjects())
		ui.Spacer(c)
		tip := m.say.ManageNewProjectTip()
		btn := ui.ButtonBase(c.Key("new-project")).Size(theme.HitTarget, theme.HitTarget).Shrink(0).Center().
			Radius(theme.RadiusControl).Label(tip).Tooltip(tip).TextColor(t.TextTertiary).Cursor(ui.CursorPointer)
		if btn.Hovered() {
			btn.Background(t.BgHover).TextColor(t.TextSecondary)
		}
		btn.Children(func() { ui.Icon(c, theme.IconPlus).Size(14, 14) })
		if btn.Clicked() {
			m.openNewProject()
		}
	})
}

func (m *Main) settingsRow(c *ui.Context, t *theme.Tokens) {
	row := sidebarRow(c, t, "settings", false)
	row.Children(func() {
		ui.Icon(c, theme.IconGear).Size(16, 16).TextColor(t.TextSecondary)
		sidebarLabel(c, t, m.say.NavSettings(), false)
		theme.Text(c, theme.Meta, t.TextTertiary, "⌘,").Shrink(0)
	})
	if row.Clicked() {
		m.openSettings(PaneGeneral)
	}
}

// sidebarRow is a 32 pt row inset 10 pt, highlighted by selection or hover.
func sidebarRow(c *ui.Context, t *theme.Tokens, key any, selected bool) ui.Element {
	row := ui.Row(c.Key(key)).Height(theme.SidebarRow).Shrink(0).Margin(0, 10).Padding(0, 10).Gap(10).
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
	style := theme.Compact
	if selected {
		style = theme.CompactStrong
	}
	theme.Text(c, style, t.Text, text).Grow(1).MinWidth(0).Ellipsis("…")
}

func sidebarCount(c *ui.Context, t *theme.Tokens, n int) {
	if n > 0 {
		theme.Text(c, theme.Meta, t.TextTertiary, itoa(n)).Shrink(0)
	}
}
