// Package theme holds every design token of the app in one place: color for light and dark, type,
// space, shape and motion, plus the vector glyphs. Views never write a color or a size of their
// own; they ask for a token.
package theme

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// Tokens are the colors of one appearance.
type Tokens struct {
	Dark bool

	Bg            ui.Color // window content
	BgSidebar     ui.Color // the sidebar (slate 2)
	BgSubtle      ui.Color // wells: search, finished cards, the idle send button (slate 3)
	BgTray        ui.Color // the Agents tray (slate 2)
	BgHover       ui.Color // row hover (slate 3)
	BgSelected    ui.Color // selected row, keyboard highlight (slate 4, neutral)
	Border        ui.Color // hairlines, always Hairline wide (slate 6)
	BorderControl ui.Color // the composer's edge, controls (slate 7)
	Text          ui.Color
	TextSecondary ui.Color
	TextTertiary  ui.Color
	Accent        ui.Color // iris 9
	AccentText    ui.Color
	AccentSoft    ui.Color // iris 3: tinted buttons, a just-added row, the focus ring
	AccentInk     ui.Color // iris 11: links, tinted button text, Undo
	AccentEdge    ui.Color // iris 8: the focus edge, the drop target's edge
	AccentWell    ui.Color // iris 2: the drop target's well
	Stale         ui.Color
	Shadow        ui.Color // floating things only
	Merged        ui.Color // a closed issue, as GitHub paints it

	status map[data.Status]ui.Color
}

// Status returns the muted color of a status.
func (t *Tokens) Status(s data.Status) ui.Color {
	if c, ok := t.status[s]; ok {
		return c
	}
	return t.TextTertiary
}

// Blocked is the only red in the app.
func (t *Tokens) Blocked() ui.Color { return t.Status(data.Blocked) }

var light = &Tokens{
	Bg:            ui.Hex("#FFFFFF"),
	BgSidebar:     ui.Hex("#F9F9FB"),
	BgSubtle:      ui.Hex("#F0F0F3"),
	BgTray:        ui.Hex("#F9F9FB"),
	BgHover:       ui.Hex("#F0F0F3"),
	BgSelected:    ui.Hex("#E8E8EC"),
	Border:        ui.Hex("#D9D9E0"),
	BorderControl: ui.Hex("#CDCED6"),
	Text:          ui.Hex("#1C2024"),
	TextSecondary: ui.Hex("#60646C"),
	TextTertiary:  ui.Hex("#8B8D98"),
	Accent:        ui.Hex("#5B5BD6"),
	AccentText:    ui.Hex("#FFFFFF"),
	AccentSoft:    ui.Hex("#F0F1FE"),
	AccentInk:     ui.Hex("#5753C6"),
	AccentEdge:    ui.Hex("#9B9EF0"),
	AccentWell:    ui.Hex("#F8F8FF"),
	Stale:         ui.Hex("#CC4E00"),
	Shadow:        ui.RGBA(0, 0, 0, 0.26),
	Merged:        ui.Hex("#8E4EC6"),
	status: map[data.Status]ui.Color{
		data.Backlog:   ui.Hex("#B9BBC6"),
		data.Todo:      ui.Hex("#8B8D98"),
		data.Doing:     ui.Hex("#0090FF"),
		data.Blocked:   ui.Hex("#E5484D"),
		data.Review:    ui.Hex("#FFC53D"),
		data.Done:      ui.Hex("#30A46C"),
		data.Cancelled: ui.Hex("#B9BBC6"),
	},
}

var dark = &Tokens{
	Dark:          true,
	Bg:            ui.Hex("#111113"),
	BgSidebar:     ui.Hex("#18191B"),
	BgSubtle:      ui.Hex("#212225"),
	BgTray:        ui.Hex("#18191B"),
	BgHover:       ui.Hex("#212225"),
	BgSelected:    ui.Hex("#272A2D"),
	Border:        ui.Hex("#363A3F"),
	BorderControl: ui.Hex("#43484E"),
	Text:          ui.Hex("#EDEEF0"),
	TextSecondary: ui.Hex("#B0B4BA"),
	TextTertiary:  ui.Hex("#696E77"),
	Accent:        ui.Hex("#5B5BD6"),
	AccentText:    ui.Hex("#FFFFFF"),
	AccentSoft:    ui.Hex("#202248"),
	AccentInk:     ui.Hex("#B1A9FF"),
	AccentEdge:    ui.Hex("#5958B1"),
	AccentWell:    ui.Hex("#171625"),
	Stale:         ui.Hex("#FF801F"),
	Shadow:        ui.RGBA(0, 0, 0, 0.5),
	Merged:        ui.Hex("#B388EB"),
	status: map[data.Status]ui.Color{
		data.Backlog:   ui.Hex("#5A6169"),
		data.Todo:      ui.Hex("#696E77"),
		data.Doing:     ui.Hex("#0090FF"),
		data.Blocked:   ui.Hex("#E5484D"),
		data.Review:    ui.Hex("#FFC53D"),
		data.Done:      ui.Hex("#30A46C"),
		data.Cancelled: ui.Hex("#5A6169"),
	},
}

