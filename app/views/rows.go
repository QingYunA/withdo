package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// rowStyle is the padding of a row. Rows bleed their highlight past the column edge by the same
// amount, so the text stays on the column's left edge.
type rowStyle struct{ vert, horz float32 }

var (
	plainRow = rowStyle{9, 10}
	needRow  = rowStyle{10, 8} // inside the Needs you container
)

// rowShell is a row of the Now screen or a list: hover, the keyboard highlight, and the fade of a
// task that was just marked done. The caller fills it and reports clicks.
func (m *Main) rowShell(c *ui.Context, t *theme.Tokens, id int, st rowStyle, opacity float32) ui.Element {
	row := ui.Row(c.Key(id)).Shrink(0).Margin(0, -st.horz).Padding(st.vert, st.horz).Gap(12).
		Radius(theme.RadiusCard).AlignItems(ui.Stretch).Cursor(ui.CursorPointer)
	switch {
	case m.sel == id || m.openID == id:
		row.Background(t.BgSelected)
	case row.Hovered():
		row.Background(t.BgHover)
	}
	if m.sel == id && m.moved {
		row.ScrollIntoView()
		m.moved = false
	}
	if opacity < 1 {
		row.Opacity(opacity)
	}
	return row.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
}

// openRow opens a row's task in the inspector and highlights it.
func (m *Main) openRow(id int) {
	m.openID, m.sel = id, id
	m.inspector.reset()
}

// body is the column that grows between a row's glyph and its right end.
func body(c *ui.Context, fn func()) {
	ui.Column(c).Grow(1).MinWidth(0).Children(fn)
}

// Content widths below which a row drops its companions: first the project name, then the branch
// and pull request, so the title keeps its room beside the inspector.
const (
	dropProjectBelow float32 = 600
	dropPillsBelow   float32 = 460
)

// titleLine is a title with its quiet companions: a branch, a pull request, a project name. The
// Needs you rows leave the pills out: their summary line already says what is waiting.
func (m *Main) titleLine(c *ui.Context, t *theme.Tokens, task data.Task, project string, pills bool) {
	ui.Row(c).Gap(8).MinWidth(0).AlignItems(ui.Center).Children(func() {
		// In a tight row the project name gives way first, then the branch, and the title last.
		theme.Text(c, theme.BodyStrong, t.Text, task.Title).Shrink(1).MinWidth(0).Ellipsis("…")
		if task.Branch != "" && pills && m.width >= dropPillsBelow {
			pill(c, t, task.Branch).MaxWidth(180).Shrink(4)
		}
		if task.PRNumber > 0 && pills && m.width >= dropPillsBelow {
			pill(c, t, "#"+itoa(task.PRNumber)).Shrink(0)
		}
		if project != "" && m.width >= dropProjectBelow {
			theme.Text(c, theme.Meta, t.TextTertiary, project).Shrink(8).MinWidth(0).Ellipsis("…")
		}
	})
}

// summaryLine is the second line of a row.
func summaryLine(c *ui.Context, t *theme.Tokens, text string) {
	if text != "" {
		theme.Text(c, theme.Meta, t.TextSecondary, text).Margin(2, 0, 0).Ellipsis("…")
	}
}

// right is the cluster at a row's right end, centered against the row's height.
func right(c *ui.Context, fn func()) {
	ui.Row(c).Shrink(0).Gap(10).AlignItems(ui.Center).Children(fn)
}

// glyphSlot puts the status glyph button where the mockup draws the bare glyph: its 24 pt hit
// target reaches past the 16 pt glyph, so it takes negative margins.
func (a *App) glyphSlot(c *ui.Context, t *theme.Tokens, task data.Task, fill float32) {
	a.glyphButton(c, t, task, fill).Margin(-3, -4, -3, -4)
}

// ageLabel is how long ago, in the stale color when the agent went quiet.
func (m *Main) ageLabel(c *ui.Context, t *theme.Tokens, it data.NowItem) {
	color := t.TextTertiary
	if it.Stale {
		color = t.Stale
	}
	theme.Text(c, theme.Meta, color, m.clock.Age(it.Since, m.app.Now())).Shrink(0)
}

// liveDot is the 6 pt doing dot with a soft ring, drawn while an agent reported in the last ten
// minutes.
func liveDot(c *ui.Context, t *theme.Tokens) {
	ui.Box(c).Size(12, 12).Shrink(0).Radius(6).Background(t.Status(data.Doing).Alpha(0.18)).Center().Children(func() {
		theme.Dot(c, t.Status(data.Doing), 6)
	})
}

// actionButton is the small tinted button at the right of a Needs you row.
func actionButton(c *ui.Context, t *theme.Tokens, key int, text, label string) ui.Element {
	btn := ui.ButtonBase(c.Key("action-"+itoa(key))).Padding(3, 9).Radius(theme.RadiusControl).Shrink(0).
		Background(t.BgSelected).Label(label).Cursor(ui.CursorPointer)
	if btn.Hovered() {
		btn.Background(t.Accent.Alpha(0.2))
	}
	btn.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	btn.Children(func() { theme.Text(c, theme.MetaStrong, t.Accent, text) })
	return btn
}
