package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The Select button sits at the right of a page title. It turns on select mode, where each row's
// leading picture becomes a pixel checkbox and a click ticks the row instead of opening it; the
// title row then reads "2 selected · Select all · Done". The first time the user ⌘-clicks in a list
// the button flashes once, two 100 ms frames, so the user finds it.

// flashFrame is one frame of the Select button's flash: lit, then not.
const flashFrame = 100 * time.Millisecond

// selectHint remembers where the Select button has flashed: once a list, for this run of the app.
type selectHint struct {
	seen map[any]bool
	at   time.Time
}

// nudgeSelectButton flashes the Select button the first time a list is ⌘-clicked.
func (m *Main) nudgeSelectButton() {
	key := m.pageKey()
	if m.hint.seen[key] {
		return
	}
	next := map[any]bool{key: true}
	for k := range m.hint.seen {
		next[k] = true
	}
	m.hint.seen, m.hint.at = next, m.app.Now()
}

// flashLit says whether the button is in the lit frame of its flash, and asks for the next frame
// while the flash plays. With Reduce Motion it does not flash.
func (m *Main) flashLit(c *ui.Context) bool {
	if m.hint.at.IsZero() || c.Preferences().ReduceMotion {
		return false
	}
	age := m.app.Now().Sub(m.hint.at)
	if age >= 2*flashFrame {
		return false
	}
	c.After(hopFrameEvery)
	return age < flashFrame
}

// canSelect says whether the page lists tasks that can be selected: every list but the Calendar and
// Summary, and not an empty one.
func (m *Main) canSelect() bool {
	if len(m.order) == 0 {
		return false
	}
	if m.searching() {
		return true
	}
	return m.nav.screen != screenCalendar && m.nav.screen != screenSummary
}

// selectControls is the Select button, or in select mode the count, Select all and Done.
func (m *Main) selectControls(c *ui.Context, t *theme.Tokens) {
	if !m.canSelect() {
		return
	}
	if !m.multi.mode {
		m.selectButton(c, t)
		return
	}
	ui.Row(c.Key("select-bar")).Shrink(0).Gap(14).AlignItems(ui.Center).Margin(0, 8, 0, 0).Children(func() {
		theme.Text(c, theme.Meta, t.TextSecondary, m.say.SelectCount(m.multi.count())).Shrink(0)
		if link(c, t, m.say.SelectAll()).Clicked() {
			m.selectAll()
		}
		if actionButtonKeyed(c, t, "select-done", m.say.ActionDone(), m.say.ActionDone()).Clicked() {
			m.clearMulti()
		}
	})
}

// selectButton is the quiet button that turns select mode on.
func (m *Main) selectButton(c *ui.Context, t *theme.Tokens) {
	btn := ui.ButtonBase(c.Key("select")).Height(28).Padding(0, 8).Gap(6).Radius(theme.RadiusControl).
		Label(m.say.A11ySelectButton()).AlignItems(ui.Center).Shrink(0).Margin(0, 8, 0, 0).Cursor(ui.CursorPointer)
	ink := t.TextSecondary
	switch {
	case m.flashLit(c):
		btn.Background(t.BgSelected)
		ink = t.Text
	case btn.Hovered():
		btn.Background(t.BgHover)
		ink = t.Text
	}
	btn.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	btn.Children(func() {
		ui.Icon(c, theme.IconBox).Size(theme.IconSmall, theme.IconSmall).TextColor(ink)
		theme.Text(c, theme.Meta, ink, m.say.SelectButton())
	})
	if btn.Clicked() {
		m.multi.mode = true
	}
}

// checkbox is the pixel checkbox of select mode: empty, or ticked in the accent color.
func (m *Main) checkbox(c *ui.Context, t *theme.Tokens, id int, size float32) ui.Element {
	if m.multi.has(id) {
		return ui.Icon(c.Key("check"), theme.IconDone).Size(size, size).TextColor(t.Accent).Shrink(0)
	}
	return ui.Icon(c.Key("check"), theme.IconBox).Size(size, size).TextColor(t.TextTertiary).Shrink(0)
}

// lead is the picture at the start of a row: the author's avatar, or in select mode the checkbox in
// its place and of its size.
func (m *Main) lead(c *ui.Context, t *theme.Tokens, key string, id int, who theme.Who, size float32) ui.Element {
	if m.multi.mode {
		return m.checkbox(c, t, id, size)
	}
	return theme.AvatarIcon(c.Key(key), who, size)
}

// indentSlot is the empty room at the start of a row of a project's Mine column, which in select mode
// holds the checkbox, 4 pt wider than the room, so the title keeps its place.
func (m *Main) indentSlot(c *ui.Context, t *theme.Tokens, id int) {
	room := listIndent - rowGap
	if !m.multi.mode {
		ui.Box(c.Key("indent")).Width(room).Shrink(0)
		return
	}
	m.checkbox(c, t, id, theme.GlyphSize).Margin(0, room-theme.GlyphSize, 0, 0)
}

// leadOf is lead for a task: its own avatar.
func (m *Main) leadOf(c *ui.Context, t *theme.Tokens, task data.Task, size float32) ui.Element {
	return m.lead(c, t, "author", task.ID, m.app.taskWho(task), size)
}

// checkSlot is the checkbox where the status glyph button is: the same 24 pt target, with the
// checkbox in its middle. The row takes the click.
func (m *Main) checkSlot(c *ui.Context, t *theme.Tokens, id int) ui.Element {
	slot := ui.Box(c.Key("glyph")).Size(theme.HitTarget, theme.HitTarget).Shrink(0).Center()
	slot.Children(func() { m.checkbox(c, t, id, theme.GlyphSize) })
	return slot
}
