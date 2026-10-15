package theme

import (
	"math"
	"time"
)

// The hop: when an agent hands something over, its sprite jumps from the Agents row of the sidebar
// to the Mine row, lands by its count, squashes and settles. The whole hop is a pure function of
// the time since it began, so the view draws it from the clock it already has and nothing runs
// while it is still.
const (
	HopSize     float32       = 24                     // the sprite, in points
	HopDuration time.Duration = 900 * time.Millisecond // from taking off to settled
	HopStagger  time.Duration = 120 * time.Millisecond // between the hops of several items
	HopMax                    = 3                      // hops shown at once; the rest count at once
	HopPeak     float32       = 30                     // the arc's top, above the Mine row
	// HopLanding is when the sprite touches down, the rest of the hop being its squash and settle.
	HopLanding time.Duration = 648 * time.Millisecond

	hopSquash = 0.3 // how much wider and shorter the sprite is as it lands
)

// HopFrame is one moment of a hop: where the sprite is drawn and how big.
type HopFrame struct {
	X, Y, W, H float32
	// Landed is true from the moment the sprite touches down: the badge counts up then.
	Landed bool
}

// HopAt is the frame of a hop elapsed after it began, from the sprite's box at the Agents row to its
// box at the Mine row (their top left corners). ok is false before the hop begins and after it is
// over.
func HopAt(elapsed time.Duration, fromX, fromY, toX, toY float32) (f HopFrame, ok bool) {
	if elapsed < 0 || elapsed >= HopDuration {
		return HopFrame{}, false
	}
	flight := HopLanding
	if elapsed < flight {
		s := float32(elapsed) / float32(flight)
		// A parabola that starts at the Agents row, peaks HopPeak above the Mine row and comes down
		// on it. rise is how far the straight line from start to finish is lifted at the middle.
		rise := HopPeak + (fromY-toY)/2
		return HopFrame{
			X: fromX + (toX-fromX)*s,
			Y: fromY + (toY-fromY)*s - 4*rise*s*(1-s),
			W: HopSize, H: HopSize,
		}, true
	}
	u := float32(elapsed-flight) / float32(HopDuration-flight)
	k := float32(hopSquash * (1 - float64(u)) * math.Cos(float64(u)*math.Pi*1.5))
	w, h := HopSize*(1+k), HopSize*(1-k)
	// The feet stay on the row: the box grows sideways around its middle and shrinks from the top.
	return HopFrame{X: toX - (w-HopSize)/2, Y: toY + (HopSize - h), W: w, H: h, Landed: true}, true
}
