package views

import (
	"sort"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// dragCard is the value a card carries while it is dragged between columns.
type dragCard struct{ id int }

// boardColumn is one status on the board.
type boardColumn struct {
	status data.Status
	tasks  []data.Task
}

// boardColumns returns the columns the board draws: those with tasks, and To do always, since it
// is where a new task goes. There are no strips and no zero-count columns. Backlog and Done follow
// the list's "Done · Backlog — show" line: folded away, they would push the live columns off the
// screen, and they stay reachable as drop targets.
func (m *Main) boardColumns() []boardColumn {
	by := map[data.Status][]data.Task{}
	for _, t := range m.selectionTasks() {
		by[t.Status] = append(by[t.Status], t)
	}
	var cols []boardColumn
	for _, s := range data.BoardStatuses {
		parked := s == data.Backlog || s == data.Done
		if (len(by[s]) == 0 && s != data.Todo) || (parked && !m.expanded[m.nav]) {
			continue
		}
		tasks := by[s]
		if s == data.Done {
			sort.SliceStable(tasks, func(i, j int) bool { return doneAt(tasks[i]).After(doneAt(tasks[j])) })
		} else {
			sortForList(tasks)
		}
		cols = append(cols, boardColumn{status: s, tasks: tasks})
	}
	return cols
}

// hiddenStatuses are the board statuses with no column, which show as drop targets while a card
// is dragged.
func hiddenStatuses(cols []boardColumn) []data.Status {
	shown := map[data.Status]bool{}
	for _, col := range cols {
		shown[col.status] = true
	}
	var out []data.Status
	for _, s := range data.BoardStatuses {
		if !shown[s] {
			out = append(out, s)
		}
	}
	return out
}

// boardActive is true when the project is shown as a board and has something to show; an empty
// project reads the same in both views.
func (m *Main) boardActive() bool { return m.showsBoard() && len(m.selectionTasks()) > 0 }

// boardScreen is the header in the page's column, and the columns under it in a horizontal
// scroller.
func (m *Main) boardScreen(c *ui.Context, t *theme.Tokens) {
	pad := m.sidePad(c)
	ui.Column(c).FillWidth().Shrink(0).AlignItems(ui.Center).Children(func() {
		ui.Column(c).FillWidth().MaxWidth(theme.ContentMax).Padding(0, pad).Children(func() { m.listHeader(c, t) })
	})
	cols := m.boardColumns()
	hidden := hiddenStatuses(cols)
	m.boardFold(c, t, pad)
	width := columnWidth(m.contentWidth(c), len(cols))
	// The drop targets float over the scroller's right edge, so they are in reach however far
	// the board is scrolled.
	ui.Box(c).Grow(1).MinHeight(0).Children(func() {
		ui.ScrollHorizontal(c).Fill().TrackScroll(&m.scroll).Children(func() {
			ui.Row(c).FillHeight().Padding(16, theme.BoardPadding, 0).Gap(theme.ColumnGap).AlignItems(ui.Stretch).
				Focusable().FocusRing(false).AutoFocus().Bind(&m.listRef).Children(func() {
				for _, col := range cols {
					m.column(c, t, col, width)
				}
			})
		})
		m.dropTargets(c, t, hidden)
	})
}

// boardFold is the list's last line, under the board's header: "Done 4 · Backlog 2 — show".
func (m *Main) boardFold(c *ui.Context, t *theme.Tokens, pad float32) {
	lm := m.listModel()
	if len(lm.done) == 0 && len(lm.backlog) == 0 {
		return
	}
	ui.Column(c).FillWidth().Shrink(0).AlignItems(ui.Center).Children(func() {
		ui.Column(c).FillWidth().MaxWidth(theme.ContentMax).Padding(0, pad).Children(func() {
			m.foldLine(c, t, lm, m.expanded[m.nav], 4, true)
		})
	})
}

// columnWidth shares the room between n columns, between the narrowest and the widest a column
// goes; beyond that the board scrolls.
func columnWidth(avail float32, n int) float32 {
	if n == 0 {
		return theme.ColumnWidth
	}
	room := avail - 2*theme.BoardPadding - float32(n-1)*theme.ColumnGap
	return min(max(room/float32(n), theme.MinColumnWidth), theme.ColumnWidth)
}

// column is one status. A card dropped on it takes its status.
func (m *Main) column(c *ui.Context, t *theme.Tokens, col boardColumn, width float32) {
	box := ui.Column(c.Key(string(col.status))).Width(width).Shrink(0).Padding(0, 8).Radius(theme.RadiusPanel).Background(t.BgSubtle)
	_, over := ui.DragOver[dragCard](box)
	if over {
		box.Border(1, t.Accent.Alpha(0.6))
	}
	if d, ok := ui.Drop[dragCard](box); ok {
		m.dropCard(d.id, col.status)
	}
	box.Children(func() {
		ui.Row(c.Key("header")).Height(40).Shrink(0).Padding(0, 0, 0, 6).Gap(8).AlignItems(ui.Center).Children(func() {
			theme.StatusGlyph(c, t, col.status, 0)
			theme.Text(c, theme.BodyStrong, t.Text, boardName(m.say, col.status))
			theme.Text(c, theme.Meta, t.TextTertiary, itoa(len(col.tasks)))
		})
		m.columnBody(c, t, col)
	})
}

func (m *Main) columnBody(c *ui.Context, t *theme.Tokens, col boardColumn) {
	tasks, hidden := col.tasks, 0
	if col.status == data.Done && !m.allDone[m.nav] && len(tasks) > theme.DoneRecentLimit {
		hidden = len(tasks) - theme.DoneRecentLimit
		tasks = tasks[:theme.DoneRecentLimit]
	}
	ui.Scroll(c.Key("scroll")).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Gap(8).Padding(0, 0, 12).Children(func() {
			if col.status == data.Todo {
				m.boardAdd(c, t)
			}
			if len(tasks) == 0 && col.status == data.Todo && !m.adding {
				theme.Paragraph(c, theme.Meta, t.TextTertiary, m.say.BoardEmpty(), 3).Padding(2, 6, 0)
			}
			for _, task := range tasks {
				m.card(c, t, task)
			}
			if hidden > 0 && link(c, t, m.say.BoardShowAll(hidden+len(tasks))).Margin(4, 0, 0, 6).Clicked() {
				m.allDone[m.nav] = true
			}
		})
	})
}

