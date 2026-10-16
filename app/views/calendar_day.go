package views

import (
	"math"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The day card. "N more" in a cell, a double click on the cell and the + of a hovered cell open a
// card over the grid with every item of that day and an input that adds a task due that day. The
// card's top left sits on the cell's, so it covers the cell and grows outward; near the right or
// bottom edge of the window it grows left or up instead. It opens as a pop-out
// (theme.DayCardAt): laid out at its full size from the first frame, then scaled up from the cell's
// corner while it fades in, drawn from the clock the app already has and asking for a frame every
// 16 ms only while it opens. Esc, a click outside it, its close button and a change of month close it.

// dayCard is the state of the card: the day it shows (zero while closed), when it opened, the cell
// it grew from, what is typed in its input and whether the input takes the keyboard on opening.
type dayCard struct {
	day    time.Time
	at     time.Time
	anchor ui.Rect
	focus  bool
	buf    string
	asked  int // frames asked for while it opens, which the tests look at
}

// isOpen reports whether the card shows.
func (d *dayCard) isOpen() bool { return !d.day.IsZero() }

// close closes the card and forgets what was typed in it.
func (d *dayCard) close() { d.day, d.buf = time.Time{}, "" }

// openDayCard opens the card on a day, grown from the cell it was asked on, and selects the day.
func (m *Main) openDayCard(cell ui.Element, day time.Time, focus bool) {
	m.selectDay(day)
	m.cal.card = dayCard{day: m.calendarDay(), at: m.app.Now(), anchor: cell.Bounds(), focus: focus}
}

// dayCardBox is where the card sits at full size and which way it grows.
type dayCardBox struct {
	x, y, w, h float32
	left, up   bool // it grows left or up from the cell, because there is no room right or below
}

// snap4 rounds a length to the 4 pt grid.
func snap4(v float32) float32 { return float32(math.Round(float64(v)/4) * 4) }

// dayCardListHeight is the room the list of n items takes: its rows, and the scrolling beyond what
// the card's greatest height leaves.
func dayCardListHeight(n int) float32 {
	if n == 0 {
		return 0
	}
	return min(float32(n)*theme.DayCardRow, theme.DayCardMaxH-dayCardChrome)
}

// dayCardChrome is the card's height without its list: edge, padding, header, input and the gap
// between each.
const dayCardChrome = 2*theme.PixelUnit + 2*theme.DayCardPad + theme.DayCardHead + theme.DayCardInput + 2*theme.DayCardGap

// dayCardPlace is the box of the card for n items, grown from the anchor, inside a window of
// winW by winH whose cells are cellW wide.
func dayCardPlace(anchor ui.Rect, winW, winH, cellW float32, n int) dayCardBox {
	b := dayCardBox{w: snap4(max(cellW*1.6, theme.DayCardMinWidth))}
	b.w = min(b.w, float32(math.Floor(float64((winW-2*theme.DayCardMargin)/4)))*4)
	b.h = dayCardChrome + dayCardListHeight(n)
	if n == 0 {
		b.h -= theme.DayCardGap // no list: one gap fewer
	}
	b.x, b.y = anchor.X, anchor.Y
	if b.x+b.w > winW-theme.DayCardMargin {
		b.x, b.left = anchor.X+anchor.W-b.w, true
	}
	if b.y+b.h > winH-theme.DayCardMargin {
		b.y, b.up = anchor.Y+anchor.H-b.h, true
	}
	b.x, b.y = max(b.x, theme.DayCardMargin), max(b.y, theme.DayCardMargin)
	return b
}

// dayCardDay is the day the card shows, as the grid has it.
func (m *Main) dayCardDay() (data.CalDay, bool) {
	for _, d := range m.calendarDays() {
		if d.Date.Equal(m.cal.card.day) {
			return d, true
		}
	}
	return data.CalDay{}, false
}

// dayCardView draws the card above the window while it is open: over a backdrop that takes the
// presses outside it, a pixel panel (theme.DrawPixelPanel), and popped out of its cell unless Reduce
// Motion is on.
func (m *Main) dayCardView(c *ui.Context, t *theme.Tokens) {
	dc := &m.cal.card
	if !dc.isOpen() {
		return
	}
	day, ok := m.dayCardDay()
	if !ok || m.nav.screen != screenCalendar || m.searching() {
		dc.close()
		return
	}
	winW, winH := c.Size()
	g := dayCardPlace(dc.anchor, winW, winH, m.calendarCellWidth(c), len(day.Items))
	progress, opacity, done := theme.DayCardAt(m.app.Now().Sub(dc.at))
	if c.Preferences().ReduceMotion {
		progress, opacity, done = 1, 1, true
	}
	if !done {
		dc.asked++
		c.After(hopFrameEvery)
	}
	// It starts about the size of the cell it grows from, never under half or over 95 % of its own.
	from := min(max(dc.anchor.W/g.w, 0.5), 0.95)
	scale := from + (1-from)*progress
	var ox, oy float32
	if g.left {
		ox = 1
	}
	if g.up {
		oy = 1
	}
	ui.Overlay(c, func() {
		back := ui.Box(c.Key("day-card-back")).Absolute().Left(0).Top(0).Right(0).Bottom(0).Modal()
		if back.Clicked() || back.OverlayShortcut(0, ui.KeyEscape) {
			dc.close()
			return
		}
		back.Children(func() {
			panel := ui.Box(c.Key("day-card")).Absolute().Left(g.x).Top(g.y).Size(g.w, g.h).Shrink(0).
				TransformOrigin(ox, oy).Scale(scale, scale).Opacity(opacity).
				Draw(func(p *ui.Painter, r ui.Rect) { theme.DrawPixelPanel(p, r, t) }).Role(ui.RoleDialog).
				Label(m.say.A11yCalendarDayCard(m.clock.CalendarDay(day.Date)))
			panel.Clicked() // presses on it stay in it
			panel.Children(func() { m.dayCardBody(c, t, day, g) })
		})
	})
}

// dayCardBody is what the card holds, inside its edge: the header, the day's items, and the input.
func (m *Main) dayCardBody(c *ui.Context, t *theme.Tokens, day data.CalDay, g dayCardBox) {
	dc := &m.cal.card
	edge := theme.PixelUnit
	ui.Column(c.Key("day-card-body")).Absolute().Left(edge).Top(edge).Size(g.w-2*edge, g.h-2*edge).Clip().Padding(theme.DayCardPad).Children(func() {
		m.dayCardHeader(c, t, day)
		if len(day.Items) > 0 {
			ui.Box(c).Height(theme.DayCardGap).Shrink(0)
			rowW := g.w - 2*edge - 2*theme.DayCardPad
			ui.Scroll(c.Key("day-card-list")).Height(dayCardListHeight(len(day.Items))).Shrink(0).Children(func() {
				ui.Column(c).FillWidth().Children(func() {
					for _, it := range day.Items {
						m.calendarItemIn(c, t, it, rowW, true)
					}
				})
			})
		}
		ui.Box(c).Height(theme.DayCardGap).Shrink(0)
		wellInput(c, t, "day-card-input", &dc.buf, m.say.CalendarDayInput(), m.say.CalendarDayPlaceholder(), theme.Body, dc.focus, m.dayCardKey).
			Radius(0).Background(ui.Transparent).Draw(func(p *ui.Painter, r ui.Rect) { theme.DrawPixelWell(p, r, t.BgSubtle) })
	})
}

// dayCardHeader is the date, how many items the day has in text.tertiary, and the close button.
func (m *Main) dayCardHeader(c *ui.Context, t *theme.Tokens, day data.CalDay) {
	ui.Row(c.Key("day-card-head")).Height(theme.DayCardHead).Shrink(0).AlignItems(ui.Center).Gap(8).Children(func() {
		theme.Text(c, theme.BodyStrong, t.Text, m.clock.DayTitle(day.Date)).Shrink(1).MinWidth(0).Ellipsis("…")
		theme.Text(c, theme.Meta, t.TextTertiary, m.say.CalendarDayCount(len(day.Items))).Shrink(0)
		ui.Box(c).Grow(1)
		label := m.say.A11yCalendarDayClose()
		btn := ui.ButtonBase(c.Key("day-card-close")).Size(24, 24).Shrink(0).Center().Radius(theme.RadiusControl).
			Label(label).Tooltip(label).Cursor(ui.CursorPointer)
		ink := t.TextTertiary
		if btn.Hovered() {
			btn.Background(t.BgHover)
			ink = t.TextSecondary
		}
		btn.Children(func() { ui.Icon(c, theme.IconClose).Size(theme.IconSmall, theme.IconSmall).TextColor(ink) })
		if btn.Clicked() {
			m.cal.card.close()
		}
	})
}

// dayCardKey is the keyboard of the card's input: Enter adds what is typed, Esc closes the card.
func (m *Main) dayCardKey(k ui.TextKey) bool {
	if k.Mods != 0 {
		return false
	}
	switch k.Key {
	case ui.KeyEnter:
		if m.addToDay(m.cal.card.day, k.Text()) {
			k.SetText("")
		}
	case ui.KeyEscape:
		m.cal.card.close()
	default:
		return false
	}
	return true
}

// addToDay adds a plain task due at the end of the day. It reports whether there was one to add:
// empty text adds nothing.
func (m *Main) addToDay(day time.Time, text string) bool {
	title := strings.Join(strings.Fields(text), " ")
	if title == "" {
		return false
	}
	due := data.EndOfDay(day)
	return m.app.AddTask(data.CreateTaskInput{Title: title, Source: data.SourceUser, DueAt: &due}) == nil
}
