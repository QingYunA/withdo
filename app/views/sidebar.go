package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// sidebarRowPad is the padding inside a sidebar row, left and right.
const sidebarRowPad float32 = 10

// sidebarRowGap is the room between the icon and the label of a sidebar row.
const sidebarRowGap float32 = 10

// sidebar is the source list over the window's material: Today, Agents and Calendar (and Recent
// changes, once it is switched on), then the projects, then Settings at the bottom.
func (m *Main) sidebar(c *ui.Context, t *theme.Tokens, vibrant bool) {
	side := ui.Column(c).Width(theme.SidebarWidth).Shrink(0)
	if !vibrant {
		side.Background(t.BgSidebar)
	}
	side.Children(func() {
		ui.Box(c).Height(titleBarHeight).Shrink(0).DragWindow()
		m.searchField(c, t)
		m.mineRow(c, t)
		m.agentsRow(c, t)
		m.navRow(c, t, "calendar", nav{screen: screenCalendar}, theme.IconCalendar, m.say.NavCalendar(), 0)
		if m.app.RecentChangesOn() {
			m.navRow(c, t, "recent", nav{screen: screenRecent}, theme.IconBranch, m.say.NavRecent(), 0)
		}
		m.projectsHeader(c, t)
		m.projectRows(c, t)
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
		if !match(t) || t.IsSubtask() || t.Status.Terminal() || t.Status == data.Backlog {
			continue
		}
		n++
		blocked = blocked || t.Status == data.Blocked
	}
	return n, blocked
}

// mineRow is Today: the pixel you, and at the right the number of handed-over rows not seen yet in
// a red badge, else how many rows wait on you.
func (m *Main) mineRow(c *ui.Context, t *theme.Tokens) {
	to := nav{}
	selected := m.shows(to)
	row := sidebarRow(c, t, "mine", selected)
	m.hop.mine = row.Bounds()
	row.Children(func() {
		avatarSlot(c, theme.You)
		sidebarLabel(c, t, m.say.NavMine(), selected)
		if n := m.hop.shown; n > 0 {
			m.unseenBadge(c, t, n)
		} else {
			sidebarCount(c, t, m.mine.Count())
		}
	})
	if row.Clicked() {
		m.goTo(to)
	}
}

// agentsRow is Agents: the robot in the accent color, and how many tasks are at work.
func (m *Main) agentsRow(c *ui.Context, t *theme.Tokens) {
	to := nav{screen: screenAgents}
	selected := m.shows(to)
	row := sidebarRow(c, t, "agents", selected)
	m.hop.agents = row.Bounds()
	row.Children(func() {
		avatarSlot(c, theme.AccentBot(t.Accent))
		sidebarLabel(c, t, m.say.NavAgents(), selected)
		sidebarCount(c, t, m.agents.Count())
	})
	if row.Clicked() {
		m.goTo(to)
	}
}

// avatarSlot is an avatar in the place of a row's 18 pt icon, so the labels stay in line.
func avatarSlot(c *ui.Context, who theme.Who) {
	theme.AvatarIcon(c.Key("avatar"), who, theme.AvatarRow)
}

// unseenBadge is the red count of what agents handed over since the user last looked.
func (m *Main) unseenBadge(c *ui.Context, t *theme.Tokens, n int) {
	badge := ui.Row(c.Key("unseen")).Height(theme.BadgeSize).MinWidth(theme.BadgeSize).Padding(0, 5).Shrink(0).
		Radius(theme.BadgeSize / 2).Background(t.Blocked()).AlignItems(ui.Center).Justify(ui.Center).Label(m.say.A11yMineUnseen(n))
	if op := m.badgeOpacity(c); op < 1 {
		badge.Opacity(op)
	}
	badge.Children(func() { theme.Text(c, theme.Badge, t.AccentText, itoa(n)) })
}

func (m *Main) navRow(c *ui.Context, t *theme.Tokens, key string, to nav, icon *ui.SVG, label string, count int) {
	selected := m.shows(to)
	row := sidebarRow(c, t, key, selected)
	row.Children(func() {
		ui.Icon(c, icon).Size(theme.IconRow, theme.IconRow).TextColor(t.TextSecondary)
		sidebarLabel(c, t, label, selected)
		sidebarCount(c, t, count)
	})
	if row.Clicked() {
		m.goTo(to)
	}
}

// projectRows lists the projects: the top-level ones in name order, each with its sub-projects
// nested under it, one indent step in. The snapshot is one flat, name-ordered list.
func (m *Main) projectRows(c *ui.Context, t *theme.Tokens) {
	projects := m.app.snap.projects
	known := make(map[int]bool, len(projects))
	for _, p := range projects {
		known[p.ID] = true
	}
	children := make(map[int][]data.Project, len(projects))
	var top []data.Project
	for _, p := range projects {
		// A sub-project whose parent is not on screen (not expected: archiving takes the whole
		// branch) reads as a top-level project rather than vanishing.
		if p.ParentID != nil && known[*p.ParentID] {
			children[*p.ParentID] = append(children[*p.ParentID], p)
			continue
		}
		top = append(top, p)
	}
	seen := make(map[int]bool, len(projects))
	var draw func(p data.Project, depth int)
	draw = func(p data.Project, depth int) {
		if seen[p.ID] { // a parent cycle is not expected; never draw forever
			return
		}
		seen[p.ID] = true
		n, blocked := openCount(m.app.snap.tasks, func(task data.Task) bool {
			return task.ProjectID != nil && *task.ProjectID == p.ID
		})
		m.projectRow(c, t, p, n, blocked, depth)
		for _, child := range children[p.ID] {
			draw(child, depth+1)
		}
	}
	for _, p := range top {
		draw(p, 0)
	}
}

// projectRow is one project in the sidebar; a sub-project sits depth steps in, its folder where its
// parent's label is.
func (m *Main) projectRow(c *ui.Context, t *theme.Tokens, p data.Project, n int, blocked bool, depth int) {
	to := nav{screen: screenProject, projectID: p.ID}
	selected := m.shows(to)
	renaming := m.manage.rename.id == p.ID && !m.manage.rename.page
	row := sidebarRow(c, t, p.ID, selected)
	if depth > 0 {
		row.Padding(0, sidebarRowPad, 0, sidebarRowPad+subRowIndent*float32(depth))
	}
	row.ContextMenu(func(mn *ui.Menu) { m.projectMenu(mn, p) })
	row.Children(func() {
		ui.Icon(c, theme.IconFolder).Size(theme.IconRow, theme.IconRow).TextColor(theme.ProjectColor(p.Name)).Shrink(0)
		if renaming {
			m.renameInput(c, t, theme.Body)
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
	if mn.Item(m.say.ManageNewSubproject()).Chosen() {
		m.openNewProject(p.ID)
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
		btn.Children(func() { ui.Icon(c, theme.IconPlus).Size(theme.IconSmall, theme.IconSmall) })
		if btn.Clicked() {
			m.openNewProject(0)
		}
	})
}

func (m *Main) settingsRow(c *ui.Context, t *theme.Tokens) {
	alert := m.app.SyncNeedsAttention()
	row := sidebarRow(c, t, "settings", false)
	row.Children(func() {
		ui.Icon(c, theme.IconGear).Size(theme.IconRow, theme.IconRow).TextColor(t.TextSecondary)
		sidebarLabel(c, t, m.say.NavSettings(), false)
		if alert {
			ui.Icon(c, theme.IconCloud).Size(theme.IconSmall, theme.IconSmall).TextColor(t.Blocked()).
				Shrink(0).Label(m.say.SettingsSyncAlert())
		}
		theme.Text(c, theme.Meta, t.TextTertiary, keyHint("⌘,")).Shrink(0)
	})
	if row.Clicked() {
		if alert {
			m.openSettings(PaneAccount)
		} else {
			m.openSettings(PaneGeneral)
		}
	}
}

// sidebarRow is a 32 pt row inset 10 pt, highlighted by selection or hover.
func sidebarRow(c *ui.Context, t *theme.Tokens, key any, selected bool) ui.Element {
	row := ui.Row(c.Key(key)).Height(theme.SidebarRow).Shrink(0).Margin(0, 10).Padding(0, sidebarRowPad).Gap(sidebarRowGap).
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
