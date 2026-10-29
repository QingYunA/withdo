package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// readmeURL is where "How to connect" goes.
const readmeURL = "https://github.com/QingYunA/withdo#connect-claude-code"

// hairline is a one pixel rule across its container.
func hairline(c *ui.Context, t *theme.Tokens) ui.Element {
	return ui.Box(c).Height(theme.Hairline).Shrink(0).Background(t.Border)
}

// vline is a one pixel rule down its container.
func vline(c *ui.Context, t *theme.Tokens) ui.Element {
	return ui.Box(c).Width(theme.Hairline).Shrink(0).Background(t.Border)
}

// pill is a branch or pull request number in the mono face, on a quiet well.
func pill(c *ui.Context, t *theme.Tokens, text string) ui.Element {
	box := ui.Row(c).Padding(1, 6).Radius(theme.RadiusControl).Background(t.BgSubtle).Shrink(1).MinWidth(0)
	return box.Children(func() {
		theme.Text(c, theme.Mono, t.TextSecondary, text).Ellipsis("…")
	})
}

// link is accent text that does something when clicked.
func link(c *ui.Context, t *theme.Tokens, text string) ui.Element {
	l := theme.Text(c, theme.Meta, t.AccentInk, text).Cursor(ui.CursorPointer)
	if l.Hovered() {
		l.Underline()
	}
	return l
}

// glyphButton is the status glyph with a 24 pt hit target. Clicking the glyph of my own open task
// marks it done at once; the glyph of an agent's task, or of a finished one, opens the task menu
// under it, so agent work is never completed by accident and a finished task can be reopened.
// fill, when above 0, plays the completion animation.
func (m *Main) glyphButton(c *ui.Context, t *theme.Tokens, task data.Task, fill float32) ui.Element {
	own := ownOpen(task)
	name := statusLabel(m.say, task.Status)
	label, tip := m.say.A11yStatus(name), name
	if own {
		label, tip = m.say.A11yMarkDone(task.Title), m.say.ActionDone()
	}
	btn := ui.ButtonBase(c.Key("glyph")).Size(theme.HitTarget, theme.HitTarget).Shrink(0).Center().Radius(theme.RadiusControl).
		Label(label).Tooltip(tip).Cursor(ui.CursorPointer)
	if btn.Hovered() {
		btn.Background(t.BgHover)
	}
	btn.Children(func() { theme.StatusGlyph(c, t, task.Status, fill) })
	if btn.Clicked() {
		if own {
			m.app.SetStatus(task, data.Done)
		} else {
			m.openMenuBelow(btn, task.ID, pageMain)
		}
	}
	return btn
}

