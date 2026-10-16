package theme

import "time"

// The day card of the calendar opens one pixel step at a time: four discrete sizes, no tween, so it
// grows the way the pixel icons are drawn. Like the hop it is a pure function of the time since it
// opened, and nothing runs once it is full size.
const (
	// DayCardSteps is how many sizes the card shows while it opens, the last being its full size.
	DayCardSteps = 4
	// DayCardDuration is from opening to done: the full size is held from 150 ms, and the card asks
	// for no more frames at DayCardDuration.
	DayCardDuration time.Duration = 200 * time.Millisecond

	dayCardStep time.Duration = 50 * time.Millisecond // how long each of the first three sizes lasts
)

// dayCardScales are the sizes of the card, as a share of its full width and height.
var dayCardScales = [DayCardSteps]float32{0.55, 0.7, 0.85, 1}

// DayCardAt is the size of the day card elapsed after it opened, as a share of its full size, and
// whether the opening is over. It steps 0.55, 0.7, 0.85 every 50 ms and is 1 from 150 ms.
func DayCardAt(elapsed time.Duration) (scale float32, done bool) {
	if elapsed >= DayCardDuration {
		return 1, true
	}
	return dayCardScales[min(max(int(elapsed/dayCardStep), 0), DayCardSteps-1)], false
}
