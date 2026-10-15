package theme

import "github.com/egoist/mygo/ui"

// The Settings window is a macOS settings window: a toolbar of panes over grouped lists on a gray
// pane (docs/design/ui-spec.md, Settings). Its three surfaces are one step apart in each
// appearance.
const (
	SettingsWidth  float32 = 680
	SettingsTab    float32 = 78 // a toolbar pane, icon over label
	SettingsRowMin float32 = 44
	SettingsRadius float32 = 10 // a group
	SettingsPad    float32 = 24 // between the window's edge and a group
)

// SettingsPane is the gray the groups sit on.
func (t *Tokens) SettingsPane() ui.Color {
	if t.Dark {
		return ui.Hex("#1C1C1E")
	}
	return ui.Hex("#F5F5F7")
}

// SettingsGroup is a grouped list: white, or #2C2C2E in dark.
func (t *Tokens) SettingsGroup() ui.Color {
	if t.Dark {
		return ui.Hex("#2C2C2E")
	}
	return ui.Hex("#FFFFFF")
}

// SettingsToolbar is the band of panes under the title.
func (t *Tokens) SettingsToolbar() ui.Color {
	if t.Dark {
		return ui.Hex("#252528")
	}
	return ui.Hex("#FAFAFA")
}

// SettingsTabOn is the well behind the open pane.
func (t *Tokens) SettingsTabOn() ui.Color {
	if t.Dark {
		return ui.RGBA(255, 255, 255, 0.1)
	}
	return ui.RGBA(0, 0, 0, 0.06)
}
