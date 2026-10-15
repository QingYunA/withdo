package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// milestoneMenu filters a project by milestone and manages them. It is drawn for every project,
// before it has a milestone too, and not for the Inbox.
func (m *Main) milestoneMenu(c *ui.Context, t *theme.Tokens) {
	if m.nav.screen != screenProject {
		return
	}
	pid := m.nav.projectID
	ms := m.app.snap.milestones[pid]
	label := m.say.MilestoneAll()
	for _, x := range ms {
		if x.ID == m.milestone[pid] {
			label = x.Title
		}
	}
	btn := ui.ButtonBase(c.Key("milestone")).Height(28).Padding(0, 8, 0, 10).Gap(4).Radius(theme.RadiusControl).
		Label(m.say.A11yMilestone(label)).TextColor(t.TextSecondary).AlignItems(ui.Center).Shrink(0)
	if btn.Hovered() {
		btn.Background(t.BgHover)
	}
	btn.Menu(func(mn *ui.Menu) { m.milestoneItems(mn, pid, ms) })
	btn.Children(func() {
		ui.Icon(c, theme.IconMilestone).Size(14, 14)
		theme.Text(c, theme.Body, t.TextSecondary, label)
		ui.Icon(c, theme.IconChevron).Size(14, 14)
	})
}

// milestoneItems is the menu: the filter, New milestone…, and a submenu to edit each milestone.
func (m *Main) milestoneItems(mn *ui.Menu, pid int, ms []data.Milestone) {
	if mn.Item(m.say.MilestoneAll()).Checked(m.milestone[pid] == 0).Chosen() {
		m.milestone[pid] = 0
	}
	if len(ms) > 0 {
		mn.Separator()
	}
	for _, x := range ms {
		if mn.Item(m.milestoneLabel(x)).Checked(m.milestone[pid] == x.ID).Chosen() {
			m.milestone[pid] = x.ID
		}
	}
	mn.Separator()
	if mn.Item(m.say.ManageMilestoneNew()).Chosen() {
		m.newMilestone(pid)
	}
	if len(ms) > 0 {
		mn.Separator()
	}
	for _, x := range ms {
		mn.Submenu(m.say.ManageMilestoneEdit(x.Title), func(sm *ui.Menu) { m.milestoneActions(sm, x) })
	}
}

// milestoneActions is the submenu of one milestone: Rename…, Due date…, Close or Reopen.
func (m *Main) milestoneActions(sm *ui.Menu, x data.Milestone) {
	if sm.Item(m.say.ManageMilestoneRename()).Chosen() {
		m.renameMilestone(x)
	}
	if sm.Item(m.say.ManageMilestoneDue()).Chosen() {
		m.dueMilestone(x)
	}
	if x.Open {
		if sm.Item(m.say.ManageMilestoneClose()).Chosen() {
			_ = m.app.UpdateMilestone(x.ID, data.MilestonePatch{Open: data.BoolPtr(false)})
		}
		return
	}
	if sm.Item(m.say.ManageMilestoneReopen()).Chosen() {
		_ = m.app.UpdateMilestone(x.ID, data.MilestonePatch{Open: data.BoolPtr(true)})
	}
}

// viewToggle is the List | Board switch: a quiet well with the chosen side raised.
func (m *Main) viewToggle(c *ui.Context, t *theme.Tokens) {
	pid := m.nav.projectID
	ui.Row(c.Key("view-toggle")).Shrink(0).Padding(2).Radius(7).Background(t.BgSubtle).Children(func() {
		for _, side := range []struct {
			key, name, label string
			board            bool
		}{
			{"List", m.say.ViewList(), m.say.A11yViewList(), false},
			{"Board", m.say.ViewBoard(), m.say.A11yViewBoard(), true},
		} {
			on := m.board[pid] == side.board
			seg := ui.ButtonBase(c.Key("view-"+side.key)).Padding(3, 10).Radius(5).Label(side.label).Cursor(ui.CursorPointer)
			switch {
			case on:
				seg.Background(t.Bg).Shadow(0, 1, 2, 0, t.Shadow.Alpha(0.35))
			case seg.Hovered():
				seg.Background(t.BgHover)
			}
			ink := t.TextSecondary
			if on {
				ink = t.Text
			}
			seg.Children(func() { theme.Text(c, theme.Meta, ink, side.name) })
			if seg.Clicked() && !on {
				m.board[pid] = side.board
			}
		}
	})
}
