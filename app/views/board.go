package views

import (
	"fmt"
	"sort"
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

// board is the six status columns in a horizontal scroller.
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
	for _, task := range tasks {
		byStatus[task.Status] = append(byStatus[task.Status], task)
	}
	ui.ScrollHorizontal(c).Grow(1).MinHeight(0).TrackScroll(&m.scroll).Children(func() {
		ui.Row(c).FillHeight().Padding(16, 16, 0).Gap(theme.ColumnGap).AlignItems(ui.Stretch).Children(func() {
			for _, s := range data.BoardStatuses {
				m.column(c, t, s, byStatus[s])
			}
		})
	})
}

// column is one status: its header and its cards. A card dropped on it takes its status.
func (m *Main) column(c *ui.Context, t *theme.Tokens, s data.Status, tasks []data.Task) {
	col := ui.Column(c.Key(string(s))).Width(theme.ColumnWidth).Shrink(0).Padding(0, 8).Radius(theme.RadiusPanel + 2)
	_, over := ui.DragOver[dragCard](col)
	if over {
		col.Background(t.BgSubtle)
	}
	col.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	if d, ok := ui.Drop[dragCard](col); ok {
		m.dropCard(d.id, s)
	}
	col.Children(func() {
		m.columnHeader(c, t, s, len(tasks))
		m.columnBody(c, t, s, tasks, over)
	})
}

func (m *Main) dropCard(id int, to data.Status) {
	task, ok := m.app.Task(id)
	m.dragging = 0
	if !ok || task.Status == to {
		return
	}
	m.app.SetStatus(task, to)
}

func (m *Main) columnHeader(c *ui.Context, t *theme.Tokens, s data.Status, n int) {
	ui.Row(c).Height(40).Shrink(0).Padding(0, 4).Gap(8).AlignItems(ui.Center).Children(func() {
		theme.StatusGlyph(c, t, s, 0)
		theme.Text(c, theme.BodyStrong, t.Text, s.Label())
		theme.Text(c, theme.Meta, t.TextTertiary, itoa(n))
	})
}

func (m *Main) columnBody(c *ui.Context, t *theme.Tokens, s data.Status, tasks []data.Task, over bool) {
	hidden := 0
	if s == data.Done {
		sort.SliceStable(tasks, func(i, j int) bool { return doneAt(tasks[i]).After(doneAt(tasks[j])) })
		if !m.showAllDone[m.nav] && len(tasks) > theme.DoneRecentLimit {
			hidden = len(tasks) - theme.DoneRecentLimit
			tasks = tasks[:theme.DoneRecentLimit]
		}
	} else {
		byPriorityThenRecent(tasks)
	}
	ui.Scroll(c.Key("scroll")).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Gap(8).Padding(0, 0, 16).Children(func() {
			if len(tasks) == 0 && !over {
				columnHint(c, t, s)
			}
			for _, task := range tasks {
				m.card(c, t, task)
			}
			if hidden > 0 {
				if link(c, t, fmt.Sprintf("Show all %d", hidden+len(tasks))).Margin(4, 0, 0, 4).Clicked() {
					m.showAllDone[m.nav] = true
				}
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
