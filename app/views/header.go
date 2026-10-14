package views

import (
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// milestoneMenu filters a project by milestone. It is drawn only for a project that has some.
func (m *Main) milestoneMenu(c *ui.Context, t *theme.Tokens) {
	if m.nav.screen != screenProject || len(m.app.snap.milestones[m.nav.projectID]) == 0 {
		return
	}
	pid := m.nav.projectID
	label := "All milestones"
	for _, ms := range m.app.snap.milestones[pid] {
		if ms.ID == m.milestone[pid] {
			label = ms.Title
		}
	}
	btn := ui.ButtonBase(c.Key("milestone")).Height(28).Padding(0, 8, 0, 10).Gap(4).Radius(theme.RadiusControl).
		Label("Milestone: " + label).TextColor(t.TextSecondary).AlignItems(ui.Center).Shrink(0)
	if btn.Hovered() {
		btn.Background(t.BgHover)
	}
	btn.Menu(func(mn *ui.Menu) {
		if mn.Item("All milestones").Checked(m.milestone[pid] == 0).Chosen() {
			m.milestone[pid] = 0
		}
		mn.Separator()
		for _, ms := range m.app.snap.milestones[pid] {
			if mn.Item(milestoneLabel(ms, m.app.Now())).Checked(m.milestone[pid] == ms.ID).Chosen() {
				m.milestone[pid] = ms.ID
			}
		}
	})
	btn.Children(func() {
		ui.Icon(c, theme.IconMilestone).Size(14, 14)
		theme.Text(c, theme.Body, t.TextSecondary, label)
		ui.Icon(c, theme.IconChevron).Size(14, 14)
	})
}

// viewToggle is the List | Board switch: a quiet well with the chosen side raised.
func (m *Main) viewToggle(c *ui.Context, t *theme.Tokens) {
	pid := m.nav.projectID
	ui.Row(c.Key("view-toggle")).Shrink(0).Padding(2).Radius(7).Background(t.BgSubtle).Children(func() {
		for _, side := range []struct {
			name  string
			board bool
		}{{"List", false}, {"Board", true}} {
			on := m.board[pid] == side.board
			seg := ui.ButtonBase(c.Key("view-"+side.name)).Padding(3, 10).Radius(5).Label(side.name + " view").Cursor(ui.CursorPointer)
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
				m.cancelAdding()
			}
		}
	})
}
