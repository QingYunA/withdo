package theme

import "time"

// The day card of the calendar pops out of its cell: it is laid out at its full size and grows from
// the cell's corner while it fades in, an ease-out over a short time. Like the hop it is a pure
// function of the time since it opened, and nothing runs once it is done.
const (
	// DayCardDuration is from opening to done: the card is full size from DayCardDuration and asks
	// for no more frames then.
	DayCardDuration time.Duration = 140 * time.Millisecond
	// DayCardFade is how long the card takes to become opaque, within DayCardDuration.
	DayCardFade time.Duration = 100 * time.Millisecond
)

// DayCardAt is how far the day card has grown elapsed after it opened, from 0 to 1 along an ease-out
// cubic, its opacity, from 0 to 1 along an ease-out over DayCardFade, and whether the opening is
// over. A negative elapsed is 0.
func DayCardAt(elapsed time.Duration) (progress, opacity float32, done bool) {
	if elapsed >= DayCardDuration {
		return 1, 1, true
	}
	elapsed = max(elapsed, 0)
	return easeOut(float32(elapsed) / float32(DayCardDuration)), easeOut(min(float32(elapsed)/float32(DayCardFade), 1)), false
}

// easeOut is the ease-out cubic of t in 0..1.
func easeOut(t float32) float32 {
	u := 1 - t
	return 1 - u*u*u
}
