package views

import (
	"slices"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The task menu is our own popover, so it can show the status glyphs and the priorities as
// one-click strips. It opens at the pointer on a right-click, under the glyph of an agent's task,
// under a row for the S, P, M and R keys, and under a value of the inspector, where it shows the
// one piece that value edits.

// menuPage is what the menu shows: the whole menu, or one piece of it.
type menuPage int

const (
	pageMain menuPage = iota
	pageStatus
	pagePriority
	pageMove
	pageRemind
	pageCalendar // Custom… of the reminders
	pageOpenIn   // where to open an agent's work
	pageCategory // a category, or none
)

const (
	popWidth float32 = 268
	popPad   float32 = 6
	popRowH  float32 = 32
	popStrip float32 = 34
	popEdge  float32 = 8 // the room the menu keeps from the window's edge
)

// popMenu is the state of the task menu: whose it is, where it hangs, which piece it shows.
type popMenu struct {
	open   bool
	task   int   // the task, or the first of the selected ones when many is set
	many   []int // the rows selected together, when the menu is for all of them
	page   menuPage
	back   bool    // Esc goes back to the whole menu rather than closing it
	x, y   float32 // where it hangs, in the window: the pointer, or under a glyph or a value
	hl     int     // the choice the keyboard highlights, -1 for none
	filter string  // what is typed on the Move to page
	seen   string  // the filter the highlight was last set for
	when   time.Time

	tip, tipNext popTip // the choice under the pointer or the arrows, as the label shows it

	frame int      // counts frames, so the press that opened the menu does not close it
	at    int      // the frame it opened in
	items []func() // the choices of the page this frame, in order, for the arrow keys
	enter bool     // Enter was pressed in the filter
	esc   bool     // Esc was pressed in the filter
}

// popTip is the name of the strip choice that is hovered or highlighted, and which strip it is in.
type popTip struct{ strip, name string }

// openMenu shows the menu for a task, hanging from the point (x, y) of the window.
func (m *Main) openMenu(id int, page menuPage, x, y float32, back bool) {
	m.pop = popMenu{open: true, task: id, page: page, back: back, x: x, y: y, hl: -1, frame: m.pop.frame, at: m.pop.frame}
	m.sel = id
	switch task, _ := m.app.Task(id); page {
	case pageMove, pageCategory:
		m.pop.hl = 0
	case pageStatus: // the arrows start from the current choice
		m.pop.hl = slices.Index(data.Statuses, task.Status)
	case pagePriority:
		m.pop.hl = slices.Index(data.Priorities, task.Priority)
	}
}

// openMenuAtPointer is the right-click: the menu opens where the pointer is on the element.
func (m *Main) openMenuAtPointer(el ui.Element, id int) {
	px, py, _ := el.PointerPosition()
	x, y := el.LocalToWindow(px, py)
	m.openMenu(id, pageMain, x, y, false)
}

// openMenuBelow opens the menu under an element: the status glyph of an agent's task, or a value
// of the inspector, which shows only the piece it edits.
func (m *Main) openMenuBelow(el ui.Element, id int, page menuPage) {
	b := el.Bounds()
	m.openMenu(id, page, b.X, b.Y+b.H+4, false)
}

func (m *Main) closeMenu() { m.pop.open = false }

// popTick counts the frames; View calls it first.
func (m *Main) popTick() { m.pop.frame++ }

// taskMenuView draws the menu over the window while it is open. It is built last, so it is on top.
func (m *Main) taskMenuView(c *ui.Context, t *theme.Tokens) {
	pm := &m.pop
	if !pm.open {
		return
	}
	task, ok := m.app.Task(pm.task)
	if !ok {
		pm.open = false // the task went away, here or elsewhere
		return
	}
	ui.Overlay(c, func() {
		anchor := ui.Box(c.Key("pop-anchor")).Absolute().Left(pm.x).Top(m.menuTop(c)).Size(1, 1)
		panel := ui.Column(c.Key("pop-panel")).Width(popWidth).Padding(popPad).Radius(theme.RadiusPanel).
			Background(t.Bg).Border(1, t.Border).Role(ui.RoleDialog).Label(m.say.A11yTaskMenu()).
			Shadow(0, 2, 6, 0, t.Shadow.Alpha(0.25)).Shadow(0, 12, 36, 0, t.Shadow.Alpha(0.6))
		panel.AttachTo(anchor, ui.AnchorBottomLeft, ui.AnchorTopLeft)
		pm.items = pm.items[:0]
		pm.tip, pm.tipNext = pm.tipNext, popTip{}
		panel.Children(func() { m.pageBody(c, t, task) })
		if pm.tip != pm.tipNext {
			c.After(time.Millisecond) // the label above a strip names the choice from the frame before
		}
		m.menuKeys(c, panel)
		if panel.PressedOutside() && pm.frame != pm.at {
			m.closeMenu()
		}
	})
}

// menuTop is where the menu hangs. The pages that do not change their height let the panel flip
// above the point when there is no room below. The Move to and Category pages shrink as the filter
// narrows them, so they would flip back and forth while typing; they are placed once, by the room
// their whole list needs: below the point, else above it, else as low as the window allows.
func (m *Main) menuTop(c *ui.Context) float32 {
	pm := &m.pop
	var rows int
	switch pm.page {
	case pageMove:
		rows = len(m.app.snap.projects) + 1
	case pageCategory:
		rows = len(m.app.snap.categories) + 1
	default:
		return pm.y
	}
	_, window := c.Size()
	full := moveFullHeight(rows)
	if pm.page == pageCategory {
		full += menuRuleH + popRowH // the hairline and New category…
	}
	switch {
	case pm.y+full+popEdge <= window:
		return pm.y
	case pm.y-full-popEdge >= 0:
		return pm.y - full - 1
	}
	return max(popEdge, window-full-popEdge)
}

// pageBody builds the page the menu is on.
func (m *Main) pageBody(c *ui.Context, t *theme.Tokens, task data.Task) {
	if len(m.pop.many) > 0 {
		m.manyPage(c, t)
		return
	}
	switch m.pop.page {
	case pageStatus:
		m.stripLabel(c, t, "status", m.say.PropStatus(), statusLabel(m.say, task.Status))
		m.statusStrip(c, t, task, true)
	case pagePriority:
		m.stripLabel(c, t, "priority", m.say.PropPriority(), m.priorityText(task.Priority))
		m.priorityStrip(c, t, task, true)
	case pageCategory:
		m.categoryPage(c, t, task)
	case pageMove:
		m.movePage(c, t, task)
	case pageRemind:
		m.remindPage(c, t, task)
	case pageCalendar:
		m.calendarPage(c, t, task)
	case pageOpenIn:
		m.openInPage(c, t, task)
	default:
		m.mainPage(c, t, task)
	}
}

// menuKeys is the keyboard of the menu, whatever has the focus. Esc steps back to the whole menu
// from a piece it was opened from, then closes; the arrows walk the choices of the page and Enter
// takes the highlighted one. The letters of the whole menu jump to a piece, as the hints show.
func (m *Main) menuKeys(c *ui.Context, panel ui.Element) {
	pm := &m.pop
	if pm.esc || panel.OverlayShortcut(0, ui.KeyEscape) {
		pm.esc = false
		m.menuBack()
		return
	}
	if pm.enter {
		pm.enter = false
		m.chooseHighlighted()
	}
	n := len(pm.items)
	step := func(d int) {
		if n > 0 {
			pm.hl = max(0, min(pm.hl+d, n-1))
		}
	}
	if n > 0 && pm.hl >= n {
		pm.hl = n - 1 // the filter narrowed the list under it
	}
	switch {
	case panel.OverlayShortcut(0, ui.KeyDown), panel.OverlayShortcut(0, ui.KeyRight):
		step(1)
	case panel.OverlayShortcut(0, ui.KeyUp), panel.OverlayShortcut(0, ui.KeyLeft):
		step(-1)
	case panel.OverlayShortcut(0, ui.KeyEnter):
		m.chooseHighlighted()
	}
	switch {
	case pm.page == pageMain && len(pm.many) > 0:
		m.manyLetters(c, panel)
	case pm.page == pageMain:
		m.menuLetters(c, panel)
	}
}

// menuLetters are the keys of the whole menu.
func (m *Main) menuLetters(c *ui.Context, panel ui.Element) {
	pm := &m.pop
	switch {
	case panel.OverlayShortcut(0, ui.KeyS):
		pm.page, pm.back, pm.hl = pageStatus, true, -1
	case panel.OverlayShortcut(0, ui.KeyP):
		pm.page, pm.back, pm.hl = pagePriority, true, -1
	case panel.OverlayShortcut(0, ui.KeyM):
		pm.page, pm.back, pm.hl, pm.filter = pageMove, true, 0, ""
	case panel.OverlayShortcut(0, ui.KeyR):
		pm.page, pm.back, pm.hl = pageRemind, true, -1
	case panel.OverlayShortcut(ui.Cmd, ui.KeyJ):
		m.askAboutTask(pm.task)
		m.closeMenu()
	case panel.OverlayShortcut(ui.Cmd, ui.KeyBackspace):
		m.deleteTask(c, pm.task)
		m.closeMenu()
	}
}

func (m *Main) chooseHighlighted() {
	pm := &m.pop
	if pm.hl >= 0 && pm.hl < len(pm.items) {
		pm.items[pm.hl]()
	}
}

// menuBack goes from a piece back to the whole menu, or closes the menu.
func (m *Main) menuBack() {
	pm := &m.pop
	if pm.page == pageCalendar {
		pm.page, pm.hl = pageRemind, -1
		return
	}
	if pm.back && pm.page != pageMain {
		pm.page, pm.back, pm.hl = pageMain, false, -1
		return
	}
	m.closeMenu()
}

// stripLabel is the name above a strip and, after a dot, the name of the choice under the pointer
// or the arrows, or else of the current one: "Status · Doing". It is in the label row because a
// tooltip would cover the label.
func (m *Main) stripLabel(c *ui.Context, t *theme.Tokens, strip, text, current string) {
	name := current
	if m.pop.tip.strip == strip {
		name = m.pop.tip.name
	}
	ui.Row(c).Shrink(0).Padding(6, 8, 4).Gap(5).AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.Section, t.TextSecondary, text)
		theme.Text(c, theme.Meta, t.TextTertiary, "· "+name)
	})
}

// priorityText is "P1 High": the code and its name.
func (m *Main) priorityText(p data.Priority) string { return string(p) + " " + priorityName(m.say, p) }
