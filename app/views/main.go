package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// titleBarHeight is the unified title bar: tall enough for the traffic lights and a 20 pt title.
const titleBarHeight float32 = 52

// nav says what the sidebar selected: the inbox, or a project.
type nav struct {
	inbox     bool
	projectID int
}

// Main is the main window: sidebar, board, inspector, and the settings sheet.
type Main struct {
	app *App

	nav         nav
	openID      int // task open in the inspector, 0 for none
	settings    bool
	milestone   map[int]int                  // milestone filter per project, 0 for all
	showAllDone map[nav]bool                 // "Show all" opened on the Done column
	pinned      map[nav]map[data.Status]bool // collapsed columns the user opened
	adding      data.Status                  // column with the inline title input, "" for none
	addNav      nav                          // the selection the input belongs to
	addBuf      string
	dragging    int // task being dragged, for the empty slot it leaves

	scroll    ui.ScrollState
	inspector inspector
	copied    time.Time
}

// NewMain opens on the first project, or the inbox when there is none.
func NewMain(a *App) *Main {
	m := &Main{app: a, milestone: map[int]int{}, showAllDone: map[nav]bool{}, pinned: map[nav]map[data.Status]bool{}}
	if ps := a.snap.projects; len(ps) > 0 {
		m.nav = nav{projectID: ps[0].ID}
	} else {
		m.nav = nav{inbox: true}
	}
	return m
}

// OpenTask shows a task in the inspector, selecting the project it belongs to.
func (m *Main) OpenTask(id int) {
	task, ok := m.app.Task(id)
	if !ok {
		return
	}
	if task.ProjectID == nil {
		m.nav = nav{inbox: true}
	} else {
		m.nav = nav{projectID: *task.ProjectID}
	}
	m.openID = id
	m.inspector.reset()
}

// OpenSettings shows the settings sheet.
func (m *Main) OpenSettings() { m.settings = true }

// View builds the window for one frame.
func (m *Main) View(c *ui.Context) {
	t := theme.Resolve(c)
	vibrant := c.Vibrancy()
	if vibrant {
		c.Root().Background(ui.Transparent)
	}
	if _, ok := m.app.Task(m.openID); !ok {
		m.openID = 0
	}
	if c.Shortcut(0, ui.KeyEscape) && !m.settings {
		switch {
		case m.adding != "":
			m.cancelAdding()
		case m.openID != 0:
			m.openID = 0
		}
	}
	row := ui.Row(c).Fill().AlignItems(ui.Stretch)
	row.Children(func() {
		m.sidebar(c, t, vibrant)
		vline(c, t)
		m.content(c, t)
		m.inspectorPane(c, t)
	})
	m.settingsSheet(c, t)
}

// content is the board with its header, which also serves as the title bar.
func (m *Main) content(c *ui.Context, t *theme.Tokens) {
	ui.Column(c).Grow(1).MinWidth(0).Background(t.Bg).Children(func() {
		m.header(c, t)
		hairline(c, t)
		m.board(c, t)
	})
}

// selection returns the tasks of the current sidebar choice, filtered by milestone.
func (m *Main) selectionTasks() []data.Task {
	in := func(t data.Task) bool {
		switch {
		case m.nav.inbox:
			return t.ProjectID == nil
		case t.ProjectID == nil || *t.ProjectID != m.nav.projectID:
			return false
		}
		ms := m.milestone[m.nav.projectID]
		return ms == 0 || (t.MilestoneID != nil && *t.MilestoneID == ms)
	}
	var out []data.Task
	for _, t := range m.app.snap.tasks {
		if in(t) {
			out = append(out, t)
		}
	}
	return m.app.withCompleting(out, in)
}
