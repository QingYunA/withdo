package views

import (
	"sync/atomic"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/locales"
	"github.com/QingYunA/adhd-todo/app/page"
	"github.com/QingYunA/adhd-todo/app/selection"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/QingYunA/adhd-todo/app/words"
	"github.com/egoist/mygo/ui"
)

// titleBarHeight is the unified title bar: tall enough for the traffic lights and a 20 pt title.
const titleBarHeight float32 = 52

// dragStrip is the quiet band above a page that still drags the window.
const dragStrip float32 = 24

type screen = page.Screen

const (
	screenMine      = page.Mine
	screenAgents    = page.Agents
	screenInbox     = page.Inbox
	screenProject   = page.Project
	screenCompleted = page.Completed
	screenCalendar  = page.Calendar
	screenSummary   = page.Summary
	screenRecent    = page.Recent
)

// nav says what the sidebar selected: Mine, Agents, the Inbox, or a project. It is comparable, so it keys
// the per-screen state.
type nav struct {
	screen    screen
	projectID int
	// cat narrows Mine to one category's page: its id, or catUnsorted; 0 is the whole of Mine.
	cat int
	// sub narrows Agents to one agent's page, by the agent's name; "" is the whole screen.
	sub string
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
	mine   data.Mine         // all of Mine this frame: the sidebar's count and the hop read it
	agents data.AgentsScreen // all of Agents this frame
	// minePage and agentsPage are what the open Mine or Agents page lists: the same as mine and
	// agents, or one slice of them when nav.cat or nav.sub names one. slices holds the parts of Mine.
	minePage   data.Mine
	agentsPage data.AgentsScreen
	slices     mineSlices
	agentRows  []data.AgentTally   // the agents the sidebar lists under Agents
	hop        hopper              // the hop to the Mine row, and the badge there
	cols       columnsState        // a project's two columns, and where they were drawn
	ho         handOffState        // handing a task to an agent, and its motion
	openID     int                 // task open in the inspector, 0 for none
	sel        int                 // task the keyboard highlights, 0 for none
	multi      selection.Selection // the rows selected together on this page
	hint       selectHint          // where the Select button has flashed
	pg         page.Page           // what the window shows this frame: its sections, and the rows in the order they are drawn
	rects      map[int]ui.Rect     // where the rows were drawn this frame, and prevRects the frame before:
	prevRects  map[int]ui.Rect     // neighbouring selected rows join into one block by them
	moved      bool
	width      float32 // the room the screen has this frame
	// focusComposer is set by Cmd+K: the page's composer takes the focus once it is drawn.
	// focusInspectorComposer is set by "Ask about this task…": the inspector's does.
	focusComposer          bool
	focusInspectorComposer bool

	milestone map[int]int  // milestone filter per project, 0 for all
	board     map[int]bool // projects shown as a board
	expanded  map[nav]bool // Done and Backlog opened under a list
	allDone   map[nav]bool // every Done task, not the recent three or five

	cal       calendarState // the Calendar page: its month and its selected day
	sum       summaryState  // the Summary page: its period, its scene, its paragraphs
	recent    recentState   // the Recent changes page: what was read, and what the assistant wrote
	manage    manageState   // the sheets, renaming, search and Completed
	searchRef ui.Handle     // the search field in the sidebar, for Cmd+F
	renameRef ui.Handle     // the input that renames a project in the sidebar

	cmp     [2]composer                  // the composer of the page and the inspector's, by surface
	convo   convo                        // the answer under the composer that sent the last request
	marks   map[int]map[string]time.Time // fields the assistant set, by task and row key
	filed   []data.Receipt               // tasks the assistant sorted by itself, waiting for their toast
	listRef ui.Handle                    // the page, so the arrow keys and Space reach it
	queue   queued                       // what a text input's last key asked of the next frame

	pop      popMenu     // the task menu
	openFail openFailure // the last failure to open an agent's work, drawn under its row
	selRect  ui.Rect     // where the selected row or card is, for the menu the S, P, M and R keys open
	// owned are the rows (by foldKey) of a section that folds as one this frame, which only fade.
	owned map[string]bool

	scroll     ui.ScrollState
	dockHeight float32 // the composer dock's height in the last frame, which the bulk bar rides above
	inspector  inspector
	thumbs     thumbCache // the pictures of tasks, decoded once
	viewer     viewer     // the picture viewer over the window
	copied     time.Time

	// onMine tells the notifier, from its own goroutine, whether the window shows all of Mine.
	onMine atomic.Bool
	// onSummary tells the reminder, from its own goroutine, whether the window shows Summary.
	onSummary atomic.Bool
}

