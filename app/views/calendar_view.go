package views

import (
	"math"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// calendarScreen is the Calendar page: the month as the title with ‹ Today › at its right, and
// the month grid, which fills the room that is left.
func (m *Main) calendarScreen(c *ui.Context, t *theme.Tokens) {
	m.ensureCalendar()
	days := m.calendarDays()
	m.pageHeader(c, t, m.clock.MonthTitle(m.cal.month), "", func() { m.calendarNav(c, t) })
	ui.Box(c).Height(18).Shrink(0)
	m.monthGrid(c, t, days)
}

// calendarNav is the quiet control at the right of the title: ‹ Today ›, in a well like the list and
// board toggle, with Today raised.
func (m *Main) calendarNav(c *ui.Context, t *theme.Tokens) {
	ui.Row(c.Key("calendar-nav")).Shrink(0).Padding(2).Gap(2).Radius(8).Background(t.BgSubtle).Children(func() {
		m.navButton(c, t, "prev", theme.IconBack, m.say.CalendarPrevious(), func() { m.shiftMonth(-1) })
		today := ui.ButtonBase(c.Key("calendar-today")).Height(26).Padding(0, 12).Center().Radius(6).
			Label(m.say.A11yCalendarToday()).Cursor(ui.CursorPointer).Background(t.Bg).Shadow(0, 1, 2, 0, t.Shadow.Alpha(0.35))
		today.Children(func() { theme.Text(c, theme.Meta, t.Text, m.say.CalendarToday()) })
		if today.Clicked() {
			m.calendarToday()
		}
		m.navButton(c, t, "next", theme.IconForward, m.say.CalendarNext(), func() { m.shiftMonth(1) })
	})
}

func (m *Main) navButton(c *ui.Context, t *theme.Tokens, key string, icon *ui.SVG, label string, do func()) {
	seg := ui.ButtonBase(c.Key("calendar-"+key)).Size(26, 26).Center().Radius(6).Label(label).Tooltip(label).Cursor(ui.CursorPointer)
	ink := t.TextTertiary
	if seg.Hovered() {
		ink = t.TextSecondary
	}
	seg.Children(func() { ui.Icon(c, icon).Size(theme.IconRow, theme.IconRow).TextColor(ink) })
	if seg.Clicked() {
		do()
	}
}

// monthGrid is the weekday names and the weeks, in hairline cells.
func (m *Main) monthGrid(c *ui.Context, t *theme.Tokens, days []data.CalDay) {
	weeks := len(days) / 7
	cellH := m.calendarCellHeight(c, weeks)
	ui.Grid(c.Key("calendar-weekdays")).Columns(7).Gap(1).Padding(0, 1).Height(theme.CalendarHeadLine).Shrink(0).Children(func() {
		for _, d := range days[:7] {
			ui.Row(c).AlignItems(ui.Center).Padding(0, theme.CalendarPadX+2).Children(func() {
				theme.Text(c, theme.Meta, t.TextTertiary, m.clock.Weekday(d.Date))
			})
		}
	})
	ui.Grid(c.Key("calendar-grid")).Columns(7).Gap(1).Padding(1).Background(t.Border).Radius(theme.RadiusCard).Clip().Shrink(0).Children(func() {
		for _, d := range days {
			m.dayCell(c, t, d, cellH)
		}
	})
}

// calendarCellHeight lets the grid fill the window's height: what the page puts above the grid is
// taken off, and the rest shared by the weeks. A cell is never shorter than CalendarMinCell; the
// page scrolls instead.
func (m *Main) calendarCellHeight(c *ui.Context, weeks int) float32 {
	_, h := c.Size()
	above := dragStrip + 36 + 18 + theme.CalendarHeadLine + 2 + 48
	return max(theme.CalendarMinCell, float32(math.Floor(float64((h-above)/float32(weeks)-1))))
}

// dayCell is one day: its number, and its items. A click selects the day, a double click opens its
// card, an item dropped on it moves to it. "N more" and the + of a hovered cell open the card too.
func (m *Main) dayCell(c *ui.Context, t *theme.Tokens, d data.CalDay, cellH float32) {
	selected := d.Date.Equal(m.calendarDay())
	cell := ui.Column(c.Key(d.Date.Format("2006-01-02"))).Height(cellH).
		Gap(theme.CalendarChipGap).ClipY().Cursor(ui.CursorPointer).Label(m.say.A11yCalendarDay(m.clock.CalendarDay(d.Date)))
	_, over := ui.DragOver[calDrag](cell)
	bg := t.Bg
	switch {
	case over:
		bg = t.CalendarDropFill()
	case d.Weekend:
		bg = t.CalendarWeekend()
	}
	if cell.Hovered() && !over {
		bg = bg.Mix(t.Text, 0.025)
	}
	ring := ui.Transparent
	if selected && !m.cal.card.isOpen() { // the open day card sits on the cell: its ring would show at the card's corner
		ring = t.CalendarRingColor()
	}
	cell.Background(bg).Border(theme.CalendarRing, ring).Padding(theme.CalendarPadTop-theme.CalendarRing, theme.CalendarPadX-theme.CalendarRing, 0)
	if dropped, ok := ui.Drop[calDrag](cell); ok {
		m.moveCalItem(dropped, d.Date)
	}
	cell.Children(func() {
		m.dayNumber(c, t, d)
		shown, more := visibleItems(len(d.Items), calendarLines(cellH))
		for _, it := range d.Items[:shown] {
			m.calendarItem(c, t, it)
		}
		if more > 0 {
			m.moreLine(c, t, cell, d, more)
		}
		if cell.Hovered() {
			m.addButton(c, t, cell, d)
		}
	})
	if cell.Clicked() {
		m.selectDay(d.Date)
	}
	if cell.DoubleClicked() {
		m.openDayCard(cell, d.Date, true)
	}
}

// moreLine is "N more" under a cell's items, a link that opens the day's card.
func (m *Main) moreLine(c *ui.Context, t *theme.Tokens, cell ui.Element, d data.CalDay, more int) {
	label := m.say.A11yCalendarShowAll(len(d.Items), m.clock.MonthDay(d.Date))
	line := ui.ButtonBase(c.Key("more")).Padding(0, 4).Shrink(0).Label(label).Cursor(ui.CursorPointer)
	ink := t.TextTertiary
	if line.Hovered() {
		ink = t.TextSecondary
	}
	line.Children(func() {
		text := theme.Text(c, theme.Meta, ink, m.say.CalendarMore(more))
		if line.Hovered() {
			text.Underline()
		}
	})
	if line.Clicked() {
		m.openDayCard(cell, d.Date, false)
	}
}

// addButton is the + at the right of the day number of a hovered cell. It opens the day's card with
// the input focused. It floats over the cell, so the layout does not move.
func (m *Main) addButton(c *ui.Context, t *theme.Tokens, cell ui.Element, d data.CalDay) {
	label := m.say.A11yCalendarAddTo(m.clock.MonthDay(d.Date))
	btn := ui.ButtonBase(c.Key("add")).Absolute().Right(theme.CalendarPadX-2).Top(theme.CalendarPadTop-theme.CalendarRing).
		Size(theme.DayCardPlus, theme.DayCardPlus).Center().Radius(theme.RadiusControl).Label(label).Tooltip(label).Cursor(ui.CursorPointer)
	ink := t.TextTertiary
	if btn.Hovered() {
		btn.Background(t.BgHover)
		ink = t.TextSecondary
	}
	btn.Children(func() { ui.Icon(c, theme.IconPlus).Size(theme.IconSmall, theme.IconSmall).TextColor(ink) })
	if btn.Clicked() {
		m.openDayCard(cell, d.Date, true)
	}
}

// calendarLines is how many items fit under the number of a cell of cellH.
func calendarLines(cellH float32) int {
	room := cellH - theme.CalendarPadTop - theme.CalendarDayLine - 4
	return max(0, int((room+theme.CalendarChipGap)/(theme.CalendarChip+theme.CalendarChipGap)))
}

// dayNumber is the day's number: quiet, grey in another month, in an accent pill for today. The
// first of a month says the month too.
func (m *Main) dayNumber(c *ui.Context, t *theme.Tokens, d data.CalDay) {
	label := itoa(d.Date.Day())
	if d.Date.Day() == 1 {
		label = m.clock.MonthDay(d.Date)
	}
	row := ui.Row(c.Key("number")).Height(20).Shrink(0).Radius(10).AlignItems(ui.Center).AlignSelf(ui.Start).Margin(0, 0, theme.CalendarDayLine-20)
	switch {
	case d.Today:
		row.Background(t.Accent).MinWidth(20).Padding(0, 6).Center().Children(func() {
			theme.Text(c, theme.MetaStrong, t.AccentText, label)
		})
	case d.InMonth:
		row.Padding(0, 4).Children(func() { theme.Text(c, theme.Meta, t.TextSecondary, label) })
	default:
		row.Padding(0, 4).Children(func() { theme.Text(c, theme.Meta, t.TextTertiary, label) })
	}
}

// calendarItem is one thing on a day: a milestone as an accent chip, a task as its author's
// avatar, the time of a reminder, and its title.
func (m *Main) calendarItem(c *ui.Context, t *theme.Tokens, it data.CalItem) {
	m.calendarItemIn(c, t, it, m.calendarCellWidth(c)-2*theme.CalendarPadX, false)
}

// calendarItemIn is calendarItem in a row width wide. In the day card the row is taller, does not
// drag, and a click closes the card as it opens the task.
func (m *Main) calendarItemIn(c *ui.Context, t *theme.Tokens, it data.CalItem, width float32, card bool) {
	height := theme.CalendarChip
	if card {
		height = theme.DayCardRow
	}
	if it.Kind == data.CalMilestone {
		ui.Row(c.Key("milestone-"+itoa(it.Milestone.ID))).Height(height).Shrink(0).Padding(0, 6).Gap(5).Radius(5).
			Background(t.BgSelected).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, theme.IconFlag).Size(theme.IconSmall, theme.IconSmall).TextColor(t.Accent).Shrink(0)
			theme.Text(c, theme.MetaStrong, t.Accent, it.Milestone.Title).Shrink(1).MinWidth(0).Ellipsis("…")
		})
		return
	}
	id := it.Task.ID
	row := ui.Row(c.Key("item-"+itoa(id)+"-"+itoa(int(it.Kind)))).Height(height).Shrink(0).Padding(0, calendarChipPad).Gap(5).
		Radius(5).AlignItems(ui.Center).Cursor(ui.CursorPointer)
	switch {
	case m.openID == id:
		row.Background(t.BgSelected)
	case row.Hovered():
		row.Background(t.BgHover)
	}
	if it.Kind != data.CalDone && !card {
		row.Drag(calDrag{id: id, kind: it.Kind})
		if row.Dragging() {
			row.Opacity(0.4)
		}
	}
	row.Children(func() {
		theme.AvatarIcon(c, m.calAvatar(it), theme.AvatarSmall)
		if it.Kind == data.CalReminder {
			m.reminderTime(c, t, it, width)
		}
		title := theme.Text(c, theme.Meta, m.calInk(t, it), it.Task.Title).Shrink(1).MinWidth(0).Ellipsis("…")
		if it.Kind == data.CalDone {
			title.Strikethrough()
		}
	})
	if row.RightClicked() {
		m.openMenuAtPointer(row, id)
	}
	if row.Clicked() {
		m.openRow(id)
		if card {
			m.cal.card.close()
		}
	}
}

// calendarChipPad is the padding at the sides of an item.
const calendarChipPad float32 = 4

// minTitle is the least room a title keeps beside the time of a reminder: below it the time
// gives way and the title has the cell to itself.
const minTitle float32 = 36

// reminderTime is the short time of a reminder before its title ("2pm", "14:00"), in
// text.tertiary, when the row, width wide, leaves the title its room beside it.
func (m *Main) reminderTime(c *ui.Context, t *theme.Tokens, it data.CalItem, width float32) {
	at := m.clock.ShortTime(it.At)
	w, _ := c.MeasureText(0, ui.Span{Text: at, Size: theme.Meta.Size, Weight: theme.Meta.Weight})
	room := width - 2*calendarChipPad - theme.AvatarSmall - 5
	if room-w-5 >= minTitle {
		theme.Text(c, theme.Meta, t.TextTertiary, at).Shrink(0)
	}
}

// calendarCellWidth is how wide a cell of the grid is this frame.
func (m *Main) calendarCellWidth(c *ui.Context) float32 {
	w := min(m.contentWidth(c)-2*m.sidePad(c), theme.CalendarMax)
	return (w - 2 - 6) / 7
}
