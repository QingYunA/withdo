package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// activity is the report timeline, newest first: a dot on a thin line, who and when, what they
// did, and what comes next.
func (m *Main) activity(c *ui.Context, t *theme.Tokens, task data.Task) {
	reports := m.app.snap.details[task.ID].Reports
	ui.Column(c).Gap(8).Children(func() {
		theme.Text(c, theme.Section, t.TextSecondary, "Activity")
		if len(reports) == 0 {
			theme.Paragraph(c, theme.Body, t.TextSecondary, "No reports yet. Agents file their progress here as they work.", 0)
			return
		}
		for i, r := range reports {
			m.reportRow(c, t, r, i == len(reports)-1, i == 0)
		}
	})
}

func (m *Main) reportRow(c *ui.Context, t *theme.Tokens, r data.Report, last, first bool) {
	ui.Row(c.Key(r.ID)).AlignItems(ui.Stretch).Gap(12).Children(func() {
		ui.Column(c).Width(6).Shrink(0).AlignItems(ui.Center).Children(func() {
			dot := t.TextTertiary
			if first {
				dot = t.Status(r.Status)
			}
			theme.Dot(c, dot, 6).Margin(6, 0, 0)
			if !last {
				ui.Box(c).Width(1).Grow(1).Margin(4, 0, 0).Background(t.Border)
			}
		})
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Padding(0, 0, 16).Children(func() {
			ui.Row(c).Gap(6).Children(func() {
				theme.Text(c, theme.Meta, t.TextSecondary, r.Agent)
				theme.Text(c, theme.Meta, t.TextTertiary, data.Age(r.CreatedAt, m.app.Now()))
			})
			theme.Paragraph(c, theme.Body, t.Text, r.Summary, 0)
			if r.NextStep != "" {
				theme.Paragraph(c, theme.Body, t.TextSecondary, "Next: "+r.NextStep, 0)
			}
		})
	})
}