// NewMain opens on Mine.
func NewMain(a *App) *Main {
	return &Main{
		app: a, milestone: map[int]int{}, board: map[int]bool{}, rects: map[int]ui.Rect{}, prevRects: map[int]ui.Rect{},
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

// ShowMine opens the Mine screen, whole.
func (m *Main) ShowMine() { m.goTo(nav{}) }

// ShowSummary opens the Summary page on today, or on this week. It plays the scene again.
func (m *Main) ShowSummary(week bool) {
	m.goTo(nav{screen: screenSummary})
	m.sum.week, m.sum.back = week, 0
	m.leaveSummary()
}

// ShowingSummary says whether the window, as last drawn, shows Summary. Safe to call from any
// goroutine.
func (m *Main) ShowingSummary() bool { return m.onSummary.Load() }

// ShowingMine says whether the window, as last drawn, shows all of Mine: no other page, no slice of
// it, no search over it. Safe to call from any goroutine.
func (m *Main) ShowingMine() bool { return m.onMine.Load() }

// openSettings asks the shell for the Settings window, on a pane.
func (m *Main) openSettings(p Pane) { m.app.Act.OpenSettings(p) }

func (m *Main) goTo(n nav) {
	m.clearSearch()
	if m.nav != n {
		m.cmp[surfacePage].keepDraft(m.nav, n)
		m.nav, m.openID, m.sel = n, 0, 0
		m.multi.Clear()
		m.scroll = ui.ScrollState{}
	}
}

// View builds the window for one frame.
func (m *Main) View(c *ui.Context) {
	t := theme.Resolve(c)
	m.say, m.clock = locales.In(c.Locale()), words.New(c.Locale(), c.Region())
	m.weekStart = firstWeekday(c)
	m.app.setWeekStart(m.weekStart)
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
	if _, ok := m.app.Category(m.nav.cat); m.nav.cat > 0 && !ok {
		m.goTo(nav{}) // deleted, here or by another process
	}
	if on := m.searching(); on != m.manage.search.was {
		m.manage.search.was, m.scroll = on, ui.ScrollState{}
		m.multi.Clear()
	}
	if m.nav.screen == screenRecent && !m.app.RecentChangesOn() {
		m.goTo(nav{}) // switched off in Settings while it was open
	}
	m.dropStaleReply()
	m.popTick()
	m.width = m.contentWidth(c)
	m.buildPages()
	m.pg = m.pageNow()
	m.onMine.Store(m.pg.Kind == page.KindMine && m.nav.cat == 0)
	m.onSummary.Store(m.pg.Kind == page.KindSummary)
	if m.pg.Kind != page.KindSummary {
		m.leaveSummary()
	}
	if m.pg.Kind != page.KindRecent {
		m.recent.shown = false // the page reads GitHub when it opens
	}
	m.hopTick(c)
	m.handOffTick(c)
	ids := m.pg.IDs()
	m.multi.Sync(ids)
	m.prevRects, m.rects = m.rects, m.prevRects
	clear(m.rects)
	m.keys(c, ids)
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		m.sidebar(c, t, vibrant)
		vline(c, t)
		m.content(c, t)
		m.inspectorPane(c, t)
	})
	m.newProjectSheet(c, t)
	m.categorySheet(c, t)
	m.milestoneSheet(c, t)
	m.confirmSheet(c, t)
	m.dayCardView(c, t)
	m.taskMenuView(c, t)
	m.viewerView(c, t)
	m.hopOverlay(c)
	m.toastsView(c, t)
}

// content is the screen the sidebar chose.
func (m *Main) content(c *ui.Context, t *theme.Tokens) {
	ui.Column(c).Grow(1).MinWidth(0).Background(t.Bg).Children(func() {
		ui.Box(c).Height(dragStrip).Shrink(0).DragWindow()
		m.errorStrip(c, t)
		switch m.pg.Kind {
		case page.KindSearch:
			m.page(c, t, func() { m.searchScreen(c, t) })
		case page.KindMine:
			m.page(c, t, func() { m.mineScreen(c, t) })
		case page.KindAgents:
			m.page(c, t, func() { m.agentsScreen(c, t) })
		case page.KindCompleted:
			m.page(c, t, func() { m.completedScreen(c, t) })
		case page.KindCalendar:
			m.pageIn(c, t, theme.CalendarMax, func() { m.calendarScreen(c, t) })
		case page.KindSummary:
			m.summaryScreen(c, t)
		case page.KindRecent:
			m.recentScreen(c, t)
		case page.KindBoard:
			m.boardScreen(c, t)
		case page.KindColumns:
			m.pageIn(c, t, theme.BoardMax+2*m.sidePad(c), func() { m.columnsScreen(c, t) })
		default:
			m.page(c, t, func() { m.listScreen(c, t) })
		}
		m.pageComposer(c, t)
		m.bulkBar(c, t)
	})
}

// pageKey keys the page's scroller: the screen, or the search results over it.
func (m *Main) pageKey() any {
	if m.pg.Kind == page.KindSearch {
		return "search"
	}
	return m.nav
}

// shows reports whether the sidebar row that goes to n is the open page: a search over it hides it.
func (m *Main) shows(n nav) bool { return m.nav == n && m.pg.Kind != page.KindSearch }

// pageNow is the page the window shows, from what the sidebar chose, the search, and the data.
func (m *Main) pageNow() page.Page {
	in := page.In{
		Screen: m.nav.screen, Searching: m.searching(), Board: m.showsBoard(),
		Mine: m.minePage, Agents: m.agentsPage, Now: m.app.Now(),
		Folds: [2]page.Fold{{Backlog: m.expanded[m.nav], AllDone: m.allDone[m.nav]}, {Backlog: m.cols.expanded[m.nav], AllDone: m.cols.allDone[m.nav]}},
	}
	switch {
	case in.Searching:
		in.Found = m.searchFound()
	case in.Screen == screenCompleted:
		in.Done = m.completedTasks()
	case in.Screen == screenInbox || in.Screen == screenProject:
		in.Tasks = m.selection(m.app.withCompleting)
		if in.Board {
			in.Cards = m.selection(m.app.withCompletingCards)
		}
	}
	return page.Build(in)
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
			col := ui.Column(c).FillWidth().MaxWidth(maxWidth).Padding(0, pad, m.pageBottom()).Focusable().FocusRing(false)
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
