package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// rowShell is a panel row: inset 6 from the edges, 8 pt radius, highlighted by selection and a
// quiet hover. It reports clicks on itself; the glyph button inside takes its own.
func (p *Panel) rowShell(c *ui.Context, t *theme.Tokens, key int, h float32, selected bool, opacity float32) ui.Element {
	row := ui.Row(c.Key(key)).Height(h).Shrink(0).Margin(0, theme.RowInset).Padding(0, 10, 0, 6).Radius(theme.RadiusCard).
		AlignItems(ui.Center).Gap(4).Cursor(ui.CursorPointer)
	switch {
	case selected:
		row.Background(t.BgSelected)
	case row.Hovered():
		row.Background(t.BgHover)
	}
	if selected && p.moved {
		row.ScrollIntoView()
		p.moved = false
	}
	if opacity < 1 {
		row.Opacity(opacity)
	}
	row.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	return row
}

func (p *Panel) agentRow(c *ui.Context, t *theme.Tokens, r agentRow, selected bool) {
	fill, opacity, _ := p.app.completionPhase(c, r.ID)
	row := p.rowShell(c, t, r.ID, theme.RowTwoLine, selected, opacity).AlignItems(ui.Start)
	row.Children(func() {
		p.app.glyphButton(c, t, r.Task, fill)
		ui.Column(c).Grow(1).MinWidth(0).Padding(5, 0, 5, 0).Children(func() {
			p.agentTitleLine(c, t, r)
			if r.LatestReport != nil {
				theme.Text(c, theme.Meta, t.TextSecondary, r.LatestReport.Summary).Ellipsis("…")
			}
		})
	})
	if row.Clicked() {
		p.open(r.ID)
	}
}

func (p *Panel) agentTitleLine(c *ui.Context, t *theme.Tokens, r agentRow) {
	ui.Row(c).Gap(8).MinWidth(0).AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.BodyStrong, t.Text, r.Title).Shrink(1).MinWidth(0).Ellipsis("…")
		if r.Branch != "" {
			pill(c, t, r.Branch).MaxWidth(150)
		}
		ui.Spacer(c)
		p.ageLabel(c, t, r)
	})
}

// ageLabel is the age of the latest report, or the stale label in the one warm color.
func (p *Panel) ageLabel(c *ui.Context, t *theme.Tokens, r agentRow) {
	if r.Stale {
		theme.Text(c, theme.Meta, t.Stale, "! "+staleWords(r.HoursSinceActivity)).Shrink(0)
		return
	}
	since := r.UpdatedAt
	if r.LatestReport != nil {
		since = r.LatestReport.CreatedAt
	}
	theme.Text(c, theme.Meta, t.TextTertiary, data.Age(since, p.app.Now())).Shrink(0)
}

// staleWords reads "3d stale" or "54h stale" from the hours since the last sign of life.
func staleWords(hours int) string {
	if hours >= 48 {
		return itoa(hours/24) + "d stale"
	}
	return itoa(hours) + "h stale"
}

func (p *Panel) mineRow(c *ui.Context, t *theme.Tokens, task data.Task, selected bool) {
	fill, opacity, _ := p.app.completionPhase(c, task.ID)
	row := p.rowShell(c, t, task.ID, theme.RowOneLine, selected, opacity)
	row.Children(func() {
		p.app.glyphButton(c, t, task, fill)
		theme.Text(c, theme.BodyStrong, t.Text, task.Title).Grow(1).MinWidth(0).Ellipsis("…")
		if words, reminder, ok := timeWords(task, p.app.Now()); ok {
			ui.Row(c).Gap(4).Shrink(0).AlignItems(ui.Center).Children(func() {
				theme.Text(c, theme.Meta, t.TextSecondary, words)
				if reminder {
					theme.ClockGlyph(c, t.TextSecondary)
				}
			})
		}
	})
	if row.Clicked() {
		p.open(task.ID)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
