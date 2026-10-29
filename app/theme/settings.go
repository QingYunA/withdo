package theme

import "github.com/egoist/mygo/ui"

// The Settings window is a macOS settings window: a sidebar of panes beside grouped lists on a gray
// pane (docs/design/ui-spec.md, Settings). Its three surfaces are one step apart in each
// appearance.
const (
	SettingsWidth float32 = 800
	// SettingsHeight is the window's height. It never changes: a pane that needs more scrolls.
	SettingsHeight  float32 = 580
	SettingsSide    float32 = 196 // the sidebar
	SettingsSideRow float32 = 34  // a pane row of the sidebar
	SettingsSidePad float32 = 10  // between the sidebar's edge and a row
	ProviderTileH   float32 = 76  // a tile of the provider grid
	SettingsRowMin  float32 = 44
	SettingsRadius  float32 = 10 // a group
	SettingsPad     float32 = 28 // between the pane's edge and a group
)

// SettingsTitle is the header of a pane: its name over the one line that says what it is for.
var SettingsTitle = Style{Size: 22, Line: 28, Weight: 700}

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

// SettingsToolbar is the sidebar's fill, behind the traffic lights and the pane rows.
func (t *Tokens) SettingsToolbar() ui.Color {
	if t.Dark {
		return ui.Hex("#232326")
	}
	return ui.Hex("#ECECF0")
}

// SettingsTabOn is the well behind the open pane.
func (t *Tokens) SettingsTabOn() ui.Color {
	if t.Dark {
		return ui.RGBA(255, 255, 255, 0.1)
	}
	return ui.RGBA(0, 0, 0, 0.06)
}
