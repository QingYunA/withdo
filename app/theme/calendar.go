package theme

import "github.com/egoist/mygo/ui"

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

// The day card's metrics, in points.
const (
	DayCardMinWidth float32 = 280 // the card is at least this wide, or 1.6 cells when that is more
	DayCardMaxH     float32 = 420 // taller than this the list of items scrolls
	DayCardPad      float32 = 12  // inside the card
	DayCardHead     float32 = 28  // the date, the count and the close button
	DayCardRow      float32 = 28  // an item of the list
	DayCardInput    float32 = 30  // the well at the foot
	DayCardGap      float32 = 8   // between the header, the list and the input
	DayCardMargin   float32 = 8   // the least room between the card and the window's edge
	DayCardPlus     float32 = 20  // the hit target of the + in a hovered cell
)

// CalendarWeekend is the faint fill of a Saturday or a Sunday.
func (t *Tokens) CalendarWeekend() ui.Color { return t.Bg.Mix(t.BgSubtle, 0.7) }

// CalendarRingColor is the ring of the selected day.
func (t *Tokens) CalendarRingColor() ui.Color { return t.Accent.Alpha(0.6) }

// CalendarDropFill is a day's fill while an item is dragged over it.
func (t *Tokens) CalendarDropFill() ui.Color { return t.Bg.Mix(t.Accent, 0.1) }
