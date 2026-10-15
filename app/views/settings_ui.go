package views

import (
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The grouped list of a settings window: a rounded block of rows with hairlines between them, an
// optional label above it, the label and its quiet line at the left of a row and the control at
// the right.

// groupLabel is the small title above a group.
func groupLabel(c *ui.Context, t *theme.Tokens, text string) {
	theme.Text(c, theme.Section, t.TextSecondary, text).Margin(4, 4, 7)
}

// group builds a block of rows. Each row is added with settingsRow.
func group(c *ui.Context, t *theme.Tokens, key string, rows func()) {
	ui.Column(c.Key("group-"+key)).Shrink(0).Margin(0, 0, 18).Radius(theme.SettingsRadius).Background(t.SettingsGroup()).
		Border(1, t.Border.Alpha(0.08)).Children(rows)
}

// settingsRow is one row of a group: a label with an optional second line at the left, and what
// control builds at the right. A row after the first gets the hairline above it.
func settingsRow(c *ui.Context, t *theme.Tokens, key, label, sub string, first bool, control func()) {
	if !first {
		ui.Box(c.Key("rule-"+key)).Height(1).Shrink(0).Margin(0, 14).Background(t.Border)
	}
	ui.Row(c.Key("row-"+key)).MinHeight(theme.SettingsRowMin).Shrink(0).Padding(8, 14).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
			if label != "" {
				theme.Text(c, theme.Body, t.Text, label).Ellipsis("…")
			}
			if sub != "" {
				theme.Paragraph(c, theme.Meta, t.TextSecondary, sub, 2)
			}
		})
		if control != nil {
			control()
		}
	})
}

// popOption is one choice of a pop-up: the value it chooses and the words it shows.
type popOption struct{ value, label string }

// popUp is a pop-up button: the chosen label and a chevron, opening the options below it. It
// returns the value chosen this frame, and false when nothing was.
func popUp(c *ui.Context, t *theme.Tokens, key, current string, opts []popOption) (string, bool) {
	sel := current
	parts := ui.SelectBase(c.Key("pop-"+key), &sel)
	label := current
	for _, o := range opts {
		if o.value == current {
			label = o.label
		}
	}
	parts.Trigger.Padding(3, 10, 3, 12).Gap(8).Radius(theme.RadiusControl).AlignItems(ui.Center).Shrink(0).
		Background(t.Bg).Border(1, t.ComposerBorder()).Shadow(0, 1, 1, 0, t.Shadow.Alpha(0.06)).Cursor(ui.CursorPointer).Label(label)
	parts.Trigger.Children(func() {
		theme.Text(c, theme.Body, t.Text, label)
		ui.Icon(c, theme.IconChevron).Size(12, 12).TextColor(t.TextSecondary)
	})
	parts.Popup(func(panel ui.Element) {
		panel.Margin(4, 0, 0).Padding(4).Radius(theme.RadiusCard).Background(t.Bg).Border(1, t.Border).Shadow(0, 8, 24, 0, t.Shadow)
		for _, o := range opts {
			item := parts.Item(o.value).Padding(5, 10).Radius(theme.RadiusControl).Gap(8).AlignItems(ui.Center)
			ink := t.Text
			if item.Highlighted() {
				item.Background(t.Accent)
				ink = t.AccentText
			}
			item.Children(func() { theme.Text(c, theme.Body, ink, o.label) })
		}
	})
	return sel, sel != current
}

// wellWidth is the width of a text field in a row.
const wellWidth float32 = 230

// fieldWell is the quiet outlined box a settings text field sits in.
func fieldWell(c *ui.Context, t *theme.Tokens, key string, width float32, body func()) {
	ui.Row(c.Key("well-"+key)).Width(width).Height(28).Shrink(0).Padding(0, 10).AlignItems(ui.Center).
		Radius(theme.RadiusControl).Background(t.Bg).Border(1, t.ComposerBorder()).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Children(body)
	})
}

// statusDot is the small colored dot of a status line.
func statusDot(c *ui.Context, color ui.Color) ui.Element { return theme.Dot(c, color, 7).Shrink(0) }
