package theme

import "github.com/egoist/mygo/ui"

func icon(shapes string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round">` + shapes + `</svg>`))
}

// Line icons in the color of the surrounding text.
var (
	IconInbox     = icon(`<path d="M2.5 9.5 4.4 3.9a1 1 0 0 1 .95-.65h5.3a1 1 0 0 1 .95.65L13.5 9.5v2.6a1 1 0 0 1-1 1h-9a1 1 0 0 1-1-1z"/><path d="M2.5 9.5h3.2l.6 1.4h3.4l.6-1.4h3.2"/>`)
	IconGear      = icon(`<circle cx="8" cy="8" r="2.1"/><path d="M8 1.9v1.5M8 12.6v1.5M13.4 5l-1.3.75M3.9 10.25 2.6 11M13.4 11l-1.3-.75M3.9 5.75 2.6 5"/>`)
	IconChevron   = icon(`<path d="m4.5 6.5 3.5 3.5 3.5-3.5"/>`)
	IconArrowOut  = icon(`<path d="M6 3.5H3.8a1.3 1.3 0 0 0-1.3 1.3v7.4a1.3 1.3 0 0 0 1.3 1.3h7.4a1.3 1.3 0 0 0 1.3-1.3V10M9 2.5h4.5V7M13.2 2.8 7.4 8.6"/>`)
	IconCalendar  = icon(`<rect x="2.5" y="3.5" width="11" height="10" rx="2"/><path d="M2.5 6.8h11M5.5 2v2.4M10.5 2v2.4"/>`)
	IconCheck     = icon(`<path d="m3.5 8.4 3 3 6-6.6"/>`)
	IconCopy      = icon(`<rect x="5.5" y="5.5" width="8" height="8" rx="1.8"/><path d="M10.5 5.5V4.3a1.8 1.8 0 0 0-1.8-1.8H4.3a1.8 1.8 0 0 0-1.8 1.8v4.4a1.8 1.8 0 0 0 1.8 1.8h1.2"/>`)
	IconMilestone = icon(`<path d="M4 13.5v-11M4 3h7.5l-1.8 2.8 1.8 2.8H4"/>`)
	IconPlus      = icon(`<path d="M8 3.5v9M3.5 8h9"/>`)
	IconSparkle   = icon(`<path d="M8 2.2c.4 2.9 1.4 4.4 5.8 5.8-4.4 1.4-5.4 2.9-5.8 5.8-.4-2.9-1.4-4.4-5.8-5.8C6.6 6.6 7.6 5.1 8 2.2z" fill="currentColor" stroke-width="1"/>`)
	IconBell      = icon(`<path d="M8 2.5a3.6 3.6 0 0 0-3.6 3.6v2.6L3.2 10.6h9.6l-1.2-1.9V6.1A3.6 3.6 0 0 0 8 2.5zM6.6 12.2a1.4 1.4 0 0 0 2.8 0"/>`)
	IconNow       = icon(`<circle cx="8" cy="8" r="5.6"/><circle cx="8" cy="8" r="1.9" fill="currentColor" stroke="none"/>`)
	IconClose     = icon(`<path d="m4.5 4.5 7 7M11.5 4.5l-7 7"/>`)
	IconSearch    = icon(`<circle cx="7" cy="7" r="4.2"/><path d="m10.2 10.2 3.3 3.3"/>`)
	IconMove      = icon(`<path d="M3 8h10M9 4l4 4-4 4"/>`)
	IconClock     = icon(`<circle cx="8" cy="8" r="5.6"/><path d="M8 4.8V8l2.2 1.4"/>`)
	IconTrash     = icon(`<path d="M3 4.5h10M6.4 4.5V3h3.2v1.5M4.4 4.5l.6 8.2h6l.6-8.2M6.9 7v3.6M9.1 7v3.6"/>`)
	IconList      = icon(`<path d="M3 4.5h10M3 8h10M3 11.5h10"/>`)
	IconBoard     = icon(`<rect x="2.5" y="3" width="11" height="10" rx="2"/><path d="M6.2 3v10M9.8 3v10"/>`)
	IconBack      = icon(`<path d="M10 3.5 5.5 8l4.5 4.5"/>`)
)

// Icons of the v4 round (docs/design/mockups/v4.html): the sidebar, the inspector's properties
// and the Settings toolbar.
var (
	IconDone     = icon(`<circle cx="8" cy="8" r="5.6"/><path d="m5.6 8.2 1.7 1.7 3.2-3.4"/>`)
	IconStatus   = icon(`<circle cx="8" cy="8" r="5.6"/><path d="M8 2.4a5.6 5.6 0 0 1 0 11.2z" fill="currentColor" stroke="none"/>`)
	IconAgent    = icon(`<rect x="3" y="4.5" width="10" height="8" rx="2"/><path d="M8 2.2v2.3M6 8.2v.6M10 8.2v.6"/>`)
	IconBranch   = icon(`<circle cx="5" cy="3.8" r="1.4"/><circle cx="5" cy="12.2" r="1.4"/><circle cx="11" cy="5.8" r="1.4"/><path d="M5 5.2v5.6M11 7.2c0 2.4-2 3-6 3.4"/>`)
	IconGeneral  = icon(`<path d="M3 4.5h6M11.5 4.5H13M3 11.5h1.5M7 11.5h6M3 8h10"/><circle cx="10.2" cy="4.5" r="1.4"/><circle cx="5.8" cy="11.5" r="1.4"/>`)
	IconTerminal = icon(`<rect x="2" y="3" width="12" height="10" rx="2"/><path d="m4.8 6.5 2 1.8-2 1.8M8.6 10.3h2.6"/>`)
	IconStar     = icon(`<path d="M8 2.2c.4 2.9 1.4 4.4 5.8 5.8-4.4 1.4-5.4 2.9-5.8 5.8-.4-2.9-1.4-4.4-5.8-5.8C6.6 6.6 7.6 5.1 8 2.2z"/>`)
	IconOctocat  = icon(`<path d="M8 2.2a5.8 5.8 0 0 0-1.8 11.3c.3 0 .4-.1.4-.3v-1c-1.6.3-2-.7-2-.7-.3-.7-.7-.9-.7-.9-.5-.4 0-.4 0-.4.6 0 .9.6.9.6.5.9 1.4.7 1.8.5 0-.4.2-.7.4-.8-1.3-.2-2.6-.7-2.6-2.9 0-.6.2-1.2.6-1.6-.1-.1-.3-.7 0-1.6 0 0 .5-.1 1.6.6a5.4 5.4 0 0 1 2.8 0c1.1-.7 1.6-.6 1.6-.6.3.9.1 1.5 0 1.6.4.4.6 1 .6 1.6 0 2.2-1.3 2.7-2.6 2.9.2.2.4.5.4 1v1.6c0 .2.1.3.4.3A5.8 5.8 0 0 0 8 2.2z"/>`)
	IconFlag     = IconMilestone
)

// ProjectColors tint a project's folder; the color follows the project's name.
var ProjectColors = []ui.Color{
	ui.Hex("#5B5BD6"), ui.Hex("#E0A800"), ui.Hex("#30A46C"), ui.Hex("#E5484D"),
	ui.Hex("#3B82F6"), ui.Hex("#D9488B"), ui.Hex("#0EA5A4"), ui.Hex("#E08A1E"),
}

// ProjectColor is the color of a project's folder in the sidebar.
func ProjectColor(name string) ui.Color {
	var h uint32 = 2166136261
	for _, b := range []byte(name) {
		h = (h ^ uint32(b)) * 16777619
	}
	return ProjectColors[h%uint32(len(ProjectColors))]
}
