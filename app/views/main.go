package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/locales"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/QingYunA/adhd-todo/app/words"
	"github.com/egoist/mygo/ui"
)

// titleBarHeight is the unified title bar: tall enough for the traffic lights and a 20 pt title.
const titleBarHeight float32 = 52

// dragStrip is the quiet band above a page that still drags the window.
const dragStrip float32 = 24

type screen int

const (
	screenMine screen = iota
	screenAgents
	screenInbox
	screenProject
	screenCompleted
	screenCalendar
)

// nav says what the sidebar selected: Mine, Agents, the Inbox, or a project. It is comparable, so it keys
// the per-screen state.
type nav struct {
	screen    screen
	projectID int
}

// Main is the window: sidebar, one screen and the inspector, with the sheets that float over them.
type Main struct {
	app *App

	// say and clock word the interface in the language and formats the window shows; View sets
	// them every frame.
	say       locales.Messages
	clock     words.Clock
	weekStart time.Weekday // the day the calendar's weeks start on

	nav    nav
	mine   data.Mine         // what the Mine screen shows this frame
	agents data.AgentsScreen // what the Agents screen shows this frame
	hop    hopper            // the hop to the Mine row, and the badge there
	cols   columnsState      // a project's two columns, and where they were drawn
	ho     handOffState      // handing a task to an agent, and its motion
	openID int               // task open in the inspector, 0 for none
	sel    int               // task the keyboard highlights, 0 for none
	moved  bool
	width  float32 // the room the screen has this frame
	// focusComposer is set by Cmd+K: the page's composer takes the focus once it is drawn.
	// focusInspectorComposer is set by "Ask about this task…": the inspector's does.
	focusComposer          bool
	focusInspectorComposer bool

	milestone map[int]int  // milestone filter per project, 0 for all
	board     map[int]bool // projects shown as a board
	expanded  map[nav]bool // Done and Backlog opened under a list
	allDone   map[nav]bool // every Done task, not the recent three or five

	cal       calendarState // the Calendar page: its month and its selected day
	manage    manageState   // the sheets, renaming, search and Completed
	searchRef ui.Handle     // the search field in the sidebar, for Cmd+F
	renameRef ui.Handle     // the input that renames a project in the sidebar

	cmp     [2]composer                  // the composer of the page and the inspector's, by surface
	convo   convo                        // the answer under the composer that sent the last request
	marks   map[int]map[string]time.Time // fields the assistant set, by task and row key
	listRef ui.Handle                    // the page, so the arrow keys and Space reach it
	queue   queued                       // what a text input's last key asked of the next frame

	pop     popMenu // the task menu
	selRect ui.Rect // where the selected row or card is, for the menu the S, P, M and R keys open

	scroll    ui.ScrollState
	inspector inspector
	copied    time.Time
}

// NewMain opens on Mine.
func NewMain(a *App) *Main {
	return &Main{
		app: a, milestone: map[int]int{}, board: map[int]bool{},
		expanded: map[nav]bool{}, allDone: map[nav]bool{},
	}
}

// OpenTask shows a task in the inspector, over the Mine screen.
func (m *Main) OpenTask(id int) {
	if _, ok := m.app.Task(id); !ok {
		return
	}
	m.nav = nav{}
	m.openID, m.sel = id, id
	m.inspector.reset()
}

// openSettings asks the shell for the Settings window, on a pane.
func (m *Main) openSettings(p Pane) { m.app.Act.OpenSettings(p) }

func (m *Main) goTo(n nav) {
	m.clearSearch()
	if m.nav != n {
		m.cmp[surfacePage].keepDraft(m.nav, n)
		m.nav, m.openID, m.sel = n, 0, 0
		m.scroll = ui.ScrollState{}
	}
}

