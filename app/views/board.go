package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// dragCard is the value a card carries while it is dragged between columns.
type dragCard struct{ id int }

// header is the unified title bar: the project's name and repository, and its milestone filter.
func (m *Main) header(c *ui.Context, t *theme.Tokens) {
	bar := c.TitleBar()
	ui.Row(c).Height(titleBarHeight).Shrink(0).Padding(0, bar.Right+24, 0, 24).Gap(12).AlignItems(ui.Center).DragWindow().Children(func() {
		title, sub := m.headerText()
		theme.Text(c, theme.Title, t.Text, title).Shrink(1).MinWidth(0).Ellipsis("…")
		if sub != "" {
			theme.Text(c, theme.Meta, t.TextSecondary, sub).Shrink(1).MinWidth(0).Ellipsis("…")
		}
		ui.Spacer(c)
		m.milestoneMenu(c, t)
	})
}

func (m *Main) headerText() (title, sub string) {
	if m.nav.inbox {
		return "Inbox", "Tasks without a project"
	}
	p, _ := m.app.Project(m.nav.projectID)
	return p.Name, p.RepoSlug
}

func (m *Main) milestoneMenu(c *ui.Context, t *theme.Tokens) {
	if m.nav.inbox {
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
		Label("Milestone: " + label).TextColor(t.TextSecondary)
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

// board is the six status columns in a horizontal scroller. Empty columns and Done are strips, and
// when the active columns do not fit they narrow and then give way, so the board seldom scrolls.
func (m *Main) board(c *ui.Context, t *theme.Tokens) {
	tasks := m.selectionTasks()
	if !m.app.snap.loaded {
		return
	}
	if len(tasks) == 0 {
		m.boardEmpty(c, t)
		return
	}
	byStatus := map[data.Status][]data.Task{}
	counts := map[data.Status]int{}
	for _, task := range tasks {
		byStatus[task.Status] = append(byStatus[task.Status], task)
		counts[task.Status]++
	}
	w, _ := c.Size()
	plans := planColumns(boardWidth(w, m.openID != 0), counts, m.forcedOpen())
	ui.ScrollHorizontal(c).Grow(1).MinHeight(0).TrackScroll(&m.scroll).Children(func() {
		ui.Row(c).FillHeight().Padding(theme.BoardPadding, theme.BoardPadding, 0).Gap(theme.ColumnGap).AlignItems(ui.Stretch).Children(func() {
			for _, s := range data.BoardStatuses {
				m.column(c, t, s, byStatus[s], plans[s])
			}
		})
	})
}

func doneAt(t data.Task) time.Time {
	if t.CompletedAt != nil {
		return *t.CompletedAt
	}
	return t.UpdatedAt
}

// columnHint says what an empty column is for and how to fill it.
func columnHint(c *ui.Context, t *theme.Tokens, s data.Status) {
	theme.Paragraph(c, theme.Meta, t.TextTertiary, hintFor(s), 3).Padding(4, 4, 0)
}

func hintFor(s data.Status) string {
	switch s {
	case data.Backlog:
		return "Someday tasks wait here. Drag a card in to park it."
	case data.Todo:
		return "Ready to start. Drag a card here, or type one into the panel."
	case data.Doing:
		return "Agents move tasks here as they report."
	case data.Blocked:
		return "Anything stuck shows up here, in red."
	case data.Review:
		return "Waiting on you. Agents move finished work here."
	case data.Done:
		return "Finished tasks land here."
	}
	return ""
}

// boardEmpty is the board of a project with no tasks at all.
func (m *Main) boardEmpty(c *ui.Context, t *theme.Tokens) {
	title, body := "No tasks in this project yet.", "Coding agents add tasks through the adhd-todo MCP server. You can add one from the menu bar panel."
	if m.nav.inbox {
		title, body = "Your inbox is clear.", "Tasks you add from the menu bar panel land here until you give them a project."
	}
	ui.Column(c).Grow(1).Center().Gap(6).Padding(0, 48).Children(func() {
		theme.Text(c, theme.Title, t.Text, title).Margin(0, 0, 4)
		theme.Paragraph(c, theme.Body, t.TextSecondary, body, 0).MaxWidth(360).TextAlign(ui.Center)
		if link(c, t, "How to connect").Margin(8, 0, 0).Clicked() {
			m.app.Act.OpenURL(readmeURL)
		}
	})
}
