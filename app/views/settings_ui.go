package views

import (
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The grouped list of a settings window: a rounded block of rows with hairlines between them, an
// optional label above it, the label and its quiet line at the left of a row and the control at
// the right.

// groupLabel is the small title above a group.
func groupLabel(c *ui.Context, t *theme.Tokens, text string) ui.Element {
	return theme.Text(c, theme.Section, t.TextSecondary, text).Margin(4, 4, 7)
}

// group builds a block of rows. Each row is added with settingsRow.
func group(c *ui.Context, t *theme.Tokens, key string, rows func()) {
	ui.Column(c.Key("group-"+key)).Shrink(0).Margin(0, 0, 18).Radius(theme.SettingsRadius).Background(t.SettingsGroup()).
		Border(1, t.Border.Alpha(0.08)).Children(rows)
}

// settingsRow is one row of a group: a label with an optional second line at the left, and what
// control builds at the right. A row after the first gets the hairline above it.
func settingsRow(c *ui.Context, t *theme.Tokens, key, label, sub string, first bool, control func()) {
	settingsRowLead(c, t, key, label, sub, first, control, nil)
}

// settingsRowLead is settingsRow with something, such as an avatar or a logo, before the label.
func settingsRowLead(c *ui.Context, t *theme.Tokens, key, label, sub string, first bool, control, lead func()) {
	settingsRowInk(c, t, key, label, sub, t.TextSecondary, first, control, lead)
}

// settingsRowInk is settingsRowLead with the second line in a color of its own, for a warning.
func settingsRowInk(c *ui.Context, t *theme.Tokens, key, label, sub string, subInk ui.Color, first bool, control, lead func()) {
	if !first {
		ui.Box(c.Key("rule-"+key)).Height(1).Shrink(0).Margin(0, 14).Background(t.Border)
	}
	ui.Row(c.Key("row-"+key)).MinHeight(theme.SettingsRowMin).Shrink(0).Padding(8, 14).Gap(12).AlignItems(ui.Center).Children(func() {
		if lead != nil {
			lead()
		}
		ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
			if label != "" {
				theme.Text(c, theme.Body, t.Text, label).Ellipsis("…")
			}
			if sub != "" {
				theme.Paragraph(c, theme.Meta, subInk, sub, 2)
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
	return popUpIn(c, t, key, current, opts, false)
}

// popUpFill is a pop-up as wide as the room it has, the label at the left and the chevron at the
// right, like a text field.
func popUpFill(c *ui.Context, t *theme.Tokens, key, current string, opts []popOption) (string, bool) {
	return popUpIn(c, t, key, current, opts, true)
}

func popUpIn(c *ui.Context, t *theme.Tokens, key, current string, opts []popOption, fill bool) (string, bool) {
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
	if fill {
		parts.Trigger.Grow(1).MinWidth(0).Justify(ui.SpaceBetween).Height(28)
	}
	parts.Trigger.Children(func() {
		theme.Text(c, theme.Body, t.Text, label).Ellipsis("…")
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
// A width of 0 leaves the width to the row it sits in.
func fieldWell(c *ui.Context, t *theme.Tokens, key string, width float32, body func()) ui.Element {
	well := ui.Row(c.Key("well-"+key)).Height(28).Shrink(0).Padding(0, 10).AlignItems(ui.Center).
		Radius(theme.RadiusControl).Background(t.Bg).Border(1, t.ComposerBorder())
	if width > 0 {
		well.Width(width)
	}
	return well.Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Children(body)
	})
}

// draftSpec describes a field whose text is kept by the caller until it is used.
type draftSpec struct {
	name        string // the key of the field, whatever the language
	label       string // what assistive technology calls it
	placeholder string
	value       string // the text it starts with
	secret      bool
	focus       bool // takes the keyboard as it appears, once
	focusNow    bool // takes the keyboard even if it was already there
	disabled    bool
}

// draftField is a text field of a form: it keeps its text in buf and tells whether Enter was
// pressed in it and whether it lost the keyboard. A field that was never typed in shows value.
func draftField(c *ui.Context, t *theme.Tokens, fs fields, s draftSpec, buf *string) (entered, left bool) {
	f := fs.get(s.name)
	var el ui.Element
	ui.Box(c.Key("box-" + s.name)).MinWidth(0).Children(func() {
		el = ui.TextInputBase(c.Key(s.name), buf)
		theme.Body.Apply(el).TextColor(t.Text).Label(s.label).OnTextKey(func(k ui.TextKey) bool {
			if k.Mods == 0 && k.Key == ui.KeyEnter {
				*buf, f.enter = k.Text(), true
				return true
			}
			return false
		})
		if s.secret {
			el.Password()
		}
		if s.disabled {
			el.Disabled(true).ReadOnly(true)
		}
		if s.focus {
			el.AutoFocus()
		}
		if s.focusNow {
			el.Focus()
		}
		if *buf == "" {
			shown := s.placeholder
			if shown == "" {
				shown = s.value
			}
			theme.Text(c, theme.Body, t.TextTertiary, shown).Absolute().Left(0).Top(0).PassThrough()
		}
	})
	focused := el.Focused()
	left = f.editing && !focused
	if focused {
		f.editing = true
	}
	if left {
		f.editing = false
	}
	entered, f.enter = f.enter, false
	return entered, left
}

// statusDot is the small colored dot of a status line.
func statusDot(c *ui.Context, color ui.Color) ui.Element { return theme.Dot(c, color, 7).Shrink(0) }
