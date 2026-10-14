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
	IconClose     = icon(`<path d="m4.5 4.5 7 7M11.5 4.5l-7 7"/>`)
)
