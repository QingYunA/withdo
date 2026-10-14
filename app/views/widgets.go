package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
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

// hint is one footer key hint. Every hint has the same tertiary style whether or not its action
// is available right now; the key glyph alone is a step stronger than its label.
func hint(c *ui.Context, t *theme.Tokens, key, action string) {
	ui.Row(c).Gap(5).Children(func() {
		theme.Text(c, theme.Meta, t.TextTertiary.Mix(t.TextSecondary, 0.5), key)
		theme.Text(c, theme.Meta, t.TextTertiary, action)
	})
}

// sectionHeader is an uppercase title with an optional count on the right.
func sectionHeader(c *ui.Context, t *theme.Tokens, title, right string) {
	ui.Row(c).Padding(14, 16, 4).Shrink(0).Children(func() {
		theme.Text(c, theme.Section, t.TextSecondary, title)
		ui.Spacer(c)
		if right != "" {
			theme.Text(c, theme.Meta, t.TextTertiary, right)
		}
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
func (a *App) glyphButton(c *ui.Context, t *theme.Tokens, task data.Task, fill float32) {
	btn := ui.ButtonBase(c.Key("glyph")).Size(theme.HitTarget, theme.HitTarget).Shrink(0).Center().Radius(theme.RadiusControl).
		Label("Status: " + task.Status.Label()).Tooltip(task.Status.Label())
	btn.Menu(func(m *ui.Menu) {
		for _, s := range data.Statuses {
			if m.Item(s.Label()).Checked(task.Status == s).Chosen() {
				a.SetStatus(task, s)
			}
		}
	})
	if btn.Hovered() {
		btn.Background(t.BgHover)
	}
	btn.Children(func() { theme.StatusGlyph(c, t, task.Status, fill) })
}

// timeWords describes when a task is due or reminds, "in 1h" or "tomorrow", and says whether the
// moment is a reminder.
func timeWords(task data.Task, now time.Time) (text string, reminder bool, ok bool) {
	switch {
	case task.RemindAt != nil && task.RemindedAt == nil:
		return data.When(*task.RemindAt, now), true, true
	case task.DueAt != nil:
		return data.When(*task.DueAt, now), false, true
	}
	return "", false, false
}

// nextMoment is the time a task asks for attention, for ordering: reminder or due, whichever
// comes first; tasks with neither come last.
func nextMoment(task data.Task) time.Time {
	var best time.Time
	for _, t := range []*time.Time{task.RemindAt, task.DueAt} {
		if t != nil && (best.IsZero() || t.Before(best)) {
			best = *t
		}
	}
	return best
}
