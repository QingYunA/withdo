package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The bulk bar is a pixel panel (the day card's shape) that floats at the bottom centre of the list
// while two rows are selected, or one in select mode: the count, Done, Move to…, Priority, Hand to
// an agent, Delete, and × that unselects. It is a toolbar: Tab reaches it, and ← → walk its buttons.
// (ui.Toolbar would hold room for an overflow button; a row with the toolbar role and a focus group
// is the same to the keyboard and to assistive technology, and its width is its buttons'.)
// In a narrow window the buttons drop their words and keep their icons.

const (
	// barNarrowBelow is the content width under which the buttons keep only their icons.
	barNarrowBelow float32 = 640
	// barTinyBelow is the content width under which the bar's count is the number alone: the five
	// icons, the count and × need about 260 pt.
	barTinyBelow float32 = 330
	// barBottom is the room under the bar, which holds its shadow.
	barBottom float32 = 22
	// barButton is the height of a button of the bar, barPad the room above and below them inside
	// its edge, and barHeight the whole bar.
	barButton float32 = 28
	barPad    float32 = 5
	barHeight         = barButton + 2*barPad
)

// bulkBar draws the bar over the bottom of the page.
func (m *Main) bulkBar(c *ui.Context, t *theme.Tokens) {
	if !m.barUp() {
		return
	}
	ids := m.selectedRows()
	narrow := m.width < barNarrowBelow
	ui.Row(c.Key("bulk-wrap")).Absolute().Left(0).Right(0).Bottom(barBottom).Justify(ui.Center).PassThrough().Children(func() {
		bar := ui.Row(c.Key("bulk-bar")).Role(ui.RoleToolbar).FocusGroup(ui.Horizontal).Label(m.say.A11yBulkBar())
		bar.Shrink(0).Padding(barPad, 8).Gap(2).AlignItems(ui.Center).Draw(func(p *ui.Painter, r ui.Rect) { theme.DrawPixelPanel(p, r, t) })
		bar.Children(func() {
			count := m.say.BulkCount(len(ids))
			if m.width < barTinyBelow { // beside the inspector in a small window only the number fits
				count = itoa(len(ids))
			}
			theme.Text(c, theme.MetaStrong, t.Text, count).Shrink(0).Margin(0, 6, 0, 8)
			if m.barButton(c, t, "done", theme.IconCheck, m.say.ActionDone(), "⌘↵", false, narrow).Clicked() {
				m.bulkSetStatus(c, ids, data.Done)
			}
			if move := m.barButton(c, t, "move", theme.IconMove, m.say.MenuMoveTo(), "", false, narrow); move.Clicked() {
				m.openManyMenuAbove(move, ids, pageMove)
			}
			if prio := m.barButton(c, t, "priority", theme.IconFlag, m.say.PropPriority(), "", false, narrow); prio.Clicked() {
				m.openManyMenuAbove(prio, ids, pagePriority)
			}
			if m.barButton(c, t, "hand", theme.IconAgent, m.say.BulkHandOff(), "⌘⇧A", false, narrow).Clicked() {
				m.handOffMany(c, ids)
			}
			if m.barButton(c, t, "delete", theme.IconTrash, m.say.MenuDelete(), "⌘⌫", true, narrow).Clicked() {
				m.askBulkDelete(ids)
			}
			if m.barClear(c, t).Clicked() {
				m.multi = m.multi.with()
			}
		})
	})
}

// barButton is a button of the bar: an icon and, unless the bar is narrow, its word. The tooltip
// says it with its key.
func (m *Main) barButton(c *ui.Context, t *theme.Tokens, key string, icon *ui.SVG, label, hint string, danger, narrow bool) ui.Element {
	tip := label
	if hint != "" {
		tip += "  " + hint
	}
	btn := ui.ButtonBase(c.Key("bulk-"+key)).Height(barButton).Padding(0, 8).Gap(6).Radius(theme.RadiusControl).
		AlignItems(ui.Center).Shrink(0).Label(label).Tooltip(tip).Cursor(ui.CursorPointer)
	ink := t.TextSecondary
	switch {
	case danger:
		ink = t.Blocked()
		if btn.Hovered() {
			btn.Background(ink.Alpha(0.12))
		}
	case btn.Hovered():
		btn.Background(t.BgHover)
		ink = t.Text
	}
	btn.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	btn.Children(func() {
		if narrow {
			ui.Icon(c, icon).Size(theme.IconRow, theme.IconRow).TextColor(ink)
			return
		}
		ui.Icon(c, icon).Size(theme.IconSmall, theme.IconSmall).TextColor(ink)
		theme.Text(c, theme.Meta, ink, label)
	})
	return btn
}

// barClear is the × at the right of the bar.
func (m *Main) barClear(c *ui.Context, t *theme.Tokens) ui.Element {
	btn := ui.ButtonBase(c.Key("bulk-clear")).Size(barButton, barButton).Center().Radius(theme.RadiusControl).Shrink(0).
		Label(m.say.BulkClear()).Tooltip(m.say.BulkClear()).Cursor(ui.CursorPointer)
	ink := t.TextTertiary
	if btn.Hovered() {
		btn.Background(t.BgHover)
		ink = t.TextSecondary
	}
	btn.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	btn.Children(func() { ui.Icon(c, theme.IconClose).Size(theme.IconSmall, theme.IconSmall).TextColor(ink) })
	return btn
}

// barUp says whether the bar is drawn: it is wanted, the page can select, and there is something
// selected on it.
func (m *Main) barUp() bool {
	return m.barShown() && m.canSelect() && len(m.selectedRows()) > 0
}

// barInset is the room the bar takes at the bottom of the page while it is up: its height and its
// margin. Every list it floats over keeps that much under its last row, so the row can be scrolled
// clear of it.
func (m *Main) barInset() float32 {
	if m.barUp() {
		return barHeight + barBottom
	}
	return 0
}

// pageBottom is the room under a page's last row, and the bar's inset while it is up.
func (m *Main) pageBottom() float32 { return 48 + m.barInset() }
