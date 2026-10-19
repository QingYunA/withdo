package views

import (
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// manageState is the state of managing things by hand: the sheets and the confirmation that float
// over the window, the project being renamed in the sidebar, the search and Completed.
type manageState struct {
	project   projectSheet
	category  categorySheet
	milestone milestoneSheet
	confirm   confirmation
	rename    renaming
	search    searchState
	completed completedState
}

// managePanelWidth is the width of the small sheets.
const managePanelWidth float32 = 400

// sheet is a small sheet over the dimmed window, the look of Settings at 400 pt. Esc and a click
// outside close it.
func sheet(c *ui.Context, t *theme.Tokens, open *bool, body func()) {
	ui.DialogBase(c, open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.32))
		panel.Width(managePanelWidth).Padding(24).Radius(theme.RadiusPanel).Background(t.Bg).Border(1, t.Border).
			Shadow(0, 20, 48, 0, t.Shadow)
		ui.Column(c).Gap(16).Children(body)
	})
}

// wellInput is a text input in a 30 pt well on bg.subtle, with a placeholder. onKey is its keyboard.
// It returns the well, for the day card that draws it in pixels.
func wellInput(c *ui.Context, t *theme.Tokens, key string, value *string, label, placeholder string, style theme.Style, focus bool, onKey func(ui.TextKey) bool) ui.Element {
	return ui.Row(c.Key(key+"-well")).Height(30).Shrink(0).Padding(0, 10).Radius(theme.RadiusControl).Background(t.BgSubtle).
		AlignItems(ui.Center).Children(func() {
		box := ui.Box(c).Grow(1).MinWidth(0)
		box.Children(func() {
			in := style.Apply(ui.TextInputBase(c.Key(key), value)).TextColor(t.Text).Label(label).OnTextKey(onKey)
			if focus {
				in.AutoFocus()
			}
			if *value == "" {
				theme.Text(c, style, t.TextTertiary, placeholder).Absolute().Left(0).Top(0).Right(0).Ellipsis("…").PassThrough()
			}
		})
	})
}

// labeled puts a label in meta above a control.
func labeled(c *ui.Context, t *theme.Tokens, label string, control func()) {
	ui.Column(c).Gap(6).Children(func() {
		theme.Text(c, theme.Meta, t.TextSecondary, label)
		control()
	})
}

// sheetButtons is the row at the foot of a sheet: Cancel and the primary action at the right.
func sheetButtons(c *ui.Context, cancel, primary string, enabled bool, focusCancel bool) (cancelled, chosen bool) {
	ui.Row(c).Gap(8).Justify(ui.End).Margin(4, 0, 0).Children(func() {
		b := ui.Button(c.Key("sheet-cancel"), cancel)
		if focusCancel {
			b.AutoFocus()
		}
		cancelled = b.Clicked()
		chosen = ui.PrimaryButton(c.Key("sheet-primary"), primary).Disabled(!enabled).Clicked()
	})
	return cancelled, chosen
}
