package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/locales"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// readmeURL is where "How to connect" goes.
const readmeURL = "https://github.com/QingYunA/adhd-todo#connect-claude-code"

// hairline is a one pixel rule across its container.
func hairline(c *ui.Context, t *theme.Tokens) ui.Element {
	return ui.Box(c).Height(1).Shrink(0).Background(t.Border)
}

// vline is a one pixel rule down its container.
func vline(c *ui.Context, t *theme.Tokens) ui.Element {
	return ui.Box(c).Width(1).Shrink(0).Background(t.Border)
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
	l := theme.Text(c, theme.Meta, t.Accent, text).Cursor(ui.CursorPointer)
	if l.Hovered() {
		l.Underline()
	}
	return l
}

// glyphButton is the status glyph with a 24 pt hit target; clicking it opens the status menu.
// fill, when above 0, plays the completion animation.
func (a *App) glyphButton(c *ui.Context, t *theme.Tokens, task data.Task, fill float32) ui.Element {
	say := locales.In(c.Locale())
	name := statusLabel(say, task.Status)
	btn := ui.ButtonBase(c.Key("glyph")).Size(theme.HitTarget, theme.HitTarget).Shrink(0).Center().Radius(theme.RadiusControl).
		Label(say.A11yStatus(name)).Tooltip(name)
	btn.Menu(func(m *ui.Menu) {
		for _, s := range data.Statuses {
			if m.Item(statusLabel(say, s)).Checked(task.Status == s).Chosen() {
				a.SetStatus(task, s)
			}
		}
	})
	if btn.Hovered() {
		btn.Background(t.BgHover)
	}
	btn.Children(func() { theme.StatusGlyph(c, t, task.Status, fill) })
	return btn
}

// segmented is a quiet well of labels with the chosen one raised, like the List | Board switch. It
// returns the index clicked this frame, or -1.
func segmented(c *ui.Context, t *theme.Tokens, key string, labels []string, selected int) int {
	clicked := -1
	ui.Row(c.Key(key)).Shrink(0).Padding(2).Radius(7).Background(t.BgSubtle).Children(func() {
		for i, name := range labels {
			on := i == selected
			seg := ui.ButtonBase(c.Key(key+"-"+itoa(i))).Padding(3, 10).Radius(5).Label(name).Cursor(ui.CursorPointer)
			switch {
			case on:
				seg.Background(t.Bg).Shadow(0, 1, 2, 0, t.Shadow.Alpha(0.35))
			case seg.Hovered():
				seg.Background(t.BgHover)
			}
			ink := t.TextSecondary
			if on {
				ink = t.Text
			}
			seg.Children(func() { theme.Text(c, theme.Meta, ink, name) })
			if seg.Clicked() && !on {
				clicked = i
			}
		}
	})
	return clicked
}

// radio is a 16 pt circle, filled when on.
func radio(c *ui.Context, t *theme.Tokens, on bool) ui.Element {
	return ui.Box(c).Size(16, 16).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		ring := ui.Rect{X: r.X + 1, Y: r.Y + 1, W: r.W - 2, H: r.H - 2}
		if !on {
			p.Stroke(ring, t.TextTertiary, ring.W/2, 1.5)
			return
		}
		p.Stroke(ring, t.Accent, ring.W/2, 1.5)
		p.Fill(ui.Rect{X: r.X + 4.5, Y: r.Y + 4.5, W: r.W - 9, H: r.H - 9}, t.Accent, (r.W-9)/2)
	})
}
