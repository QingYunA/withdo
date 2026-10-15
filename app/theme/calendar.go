package theme

import "github.com/egoist/mygo/ui"

// IconForward is the chevron pointing right, the pair of IconBack.
var IconForward = icon(`<path d="M6 3.5 10.5 8 6 12.5"/>`)

// The calendar's metrics, in points.
const (
	CalendarMax      float32 = 1240 // the month grid at its widest, centered
	CalendarMinCell  float32 = 76   // a cell never gets shorter; the page scrolls instead
	CalendarPadX     float32 = 6    // inside a cell
	CalendarPadTop   float32 = 5
	CalendarDayLine  float32 = 22 // the row of the day number: its pill is 20 pt tall
	CalendarChip     float32 = 20 // an item of a cell
	CalendarChipGap  float32 = 2
	CalendarHeadLine float32 = 28 // the weekday names above the grid
	CalendarMaxItems         = 3  // items a cell lists before "N more"
	CalendarRing     float32 = 2  // the ring of the selected day
)

// CalendarWeekend is the faint fill of a Saturday or a Sunday.
func (t *Tokens) CalendarWeekend() ui.Color { return t.Bg.Mix(t.BgSubtle, 0.7) }

// CalendarRingColor is the ring of the selected day.
func (t *Tokens) CalendarRingColor() ui.Color { return t.Accent.Alpha(0.6) }

// CalendarDropFill is a day's fill while an item is dragged over it.
func (t *Tokens) CalendarDropFill() ui.Color { return t.Bg.Mix(t.Accent, 0.1) }