// View builds the window for one frame.
func (m *Main) View(c *ui.Context) {
	t := theme.Resolve(c)
	m.say, m.clock = locales.In(c.Locale()), words.New(c.Locale(), c.Region())
	m.weekStart = firstWeekday(c)
	vibrant := c.Vibrancy()
	if vibrant {
		c.Root().Background(ui.Transparent)
	}
	if _, ok := m.app.Task(m.openID); !ok {
		m.openID = 0
	}
	if _, ok := m.app.Project(m.nav.projectID); m.nav.screen == screenProject && !ok {
		m.goTo(nav{}) // archived or deleted, here or by another process
	}
	if on := m.searching(); on != m.manage.search.was {
		m.manage.search.was, m.scroll = on, ui.ScrollState{}
	}
	m.dropStaleReply()
	m.popTick()
	m.width = m.contentWidth(c)
	m.mine, m.agents = m.app.buildMine(), m.app.buildAgents()
	m.hopTick(c)
	m.handOffTick(c)
	ids := m.rowIDs()
	m.keys(c, ids)
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		m.sidebar(c, t, vibrant)
		vline(c, t)
		m.content(c, t)
		m.inspectorPane(c, t)
	})
	m.newProjectSheet(c, t)
	m.milestoneSheet(c, t)
	m.confirmSheet(c, t)
	m.taskMenuView(c, t)
	m.hopOverlay(c)
	m.handOffOverlay(c, t)
}

// content is the screen the sidebar chose.
func (m *Main) content(c *ui.Context, t *theme.Tokens) {
	ui.Column(c).Grow(1).MinWidth(0).Background(t.Bg).Children(func() {
		ui.Box(c).Height(dragStrip).Shrink(0).DragWindow()
		m.errorStrip(c, t)
		switch {
		case m.searching():
			m.page(c, t, func() { m.searchScreen(c, t) })
		case m.nav.screen == screenMine:
			m.page(c, t, func() { m.mineScreen(c, t) })
		case m.nav.screen == screenAgents:
			m.page(c, t, func() { m.agentsScreen(c, t) })
		case m.nav.screen == screenCompleted:
			m.page(c, t, func() { m.completedScreen(c, t) })
		case m.nav.screen == screenCalendar:
			m.pageIn(c, t, theme.CalendarMax, func() { m.calendarScreen(c, t) })
		case m.boardActive():
			m.boardScreen(c, t)
		case m.columnsActive():
			m.pageIn(c, t, theme.BoardMax+2*m.sidePad(c), func() { m.listScreen(c, t) })
		default:
			m.page(c, t, func() { m.listScreen(c, t) })
		}
	})
}

// pageKey keys the page's scroller: the screen, or the search results over it.
func (m *Main) pageKey() any {
	if m.searching() {
		return "search"
	}
	return m.nav
}

func (m *Main) showsBoard() bool {
	return m.nav.screen == screenProject && m.board[m.nav.projectID]
}

// contentWidth is the room the screen has: the window less the sidebar and the open inspector.
func (m *Main) contentWidth(c *ui.Context) float32 {
	w, _ := c.Size()
	w -= theme.SidebarWidth + 1
	if m.openID != 0 {
		w -= theme.InspectorWidth + 1
	}
	return w
}

// sidePad is 56 pt when the 820 pt column fits, less as the window or the inspector squeezes it.
func (m *Main) sidePad(c *ui.Context) float32 {
	w := m.contentWidth(c)
	switch {
	case w >= theme.ContentMax:
		return theme.ContentPad
	case w >= 640:
		return 40
	}
	return 24
}

// page scrolls a centered column of at most 820 pt, the way a document reads. The column is the
// keyboard's focus, so the arrow keys and Space work as soon as the screen opens.
func (m *Main) page(c *ui.Context, t *theme.Tokens, body func()) {
	m.pageIn(c, t, theme.ContentMax, body)
}

// pageIn is page for a column of at most maxWidth.
func (m *Main) pageIn(c *ui.Context, t *theme.Tokens, maxWidth float32, body func()) {
	pad := m.sidePad(c)
	ui.Scroll(c.Key(m.pageKey())).Grow(1).MinHeight(0).TrackScroll(&m.scroll).Children(func() {
		ui.Column(c).FillWidth().AlignItems(ui.Center).Children(func() {
			col := ui.Column(c).FillWidth().MaxWidth(maxWidth).Padding(0, pad, 48).Focusable().FocusRing(false)
			if !m.focusComposer && !m.searchRef.Focused(c) { // typing a search keeps the keyboard
				col.AutoFocus()
			}
			col.Bind(&m.listRef).Children(body)
		})
	})
}

func (m *Main) errorStrip(c *ui.Context, t *theme.Tokens) {
	if m.app.Err() == nil {
		return
	}
	ui.Row(c).Shrink(0).Padding(8, 24).Gap(8).Background(t.BgSubtle).Children(func() {
		theme.Text(c, theme.Meta, t.TextSecondary, m.say.ErrNoStore())
		ui.Spacer(c)
		if link(c, t, m.say.ConvoRetry()).Clicked() {
			m.app.Refresh()
		}
	})
}
