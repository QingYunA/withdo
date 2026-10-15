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
	screenNow screen = iota
	screenInbox
	screenProject
	screenLogbook
)

// nav says what the sidebar selected: Now, the Inbox, or a project. It is comparable, so it keys
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
	say   locales.Messages
	clock words.Clock

	nav       nav
	openID    int // task open in the inspector, 0 for none
	sel       int // task the keyboard highlights, 0 for none
	moved     bool
	width     float32 // the room the screen has this frame
	wantAsk   bool    // Cmd+K was pressed: the ask field takes the focus once it is drawn
	askFrames int     // how many frames the ask field has been drawn since the screen changed

	milestone map[int]int  // milestone filter per project, 0 for all
	board     map[int]bool // projects shown as a board
	expanded  map[nav]bool // Done and Backlog opened under a list
	allDone   map[nav]bool // every Done task, not the recent ten
	adding    bool         // the inline "Add a task" input is open
	addNav    nav          // the screen it belongs to
	addBuf    string
	dragging  int // task being dragged this frame, so the hidden statuses show as drop targets
	wasDrag   int // the one dragged in the frame before: the targets live a frame past the drop

	manage    manageState // the sheets, renaming, search and the Logbook
	searchRef ui.Handle   // the search field in the sidebar, for Cmd+F
	renameRef ui.Handle   // the input that renames a project in the sidebar

	ask     askState
	convo   convo     // the conversation sheet under the ask field
	hintOn  bool      // the "Connect a model" line is showing under the field
	askRef  ui.Handle // the ask field, for Cmd+K
	addRef  ui.Handle // the inline add input
	listRef ui.Handle // the page, so the arrow keys and Space reach it
	queue   queued    // what a text input's last key asked of the next frame

	scroll    ui.ScrollState
	inspector inspector
	copied    time.Time
}

// NewMain opens on Now.
func NewMain(a *App) *Main {
	return &Main{
		app: a, milestone: map[int]int{}, board: map[int]bool{},
		expanded: map[nav]bool{}, allDone: map[nav]bool{},
	}
}

// OpenTask shows a task in the inspector, over the Now screen.
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
		m.nav, m.openID, m.sel, m.askFrames, m.hintOn = n, 0, 0, 0, false
		m.cancelAdding()
		m.scroll = ui.ScrollState{}
	}
}

// View builds the window for one frame.
func (m *Main) View(c *ui.Context) {
	t := theme.Resolve(c)
	m.say, m.clock = locales.In(c.Locale()), words.New(c.Locale(), c.Region())
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
	m.wasDrag, m.dragging = m.dragging, 0
	m.width = m.contentWidth(c)
	now := m.app.buildNow()
	ids := m.rowIDs(now)
	m.keys(c, ids)
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		m.sidebar(c, t, vibrant, now)
		vline(c, t)
		m.content(c, t, now)
		m.inspectorPane(c, t)
	})
	m.newProjectSheet(c, t)
	m.milestoneSheet(c, t)
	m.confirmSheet(c, t)
}

// content is the screen the sidebar chose.
func (m *Main) content(c *ui.Context, t *theme.Tokens, now data.Now) {
	ui.Column(c).Grow(1).MinWidth(0).Background(t.Bg).Children(func() {
		ui.Box(c).Height(dragStrip).Shrink(0).DragWindow()
		m.errorStrip(c, t)
		switch {
		case m.searching():
			m.page(c, t, func() { m.searchScreen(c, t) })
		case m.nav.screen == screenNow:
			m.page(c, t, func() { m.nowScreen(c, t, now) })
		case m.nav.screen == screenLogbook:
			m.page(c, t, func() { m.logbookScreen(c, t) })
		case m.boardActive():
			m.boardScreen(c, t)
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
	pad := m.sidePad(c)
	ui.Scroll(c.Key(m.pageKey())).Grow(1).MinHeight(0).TrackScroll(&m.scroll).Children(func() {
		ui.Column(c).FillWidth().AlignItems(ui.Center).Children(func() {
			col := ui.Column(c).FillWidth().MaxWidth(theme.ContentMax).Padding(0, pad, 48).Focusable().FocusRing(false)
			if !m.wantAsk && !m.searchRef.Focused(c) { // typing a search keeps the keyboard
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
