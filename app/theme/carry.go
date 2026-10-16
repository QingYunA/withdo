package theme

import (
	"math"
	"time"
)

// Handing a task over is a drag, and the sprites play along. The user's sprite carries the lifted
// row and walks while the pointer moves; the agent's sprite in the Agents header hops up and reaches
// while the row is over the tray, and squashes once as it catches the row on the drop. Everything is
// a function of the time on the app's clock, so nothing runs while it is still.
const (
	CarryStepsPerSecond         = 4                      // the walking sprite's steps
	CarryBobHeight      float32 = 2                      // how far up a step lifts it, in points
	CarryStopsAfter             = 120 * time.Millisecond // the sprite stands still this long after the pointer did
	ReachEvery                  = 800 * time.Millisecond // the agent hops again this often while the row is over the tray
	WashFor                     = 2 * time.Second        // the new row's accent wash fades over this
	WashMax             float32 = 0.08                   // and starts at this strength
)

// CarryBob is how far the walking sprite is lifted, elapsed after it started walking: a small bounce
// with CarryStepsPerSecond steps a second.
func CarryBob(elapsed time.Duration) float32 {
	if elapsed < 0 {
		return 0
	}
	return CarryBobHeight * float32(math.Abs(math.Sin(math.Pi*CarryStepsPerSecond*elapsed.Seconds())))
}

// Reach is one short hop of the agent's sprite on the spot, (x, y) being its top left corner at rest,
// a time hovering after the row came over the tray; it repeats every ReachEvery. The frame is HopAt's
// with the same start and landing, so its top is HopPeak above the rest.
func Reach(hovering time.Duration, x, y float32) HopFrame {
	if hovering < 0 {
		hovering = 0
	}
	f, ok := HopAt(hovering%ReachEvery, x, y, x, y)
	if !ok {
		return HopFrame{X: x, Y: y, W: HopSize, H: HopSize}
	}
	return f
}

// CatchFor is how long the squash of a catch lasts.
const CatchFor = HopDuration - HopLanding

// Catch is the agent's squash as the row is dropped on it, elapsed after the drop: the landing of the
// hop, wider and shorter with its feet on the ground, then settled. ok is false once it is over.
func Catch(elapsed time.Duration, x, y float32) (HopFrame, bool) {
	if elapsed < 0 {
		return HopFrame{}, false
	}
	return HopAt(HopLanding+elapsed, x, y, x, y)
}

// Wash is the strength of the accent wash of a row that landed in Waiting for an agent, elapsed after
// the drop: WashMax, fading to nothing over WashFor.
func Wash(elapsed time.Duration) float32 {
	if elapsed < 0 || elapsed >= WashFor {
		return 0
	}
	return WashMax * (1 - float32(elapsed)/float32(WashFor))
}