// For returns the tokens of an appearance. accent, when it has alpha, replaces the default
// accent: the system accent the user chose.
func For(isDark bool, accent ui.Color) *Tokens {
	base := light
	if isDark {
		base = dark
	}
	t := *base
	if accent.A > 0 {
		t.Accent = accent
		t.AccentText = ui.Hex("#FFFFFF")
	}
	return &t
}

// Resolve returns the tokens for the frame and makes the window's widget theme follow them. It
// uses the system accent only when the user changed it from the macOS default blue.
func Resolve(c *ui.Context) *Tokens {
	sys := c.Preferences().Accent
	if isDefaultSystemAccent(sys) {
		sys = ui.Transparent
	}
	t := For(c.Theme().Dark, sys)
	c.SetTheme(t.widgetTheme())
	return t
}

// isDefaultSystemAccent recognizes macOS's default blue in both appearances.
func isDefaultSystemAccent(c ui.Color) bool {
	if c.A == 0 {
		return true
	}
	near := func(a, b uint8) bool { return int(a)+6 >= int(b) && int(b)+6 >= int(a) }
	for _, d := range [][3]uint8{{0, 122, 255}, {10, 132, 255}, {0, 125, 255}} {
		if near(c.R, d[0]) && near(c.G, d[1]) && near(c.B, d[2]) {
			return true
		}
	}
	return false
}

// widgetTheme maps the tokens onto mygo's theme so the controls (text inputs, menus, scroll bars)
// match the views drawn by hand.
func (t *Tokens) widgetTheme() *ui.Theme {
	w := ui.LightTheme()
	if t.Dark {
		w = ui.DarkTheme()
	}
	w.Background, w.Surface = t.Bg, t.BgSubtle
	w.SurfaceHover = t.BgSubtle.Mix(t.Text, 0.06)
	w.SurfacePressed = t.BgSubtle.Mix(t.Text, 0.12)
	w.Border = t.Border
	w.Text, w.TextMuted = t.Text, t.TextSecondary
	w.Accent, w.AccentText = t.Accent, t.AccentText
	w.AccentHover, w.AccentPressed = t.Accent.Mix(t.Text, 0.12), t.Accent.Mix(t.Text, 0.24)
	w.Danger, w.Warning, w.Success = t.Blocked(), t.Stale, t.Status(data.Done)
	w.Selection, w.Focus = t.Accent.Alpha(0.3), t.Accent.Alpha(0.55)
	w.Scrollbar = t.Text.Alpha(0.28)
	w.Radius, w.Spacing, w.FontSize = RadiusControl, 4, Body.Size
	return w
}

// Hairline is the width of every border, divider and hairline (Craft rule 6). mygo takes a float
// width, so the half point is drawn as is.
const Hairline float32 = 0.5

// Radii in points.
const (
	RadiusControl float32 = 6  // pills, inputs
	RadiusCard    float32 = 9  // cards
	RadiusRow     float32 = 8  // rows (v34)
	RadiusPanel   float32 = 12 // composer, From your agents container, popovers, sheets
	RadiusTile    float32 = 10 // board cards
)

// Space on the 4 pt grid.
const (
	Space4  float32 = 4
	Space8  float32 = 8
	Space12 float32 = 12
	Space16 float32 = 16
	Space24 float32 = 24
)

// Motion durations.
const (
	MotionFast   = 160 * time.Millisecond // hover, selection, drawer
	MotionSpring = 220 * time.Millisecond // card settling, completing
)

// Layout metrics shared by several surfaces.
const (
	GlyphSize      float32 = 18
	BadgeSize      float32 = 18      // the red count beside Mine
	IconProp       float32 = IconRow // the icon before a property's label in the inspector
	IconMenu       float32 = IconRow // the icon before an item of the drawn task menu
	HitTarget      float32 = 24
	RowOneLine     float32 = 32
	SidebarWidth   float32 = 216
	InspectorWidth float32 = 360
	SidebarRow     float32 = 32  // a row of the sidebar, in the 14 pt sidebar size
	ContentMax     float32 = 780 // the centered column of lists
	ContentPad     float32 = 48  // its side padding when there is room
	AskHeight      float32 = 46
	BoardMax       float32 = 980 // the board's block (header, composer and lanes) at its widest
	MinColumnWidth float32 = 176 // a board lane at its narrowest, before the board scrolls
	ColumnGap      float32 = 20

	WindowMinWidth  = 860
	WindowMinHeight = 540
	WindowWidth     = 1100
	WindowHeight    = 720
)
