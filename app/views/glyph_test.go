package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/locales"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// TestScreensGlyphs renders every status glyph and the priority badges large, for the review.
func TestScreensGlyphs(t *testing.T) {
	for _, dark := range []bool{false, true} {
		suffix := "light"
		if dark {
			suffix = "dark"
		}
		view := func(c *ui.Context) {
			tk := theme.Resolve(c)
			ui.Column(c).Fill().Padding(20).Gap(14).Children(func() {
				ui.Row(c).Gap(18).AlignItems(ui.Center).Children(func() {
					for _, s := range data.Statuses {
						ui.Column(c.Key(string(s))).Gap(6).AlignItems(ui.Center).Children(func() {
							theme.StatusGlyph(c, tk, s, 0)
							theme.Text(c, theme.Meta, tk.TextSecondary, statusLabel(locales.In(""), s))
						})
					}
				})
				ui.Row(c).Gap(18).AlignItems(ui.Center).Children(func() {
					for _, p := range []data.Priority{data.P0, data.P1} {
						ui.Column(c.Key(string(p))).Gap(6).AlignItems(ui.Center).Children(func() {
							theme.PriorityBadge(c, tk, p)
							theme.Text(c, theme.Meta, tk.TextSecondary, "badge")
						})
					}
					for i, k := range []float32{0.25, 0.5, 0.75, 1} {
						ui.Column(c.Key(i + 10)).Gap(6).AlignItems(ui.Center).Children(func() {
							theme.StatusGlyph(c, tk, data.Doing, k)
							theme.Text(c, theme.Meta, tk.TextSecondary, "done "+itoa(int(k*100)))
						})
					}
				})
			})
		}
		tt := ui.NewTester(view, 560, 150)
		tt.SetScale(4)
		tt.SetDark(dark)
		still(tt)
		settle(tt)
		shot(t, "glyphs-"+suffix+".png", tt.Image(), false, dark)
	}
}