// boardAdd is "+ Add a task" at the top of To do, and the input it opens.
func (m *Main) boardAdd(c *ui.Context, t *theme.Tokens) {
	if m.adding && m.addNav == m.nav {
		ui.Row(c.Key("board-add-input")).Padding(8, 12).Radius(theme.RadiusCard).Background(t.Bg).Border(1, t.Accent.Alpha(0.6)).Children(func() {
			box := ui.Box(c).Grow(1).MinWidth(0)
			box.Children(func() {
				theme.BodyStrong.Apply(ui.TextInputBase(c.Key("add-input"), &m.addBuf)).TextColor(t.Text).
					Label(m.say.ListNewTaskTitle()).AutoFocus().Bind(&m.addRef).OnTextKey(m.addKey)
				if m.addBuf == "" {
					theme.Text(c, theme.BodyStrong, t.TextTertiary, m.say.ListTaskTitle()).Absolute().Left(0).Top(0).PassThrough()
				}
			})
		})
		return
	}
	row := ui.Row(c.Key("board-add")).Height(theme.RowOneLine).Padding(0, 6).Radius(theme.RadiusCard).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(m.say.ListAddTask())
	if row.Hovered() {
		row.Background(t.BgHover)
	}
	row.Children(func() { theme.Text(c, theme.Body, t.TextTertiary, m.say.ListAddTaskRow()) })
	if row.Clicked() {
		m.adding, m.addNav, m.addBuf = true, m.nav, ""
	}
}

// dropTargets are thin targets for the statuses that have no column. They appear while a card is
// dragged, at the right edge.
func (m *Main) dropTargets(c *ui.Context, t *theme.Tokens, hidden []data.Status) {
	if (m.dragging == 0 && m.wasDrag == 0) || len(hidden) == 0 {
		return
	}
	ui.Column(c.Key("drop-targets")).Absolute().Right(16).Top(16).Width(148).Padding(8).Gap(6).
		Radius(theme.RadiusPanel).Background(t.Bg).Border(1, t.Border).Shadow(0, 8, 24, 0, t.Shadow).Children(func() {
		for _, s := range hidden {
			target := ui.Row(c.Key("drop-"+string(s))).Height(40).Shrink(0).Padding(0, 12).Gap(8).AlignItems(ui.Center).
				Radius(theme.RadiusCard).Border(1, t.Border).BorderStyle(ui.BorderDashed)
			if _, over := ui.DragOver[dragCard](target); over {
				target.Background(t.BgSelected).Border(1, t.Accent.Alpha(0.6))
			}
			if d, ok := ui.Drop[dragCard](target); ok {
				m.dropCard(d.id, s)
			}
			target.Children(func() {
				theme.StatusGlyph(c, t, s, 0)
				theme.Text(c, theme.Body, t.TextSecondary, boardName(m.say, s))
			})
		}
	})
}

func (m *Main) dropCard(id int, to data.Status) {
	task, ok := m.app.Task(id)
	if !ok || task.Status == to {
		return
	}
	m.app.SetStatus(task, to)
}

func doneAt(t data.Task) time.Time {
	if t.CompletedAt != nil {
		return *t.CompletedAt
	}
	return t.UpdatedAt
}
